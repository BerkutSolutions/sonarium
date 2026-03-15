import { API } from './api.js';
import { t } from './i18n.js';

const UPLOAD_CONCURRENCY_KEY = 'sonarium.uploadConcurrency';
const UPLOAD_HISTORY_KEY = 'sonarium.uploadHistory';
const UPLOAD_HISTORY_LIMIT = 30;
let uploadConcurrency = Math.min(10, Math.max(1, Number(window.localStorage.getItem(UPLOAD_CONCURRENCY_KEY) || 4)));

window.addEventListener('soundhub:upload-concurrency-changed', (event) => {
  const value = Math.min(10, Math.max(1, Number(event?.detail?.value || 4)));
  uploadConcurrency = value;
  window.localStorage.setItem(UPLOAD_CONCURRENCY_KEY, String(value));
});

const uploadManager = (() => {
  const listeners = new Set();
  let batch = null;

  window.addEventListener('beforeunload', (event) => {
    if (!batch?.running) return;
    event.preventDefault();
    event.returnValue = '';
  });

  function emptyState() {
    return {
      active: false,
      running: false,
      cancelled: false,
      items: [],
      summary: '',
      uploaded: 0,
      skipped: 0,
      total: 0,
      failed: 0,
      progressPercent: 0,
    };
  }

  function snapshot() {
    if (!batch) return emptyState();

    const uploaded = batch.items.filter((item) => item.status === 'done').length;
    const skipped = batch.items.filter((item) => item.status === 'skipped').length;
    const failed = batch.items.filter((item) => item.status === 'failed').length;
    const totalBytes = batch.items.reduce((sum, item) => sum + (item.total || 0), 0);
    const loadedBytes = batch.items.reduce((sum, item) => {
      if (item.status === 'done') return sum + (item.total || 0);
      return sum + (item.loaded || 0);
    }, 0);
    const progressPercent = totalBytes > 0 ? Math.min(100, Math.round((loadedBytes / totalBytes) * 100)) : 0;
    const inFlight = batch.items.filter((item) => item.status === 'uploading').length;
    const cancelledCount = batch.items.filter((item) => item.status === 'cancelled').length;

    let summary = '';
    if (batch.running) {
      summary = `${t('uploading', 'Uploading...')} ${uploaded}/${batch.items.length}`;
      if (inFlight > 0) {
        summary += ` | ${t('upload_in_progress', 'in progress')}: ${inFlight}`;
      }
      if (failed > 0) {
        summary += ` | ${t('upload_failed_count', 'Failed')}: ${failed}`;
      }
      if (skipped > 0) {
        summary += ` | ${t('upload_skipped_count', 'Skipped')}: ${skipped}`;
      }
    } else if (batch.cancelled) {
      summary = `${t('upload_cancelled', 'Upload cancelled')}: ${uploaded}/${batch.items.length}`;
      if (cancelledCount > 0) {
        summary += ` | ${t('cancelled', 'Cancelled')}: ${cancelledCount}`;
      }
    } else if (failed > 0) {
      summary = `${t('upload_failed_count', 'Failed')}: ${failed}`;
    }

    return {
      active: batch.running || batch.cancelled || failed > 0,
      running: batch.running,
      cancelled: batch.cancelled,
      items: batch.items,
      summary,
      uploaded,
      skipped,
      total: batch.items.length,
      failed,
      progressPercent,
    };
  }

  function notify() {
    const state = snapshot();
    listeners.forEach((listener) => listener(state));
  }

  function subscribe(listener) {
    listeners.add(listener);
    listener(snapshot());
    return () => listeners.delete(listener);
  }

  async function start(files, options = {}) {
    if (batch?.running) {
      abort();
    }

    batch = {
      running: true,
      cancelled: false,
      items: files.map((file) => ({
        file,
        name: file.name,
        loaded: 0,
        total: file.size || 0,
        progress: 0,
        status: 'queued',
        error: '',
      })),
      controllers: new Map(),
    };
    notify();

    let nextIndex = 0;
    const workerCount = Math.min(uploadConcurrency, files.length);

    async function worker() {
      while (batch && !batch.cancelled) {
        const current = nextIndex;
        nextIndex += 1;
        if (current >= batch.items.length) break;

        const item = batch.items[current];
        const controller = new AbortController();
        batch.controllers.set(current, controller);
        item.status = 'uploading';
        notify();

        try {
          const payload = await API.uploadLibraryFileWithProgress(item.file, {
            signal: controller.signal,
            skipDuplicates: Boolean(options?.skipDuplicates),
            duplicatePolicy: String(options?.duplicatePolicy || 'keep'),
            onProgress(progress) {
              item.loaded = progress.loaded;
              item.total = progress.total;
              item.progress = progress.percent;
              notify();
            },
          });
          item.status = payload?.status === 'skipped' ? 'skipped' : 'done';
          item.progress = 100;
          item.loaded = item.total || item.loaded;
        } catch (error) {
          if (batch?.cancelled || controller.signal.aborted) {
            item.status = 'cancelled';
          } else {
            item.status = 'failed';
            item.error = error.message || '';
          }
        } finally {
          batch?.controllers.delete(current);
          notify();
        }
      }
    }

    await Promise.all(Array.from({ length: workerCount }, () => worker()));
    if (!batch) return emptyState();

    batch.running = false;
    notify();

    const finalState = snapshot();
    if (!batch.cancelled && finalState.failed === 0) {
      batch = null;
      notify();
      return finalState;
    }
    return finalState;
  }

  function abort() {
    if (!batch) return;
    batch.cancelled = true;
    batch.running = false;
    batch.controllers.forEach((controller) => controller.abort());
    batch.items.forEach((item) => {
      if (item.status === 'queued' || item.status === 'uploading') {
        item.status = 'cancelled';
      }
    });
    notify();
  }

  function clear() {
    batch = null;
    notify();
  }

  return { subscribe, start, abort, clear };
})();

export async function renderLibrary(context, root) {
  const pathEl = root.querySelector('#library-path-value');
  const statusEl = root.querySelector('#library-scan-status');
  const statusHeroEl = root.querySelector('#library-scan-status-hero');
  const statusBadgeEl = root.querySelector('#library-scan-status-badge');
  const scanBtn = root.querySelector('#scan-library-btn');
  const form = root.querySelector('#library-upload-form');
  const fileInput = root.querySelector('#library-upload-file');
  const folderInput = root.querySelector('#library-upload-folder');
  const skipDuplicatesInput = root.querySelector('#library-upload-skip-duplicates');
  const replaceDuplicatesInput = root.querySelector('#library-upload-replace-duplicates');
  const selectedEl = root.querySelector('#library-upload-selected');
  const resultEl = root.querySelector('#library-upload-result');
  const dropzone = root.querySelector('#library-upload-dropzone');
  const progressSection = root.querySelector('#library-upload-progress');
  const summaryEl = root.querySelector('#library-upload-summary');
  const overallBarEl = root.querySelector('#library-upload-overall-bar');
  const listEl = root.querySelector('#library-upload-list');
  const cancelBtn = root.querySelector('#library-upload-cancel');
  const historyListEl = root.querySelector('#library-upload-history-list');
  const historyClearBtn = root.querySelector('#library-upload-history-clear');
  const integrityBtn = root.querySelector('#library-integrity-btn');
  const integritySection = root.querySelector('#library-integrity-section');
  const integritySummaryEl = root.querySelector('#library-integrity-summary');
  const integrityListEl = root.querySelector('#library-integrity-list');
  let uploadHistory = loadUploadHistory();
  let unsubscribe = null;

  try {
    const settings = await API.getSettings();
    const value = Math.min(10, Math.max(1, Number(settings?.upload_concurrency || uploadConcurrency)));
    uploadConcurrency = value;
    window.localStorage.setItem(UPLOAD_CONCURRENCY_KEY, String(value));
  } catch {}

  async function reloadStatus() {
    const payload = await API.getLibraryScanStatus();
    const status = mapScannerStatus(payload?.scan?.status);
    pathEl.textContent = payload?.library_path || '-';
    statusEl.textContent = status;
    if (statusHeroEl) statusHeroEl.textContent = status;
    if (statusBadgeEl) {
      statusBadgeEl.textContent = status;
      statusBadgeEl.dataset.state = String(payload?.scan?.status || 'idle').toLowerCase();
    }
  }

  function renderBatch(state) {
    window.dispatchEvent(new CustomEvent('soundhub:upload-state', {
      detail: { active: Boolean(state.running), running: Boolean(state.running) }
    }));
    progressSection.hidden = !state.active;
    if (!state.active) {
      overallBarEl.style.width = '0%';
      listEl.innerHTML = '';
      cancelBtn.hidden = true;
      return;
    }

    summaryEl.textContent = state.summary || t('upload_waiting', 'Waiting to start upload.');
    overallBarEl.style.width = `${state.progressPercent || 0}%`;
    listEl.innerHTML = state.items.map((item) => `
      <div class="sh-upload-item" data-state="${escapeHtml(item.status)}">
        <div class="sh-upload-item-head">
          <strong title="${escapeHtml(item.name)}">${escapeHtml(item.name)}</strong>
          <span>${escapeHtml(renderUploadState(item))}</span>
        </div>
        <div class="sh-upload-item-bar"><span style="width:${Math.max(0, Math.min(100, item.progress || 0))}%"></span></div>
      </div>
    `).join('');
    cancelBtn.hidden = !state.running;
  }

  function renderIntegrity(report) {
    if (!integritySection || !integritySummaryEl || !integrityListEl) return;
    integritySection.hidden = false;
    const issues = Array.isArray(report?.issues) ? report.issues : [];
    if (!issues.length) {
      integritySummaryEl.textContent = t('integrity_ok', 'No broken tracks found.');
      integrityListEl.innerHTML = `<p class="sh-library-panel-copy">${escapeHtml(t('integrity_ok', 'No broken tracks found.'))}</p>`;
      return;
    }
    integritySummaryEl.textContent = `${t('integrity_found', 'Broken tracks found')}: ${issues.length}`;
    integrityListEl.innerHTML = issues.map((issue) => `
      <article class="sh-library-upload-history-entry">
        <header>
          <strong>${escapeHtml(issue.title || t('unknown_title', 'Unknown title'))}</strong>
          <span>${escapeHtml(issue.severity || 'error')}</span>
        </header>
        <div class="sh-library-panel-copy">${escapeHtml(issue.reason || '')}</div>
        <div class="sh-library-panel-copy">${escapeHtml(issue.file_path || '')}</div>
      </article>
    `).join('');
  }

  function currentDuplicatePolicy() {
    if (replaceDuplicatesInput?.checked) return 'replace';
    if (skipDuplicatesInput?.checked) return 'skip';
    return 'keep';
  }

  function renderHistory() {
    if (!historyListEl) return;
    if (!uploadHistory.length) {
      historyListEl.innerHTML = `<p class="sh-library-panel-copy">${escapeHtml(t('upload_history_empty', 'No upload history yet.'))}</p>`;
      return;
    }
    historyListEl.innerHTML = uploadHistory.map((entry) => {
      const items = (Array.isArray(entry.items) ? entry.items : []).map((item) => `
        <div class="sh-upload-item" data-state="${escapeHtml(item.status)}">
          <div class="sh-upload-item-head">
            <strong title="${escapeHtml(item.name)}">${escapeHtml(item.name)}</strong>
            <span>${escapeHtml(renderUploadState(item))}</span>
          </div>
        </div>
      `).join('');
      return `
        <article class="sh-library-upload-history-entry">
          <header>
            <strong>${escapeHtml(entry.summary || '')}</strong>
            <span>${escapeHtml(formatHistoryTime(entry.at))}</span>
          </header>
          <div class="sh-library-upload-history-items">${items}</div>
        </article>
      `;
    }).join('');
  }

  function appendHistory(state) {
    const items = Array.isArray(state?.items) ? state.items : [];
    if (!items.length) return;
    const uploaded = items.filter((item) => item.status === 'done').length;
    const skipped = items.filter((item) => item.status === 'skipped').length;
    const failed = items.filter((item) => item.status === 'failed').length;
    const cancelled = items.filter((item) => item.status === 'cancelled').length;
    const total = items.length;
    let summary = `${t('uploaded_count', 'Uploaded files')}: ${uploaded}/${total}`;
    if (skipped > 0) summary += ` | ${t('upload_skipped_count', 'Skipped')}: ${skipped}`;
    if (failed > 0) summary += ` | ${t('upload_failed_count', 'Failed')}: ${failed}`;
    if (cancelled > 0) summary += ` | ${t('cancelled', 'Cancelled')}: ${cancelled}`;
    uploadHistory.unshift({
      at: Date.now(),
      summary,
      items: items.map((item) => ({
        name: String(item.name || ''),
        status: String(item.status || 'queued'),
        error: String(item.error || ''),
      })),
    });
    uploadHistory = uploadHistory.slice(0, UPLOAD_HISTORY_LIMIT);
    saveUploadHistory(uploadHistory);
    renderHistory();
  }

  unsubscribe = uploadManager.subscribe(renderBatch);
  renderHistory();

  scanBtn.addEventListener('click', async () => {
    scanBtn.disabled = true;
    try {
      await API.scanLibrary();
      await reloadStatus();
    } finally {
      scanBtn.disabled = false;
    }
  });

  cancelBtn?.addEventListener('click', () => uploadManager.abort());
  historyClearBtn?.addEventListener('click', () => {
    uploadHistory = [];
    saveUploadHistory(uploadHistory);
    renderHistory();
  });

  integrityBtn?.addEventListener('click', async () => {
    integrityBtn.disabled = true;
    try {
      const report = await API.getLibraryIntegrity();
      renderIntegrity(report);
    } finally {
      integrityBtn.disabled = false;
    }
  });

  form.addEventListener('submit', (event) => {
    event.preventDefault();
  });

  skipDuplicatesInput?.addEventListener('change', () => {
    if (!skipDuplicatesInput.checked) return;
    if (replaceDuplicatesInput) replaceDuplicatesInput.checked = false;
  });
  replaceDuplicatesInput?.addEventListener('change', () => {
    if (!replaceDuplicatesInput.checked) return;
    if (skipDuplicatesInput) skipDuplicatesInput.checked = false;
  });

  fileInput.addEventListener('change', async () => {
    const files = filterAudioFiles(Array.from(fileInput.files || []));
    if (!files.length) return;
    updateSelectedFiles(files);
    const state = await uploadManager.start(files, {
      skipDuplicates: Boolean(skipDuplicatesInput?.checked),
      duplicatePolicy: currentDuplicatePolicy()
    });
    await reloadStatus();
    window.dispatchEvent(new CustomEvent('soundhub:library-updated'));
    appendHistory(state);
    resultEl.textContent = renderBatchResult(state);
    fileInput.value = '';
    updateSelectedFiles([]);
  });

  folderInput?.addEventListener('change', async () => {
    const files = filterAudioFiles(Array.from(folderInput.files || []));
    if (!files.length) return;
    updateSelectedFiles(files);
    const state = await uploadManager.start(files, {
      skipDuplicates: Boolean(skipDuplicatesInput?.checked),
      duplicatePolicy: currentDuplicatePolicy()
    });
    await reloadStatus();
    window.dispatchEvent(new CustomEvent('soundhub:library-updated'));
    appendHistory(state);
    resultEl.textContent = renderBatchResult(state);
    folderInput.value = '';
    updateSelectedFiles([]);
  });

  dropzone.addEventListener('dragover', (event) => {
    event.preventDefault();
    dropzone.classList.add('drag-over');
  });

  dropzone.addEventListener('dragleave', () => {
    dropzone.classList.remove('drag-over');
  });

  dropzone.addEventListener('drop', async (event) => {
    event.preventDefault();
    dropzone.classList.remove('drag-over');
    const files = filterAudioFiles(await getDroppedAudioFiles(event));
    if (!files.length) return;
    updateSelectedFiles(files);
    const state = await uploadManager.start(files, {
      skipDuplicates: Boolean(skipDuplicatesInput?.checked),
      duplicatePolicy: currentDuplicatePolicy()
    });
    await reloadStatus();
    window.dispatchEvent(new CustomEvent('soundhub:library-updated'));
    appendHistory(state);
    resultEl.textContent = renderBatchResult(state);
    updateSelectedFiles([]);
  });

  context.registerPageCleanup?.(() => {
    unsubscribe?.();
    window.dispatchEvent(new CustomEvent('soundhub:upload-state', { detail: { active: false, running: false } }));
  });

  function updateSelectedFiles(files) {
    if (!selectedEl) return;
    if (!files.length) {
      selectedEl.textContent = t('library_no_files_selected', 'No files selected');
      return;
    }
    if (files.length === 1) {
      selectedEl.textContent = files[0].name;
      return;
    }
    selectedEl.textContent = `${files.length} ${t('library_files_selected', 'files selected')}`;
  }

  await reloadStatus();
}

function renderBatchResult(state) {
  if (state.cancelled) {
    return t('upload_cancelled', 'Upload cancelled');
  }
  let result = `${t('uploaded_count', 'Uploaded files')}: ${state.uploaded}/${state.total}`;
  if (state.skipped > 0) {
    result += ` | ${t('upload_skipped_count', 'Skipped')}: ${state.skipped}`;
  }
  return result;
}

function renderUploadState(item) {
  switch (item.status) {
    case 'done':
      return t('uploaded', 'Uploaded');
    case 'skipped':
      return t('upload_skipped', 'Skipped duplicate');
    case 'failed':
      return `${t('upload_failed', 'Upload failed')}${item.error ? `: ${item.error}` : ''}`;
    case 'cancelled':
      return t('upload_cancelled', 'Upload cancelled');
    case 'uploading':
      return `${item.progress || 0}%`;
    default:
      return t('queued', 'Queued');
  }
}

function loadUploadHistory() {
  try {
    const raw = window.localStorage.getItem(UPLOAD_HISTORY_KEY);
    const parsed = JSON.parse(raw || '[]');
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

function saveUploadHistory(history) {
  try {
    window.localStorage.setItem(UPLOAD_HISTORY_KEY, JSON.stringify(Array.isArray(history) ? history : []));
  } catch {}
}

function formatHistoryTime(value) {
  const ts = Number(value || 0);
  if (!Number.isFinite(ts) || ts <= 0) return '';
  return new Date(ts).toLocaleString('ru-RU');
}

function mapScannerStatus(status) {
  const value = String(status || '').toLowerCase();
  if (value === 'scanning') {
    return t('scanner_status_scanning', 'Scanning');
  }
  if (value === 'completed') {
    return t('scanner_status_completed', 'Completed');
  }
  return t('scanner_status_idle', 'Idle');
}

async function getDroppedAudioFiles(event) {
  const items = Array.from(event.dataTransfer?.items || []);
  if (!items.length) {
    return Array.from(event.dataTransfer?.files || []);
  }

  const files = [];
  for (const item of items) {
    const entry = typeof item.webkitGetAsEntry === 'function' ? item.webkitGetAsEntry() : null;
    if (entry) {
      files.push(...await walkEntry(entry));
      continue;
    }
    const file = item.getAsFile?.();
    if (file) files.push(file);
  }
  return files;
}

async function walkEntry(entry) {
  if (!entry) return [];
  if (entry.isFile) {
    const file = await new Promise((resolve) => entry.file(resolve, () => resolve(null)));
    return file ? [file] : [];
  }
  if (!entry.isDirectory) return [];

  const reader = entry.createReader();
  const out = [];
  while (true) {
    const batch = await new Promise((resolve) => reader.readEntries(resolve, () => resolve([])));
    if (!batch.length) break;
    for (const child of batch) {
      out.push(...await walkEntry(child));
    }
  }
  return out;
}

function filterAudioFiles(files) {
  return files.filter(isAudioFile);
}

function isAudioFile(file) {
  const name = String(file?.name || '').toLowerCase();
  const type = String(file?.type || '').toLowerCase();
  return type.startsWith('audio/')
    || name.endsWith('.mp3')
    || name.endsWith('.flac')
    || name.endsWith('.ogg')
    || name.endsWith('.m4a')
    || name.endsWith('.wav')
    || name.endsWith('.aac')
    || name.endsWith('.opus')
    || name.endsWith('.wma')
    || name.endsWith('.alac');
}

function escapeHtml(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

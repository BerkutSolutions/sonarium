import { API } from './api.js';
import { t } from './i18n.js';

const UPLOAD_CONCURRENCY_KEY = 'sonarium.uploadConcurrency';
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
      total: 0,
      failed: 0,
      progressPercent: 0,
    };
  }

  function snapshot() {
    if (!batch) return emptyState();

    const uploaded = batch.items.filter((item) => item.status === 'done').length;
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

  async function start(files) {
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
          await API.uploadLibraryFileWithProgress(item.file, {
            signal: controller.signal,
            onProgress(progress) {
              item.loaded = progress.loaded;
              item.total = progress.total;
              item.progress = progress.percent;
              notify();
            },
          });
          item.status = 'done';
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
      return emptyState();
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

export async function renderLibrary(_context, root) {
  const pathEl = root.querySelector('#library-path-value');
  const statusEl = root.querySelector('#library-scan-status');
  const statusHeroEl = root.querySelector('#library-scan-status-hero');
  const statusBadgeEl = root.querySelector('#library-scan-status-badge');
  const scanBtn = root.querySelector('#scan-library-btn');
  const form = root.querySelector('#library-upload-form');
  const fileInput = root.querySelector('#library-upload-file');
  const folderInput = root.querySelector('#library-upload-folder');
  const selectedEl = root.querySelector('#library-upload-selected');
  const resultEl = root.querySelector('#library-upload-result');
  const dropzone = root.querySelector('#library-upload-dropzone');
  const progressSection = root.querySelector('#library-upload-progress');
  const summaryEl = root.querySelector('#library-upload-summary');
  const overallBarEl = root.querySelector('#library-upload-overall-bar');
  const listEl = root.querySelector('#library-upload-list');
  const cancelBtn = root.querySelector('#library-upload-cancel');
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

  unsubscribe = uploadManager.subscribe(renderBatch);

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

  form.addEventListener('submit', (event) => {
    event.preventDefault();
  });

  fileInput.addEventListener('change', async () => {
    const files = filterAudioFiles(Array.from(fileInput.files || []));
    if (!files.length) return;
    updateSelectedFiles(files);
    const state = await uploadManager.start(files);
    await reloadStatus();
    window.dispatchEvent(new CustomEvent('soundhub:library-updated'));
    resultEl.textContent = state.cancelled
      ? t('upload_cancelled', 'Upload cancelled')
      : `${t('uploaded_count', 'Uploaded files')}: ${state.uploaded}/${state.total}`;
    fileInput.value = '';
    updateSelectedFiles([]);
  });

  folderInput?.addEventListener('change', async () => {
    const files = filterAudioFiles(Array.from(folderInput.files || []));
    if (!files.length) return;
    updateSelectedFiles(files);
    const state = await uploadManager.start(files);
    await reloadStatus();
    window.dispatchEvent(new CustomEvent('soundhub:library-updated'));
    resultEl.textContent = state.cancelled
      ? t('upload_cancelled', 'Upload cancelled')
      : `${t('uploaded_count', 'Uploaded files')}: ${state.uploaded}/${state.total}`;
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
    const state = await uploadManager.start(files);
    await reloadStatus();
    window.dispatchEvent(new CustomEvent('soundhub:library-updated'));
    resultEl.textContent = state.cancelled
      ? t('upload_cancelled', 'Upload cancelled')
      : `${t('uploaded_count', 'Uploaded files')}: ${state.uploaded}/${state.total}`;
    updateSelectedFiles([]);
  });

  root.addEventListener('DOMNodeRemoved', () => {
    unsubscribe?.();
  }, { once: true });

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

function renderUploadState(item) {
  switch (item.status) {
    case 'done':
      return t('uploaded', 'Uploaded');
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

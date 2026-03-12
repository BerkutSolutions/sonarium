import { API } from './api.js';
import { t } from './i18n.js';

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

  scanBtn.addEventListener('click', async () => {
    scanBtn.disabled = true;
    try {
      await API.scanLibrary();
      await reloadStatus();
    } finally {
      scanBtn.disabled = false;
    }
  });

  form.addEventListener('submit', (event) => {
    event.preventDefault();
  });
  fileInput.addEventListener('change', async () => {
    const files = filterAudioFiles(Array.from(fileInput.files || []));
    if (!files.length) return;
    updateSelectedFiles(files);
    await uploadMany(files);
    fileInput.value = '';
    updateSelectedFiles([]);
  });
  folderInput?.addEventListener('change', async () => {
    const files = filterAudioFiles(Array.from(folderInput.files || []));
    if (!files.length) return;
    updateSelectedFiles(files);
    await uploadMany(files);
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
    await uploadMany(files);
    updateSelectedFiles([]);
  });

  async function uploadMany(files) {
    const total = files.length;
    let uploaded = 0;
    for (let index = 0; index < total; index += 1) {
      const file = files[index];
      resultEl.textContent = `${t('uploading', 'Uploading...')} ${index + 1}/${total}: ${file.name}`;
      try {
        await upload(file);
        uploaded += 1;
      } catch (error) {
        resultEl.textContent = `${t('upload_failed', 'Upload failed')}: ${file.name} (${error.message})`;
      }
    }
    await reloadStatus();
    window.dispatchEvent(new CustomEvent('soundhub:library-updated'));
    resultEl.textContent = `${t('uploaded_count', 'Uploaded files')}: ${uploaded}/${total}`;
  }

  async function upload(file) {
    const payload = await API.uploadLibraryFile(file);
    return payload?.stored_path || file.name;
  }

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

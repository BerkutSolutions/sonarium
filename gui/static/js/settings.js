import { API } from './api.js';
import { t } from './i18n.js';

export async function renderSettings(_context, root) {
  const content = root.querySelector('#settings-content');
  const updatesPanel = root.querySelector('#settings-updates-panel');
  const updatesContent = root.querySelector('#settings-updates-content');
  const updatesError = root.querySelector('#settings-updates-error');
  const checkButton = root.querySelector('#settings-check-updates-btn');
  const autoCheckUpdates = root.querySelector('#settings-auto-check-updates');
  const uploadPanel = root.querySelector('#settings-upload-panel');
  const uploadConcurrencySelect = root.querySelector('#settings-upload-concurrency');
  const uploadConcurrencySave = root.querySelector('#settings-upload-concurrency-save');
  const uploadError = root.querySelector('#settings-upload-error');
  const storagePanel = root.querySelector('#settings-storage-panel');
  const storageContent = root.querySelector('#settings-storage-content');
  const storageError = root.querySelector('#settings-storage-error');
  const deleteAllMusicButton = root.querySelector('#settings-delete-all-music-btn');
  const deleteModal = root.querySelector('#settings-delete-modal');
  const deleteModalClose = root.querySelector('#settings-delete-modal-close');
  const deleteModalCancel = root.querySelector('#settings-delete-modal-cancel');
  const deleteModalConfirm = root.querySelector('#settings-delete-modal-confirm');
  const data = await API.getSettings();

  content.innerHTML = `
    <div class="sh-list-row"><span>${escapeHtml(t('server', 'Server'))}</span><strong>${escapeHtml(data?.server_name || '-')}</strong></div>
    <div class="sh-list-row"><span>${escapeHtml(t('version', 'Version'))}</span><strong>${escapeHtml(data?.version || '-')}</strong></div>
    <div class="sh-list-row"><span>${escapeHtml(t('library_path', 'Library Path'))}</span><strong>${escapeHtml(data?.library_path || '-')}</strong></div>
    <div class="sh-list-row"><span>${escapeHtml(t('database', 'Database'))}</span><strong>${data?.database_ok ? escapeHtml(t('ok', 'OK')) : escapeHtml(t('unavailable', 'Unavailable'))}</strong></div>
    <div class="sh-list-row"><span>${escapeHtml(t('scanner_status', 'Scanner Status'))}</span><strong>${escapeHtml(mapScannerStatus(data?.scanner_status?.status || ''))}</strong></div>
  `;

  if (!updatesPanel || !updatesContent) {
    return;
  }

  const isAdmin = Boolean(data?.can_check_updates);
  updatesPanel.hidden = !isAdmin;
  if (uploadPanel) uploadPanel.hidden = !isAdmin;
  if (storagePanel) storagePanel.hidden = !isAdmin;

  if (uploadConcurrencySelect) {
    const currentValue = Math.min(10, Math.max(1, Number(data?.upload_concurrency || 4)));
    uploadConcurrencySelect.value = String(currentValue);
  }
  if (autoCheckUpdates) {
    autoCheckUpdates.checked = Boolean(data?.auto_check_updates);
  }

  if (!isAdmin) {
    return;
  }

  renderUpdateState(updatesContent, data?.update_check, data?.version || '-');
  await reloadStorageUsage();

  checkButton?.addEventListener('click', async () => {
    if (updatesError) {
      updatesError.hidden = true;
      updatesError.textContent = '';
    }
    if (checkButton) {
      checkButton.disabled = true;
      checkButton.textContent = t('checking_updates', 'Checking...');
    }
    try {
      const result = await API.checkUpdates();
      renderUpdateState(updatesContent, result, data?.version || '-');
    } catch (error) {
      if (updatesError) {
        updatesError.hidden = false;
        updatesError.textContent = error.message || t('update_check_failed', 'Failed to check updates');
      }
    } finally {
      if (checkButton) {
        checkButton.disabled = false;
        checkButton.textContent = t('check_updates', 'Check updates');
      }
    }
  });

  autoCheckUpdates?.addEventListener('change', async () => {
    if (updatesError) {
      updatesError.hidden = true;
      updatesError.textContent = '';
    }
    autoCheckUpdates.disabled = true;
    try {
      await API.setAutoCheckUpdates(Boolean(autoCheckUpdates.checked));
      if (autoCheckUpdates.checked) {
        const result = await API.checkUpdates();
        renderUpdateState(updatesContent, result, data?.version || '-');
      }
    } catch (error) {
      autoCheckUpdates.checked = !autoCheckUpdates.checked;
      if (updatesError) {
        updatesError.hidden = false;
        updatesError.textContent = error.message || t('save_failed', 'Failed to save changes');
      }
    } finally {
      autoCheckUpdates.disabled = false;
    }
  });

  uploadConcurrencySave?.addEventListener('click', async () => {
    if (!uploadConcurrencySelect) return;
    if (uploadError) {
      uploadError.hidden = true;
      uploadError.textContent = '';
    }
    const value = Math.min(10, Math.max(1, Number(uploadConcurrencySelect.value || 4)));
    uploadConcurrencySave.disabled = true;
    try {
      await API.setUploadConcurrency(value);
      window.dispatchEvent(new CustomEvent('soundhub:upload-concurrency-changed', { detail: { value } }));
    } catch (error) {
      if (uploadError) {
        uploadError.hidden = false;
        uploadError.textContent = error.message || t('save_failed', 'Failed to save changes');
      }
    } finally {
      uploadConcurrencySave.disabled = false;
    }
  });

  deleteAllMusicButton?.addEventListener('click', openDeleteModal);
  deleteModalClose?.addEventListener('click', closeDeleteModal);
  deleteModalCancel?.addEventListener('click', closeDeleteModal);
  deleteModal?.addEventListener('click', (event) => {
    if (event.target === deleteModal) {
      closeDeleteModal();
    }
  });
  deleteModalConfirm?.addEventListener('click', async () => {
    if (storageError) {
      storageError.hidden = true;
      storageError.textContent = '';
    }
    if (deleteModalConfirm) {
      deleteModalConfirm.disabled = true;
    }
    try {
      await API.deleteAllMusic();
      closeDeleteModal();
      await reloadStorageUsage();
      window.dispatchEvent(new CustomEvent('soundhub:library-updated'));
    } catch (error) {
      if (storageError) {
        storageError.hidden = false;
        storageError.textContent = error.message || t('delete_all_music_failed', 'Failed to delete music library');
      }
    } finally {
      if (deleteModalConfirm) {
        deleteModalConfirm.disabled = false;
      }
    }
  });

  if (autoCheckUpdates?.checked && !data?.update_check) {
    try {
      const result = await API.checkUpdates();
      renderUpdateState(updatesContent, result, data?.version || '-');
    } catch {
      // Keep the page usable even if the automatic check fails.
    }
  }

  function openDeleteModal() {
    if (!deleteModal) return;
    deleteModal.hidden = false;
    document.body.classList.add('sh-modal-open');
  }

  function closeDeleteModal() {
    if (!deleteModal) return;
    deleteModal.hidden = true;
    document.body.classList.remove('sh-modal-open');
  }

  async function reloadStorageUsage() {
    if (!storageContent) return;
    if (storageError) {
      storageError.hidden = true;
      storageError.textContent = '';
    }
    try {
      const usage = await API.getStorageUsage();
      renderStorageUsage(storageContent, usage);
    } catch (error) {
      if (storageError) {
        storageError.hidden = false;
        storageError.textContent = error.message || t('storage_usage_failed', 'Failed to load storage usage');
      }
    }
  }
}

function mapScannerStatus(status) {
  const value = String(status || '').toLowerCase();
  if (value === 'scanning') return t('scanner_status_scanning', 'Scanning');
  if (value === 'completed') return t('scanner_status_completed', 'Completed');
  if (value === 'failed') return t('scanner_status_failed', 'Failed');
  return t('scanner_status_idle', 'Idle');
}

function renderUpdateState(root, update, currentVersion) {
  const latestVersion = String(update?.latest_version || currentVersion || '-').trim() || '-';
  const releaseURL = String(update?.release_url || '').trim();
  const checkedAt = update?.checked_at ? formatDateTime(update.checked_at) : t('not_checked_yet', 'Not checked yet');
  const hasUpdate = Boolean(update?.has_update);
  const source = String(update?.source || '').trim();

  root.innerHTML = `
    <div class="sh-list-row"><span>${escapeHtml(t('current_version', 'Current version'))}</span><strong>${escapeHtml(currentVersion || '-')}</strong></div>
    <div class="sh-list-row"><span>${escapeHtml(t('latest_version', 'Latest version'))}</span><strong>${escapeHtml(latestVersion)}</strong></div>
    <div class="sh-list-row"><span>${escapeHtml(t('update_status', 'Status'))}</span><strong>${escapeHtml(hasUpdate ? t('update_available', 'Update available') : t('up_to_date', 'Up to date'))}</strong></div>
    <div class="sh-list-row"><span>${escapeHtml(t('last_checked', 'Last checked'))}</span><strong>${escapeHtml(checkedAt)}</strong></div>
    <div class="sh-list-row"><span>${escapeHtml(t('update_source', 'Source'))}</span><strong>${escapeHtml(source || '-')}</strong></div>
    ${releaseURL ? `<div class="sh-list-row"><span>${escapeHtml(t('release_page', 'Release page'))}</span><strong><a class="sh-link" href="${escapeHtml(releaseURL)}" target="_blank" rel="noopener noreferrer">${escapeHtml(t('open_release_page', 'Open release page'))}</a></strong></div>` : ''}
  `;
}

function renderStorageUsage(root, usage) {
  const totalBytes = Number(usage?.total_bytes || 0);
  const users = Array.isArray(usage?.users) ? usage.users : [];
  root.innerHTML = `
    <div class="sh-list-row"><span>${escapeHtml(t('music_storage_total', 'Total music size'))}</span><strong>${escapeHtml(formatBytes(totalBytes))}</strong></div>
    ${users.map((user) => `
      <div class="sh-list-row">
        <span>${escapeHtml(user.display_name || t('system', 'System'))}</span>
        <strong>${escapeHtml(formatBytes(Number(user.bytes_used || 0)))} • ${escapeHtml(String(user.track_count || 0))} ${escapeHtml(t('tracks_lower', 'tracks'))}</strong>
      </div>
    `).join('')}
  `;
}

function formatDateTime(value) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return String(value || '');
  return date.toLocaleString();
}

function escapeHtml(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

function formatBytes(value) {
  const bytes = Number(value || 0);
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let size = bytes;
  let unit = 0;
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024;
    unit += 1;
  }
  return `${size.toFixed(size >= 10 || unit === 0 ? 0 : 1)} ${units[unit]}`;
}

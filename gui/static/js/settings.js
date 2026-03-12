import { API } from './api.js';
import { t } from './i18n.js';

export async function renderSettings(_context, root) {
  const content = root.querySelector('#settings-content');
  const updatesPanel = root.querySelector('#settings-updates-panel');
  const updatesContent = root.querySelector('#settings-updates-content');
  const updatesError = root.querySelector('#settings-updates-error');
  const checkButton = root.querySelector('#settings-check-updates-btn');
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

  const canCheckUpdates = Boolean(data?.can_check_updates);
  updatesPanel.hidden = !canCheckUpdates;
  if (!canCheckUpdates) {
    return;
  }

  renderUpdateState(updatesContent, data?.update_check, data?.version || '-');
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

function formatDateTime(value) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return String(value || '');
  }
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

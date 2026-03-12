import { API } from './api.js';
import { t } from './i18n.js';

export function createShareModal() {
  const modal = document.getElementById('share-modal');
  const closeButton = document.getElementById('share-modal-close');
  const errorBox = document.getElementById('share-error');
  const publicToggle = document.getElementById('share-public-toggle');
  const publicLinkInput = document.getElementById('share-public-link');
  const copyButton = document.getElementById('share-copy-link');
  const userSelect = document.getElementById('share-user-select');
  const permissionSelect = document.getElementById('share-permission-select');
  const addButton = document.getElementById('share-user-add');
  const shareList = document.getElementById('share-list');

  const state = {
    entityType: '',
    entityId: '',
    users: []
  };

  const close = () => {
    if (!modal) return;
    modal.hidden = true;
    document.body.classList.remove('sh-modal-open');
    setError('');
  };

  closeButton?.addEventListener('click', close);
  modal?.addEventListener('click', (event) => {
    if (event.target === modal) close();
  });

  publicToggle?.addEventListener('change', async () => {
    if (!state.entityType || !state.entityId) return;
    publicToggle.disabled = true;
    setError('');
    try {
      const share = await API.setEntityPublicShare(state.entityType, state.entityId, publicToggle.checked);
      publicLinkInput.value = share?.share_token ? buildShareLink(state.entityType, state.entityId, share.share_token) : '';
      await reloadShares();
    } catch (error) {
      publicToggle.checked = !publicToggle.checked;
      setError(error.message || t('save_failed', 'Failed to save changes'));
    } finally {
      publicToggle.disabled = false;
    }
  });

  copyButton?.addEventListener('click', async () => {
    if (!publicLinkInput?.value) return;
    try {
      await navigator.clipboard.writeText(publicLinkInput.value);
      copyButton.textContent = t('copied', 'Copied');
      setTimeout(() => {
        copyButton.textContent = t('copy_link', 'Copy link');
      }, 1200);
    } catch {
      publicLinkInput.select();
      document.execCommand('copy');
    }
  });

  addButton?.addEventListener('click', async () => {
    if (!state.entityType || !state.entityId) return;
    const userID = String(userSelect?.value || '').trim();
    if (!userID) return;
    addButton.disabled = true;
    setError('');
    try {
      await API.shareEntityWithUser(state.entityType, state.entityId, {
        user_id: userID,
        permission: permissionSelect?.value || 'listener'
      });
      await reloadShares();
    } catch (error) {
      setError(error.message || t('save_failed', 'Failed to save changes'));
    } finally {
      addButton.disabled = false;
    }
  });

  async function reloadShares() {
    const [shares, users] = await Promise.all([
      API.getEntityShares(state.entityType, state.entityId),
      state.users.length ? Promise.resolve(state.users) : API.getShareableUsers()
    ]);
    state.users = Array.isArray(users) ? users : [];
    renderUsers(state.users);
    renderShares(Array.isArray(shares) ? shares : []);
  }

  function renderUsers(users) {
    if (!userSelect) return;
    userSelect.innerHTML = `<option value="">${escapeHtml(t('select_user', 'Select user'))}</option>` + users.map((user) => `
      <option value="${escapeHtml(user.id)}">${escapeHtml(user.display_name || user.username)}</option>
    `).join('');
  }

  function renderShares(shares) {
    if (!shareList) return;
    const publicShare = shares.find((share) => share.is_public);
    publicToggle.checked = Boolean(publicShare);
    publicLinkInput.value = publicShare?.share_token ? buildShareLink(state.entityType, state.entityId, publicShare.share_token) : '';

    const directShares = shares.filter((share) => !share.is_public);
    if (!directShares.length) {
      shareList.innerHTML = `<p class="sh-detail-empty">${escapeHtml(t('share_empty', 'No direct shares yet.'))}</p>`;
      return;
    }

    shareList.innerHTML = directShares.map((share) => `
      <div class="sh-list-row sh-share-row" data-share-id="${escapeHtml(share.id)}">
        <div>
          <strong>${escapeHtml(share.recipient_display_name || share.recipient_username || '-')}</strong>
          <small>${escapeHtml(share.permission || 'listener')}</small>
        </div>
        <div class="sh-actions">
          <button type="button" class="btn ghost" data-share-revoke="${escapeHtml(share.id)}">${escapeHtml(t('remove_access', 'Remove access'))}</button>
        </div>
      </div>
    `).join('');

    shareList.querySelectorAll('[data-share-revoke]').forEach((button) => {
      button.addEventListener('click', async () => {
        const shareID = button.getAttribute('data-share-revoke');
        if (!shareID) return;
        button.disabled = true;
        try {
          await API.deleteEntityShare(shareID);
          await reloadShares();
        } catch (error) {
          setError(error.message || t('save_failed', 'Failed to save changes'));
          button.disabled = false;
        }
      });
    });
  }

  function setError(message) {
    if (!errorBox) return;
    errorBox.hidden = !message;
    errorBox.textContent = message || '';
  }

  return {
    async open(entityType, entityId) {
      state.entityType = String(entityType || '').trim().toLowerCase();
      state.entityId = String(entityId || '').trim();
      if (!state.entityType || !state.entityId || !modal) return;
      permissionSelect.innerHTML = state.entityType === 'playlist'
        ? `
            <option value="listener">${escapeHtml(t('listener', 'Listener'))}</option>
            <option value="editor">${escapeHtml(t('editor', 'Editor'))}</option>
          `
        : `<option value="viewer">${escapeHtml(t('viewer', 'Viewer'))}</option>`;
      modal.hidden = false;
      document.body.classList.add('sh-modal-open');
      await reloadShares();
    },
    close
  };
}

function buildShareLink(entityType, entityId, token) {
  return `${window.location.origin}/${entityType}s/${encodeURIComponent(entityId)}?share=${encodeURIComponent(token)}`;
}

function escapeHtml(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

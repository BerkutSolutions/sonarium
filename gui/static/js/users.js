import { API } from './api.js';
import { t } from './i18n.js';

export async function renderUsers(context, root) {
  const list = root.querySelector('#users-list');
  const registrationToggle = root.querySelector('#users-registration-open');
  const errorBox = root.querySelector('#users-error');
  const payload = await API.getUsers();
  const users = Array.isArray(payload?.users) ? payload.users : [];
  const currentUser = context?.auth?.getStatus?.()?.user || null;

  registrationToggle.checked = Boolean(payload?.registration_open);
  registrationToggle.addEventListener('change', async () => {
    registrationToggle.disabled = true;
    setError(errorBox, '');
    try {
      await API.setRegistrationOpen(registrationToggle.checked);
    } catch (error) {
      registrationToggle.checked = !registrationToggle.checked;
      setError(errorBox, error.message || t('save_failed', 'Failed to save changes'));
    } finally {
      registrationToggle.disabled = false;
    }
  });

  if (!users.length) {
    list.innerHTML = `<p>${escapeHtml(t('users_empty', 'No users found.'))}</p>`;
    return;
  }

  list.innerHTML = users.map((user) => {
    const self = currentUser?.id === user.id;
    return `
      <div class="sh-list-row sh-user-row" data-user-id="${escapeHtml(user.id)}">
        <div>
          <strong><a href="/profile/${encodeURIComponent(user.id)}" data-user-profile-link>${escapeHtml(user.display_name || user.username)}</a></strong>
          <small>@${escapeHtml(user.username)} - ${escapeHtml(user.role)} - ${escapeHtml(user.active ? t('active', 'Active') : t('blocked', 'Blocked'))}</small>
        </div>
        <div class="sh-actions">
          <button type="button" data-action="toggle-active"${self ? ' disabled' : ''}>${escapeHtml(user.active ? t('block', 'Block') : t('unblock', 'Unblock'))}</button>
          <button type="button" data-action="delete"${self ? ' disabled' : ''}>${escapeHtml(t('delete', 'Delete'))}</button>
        </div>
      </div>
    `;
  }).join('');

  list.querySelectorAll('[data-action="toggle-active"]').forEach((button) => {
    button.addEventListener('click', async () => {
      const row = button.closest('[data-user-id]');
      const userID = row?.getAttribute('data-user-id') || '';
      if (!userID) return;
      button.disabled = true;
      setError(errorBox, '');
      try {
        const shouldActivate = button.textContent.trim() === t('unblock', 'Unblock');
        await API.setUserActive(userID, shouldActivate);
        await renderUsers(context, root);
      } catch (error) {
        setError(errorBox, error.message || t('save_failed', 'Failed to save changes'));
      } finally {
        button.disabled = false;
      }
    });
  });

  list.querySelectorAll('[data-action="delete"]').forEach((button) => {
    button.addEventListener('click', async () => {
      const row = button.closest('[data-user-id]');
      const userID = row?.getAttribute('data-user-id') || '';
      if (!userID) return;
      button.disabled = true;
      setError(errorBox, '');
      try {
        await API.deleteUser(userID);
        await renderUsers(context, root);
      } catch (error) {
        setError(errorBox, error.message || t('save_failed', 'Failed to save changes'));
      } finally {
        button.disabled = false;
      }
    });
  });

  list.querySelectorAll('[data-user-profile-link]').forEach((link) => {
    link.addEventListener('click', (event) => {
      event.preventDefault();
      window.dispatchEvent(new CustomEvent('soundhub:navigate', { detail: { path: link.getAttribute('href') } }));
    });
  });
}

function setError(node, message) {
  if (!node) return;
  node.hidden = !message;
  node.textContent = message || '';
}

function escapeHtml(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

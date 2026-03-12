import { API } from './api.js';
import { t } from './i18n.js';

export async function renderProfile(context, root, params = {}) {
  const status = context?.auth?.getStatus?.();
  const currentUser = status?.user || null;
  const requestedUserID = String(params.userId || currentUser?.id || '').trim();
  const ownProfile = currentUser?.id === requestedUserID;
  const errorBox = root.querySelector('#profile-error');
  const passwordError = root.querySelector('#profile-password-error');
  const form = root.querySelector('#profile-form');
  const displayNameInput = root.querySelector('#profile-display-name');
  const usernameInput = root.querySelector('#profile-username');
  const profilePublicInput = root.querySelector('#profile-public');
  const editButton = root.querySelector('#profile-edit-btn');
  const saveButton = root.querySelector('#profile-save-btn');
  const passwordForm = root.querySelector('#profile-password-form');
  const sharedList = root.querySelector('#profile-shared-list');

  let profile = ownProfile
    ? currentUser
    : (await API.getProfile(requestedUserID))?.user;

  renderProfileState(root, profile, ownProfile);
  renderSharedWithMe(sharedList, ownProfile ? await API.getReceivedShares(currentUser?.id || '') : []);

  displayNameInput.value = profile?.display_name || '';
  usernameInput.value = profile?.username || '';
  profilePublicInput.checked = Boolean(profile?.profile_public);

  if (!ownProfile) {
    return;
  }

  editButton?.addEventListener('click', () => {
    const editing = displayNameInput.disabled;
    displayNameInput.disabled = !editing;
    usernameInput.disabled = !editing;
    profilePublicInput.disabled = !editing;
    saveButton.disabled = !editing;
    if (editing) {
      displayNameInput.focus();
      return;
    }
    displayNameInput.value = profile?.display_name || '';
    usernameInput.value = profile?.username || '';
    profilePublicInput.checked = Boolean(profile?.profile_public);
  });

  form?.addEventListener('submit', async (event) => {
    event.preventDefault();
    setError(errorBox, '');
    saveButton.disabled = true;
    try {
      const payload = await API.updateProfile({
        display_name: displayNameInput.value.trim(),
        username: usernameInput.value.trim(),
        profile_public: profilePublicInput.checked
      });
      await context.auth.refresh();
      profile = payload?.user || profile;
      displayNameInput.value = profile?.display_name || displayNameInput.value;
      usernameInput.value = profile?.username || usernameInput.value;
      profilePublicInput.checked = Boolean(profile?.profile_public);
      displayNameInput.disabled = true;
      usernameInput.disabled = true;
      profilePublicInput.disabled = true;
      saveButton.disabled = true;
    } catch (error) {
      setError(errorBox, error.message || t('save_failed', 'Failed to save changes'));
      saveButton.disabled = false;
    }
  });

  passwordForm?.addEventListener('submit', async (event) => {
    event.preventDefault();
    setError(passwordError, '');
    const currentPassword = root.querySelector('#profile-current-password').value;
    const newPassword = root.querySelector('#profile-new-password').value;
    const repeatPassword = root.querySelector('#profile-repeat-password').value;
    if (newPassword !== repeatPassword) {
      setError(passwordError, t('passwords_no_match', 'Passwords do not match'));
      return;
    }
    try {
      await API.changePassword({
        current_password: currentPassword,
        new_password: newPassword
      });
      passwordForm.reset();
    } catch (error) {
      setError(passwordError, error.message || t('save_failed', 'Failed to save changes'));
    }
  });
}

function renderProfileState(root, profile, ownProfile) {
  root.querySelector('#profile-display-name')?.setAttribute('placeholder', profile?.display_name || '');
  root.querySelector('#profile-username')?.setAttribute('placeholder', profile?.username || '');
  root.querySelector('#profile-edit-btn')?.toggleAttribute('hidden', !ownProfile);
  root.querySelector('#profile-save-btn')?.toggleAttribute('hidden', !ownProfile);
  root.querySelector('#profile-password-form')?.closest('.sh-library-panel')?.toggleAttribute('hidden', !ownProfile);
  root.querySelector('#profile-shared-list')?.closest('.sh-library-panel')?.toggleAttribute('hidden', !ownProfile);
}

function renderSharedWithMe(node, shares) {
  if (!node) return;
  const items = Array.isArray(shares) ? shares : [];
  if (!items.length) {
    node.innerHTML = `<p class="sh-detail-empty">${escapeHtml(t('shared_with_me_empty', 'Nothing has been shared with you yet.'))}</p>`;
    return;
  }
  node.innerHTML = items.map((share) => `
    <a class="sh-list-row sh-user-row" href="${escapeHtml(buildEntityHref(share))}" data-share-link>
      <div>
        <strong>${escapeHtml(entityLabel(share.entity_type))}</strong>
        <small>${escapeHtml(share.recipient_display_name || share.recipient_username || '')} • ${escapeHtml(share.permission || '')}</small>
      </div>
    </a>
  `).join('');
  node.querySelectorAll('[data-share-link]').forEach((link) => {
    link.addEventListener('click', (event) => {
      event.preventDefault();
      window.dispatchEvent(new CustomEvent('soundhub:navigate', { detail: { path: link.getAttribute('href') } }));
    });
  });
}

function entityLabel(entityType) {
  switch (String(entityType || '').toLowerCase()) {
    case 'album':
      return t('album_contents', 'Album');
    case 'artist':
      return t('artist', 'Artist');
    case 'playlist':
      return t('playlist_contents', 'Playlist');
    case 'track':
      return t('track_info', 'Track');
    default:
      return entityType || '-';
  }
}

function buildEntityHref(share) {
  const id = encodeURIComponent(String(share?.entity_id || '').trim());
  switch (String(share?.entity_type || '').toLowerCase()) {
    case 'album':
      return `/albums/${id}`;
    case 'artist':
      return `/artists/${id}`;
    case 'playlist':
      return `/playlists/${id}`;
    case 'track':
      return `/tracks/${id}`;
    default:
      return '/';
  }
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

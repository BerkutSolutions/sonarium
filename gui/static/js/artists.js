import { API } from './api.js';
import { t } from './i18n.js';
import { showContextMenu } from './context-menu.js';
import { loadArtistNameMap, resolveArtistName } from './artist-map.js';

export async function renderArtists(context, root) {
  const list = root.querySelector('#artists-list');
  const editModal = root.querySelector('#artist-edit-modal');
  const editForm = root.querySelector('#artist-edit-form');
  const editName = root.querySelector('#artist-edit-name');
  const editMergeTarget = root.querySelector('#artist-edit-merge-target');
  const editClose = root.querySelector('#artist-edit-close');
  const editCancel = root.querySelector('#artist-edit-cancel');
  const modal = root.querySelector('#artist-modal');
  const modalClose = root.querySelector('#artist-modal-close');
  const modalCover = root.querySelector('#artist-modal-cover');
  const modalNameView = root.querySelector('#artist-modal-name-view');
  const modalName = root.querySelector('#artist-modal-name');
  const modalID = root.querySelector('#artist-modal-id');
  const modalIDRow = root.querySelector('#artist-modal-id-row');
  const modalAlbums = root.querySelector('#artist-modal-albums');
  const modalEdit = root.querySelector('#artist-modal-edit');
  const modalSave = root.querySelector('#artist-modal-save');
  const modalDelete = root.querySelector('#artist-modal-delete');

  let artists = [];
  let countMap = new Map();
  let selected = null;
  let editingArtist = null;
  let editMode = false;

  async function load() {
    const [artistsRaw, countsRaw] = await Promise.all([
      API.getArtists({ limit: 500, offset: 0, sort: 'name' }),
      API.getArtistAlbumCounts(),
    ]);
    countMap = new Map((Array.isArray(countsRaw) ? countsRaw : []).map((item) => [item.artist_id || item.artistId, item.album_count || item.albumCount || 0]));
    artists = normalizeArtists(artistsRaw);
    renderList();
  }

  function renderList() {
    if (!artists.length) {
      list.innerHTML = `<p>${escapeHtml(t('empty_no_artists', 'No artists found.'))}</p>`;
      return;
    }

    list.innerHTML = artists
      .map((artist, idx) => `
      <a class="sh-card sh-artist-card" href="/artists/${encodeURIComponent(artist.id)}" data-artist-index="${idx}" data-detail-link>
        <img src="/api/covers/artist/${encodeURIComponent(artist.id)}" alt="${escapeHtml(artist.name)}" class="sh-album-cover" loading="lazy" />
        <h3>${escapeHtml(artist.name)}</h3>
        <small>${escapeHtml(t('albums_count', 'Albums'))}: ${countMap.get(artist.id) || 0}</small>
        <div class="sh-actions">
          <button type="button" data-action="play-artist" data-artist-id="${escapeHtml(artist.id)}">${escapeHtml(t('play', 'Play'))}</button>
          <button type="button" data-action="fav-artist" data-artist-id="${escapeHtml(artist.id)}" title="${escapeHtml(t('toggle_favorite', 'Toggle favorite'))}">&#10084;</button>
        </div>
      </a>
    `)
      .join('');

    list.querySelectorAll('[data-action="play-artist"]').forEach((button) => {
      button.addEventListener('click', async (event) => {
        event.preventDefault();
        event.stopPropagation();
        const artistID = button.getAttribute('data-artist-id');
        button.disabled = true;
        try {
          const queue = await buildArtistQueue(artistID);
          if (!queue.length) return;
          context.player.replaceQueueFromTracks(queue, 0, 'artist', artistID, true);
        } finally {
          button.disabled = false;
        }
      });
    });

    list.querySelectorAll('[data-action="fav-artist"]').forEach((button) => {
      button.addEventListener('click', async (event) => {
        event.preventDefault();
        event.stopPropagation();
        const artistID = button.getAttribute('data-artist-id');
        await API.toggleFavoriteArtist(artistID);
        button.classList.toggle('active');
      });
    });

    list.querySelectorAll('[data-artist-index]').forEach((card) => {
      card.addEventListener('click', async (event) => {
        if (event.target.closest('button')) return;
        event.preventDefault();
        const href = card.getAttribute('href');
        if (href) {
          await context.router?.go(href);
        }
      });
      card.addEventListener('contextmenu', async (event) => {
        event.preventDefault();
        const idx = Number(card.getAttribute('data-artist-index'));
        const artist = artists[idx];
        showContextMenu(event, [
          {
            label: t('edit', 'Edit'),
            action: async () => {
              openEditModal(artist);
            }
          },
          {
            label: t('play', 'Play'),
            action: async () => {
              const queue = await buildArtistQueue(artist.id);
              if (!queue.length) return;
              context.player.replaceQueueFromTracks(queue, 0, 'artist', artist.id, true);
            }
          },
          {
            label: t('add_to_queue', 'Add to queue'),
            action: async () => {
              const queue = await buildArtistQueue(artist.id);
              if (!queue.length) return;
              context.player.appendTracks(queue);
            }
          },
          { label: t('view', 'View'), action: async () => context.router?.go(`/artists/${encodeURIComponent(artist.id)}`) },
          {
            label: t('delete', 'Delete'),
            danger: true,
            action: async () => {
              await API.deleteArtist(artist.id);
              closeModal();
              await load();
            }
          }
        ]);
      });
    });
  }

  async function openModal(artist, editable) {
    if (!artist) return;
    selected = artist;
    modalNameView.textContent = artist.name || '';
    modalName.value = artist.name || '';
    modalID.textContent = artist.id || '-';
    modalCover.src = `/api/covers/artist/${encodeURIComponent(artist.id)}`;
    modalCover.onerror = () => {
      modalCover.src = '/static/logo.png';
    };

    const albumsRaw = await API.getArtistAlbums(artist.id, { limit: 500, offset: 0, sort: 'year' });
    const albums = Array.isArray(albumsRaw) ? albumsRaw : [];
    modalAlbums.innerHTML = albums.length
      ? albums.map((item) => `
        <article class="sh-artist-album-card" data-album-id="${escapeHtml(item.id || item.ID || '')}">
          <img src="${API.albumCoverThumbUrl(item.id || item.ID || '', 256)}" alt="${escapeHtml(item.title || item.Title || '')}" class="sh-artist-album-cover" />
          <div class="sh-artist-album-meta">
            <strong>${escapeHtml(item.title || item.Title || '')}</strong>
            <small>${escapeHtml(String(item.year || item.Year || ''))}</small>
          </div>
        </article>
      `).join('')
      : `<p>${escapeHtml(t('empty_no_albums', 'No albums found.'))}</p>`;

    modalAlbums.querySelectorAll('[data-album-id]').forEach((card) => {
      card.addEventListener('click', async () => {
        const albumID = card.getAttribute('data-album-id') || '';
        if (!albumID) return;
        await context.router?.go(`/albums/${encodeURIComponent(albumID)}`);
      });
    });

    setEditMode(Boolean(editable));
    modalIDRow.hidden = !editable;
    modal.hidden = false;
    document.body.classList.add('sh-modal-open');
  }

  function closeModal() {
    modal.hidden = true;
    document.body.classList.remove('sh-modal-open');
    selected = null;
  }

  modalClose?.addEventListener('click', closeModal);
  modal?.addEventListener('click', (event) => {
    if (event.target === modal) closeModal();
  });

  modalEdit?.addEventListener('click', () => {
    setEditMode(!editMode);
  });

  modalSave?.addEventListener('click', async () => {
    if (!editMode || !selected) return;
    const name = modalName.value.trim();
    if (!name) return;
    await API.updateArtist(selected.id, name);
    closeModal();
    await load();
  });

  modalDelete?.addEventListener('click', async () => {
    if (!editMode || !selected) return;
    await API.deleteArtist(selected.id);
    closeModal();
    await load();
  });

  function setEditMode(enabled) {
    editMode = Boolean(enabled);
    modalName.hidden = !editMode;
    modalNameView.hidden = editMode;
    modalName.disabled = !editMode;
    modalSave.disabled = !editMode;
    modalDelete.disabled = !editMode;
    modalIDRow.hidden = !editMode;
    if (modalEdit) {
      modalEdit.classList.toggle('active', editMode);
    }
  }

  function openEditModal(artist) {
    if (!artist?.id || !editModal || !editName || !editMergeTarget) return;
    editingArtist = artist;
    editName.value = artist.name || '';
    editMergeTarget.innerHTML = [
      `<option value="">${escapeHtml(t('artist_merge_none', 'Do not merge'))}</option>`,
      ...artists
        .filter((item) => item.id !== artist.id)
        .map((item) => `<option value="${escapeHtml(item.id)}">${escapeHtml(item.name)}</option>`)
    ].join('');
    editMergeTarget.value = '';
    editModal.hidden = false;
    document.body.classList.add('sh-modal-open');
    queueMicrotask(() => editName.focus());
  }

  function closeEditModal() {
    if (!editModal) return;
    editModal.hidden = true;
    document.body.classList.remove('sh-modal-open');
    editingArtist = null;
  }

  editClose?.addEventListener('click', closeEditModal);
  editCancel?.addEventListener('click', closeEditModal);
  editModal?.addEventListener('click', (event) => {
    if (event.target === editModal) closeEditModal();
  });

  editForm?.addEventListener('submit', async (event) => {
    event.preventDefault();
    if (!editingArtist?.id) return;
    const nextName = String(editName?.value || '').trim();
    const mergeTargetId = String(editMergeTarget?.value || '').trim();
    if (!nextName) return;

    await API.updateArtist(editingArtist.id, {
      name: nextName,
      existing_artist_id: mergeTargetId || undefined
    });

    closeEditModal();
    await load();
    window.dispatchEvent(new CustomEvent('soundhub:library-updated'));
  });

  await load();

  async function buildArtistQueue(artistID) {
    const artistNameMap = await loadArtistNameMap();
    const albumsRaw = await API.getArtistAlbums(artistID, { limit: 500, offset: 0, sort: 'year' });
    const albumIDs = (Array.isArray(albumsRaw) ? albumsRaw : []).map((item) => String(item.id || item.ID || ''));
    const tracks = [];
    for (const albumID of albumIDs) {
      const raw = await API.getAlbumTracks(albumID, { limit: 500, offset: 0, sort: 'name' });
      tracks.push(...normalizeTracks(raw, artistNameMap));
    }
    if (tracks.length > 1) return tracks;

    const allTracksRaw = await API.getTracks({ limit: 5000, offset: 0, sort: 'name' });
    const allTracks = normalizeTracks(allTracksRaw, artistNameMap);
    return allTracks.filter((track) => track.artistId === artistID);
  }
}

function normalizeArtists(raw) {
  return (Array.isArray(raw) ? raw : []).map((item) => ({
    id: item.id || item.ID || '',
    name: item.name || item.Name || t('unknown_artist', 'Unknown artist')
  }));
}

function normalizeTracks(raw, artistMap) {
  return (Array.isArray(raw) ? raw : []).map((track) => ({
    id: track.id || track.ID || '',
    title: track.title || track.Title || '',
    artist: resolveArtistName(track, artistMap),
    artistId: track.artistId || track.ArtistID || '',
    albumId: track.albumId || track.AlbumID || ''
  }));
}

function escapeHtml(value) {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

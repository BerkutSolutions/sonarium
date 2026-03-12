import { API } from './api.js';
import { t } from './i18n.js';
import { loadArtistNameMap, resolveArtistName } from './artist-map.js';
import { showContextMenu } from './context-menu.js';

export async function renderAlbums(context, root) {
  const list = root.querySelector('#albums-list');
  const modal = root.querySelector('#album-modal');
  const modalClose = root.querySelector('#album-modal-close');
  const modalCover = root.querySelector('#album-modal-cover');
  const modalCoverInput = root.querySelector('#album-cover-input');
  const modalTitleView = root.querySelector('#album-modal-title-view');
  const modalTitleInput = root.querySelector('#album-modal-title');
  const modalTitleEditBtn = root.querySelector('#album-name-edit-btn');
  const modalArtist = root.querySelector('#album-modal-artist');
  const modalYear = root.querySelector('#album-modal-year');
  const modalID = root.querySelector('#album-modal-id');
  const modalIDRow = root.querySelector('#album-modal-id-row');
  const modalHeading = root.querySelector('#album-modal-heading');
  const modalTracks = root.querySelector('#album-modal-tracks');
  const modalTracksCount = root.querySelector('#album-modal-tracks-count');
  const modalSearch = root.querySelector('#album-modal-track-search');
  const modalSearchResults = root.querySelector('#album-modal-search-results');
  const modalEdit = root.querySelector('#album-modal-edit');
  const modalSave = root.querySelector('#album-modal-save');
  const modalDelete = root.querySelector('#album-modal-delete');
  const modalPlay = root.querySelector('#album-modal-play');
  const modalQueue = root.querySelector('#album-modal-queue');
  const mergeModal = root.querySelector('#album-merge-modal');
  const mergeForm = root.querySelector('#album-merge-form');
  const mergeClose = root.querySelector('#album-merge-close');
  const mergeCancel = root.querySelector('#album-merge-cancel');
  const mergeSource = root.querySelector('#album-merge-source');
  const mergeTarget = root.querySelector('#album-merge-target');

  let albums = [];
  let allTracks = [];
  let artistMap = new Map();
  let selectedAlbum = null;
  let selectedAlbumTracks = [];
  let editMode = false;
  let mergeSourceAlbum = null;

  const pendingAlbumID = String(window.__soundhubOpenAlbumId || '').trim();
  if (pendingAlbumID) {
    delete window.__soundhubOpenAlbumId;
  }

  async function loadData() {
    const [response, artists, tracksRaw] = await Promise.all([
      API.getAlbums({ limit: 500, offset: 0, sort: 'year' }),
      loadArtistNameMap(),
      API.getTracks({ limit: 3000, offset: 0, sort: 'name' })
    ]);
    artistMap = artists;
    albums = normalizeAlbums(response, artistMap);
    allTracks = normalizeTracks(tracksRaw, artistMap);
    renderList();
    if (pendingAlbumID) {
      const target = albums.find((item) => String(item.id) === pendingAlbumID);
      if (target) {
        await openModal(target, 'content');
      }
    }
  }

  function renderList() {
    if (!albums.length) {
      list.innerHTML = `<p>${escapeHtml(t('empty_no_albums', 'No albums found.'))}</p>`;
      return;
    }

    list.innerHTML = albums.map((album, idx) => `
      <a class="sh-card sh-album-card" href="/albums/${encodeURIComponent(album.id)}" data-album-index="${idx}" data-detail-link>
        <div class="sh-album-cover-wrap">
          <img src="${coverSrc(album.id, album.title)}" alt="${escapeHtml(album.title)} ${escapeHtml(t('cover', 'cover'))}" class="sh-album-cover" loading="lazy" />
          <div class="sh-album-cover-actions">
            <button type="button" class="sh-cover-play-btn" data-action="play-album" data-album-id="${escapeHtml(album.id)}" aria-label="${escapeHtml(t('play', 'Play'))}">&#9654;</button>
          </div>
          <div class="sh-cover-menu-slot">
            <button type="button" class="sh-cover-menu-btn" data-action="album-menu" data-album-index="${idx}" aria-label="Menu">&#8942;</button>
          </div>
        </div>
        <h3>${escapeHtml(album.title)}</h3>
        <div>${escapeHtml(t('year', 'Year'))}: ${album.year || '-'}</div>
        <small>${escapeHtml(album.artist || t('unknown_artist', 'Unknown artist'))}</small>
      </a>
    `).join('');

    list.querySelectorAll('[data-action="play-album"]').forEach((button) => {
      button.addEventListener('click', async (event) => {
        event.preventDefault();
        event.stopPropagation();
        const albumID = button.getAttribute('data-album-id');
        const tracks = await loadAlbumTracks(albumID);
        if (!tracks.length) return;
        context.player.replaceQueueFromTracks(tracks, 0, 'album', albumID, true);
      });
    });

    list.querySelectorAll('[data-action="album-menu"]').forEach((button) => {
      button.addEventListener('click', (event) => {
        event.preventDefault();
        event.stopPropagation();
        const idx = Number(button.getAttribute('data-album-index'));
        const album = albums[idx];
        if (!album) return;
        showContextMenu(event, buildAlbumContextMenu(album));
      });
    });

    list.querySelectorAll('[data-album-index]').forEach((card) => {
      card.addEventListener('click', async (event) => {
        if (event.target.closest('button')) return;
        event.preventDefault();
        const href = card.getAttribute('href');
        if (href) {
          await context.router?.go(href);
        }
      });
      card.addEventListener('contextmenu', (event) => {
        event.preventDefault();
        const idx = Number(card.getAttribute('data-album-index'));
        const album = albums[idx];
        if (!album) return;
        showContextMenu(event, buildAlbumContextMenu(album));
      });
    });
  }

  function buildAlbumContextMenu(album) {
    return [
      { label: t('view', 'View'), action: async () => context.router?.go(`/albums/${encodeURIComponent(album.id)}`) },
      {
        label: t('add_to_queue', 'Add to queue'),
        action: async () => {
          const tracks = await loadAlbumTracks(album.id);
          context.player.appendTracks(tracks);
        }
      },
      {
        label: t('merge', 'Merge'),
        action: async () => {
          openMergeModal(album);
        }
      },
      {
        label: t('delete', 'Delete'),
        danger: true,
        action: async () => {
          await API.deleteAlbum(album.id);
          closeModal();
          await loadData();
        }
      }
    ];
  }

  function openMergeModal(album) {
    if (!album?.id || !mergeModal || !mergeSource || !mergeTarget) return;
    mergeSourceAlbum = album;
    mergeSource.value = albumLabel(album);
    const candidates = albums.filter((item) => item.id !== album.id);
    mergeTarget.innerHTML = [
      `<option value="">${escapeHtml(t('select_album', 'Select album'))}</option>`,
      ...candidates.map((item) => `<option value="${escapeHtml(item.id)}">${escapeHtml(albumLabel(item))}</option>`)
    ].join('');
    mergeTarget.value = '';
    mergeModal.hidden = false;
    document.body.classList.add('sh-modal-open');
    queueMicrotask(() => mergeTarget.focus());
  }

  function closeMergeModal() {
    if (!mergeModal) return;
    mergeModal.hidden = true;
    document.body.classList.remove('sh-modal-open');
    mergeSourceAlbum = null;
  }

  async function openModal(album, mode = 'content') {
    if (!album) return;
    selectedAlbum = album;
    modalTitleView.textContent = album.title || t('unnamed', 'Unnamed');
    modalTitleInput.value = album.title || '';
    modalYear.value = album.year || 0;
    modalID.textContent = album.id || '-';
    modalCover.src = API.albumCoverThumbUrl(album.id || '', 256);
    modalCover.onerror = () => { modalCover.src = '/static/logo.png'; };

    modalArtist.innerHTML = Array.from(artistMap.entries())
      .map(([id, name]) => `<option value="${escapeHtml(id)}">${escapeHtml(name)}</option>`)
      .join('');
    modalArtist.value = album.artistId || '';

    selectedAlbumTracks = await loadAlbumTracks(album.id);
    modalTracksCount.textContent = String(selectedAlbumTracks.length);
    renderAlbumTracks();
    renderSearchResults(modalSearch.value || '');

    editMode = mode === 'edit';
    setEditMode(editMode);
    modalIDRow.hidden = mode !== 'info';
    modalHeading.textContent = mode === 'info' ? t('album_info', 'Album Info') : t('album_contents', 'Album');

    modal.hidden = false;
    document.body.classList.add('sh-modal-open');
  }

  function renderAlbumTracks() {
    if (!selectedAlbumTracks.length) {
      modalTracks.innerHTML = `<p>${escapeHtml(t('empty_no_tracks', 'No tracks found.'))}</p>`;
      return;
    }

    modalTracks.innerHTML = selectedAlbumTracks.map((track) => `
      <article class="sh-ya-track-row" data-track-id="${escapeHtml(track.id)}">
        <div class="sh-ya-track-left">
          <img src="${API.albumCoverThumbUrl(track.albumId || selectedAlbum?.id || '', 64)}" alt="${escapeHtml(track.title)}" class="sh-ya-track-cover" />
          <div>
            <div class="sh-ya-track-title">${escapeHtml(track.title || t('unknown_title', 'Unknown title'))}</div>
            <div class="sh-ya-track-artist">${escapeHtml(track.artist || t('unknown_artist', 'Unknown artist'))}</div>
          </div>
        </div>
        <div class="sh-ya-track-right">
          <span class="sh-ya-track-duration">${escapeHtml(formatDuration(track.durationSeconds || 0))}</span>
          <div class="sh-ya-track-hover-actions">
            <button type="button" class="sh-ya-icon-btn" data-action="favorite" title="${escapeHtml(t('toggle_favorite', 'Toggle favorite'))}">&#9825;</button>
            <button type="button" class="sh-ya-icon-btn" data-action="menu" title="Menu">&#8942;</button>
          </div>
        </div>
      </article>
    `).join('');

    modalTracks.querySelectorAll('[data-action="favorite"]').forEach((button) => {
      button.addEventListener('click', async () => {
        const row = button.closest('.sh-ya-track-row');
        const trackID = row?.getAttribute('data-track-id') || '';
        if (!trackID) return;
        await API.toggleFavoriteTrack(trackID);
        button.classList.toggle('active');
      });
    });

    modalTracks.querySelectorAll('[data-action="menu"]').forEach((button) => {
      button.addEventListener('click', (event) => {
        const row = button.closest('.sh-ya-track-row');
        const trackID = row?.getAttribute('data-track-id') || '';
        const track = selectedAlbumTracks.find((item) => String(item.id) === String(trackID));
        if (!track) return;
        showContextMenu(event, buildTrackMenu(track));
      });
    });
  }

  function buildTrackMenu(track) {
    return [
      { label: 'Р В Р’В Р РЋРЎС™Р В Р Р‹Р В РІР‚С™Р В Р’В Р вЂ™Р’В°Р В Р’В Р В РІР‚В Р В Р’В Р РЋРІР‚ВР В Р Р‹Р Р†Р вЂљРЎв„ўР В Р Р‹Р В РЎвЂњР В Р Р‹Р В Р РЏ', action: async () => API.toggleFavoriteTrack(track.id) },
      { label: 'Р В Р’В Р вЂ™Р’ВР В Р’В Р РЋРІР‚вЂњР В Р Р‹Р В РІР‚С™Р В Р’В Р вЂ™Р’В°Р В Р Р‹Р Р†Р вЂљРЎв„ўР В Р Р‹Р В Р вЂ° Р В Р Р‹Р В РЎвЂњР В Р’В Р вЂ™Р’В»Р В Р’В Р вЂ™Р’ВµР В Р’В Р СћРІР‚ВР В Р Р‹Р РЋРІР‚СљР В Р Р‹Р В РІР‚в„–Р В Р Р‹Р Р†Р вЂљР’В°Р В Р’В Р РЋРІР‚ВР В Р’В Р РЋР’В', action: async () => context.player.addNext(track) },
      { label: 'Р В Р’В Р Р†Р вЂљРЎСљР В Р’В Р РЋРІР‚СћР В Р’В Р вЂ™Р’В±Р В Р’В Р вЂ™Р’В°Р В Р’В Р В РІР‚В Р В Р’В Р РЋРІР‚ВР В Р Р‹Р Р†Р вЂљРЎв„ўР В Р Р‹Р В Р вЂ° Р В Р’В Р В РІР‚В  Р В Р’В Р РЋРІР‚СњР В Р’В Р РЋРІР‚СћР В Р’В Р В РІР‚В¦Р В Р’В Р вЂ™Р’ВµР В Р Р‹Р Р†Р вЂљР’В  Р В Р’В Р РЋРІР‚СћР В Р Р‹Р Р†Р вЂљР Р‹Р В Р’В Р вЂ™Р’ВµР В Р Р‹Р В РІР‚С™Р В Р’В Р вЂ™Р’ВµР В Р’В Р СћРІР‚ВР В Р’В Р РЋРІР‚В', action: async () => context.player.appendTracks([track]) },
      { label: 'Р В Р’В Р РЋРЎС™Р В Р’В Р вЂ™Р’Вµ Р В Р’В Р В РІР‚В¦Р В Р Р‹Р В РІР‚С™Р В Р’В Р вЂ™Р’В°Р В Р’В Р В РІР‚В Р В Р’В Р РЋРІР‚ВР В Р Р‹Р Р†Р вЂљРЎв„ўР В Р Р‹Р В РЎвЂњР В Р Р‹Р В Р РЏ', action: async () => API.toggleFavoriteTrack(track.id) },
      { label: 'Р В Р’В Р РЋРЎСџР В Р’В Р вЂ™Р’ВµР В Р Р‹Р В РІР‚С™Р В Р’В Р вЂ™Р’ВµР В Р’В Р Р†РІР‚С›РІР‚вЂњР В Р Р‹Р Р†Р вЂљРЎв„ўР В Р’В Р РЋРІР‚В Р В Р’В Р РЋРІР‚Сњ Р В Р’В Р вЂ™Р’В°Р В Р’В Р вЂ™Р’В»Р В Р Р‹Р В Р вЂ°Р В Р’В Р вЂ™Р’В±Р В Р’В Р РЋРІР‚СћР В Р’В Р РЋР’ВР В Р Р‹Р РЋРІР‚Сљ', action: async () => { if (selectedAlbum?.id) await context.router?.go(`/albums/${encodeURIComponent(selectedAlbum.id)}`); } },
      { label: 'Р В Р’В Р РЋРЎСџР В Р’В Р вЂ™Р’ВµР В Р Р‹Р В РІР‚С™Р В Р’В Р вЂ™Р’ВµР В Р’В Р Р†РІР‚С›РІР‚вЂњР В Р Р‹Р Р†Р вЂљРЎв„ўР В Р’В Р РЋРІР‚В Р В Р’В Р РЋРІР‚Сњ Р В Р’В Р РЋРІР‚ВР В Р Р‹Р В РЎвЂњР В Р’В Р РЋРІР‚вЂќР В Р’В Р РЋРІР‚СћР В Р’В Р вЂ™Р’В»Р В Р’В Р В РІР‚В¦Р В Р’В Р РЋРІР‚ВР В Р Р‹Р Р†Р вЂљРЎв„ўР В Р’В Р вЂ™Р’ВµР В Р’В Р вЂ™Р’В»Р В Р Р‹Р В РІР‚в„–', action: async () => { if (track.artistId) await context.router?.go(`/artists/${encodeURIComponent(track.artistId)}`); } },
      { label: 'Р В Р’В Р РЋРІР‚С” Р В Р Р‹Р Р†Р вЂљРЎв„ўР В Р Р‹Р В РІР‚С™Р В Р’В Р вЂ™Р’ВµР В Р’В Р РЋРІР‚СњР В Р’В Р вЂ™Р’Вµ', action: async () => { if (track.id) await context.router?.go(`/tracks/${encodeURIComponent(track.id)}`); } }
    ];
  }

  function renderSearchResults(query) {
    const q = String(query || '').trim().toLowerCase();
    if (!q || !selectedAlbum) {
      modalSearchResults.innerHTML = '';
      return;
    }

    const selectedIDs = new Set(selectedAlbumTracks.map((item) => item.id));
    const matches = allTracks
      .filter((track) => !selectedIDs.has(track.id))
      .filter((track) => {
        const title = String(track.title || '').toLowerCase();
        const artist = String(track.artist || '').toLowerCase();
        return title.includes(q) || artist.includes(q);
      })
      .slice(0, 12);

    if (!matches.length) {
      modalSearchResults.innerHTML = `<p>${escapeHtml(t('empty_no_results', 'No results'))}</p>`;
      return;
    }

    modalSearchResults.innerHTML = matches.map((track) => `
      <div class="sh-list-row" data-action="add-to-album" data-track-id="${escapeHtml(track.id)}">
        <span>${escapeHtml(track.title)} - ${escapeHtml(track.artist || t('unknown_artist', 'Unknown artist'))}</span>
      </div>
    `).join('');

    modalSearchResults.querySelectorAll('[data-action="add-to-album"]').forEach((row) => {
      row.addEventListener('click', async () => {
        if (!selectedAlbum) return;
        const trackID = row.getAttribute('data-track-id') || '';
        const track = allTracks.find((item) => item.id === trackID);
        if (!track) return;
        await API.updateTrack(track.id, {
          title: track.title,
          album_id: selectedAlbum.id,
          artist_id: selectedAlbum.artistId || track.artistId
        });
        modalSearch.value = '';
        await openModal(selectedAlbum, editMode ? 'edit' : 'content');
      });
    });
  }

  function closeModal() {
    modal.hidden = true;
    document.body.classList.remove('sh-modal-open');
    selectedAlbum = null;
    selectedAlbumTracks = [];
  }

  modalClose?.addEventListener('click', closeModal);
  modal?.addEventListener('click', (event) => {
    if (event.target === modal) closeModal();
  });

  modalTitleEditBtn?.addEventListener('click', () => {
    editMode = true;
    setEditMode(true);
  });

  modalEdit?.addEventListener('click', () => {
    editMode = !editMode;
    setEditMode(editMode);
  });

  modalSave?.addEventListener('click', async () => {
    if (!selectedAlbum) return;
    const title = modalTitleInput.value.trim();
    const artistID = modalArtist.value;
    const year = Number.parseInt(modalYear.value || '0', 10) || 0;
    if (!title || !artistID) return;
    await API.updateAlbum(selectedAlbum.id, { title, artist_id: artistID, year });
    selectedAlbum.title = title;
    selectedAlbum.artistId = artistID;
    selectedAlbum.year = year;
    modalTitleView.textContent = title;
    setEditMode(false);
    await loadData();
  });

  modalDelete?.addEventListener('click', async () => {
    if (!selectedAlbum) return;
    await API.deleteAlbum(selectedAlbum.id);
    closeModal();
    await loadData();
  });

  modalPlay?.addEventListener('click', () => {
    if (!selectedAlbum?.id || !selectedAlbumTracks.length) return;
    context.player.replaceQueueFromTracks(selectedAlbumTracks, 0, 'album', selectedAlbum.id, true);
  });

  modalQueue?.addEventListener('click', () => {
    context.player.appendTracks(selectedAlbumTracks);
  });

  modalSearch?.addEventListener('input', () => {
    renderSearchResults(modalSearch.value);
  });

  modalCoverInput?.addEventListener('change', () => {
    const file = modalCoverInput.files?.[0];
    if (!file) return;
    const url = URL.createObjectURL(file);
    modalCover.src = url;
  });

  mergeClose?.addEventListener('click', closeMergeModal);
  mergeCancel?.addEventListener('click', closeMergeModal);
  mergeModal?.addEventListener('click', (event) => {
    if (event.target === mergeModal) closeMergeModal();
  });
  mergeForm?.addEventListener('submit', async (event) => {
    event.preventDefault();
    if (!mergeSourceAlbum?.id) return;
    const targetAlbumId = String(mergeTarget?.value || '').trim();
    if (!targetAlbumId) return;
    await API.mergeAlbum(mergeSourceAlbum.id, targetAlbumId);
    closeMergeModal();
    closeModal();
    await loadData();
    window.dispatchEvent(new CustomEvent('soundhub:library-updated'));
  });

  function setEditMode(enabled) {
    const next = Boolean(enabled);
    modalTitleInput.hidden = !next;
    modalTitleView.hidden = next;
    modalTitleEditBtn.style.visibility = next ? 'hidden' : 'visible';
    modalArtist.disabled = !next;
    modalYear.disabled = !next;
    modalSave.disabled = !next;
    if (next) {
      modalTitleInput.focus();
      modalTitleInput.select();
    }
  }

  await loadData();
}

async function loadAlbumTracks(albumID) {
  const artistMap = await loadArtistNameMap();
  const tracksRaw = await API.getAlbumTracks(albumID, { limit: 500, offset: 0, sort: 'name' });
  return normalizeTracks(tracksRaw, artistMap);
}

function normalizeAlbums(response, artistMap) {
  return (Array.isArray(response) ? response : []).map((album) => ({
    id: String(album.id || album.ID || '').trim(),
    title: String(album.title || album.Title || '').trim(),
    year: Number(album.year || album.Year || 0) || 0,
    artistId: String(album.artistId || album.ArtistID || '').trim(),
    artist: artistMap.get(String(album.artistId || album.ArtistID || '').trim()) || '',
  }));
}

function normalizeTracks(response, artistMap) {
  return (Array.isArray(response) ? response : []).map((track) => ({
    id: String(track.id || track.ID || '').trim(),
    title: String(track.title || track.Title || '').trim(),
    artist: resolveArtistName(track, artistMap),
    artistId: String(track.artistId || track.ArtistID || '').trim(),
    albumId: String(track.albumId || track.AlbumID || '').trim(),
    durationSeconds: normalizeDurationSeconds(track)
  }));
}

function formatDuration(seconds) {
  const safe = Math.max(0, Number.parseInt(String(seconds || 0), 10) || 0);
  const minutes = Math.floor(safe / 60);
  const sec = safe % 60;
  return `${minutes}:${String(sec).padStart(2, '0')}`;
}

function normalizeDurationSeconds(track) {
  const raw = Number(track.duration_seconds ?? track.durationSeconds ?? track.DurationSeconds ?? track.duration ?? track.Duration ?? 0);
  if (!Number.isFinite(raw) || raw <= 0) return 0;
  if (raw > 1000000) return Math.round(raw / 1e9);
  return Math.round(raw);
}

function escapeHtml(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

function coverSrc(albumId, fallbackLabel) {
  if (albumId) {
    return API.albumCoverThumbUrl(albumId, 256);
  }
  return API.placeholderCoverUrl(fallbackLabel || 'Album');
}

function albumLabel(album) {
  const title = String(album?.title || t('unnamed', 'Unnamed')).trim();
  const artist = String(album?.artist || t('unknown_artist', 'Unknown artist')).trim();
  const year = Number(album?.year || 0) || 0;
  return year ? `${title} · ${artist} · ${year}` : `${title} · ${artist}`;
}


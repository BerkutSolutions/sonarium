import { API } from './api.js';
import { t } from './i18n.js';
import { loadArtistNameMap, resolveArtistName } from './artist-map.js';
import { showContextMenu } from './context-menu.js';

export async function renderPlaylists(context, root) {
  const list = root.querySelector('#playlists-list');
  const form = root.querySelector('#create-playlist-form');
  const input = root.querySelector('#playlist-name');

  const modal = root.querySelector('#playlist-modal');
  const modalClose = root.querySelector('#playlist-modal-close');
  const modalHeading = root.querySelector('#playlist-modal-heading');
  const modalCover = root.querySelector('#playlist-modal-cover');
  const modalCoverInput = root.querySelector('#playlist-cover-input');
  const modalID = root.querySelector('#playlist-modal-id');
  const modalIDRow = root.querySelector('#playlist-modal-id-row');
  const modalTitleView = root.querySelector('#playlist-modal-title-view');
  const modalNameInput = root.querySelector('#playlist-modal-name');
  const modalNameEditBtn = root.querySelector('#playlist-name-edit-btn');
  const modalTracks = root.querySelector('#playlist-modal-tracks');
  const modalSearch = root.querySelector('#playlist-modal-track-search');
  const modalSearchResults = root.querySelector('#playlist-modal-search-results');
  const modalEdit = root.querySelector('#playlist-modal-edit');
  const modalSave = root.querySelector('#playlist-modal-save');
  const modalDelete = root.querySelector('#playlist-modal-delete');
  const modalPlay = root.querySelector('#playlist-modal-play');
  const modalQueue = root.querySelector('#playlist-modal-queue');

  let artistMap = await loadArtistNameMap();
  let playlists = [];
  let selected = null;
  let selectedTracks = [];
  let allTracks = [];
  let editMode = false;

  async function reload() {
    const [playlistsRaw, home] = await Promise.all([
      API.getPlaylists({ limit: 200, offset: 0, sort: 'name' }),
      API.getHome(500)
    ]);
    playlists = normalizePlaylists(playlistsRaw);
    const favoriteTracks = normalizeTracks(home?.favorites?.tracks || [], artistMap);
    playlists = injectFavoritesPlaylist(playlists, favoriteTracks);

    if (!playlists.length) {
      list.className = 'sh-list';
      list.innerHTML = `<p>${escapeHtml(t('empty_no_playlists', 'No playlists yet.'))}</p>`;
      return;
    }

    list.className = 'sh-grid';
    list.innerHTML = playlists.map((playlist, idx) => `
      <a class="sh-card sh-album-card sh-playlist-card" href="/playlists/${encodeURIComponent(playlist.id)}" data-playlist-index="${idx}" data-detail-link>
        <div class="sh-album-cover-wrap">
          <img src="${playlistCoverSrc(playlist)}" alt="${escapeHtml(playlist.name)}" class="sh-album-cover" loading="lazy" />
          <div class="sh-album-cover-actions">
            <button type="button" class="sh-cover-play-btn" data-action="play-playlist" data-playlist-id="${escapeHtml(playlist.id)}" aria-label="${escapeHtml(t('play', 'Play'))}">&#9654;</button>
          </div>
          <div class="sh-cover-menu-slot">
            <button type="button" class="sh-cover-menu-btn" data-action="playlist-menu" data-playlist-index="${idx}" aria-label="Menu">&#8942;</button>
          </div>
        </div>
        <h3>${escapeHtml(playlist.name)}</h3>
        <div>${escapeHtml(playlist.system ? t('favorites', 'Favorites') : t('playlist_contents', 'Playlist'))}</div>
        <small>${escapeHtml(playlist.system ? t('favorite_tracks', 'Favorite tracks') : t('playlist_contents', 'Playlist'))}</small>
      </a>
    `).join('');

    bindPlaylistListActions();
  }

  function bindPlaylistListActions() {
    list.querySelectorAll('[data-action="play-playlist"]').forEach((button) => {
      button.addEventListener('click', async (event) => {
        event.preventDefault();
        event.stopPropagation();
        const playlistID = button.getAttribute('data-playlist-id');
        const playlist = playlists.find((item) => String(item.id) === String(playlistID));
        const tracks = playlist?.system
          ? (Array.isArray(playlist.tracks) ? playlist.tracks : [])
          : normalizeTracks((await API.getPlaylist(playlistID))?.tracks || [], artistMap);
        if (!tracks.length) return;
        context.player.replaceQueueFromTracks(tracks, 0, 'playlist', playlistID, true);
      });
    });

    list.querySelectorAll('[data-action="playlist-menu"]').forEach((button) => {
      button.addEventListener('click', (event) => {
        event.preventDefault();
        event.stopPropagation();
        const idx = Number(button.getAttribute('data-playlist-index'));
        const playlist = playlists[idx];
        if (!playlist) return;
        showContextMenu(event, buildPlaylistContextMenu(playlist));
      });
    });

    list.querySelectorAll('[data-playlist-index]').forEach((card) => {
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
        const idx = Number(card.getAttribute('data-playlist-index'));
        const playlist = playlists[idx];
        if (!playlist) return;
        showContextMenu(event, buildPlaylistContextMenu(playlist));
      });
    });
  }

  function buildPlaylistContextMenu(playlist) {
    const items = [
      { label: t('view', 'View'), action: async () => context.router?.go(`/playlists/${encodeURIComponent(playlist.id)}`) }
    ];
    if (!playlist.system) {
      items.push({
        label: t('delete', 'Delete'),
        danger: true,
        action: async () => {
          await API.deletePlaylist(playlist.id);
          closeModal();
          await reload();
        }
      });
    }
    return items;
  }

  async function openModal(playlist, mode = 'content') {
    if (!playlist) return;
    selected = playlist;
    modalTitleView.textContent = playlist.name || t('unnamed', 'Unnamed');
    modalNameInput.value = playlist.name || '';
    modalID.textContent = playlist.id || '-';
    modalIDRow.hidden = mode !== 'info';
    modalCover.src = playlistCoverSrc(playlist);

    if (playlist.system && playlist.id === '__favorites__') {
      const home = await API.getHome(500);
      selectedTracks = normalizeTracks(home?.favorites?.tracks || [], artistMap);
    } else {
      const payload = await API.getPlaylist(playlist.id);
      selectedTracks = normalizeTracks(payload?.tracks || [], artistMap);
    }
    allTracks = normalizeTracks(await API.getTracks({ limit: 3000, offset: 0, sort: 'name' }), artistMap);

    renderPlaylistTracks();
    renderSearchResults(modalSearch.value || '');

    editMode = mode === 'edit';
    setEditMode(editMode);
    modalHeading.textContent = mode === 'info' ? t('playlist_info', 'Playlist Info') : t('playlist_contents', 'Playlist');

    modal.hidden = false;
    document.body.classList.add('sh-modal-open');
  }

  function renderPlaylistTracks() {
    if (!selectedTracks.length) {
      modalTracks.innerHTML = `<p>${escapeHtml(t('empty_no_tracks', 'No tracks found.'))}</p>`;
      return;
    }

    modalTracks.innerHTML = selectedTracks.map((track) => `
      <article class="sh-ya-track-row" data-track-id="${escapeHtml(track.id)}">
        <div class="sh-ya-track-left">
          <img src="${trackCoverSrc(track)}" alt="${escapeHtml(track.title)}" class="sh-ya-track-cover" />
          <div>
            <div class="sh-ya-track-title">${escapeHtml(track.title || t('unknown_title', 'Unknown title'))}</div>
            <div class="sh-ya-track-artist">${escapeHtml(track.artist || t('unknown_artist', 'Unknown artist'))}</div>
          </div>
        </div>
        <div class="sh-ya-track-right">
          <span class="sh-ya-track-duration">${escapeHtml(formatDuration(track.durationSeconds || track.duration || 0))}</span>
          <div class="sh-ya-track-hover-actions">
            <button type="button" class="sh-ya-icon-btn" data-action="remove" title="${escapeHtml(t('delete', 'Delete'))}">&times;</button>
            <button type="button" class="sh-ya-icon-btn ${selected?.system ? 'active' : ''}" data-action="favorite" title="${escapeHtml(t('toggle_favorite', 'Toggle favorite'))}">${selected?.system ? '&#9829;' : '&#9825;'}</button>
            <button type="button" class="sh-ya-icon-btn" data-action="menu" title="Menu">&#8942;</button>
          </div>
        </div>
      </article>
    `).join('');

    modalTracks.querySelectorAll('[data-action="remove"]').forEach((button) => {
      button.addEventListener('click', async () => {
        const row = button.closest('.sh-ya-track-row');
        const trackID = row?.getAttribute('data-track-id') || '';
        if (!selected?.id || selected.system || !trackID) return;
        await API.removeTrackFromPlaylist(selected.id, trackID);
        await openModal(selected, editMode ? 'edit' : 'content');
      });
    });

    modalTracks.querySelectorAll('[data-action="favorite"]').forEach((button) => {
      button.addEventListener('click', async () => {
        const row = button.closest('.sh-ya-track-row');
        const trackID = row?.getAttribute('data-track-id') || '';
        if (!trackID) return;
        await API.toggleFavoriteTrack(trackID);
        button.classList.toggle('active');
        button.innerHTML = button.classList.contains('active') ? '&#9829;' : '&#9825;';
        if (selected?.system) {
          selectedTracks = selectedTracks.filter((item) => String(item.id) !== String(trackID));
          renderPlaylistTracks();
        }
      });
    });

    modalTracks.querySelectorAll('[data-action="menu"]').forEach((button) => {
      button.addEventListener('click', (event) => {
        const row = button.closest('.sh-ya-track-row');
        const trackID = row?.getAttribute('data-track-id') || '';
        const track = selectedTracks.find((item) => String(item.id) === String(trackID));
        if (!track) return;
        showContextMenu(event, buildTrackMenu(track));
      });
    });

    modalTracks.querySelectorAll('.sh-ya-track-row').forEach((row) => {
      row.addEventListener('contextmenu', (event) => {
        event.preventDefault();
        const trackID = row.getAttribute('data-track-id') || '';
        const track = selectedTracks.find((item) => String(item.id) === String(trackID));
        if (!track) return;
        showContextMenu(event, buildTrackMenu(track));
      });
    });
  }

  function buildTrackMenu(track) {
    return [
      { label: 'Р СњРЎР‚Р В°Р Р†Р С‘РЎвЂљРЎРѓРЎРЏ', action: async () => API.toggleFavoriteTrack(track.id) },
      { label: 'Р ВР С–РЎР‚Р В°РЎвЂљРЎРЉ РЎРѓР В»Р ВµР Т‘РЎС“РЎР‹РЎвЂ°Р С‘Р С', action: async () => context.player.addNext(track) },
      { label: 'Р вЂќР С•Р В±Р В°Р Р†Р С‘РЎвЂљРЎРЉ Р Р† Р С”Р С•Р Р…Р ВµРЎвЂ  Р С•РЎвЂЎР ВµРЎР‚Р ВµР Т‘Р С‘', action: async () => context.player.appendTracks([track]) },
      { label: 'Р СњР Вµ Р Р…РЎР‚Р В°Р Р†Р С‘РЎвЂљРЎРѓРЎРЏ', action: async () => API.toggleFavoriteTrack(track.id) },
      {
        label: 'Р вЂќР С•Р В±Р В°Р Р†Р С‘РЎвЂљРЎРЉ Р Р† Р С—Р В»Р ВµР в„–Р В»Р С‘РЎРѓРЎвЂљ',
        action: async () => {
          const all = normalizePlaylists(await API.getPlaylists({ limit: 200, offset: 0, sort: 'name' }));
          const name = window.prompt('Р СџР В»Р ВµР в„–Р В»Р С‘РЎРѓРЎвЂљ:', all.map((item) => item.name).join(', '));
          if (!name) return;
          const found = all.find((item) => item.name.toLowerCase() === name.toLowerCase());
          if (!found) return;
          await API.addTrackToPlaylist(found.id, track.id, 9999);
        }
      },
      { label: 'Р СџР ВµРЎР‚Р ВµР в„–РЎвЂљР С‘ Р С” Р В°Р В»РЎРЉР В±Р С•Р СРЎС“', action: async () => { if (track.albumId) await context.router?.go(`/albums/${encodeURIComponent(track.albumId)}`); } },
      { label: 'Р СџР ВµРЎР‚Р ВµР в„–РЎвЂљР С‘ Р С” Р С‘РЎРѓР С—Р С•Р В»Р Р…Р С‘РЎвЂљР ВµР В»РЎР‹', action: async () => { if (track.artistId) await context.router?.go(`/artists/${encodeURIComponent(track.artistId)}`); } },
      { label: 'Р С› РЎвЂљРЎР‚Р ВµР С”Р Вµ', action: async () => { window.alert(`${track.title}\n${track.artist}`); } }
    ];
  }

  function renderSearchResults(query) {
    const q = String(query || '').trim().toLowerCase();
    if (!q) {
      modalSearchResults.innerHTML = '';
      return;
    }
    const selectedIDs = new Set(selectedTracks.map((item) => item.id));
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
      <div class="sh-list-row" data-action="add-track" data-track-id="${escapeHtml(track.id)}">
        <span>${escapeHtml(track.title)} - ${escapeHtml(track.artist || t('unknown_artist', 'Unknown artist'))}</span>
      </div>
    `).join('');

    modalSearchResults.querySelectorAll('[data-action="add-track"]').forEach((row) => {
      row.addEventListener('click', async () => {
        if (!selected?.id || selected.system) return;
        const trackID = row.getAttribute('data-track-id') || '';
        if (!trackID) return;
        await API.addTrackToPlaylist(selected.id, trackID, selectedTracks.length + 1);
        modalSearch.value = '';
        await openModal(selected, editMode ? 'edit' : 'content');
      });
    });
  }

  function closeModal() {
    modal.hidden = true;
    document.body.classList.remove('sh-modal-open');
    selected = null;
    selectedTracks = [];
    allTracks = [];
  }

  modalClose?.addEventListener('click', closeModal);
  modal?.addEventListener('click', (event) => {
    if (event.target === modal) closeModal();
  });

  modalNameEditBtn?.addEventListener('click', () => {
    editMode = true;
    setEditMode(true);
  });

  modalEdit?.addEventListener('click', () => {
    editMode = !editMode;
    setEditMode(editMode);
  });

  modalSave?.addEventListener('click', async () => {
    if (!selected || selected.system) return;
    const name = modalNameInput.value.trim();
    if (!name) return;
    await API.renamePlaylist(selected.id, name);
    selected.name = name;
    modalTitleView.textContent = name;
    setEditMode(false);
    await reload();
  });

  modalDelete?.addEventListener('click', async () => {
    if (!selected || selected.system) return;
    await API.deletePlaylist(selected.id);
    closeModal();
    await reload();
  });

  modalPlay?.addEventListener('click', () => {
    if (!selected?.id || !selectedTracks.length) return;
    context.player.replaceQueueFromTracks(selectedTracks, 0, 'playlist', selected.id, true);
  });

  modalQueue?.addEventListener('click', () => {
    context.player.appendTracks(selectedTracks);
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

  form.addEventListener('submit', async (event) => {
    event.preventDefault();
    const name = input.value.trim();
    if (!name) return;
    await API.createPlaylist({ name });
    input.value = '';
    await reload();
  });

  function setEditMode(enabled) {
    const next = Boolean(enabled) && !selected?.system;
    modalNameInput.hidden = !next;
    modalTitleView.hidden = next;
    modalNameEditBtn.style.visibility = next ? 'hidden' : 'visible';
    modalSave.disabled = !next;
    modalDelete.disabled = !next;
    modalEdit.disabled = Boolean(selected?.system);
    if (selected?.system) {
      modalSave.disabled = true;
      modalDelete.disabled = true;
      modalNameEditBtn.style.visibility = 'hidden';
    }
    if (next) {
      modalNameInput.focus();
      modalNameInput.select();
    }
  }

  await reload();
}

function normalizePlaylists(raw) {
  return (Array.isArray(raw) ? raw : []).map((playlist) => ({
    id: String(playlist.id || playlist.ID || '').trim(),
    name: String(playlist.name || playlist.Name || t('unnamed', 'Unnamed')).trim(),
    system: false
  }));
}

function normalizeTracks(raw, artistMap) {
  return (Array.isArray(raw) ? raw : []).map((track) => ({
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

function injectFavoritesPlaylist(playlists, tracks) {
  const favorites = {
    id: '__favorites__',
    name: t('favorites', 'Favorites'),
    system: true,
    tracks: Array.isArray(tracks) ? tracks : []
  };
  const filtered = (Array.isArray(playlists) ? playlists : []).filter((item) => item.id !== favorites.id);
  return [favorites, ...filtered];
}

function playlistCoverSrc(playlist) {
  if (playlist?.system) {
    return API.placeholderCoverUrl(t('favorites', 'Favorites'));
  }
  return API.placeholderCoverUrl(playlist?.name || 'Playlist');
}

function trackCoverSrc(track) {
  if (track?.albumId) {
    return API.albumCoverThumbUrl(track.albumId, 64);
  }
  return API.placeholderCoverUrl(track?.title || 'Track');
}

function escapeHtml(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

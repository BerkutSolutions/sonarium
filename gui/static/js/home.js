import { API } from './api.js';
import { t } from './i18n.js';
import { showContextMenu } from './context-menu.js';

export async function renderHome(context, root) {
  const blocksRoot = root.querySelector('#home-blocks');
  const waveBtn = root.querySelector('#home-wave-btn');
  const waveGenreSelect = root.querySelector('#home-wave-genre');
  let selectedWaveGenre = '';
  if (waveBtn) {
    waveBtn.textContent = t('play', 'Play');
  }
  const home = await API.getHome(12);
  const blocks = [
    { id: 'recent_albums', title: t('home_recent_albums'), type: 'albums', items: home?.recent_albums || [] },
    { id: 'recent_tracks', title: t('home_recent_tracks'), type: 'tracks', items: home?.recent_tracks || [] },
    { id: 'continue_listening', title: t('home_continue_listening'), type: 'tracks', items: home?.continue_listening || [] },
    { id: 'random_albums', title: t('home_random_albums'), type: 'albums', items: home?.random_albums || [] },
    { id: 'favorite_albums', title: t('home_favorite_albums'), type: 'albums', items: home?.favorites?.albums || [] },
    { id: 'favorite_artists', title: t('home_favorite_artists', 'Favorite Artists'), type: 'artists', items: home?.favorites?.artists || [] }
  ];

  blocksRoot.innerHTML = blocks.map((block) => `
    <section class="sh-home-block">
      <h3>${escapeHtml(block.title)}</h3>
      <div class="sh-horizontal" data-block-id="${block.id}">
        ${renderItems(block.type, block.items, block.id)}
      </div>
    </section>
  `).join('');

  const blockMap = new Map(blocks.map((block) => [block.id, block]));

  blocksRoot.querySelectorAll('[data-action="play-album"]').forEach((button) => {
    button.addEventListener('click', async () => {
      const albumID = button.getAttribute('data-album-id');
      const tracks = await API.getAlbumTracks(albumID, { limit: 500, offset: 0, sort: 'name' });
      context.player.replaceQueueFromTracks(tracks || [], 0, 'album', albumID, true);
    });
  });
  blocksRoot.querySelectorAll('[data-action="play-track"]').forEach((button) => {
    button.addEventListener('click', async () => {
      const trackID = button.getAttribute('data-track-id');
      const tracks = (home?.continue_listening || []).concat(home?.recent_tracks || []);
      const currentIdx = tracks.findIndex((track) => String(track.id || track.ID) === trackID);
      const queue = tracks.length ? tracks : [{ id: trackID }];
      context.player.replaceQueueFromTracks(queue, currentIdx >= 0 ? currentIdx : 0, 'home', 'dashboard', true);
    });
  });
  blocksRoot.querySelectorAll('[data-action="play-artist"]').forEach((button) => {
    button.addEventListener('click', async () => {
      const artistID = button.getAttribute('data-artist-id');
      const queue = await buildArtistQueue(artistID);
      if (!queue.length) return;
      context.player.replaceQueueFromTracks(queue, 0, 'artist', artistID, true);
    });
  });

  blocksRoot.querySelectorAll('[data-home-item]').forEach((card) => {
    card.addEventListener('click', async (event) => {
      if (event.target && event.target.closest('button')) return;
      event.preventDefault();
      const type = card.getAttribute('data-home-type');
      const id = card.getAttribute('data-home-id');
      const blockID = card.getAttribute('data-home-block');
      const block = blockMap.get(blockID);
      const blockItems = Array.isArray(block?.items) ? block.items : [];
      if (!type || !id) return;
      await openDetail(context, type, id, blockItems);
    });
    card.addEventListener('contextmenu', async (event) => {
      event.preventDefault();
      const type = card.getAttribute('data-home-type');
      const id = card.getAttribute('data-home-id');
      const blockID = card.getAttribute('data-home-block');
      const block = blockMap.get(blockID);
      const blockItems = Array.isArray(block?.items) ? block.items : [];
      if (!type || !id) return;
      showContextMenu(event, buildMenu(context, type, id, blockItems));
    });
  });

  await populateWaveGenres(waveGenreSelect);
  waveGenreSelect?.addEventListener('change', () => {
    selectedWaveGenre = String(waveGenreSelect.value || '').trim();
  });

  waveBtn?.addEventListener('click', async () => {
    const tracks = await API.getTracks({ limit: 2000, offset: 0, sort: 'name' });
    let queue = Array.isArray(tracks) ? tracks.slice() : [];
    if (selectedWaveGenre) {
      queue = queue.filter((track) => trackMatchesGenre(track, selectedWaveGenre));
    }
    if (!queue.length) return;
    shuffle(queue);
    context.player.replaceQueueFromTracks(queue, 0, 'wave', selectedWaveGenre || 'all_tracks', true);
  });
}

function renderItems(type, items, blockID) {
  if (!Array.isArray(items) || !items.length) {
    return `<div class="sh-empty">${escapeHtml(t('empty_no_items', 'No items'))}</div>`;
  }
  if (type === 'tracks') {
    return items.map((track) => `
      <a class="sh-inline-card sh-inline-track" href="/tracks/${encodeURIComponent(track.id || track.ID || '')}" data-home-item data-home-type="track" data-home-id="${escapeHtml(track.id || track.ID || '')}" data-home-block="${escapeHtml(blockID || '')}">
        <img src="${API.albumCoverThumbUrl(track.album_id || track.albumId || track.AlbumID || '', 256)}" alt="${escapeHtml(track.title || '')}" />
        <div>
          <strong>${escapeHtml(track.title || '')}</strong>
          <small>${escapeHtml(track.artist_name || track.artist || '')}</small>
        </div>
      </a>
    `).join('');
  }
  if (type === 'artists') {
    return items.map((artist) => `
      <a class="sh-inline-card" href="/artists/${encodeURIComponent(artist.id || artist.ID || '')}" data-home-item data-home-type="artist" data-home-id="${escapeHtml(artist.id || artist.ID || '')}" data-home-block="favorite_artists">
        <img src="/api/covers/artist/${encodeURIComponent(artist.id || artist.ID || '')}" alt="${escapeHtml(artist.name || '')}" />
        <div>
          <strong>${escapeHtml(artist.name || '')}</strong>
          <small>${escapeHtml(t('artist', 'Artist'))}</small>
        </div>
      </a>
    `).join('');
  }
  return items.map((album) => `
    <a class="sh-inline-card" href="/albums/${encodeURIComponent(album.id || album.ID || '')}" data-home-item data-home-type="album" data-home-id="${escapeHtml(album.id || album.ID || '')}" data-home-block="${escapeHtml(blockID || '')}">
      <img src="${API.albumCoverThumbUrl(album.id || album.ID || '', 256)}" alt="${escapeHtml(album.title || '')}" />
      <div>
        <strong>${escapeHtml(album.title || '')}</strong>
        <small>${escapeHtml(album.artist_name || album.artistName || '')}</small>
      </div>
    </a>
  `).join('');
}

function escapeHtml(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

function buildMenu(context, type, id, blockItems) {
  if (type === 'album') {
    return [
      {
        label: t('view', 'View'),
        action: async () => openDetail(context, type, id, blockItems)
      },
      {
        label: t('play', 'Play'),
        action: async () => {
          const tracks = await API.getAlbumTracks(id, { limit: 500, offset: 0, sort: 'name' });
          context.player.replaceQueueFromTracks(tracks || [], 0, 'album', id, true);
        }
      },
      {
        label: t('add_to_queue', 'Add to queue'),
        action: async () => {
          const tracks = await API.getAlbumTracks(id, { limit: 500, offset: 0, sort: 'name' });
          context.player.appendTracks(tracks || []);
        }
      },
      {
        label: t('toggle_favorite', 'Toggle favorite'),
        action: async () => {
          await API.toggleFavoriteAlbum(id);
        }
      }
    ];
  }
  if (type === 'artist') {
    return [
      {
        label: t('view', 'View'),
        action: async () => openDetail(context, type, id, blockItems)
      },
      {
        label: t('play', 'Play'),
        action: async () => {
          const queue = await buildArtistQueue(id);
          if (!queue.length) return;
          context.player.replaceQueueFromTracks(queue, 0, 'artist', id, true);
        }
      },
      {
        label: t('add_to_queue', 'Add to queue'),
        action: async () => {
          const queue = await buildArtistQueue(id);
          if (!queue.length) return;
          context.player.appendTracks(queue);
        }
      },
      {
        label: t('toggle_favorite', 'Toggle favorite'),
        action: async () => {
          await API.toggleFavoriteArtist(id);
        }
      }
    ];
  }

  return [
    {
      label: t('view', 'View'),
      action: async () => openDetail(context, type, id, blockItems)
    },
    {
      label: t('play', 'Play'),
      action: async () => {
        const idx = blockItems.findIndex((item) => String(item.id || item.ID || '') === String(id));
        context.player.replaceQueueFromTracks(blockItems, idx >= 0 ? idx : 0, 'home', 'dashboard', true);
      }
    },
    {
      label: t('add_to_queue', 'Add to queue'),
      action: async () => {
        const item = blockItems.find((candidate) => String(candidate.id || candidate.ID || '') === String(id));
        if (item) context.player.appendTracks([item]);
      }
    },
    {
      label: t('toggle_favorite', 'Toggle favorite'),
      action: async () => {
        await API.toggleFavoriteTrack(id);
      }
    }
  ];
}

async function buildArtistQueue(artistID) {
  const albumsRaw = await API.getArtistAlbums(artistID, { limit: 500, offset: 0, sort: 'year' });
  const albumIDs = (Array.isArray(albumsRaw) ? albumsRaw : []).map((item) => String(item.id || item.ID || ''));
  const tracks = [];
  for (const albumID of albumIDs) {
    const raw = await API.getAlbumTracks(albumID, { limit: 500, offset: 0, sort: 'name' });
    tracks.push(...(Array.isArray(raw) ? raw : []));
  }
  if (tracks.length) return tracks;
  const allTracksRaw = await API.getTracks({ limit: 5000, offset: 0, sort: 'name' });
  return (Array.isArray(allTracksRaw) ? allTracksRaw : []).filter((track) => String(track.artistId || track.ArtistID || '') === String(artistID));
}

function shuffle(items) {
  for (let i = items.length - 1; i > 0; i -= 1) {
    const j = Math.floor(Math.random() * (i + 1));
    const tmp = items[i];
    items[i] = items[j];
    items[j] = tmp;
  }
}

async function populateWaveGenres(select) {
  if (!select) return;
  const baseOption = `<option value="" data-i18n="all_genres">${escapeHtml(t('all_genres', 'All genres'))}</option>`;
  try {
    const tracks = await API.getTracks({ limit: 5000, offset: 0, sort: 'name' });
    const genres = collectGenres(Array.isArray(tracks) ? tracks : []);
    select.innerHTML = [
      baseOption,
      ...genres.map((genre) => `<option value="${escapeHtml(genre)}">${escapeHtml(genre)}</option>`)
    ].join('');
  } catch {
    select.innerHTML = baseOption;
  }
}

function collectGenres(tracks) {
  const set = new Set();
  tracks.forEach((track) => {
    splitGenres(track.genre || track.Genre || '').forEach((genre) => {
      if (genre) set.add(genre);
    });
  });
  return Array.from(set).sort((a, b) => a.localeCompare(b, undefined, { sensitivity: 'base' }));
}

function trackMatchesGenre(track, genre) {
  const normalizedGenre = String(genre || '').trim().toLowerCase();
  if (!normalizedGenre) return true;
  return splitGenres(track.genre || track.Genre || '').some((value) => value.toLowerCase() === normalizedGenre);
}

function splitGenres(value) {
  const raw = String(value || '').trim();
  if (!raw) return [];
  const normalized = raw
    .replaceAll('|', ';')
    .replaceAll(' / ', ';')
    .replaceAll('\\', ';');
  return normalized
    .split(/[;,]/)
    .map((item) => String(item || '').trim())
    .filter(Boolean);
}

async function openDetail(context, type, id, blockItems) {
  if (type === 'album') {
    await context.router?.go(`/albums/${encodeURIComponent(id)}`);
    return;
  }
  if (type === 'artist') {
    await context.router?.go(`/artists/${encodeURIComponent(id)}`);
    return;
  }
  if (type === 'track') {
    await context.router?.go(`/tracks/${encodeURIComponent(id)}`);
    return;
  }
  const idx = blockItems.findIndex((item) => String(item.id || item.ID || '') === String(id));
  context.player.replaceQueueFromTracks(blockItems, idx >= 0 ? idx : 0, 'home', 'dashboard', true);
}

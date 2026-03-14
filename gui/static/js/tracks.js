import { API } from './api.js';
import { t } from './i18n.js';
import { loadArtistNameMap, resolveArtistName } from './artist-map.js';
import { showContextMenu } from './context-menu.js';

export async function renderTracks(context, root) {
  const list = root.querySelector('#tracks-list');
  const searchInput = root.querySelector('#tracks-search');
  const sortSelect = root.querySelector('#tracks-sort');
  const duplicatesOnlyToggle = root.querySelector('#tracks-duplicates-only');

  let tracks = [];
  let albums = [];
  let artistMap = new Map();
  let searchTerm = '';
  let duplicatesOnly = false;
  let currentSort = String(sortSelect?.value || 'name').trim().toLowerCase() || 'name';

  async function loadData() {
    const [tracksRaw, albumsRaw, artists] = await Promise.all([
      API.getTracks({ limit: 5000, offset: 0, sort: toBackendTrackSort(currentSort) }),
      API.getAlbums({ limit: 500, offset: 0, sort: 'year' }),
      loadArtistNameMap(),
    ]);
    artistMap = artists;
    albums = normalizeAlbums(albumsRaw, artistMap);
    tracks = normalizeTracks(tracksRaw, artistMap, albums);
    renderList();
  }

  function renderList() {
    const visibleTracks = filterTracks(tracks, {
      searchTerm,
      duplicatesOnly
    });
    const sortedTracks = sortTracks(visibleTracks, currentSort);

    if (!sortedTracks.length) {
      list.innerHTML = `<p>${escapeHtml(t('empty_no_tracks', 'No tracks found.'))}</p>`;
      return;
    }

    list.innerHTML = sortedTracks
      .map((track, idx) => `
      <article class="sh-detail-track-row sh-track-page-row" data-track-index="${idx}" data-track-id="${escapeHtml(track.id)}">
        <span class="sh-detail-track-index-slot">
          <span class="sh-detail-track-index">${idx + 1}</span>
          <button type="button" class="sh-detail-track-play" data-play-index="${idx}" aria-label="${escapeHtml(t('play', 'Play'))}">&#9654;</button>
        </span>
        <img src="${track.albumId ? API.albumCoverThumbUrl(track.albumId, 64) : '/static/logo.png'}" alt="${escapeHtml(track.title)} ${escapeHtml(t('cover', 'cover'))}" class="sh-detail-track-cover" loading="lazy" />
        <span class="sh-detail-track-text">
          <strong>${escapeHtml(track.title)}</strong>
          <small>${escapeHtml(track.artist || t('unknown_artist', 'Unknown artist'))}</small>
        </span>
        <span class="sh-detail-track-row-meta">
          <span class="sh-detail-track-duration">${escapeHtml(formatDuration(track.durationSeconds || 0))}</span>
        </span>
      </article>
    `)
      .join('');

    list.querySelectorAll('[data-play-index]').forEach((btn) => {
      btn.addEventListener('click', (event) => {
        event.preventDefault();
        event.stopPropagation();
        const index = Number(btn.getAttribute('data-play-index'));
        context.player.replaceQueueFromTracks(sortedTracks, index, 'tracks', 'list', true);
      });
    });

    list.querySelectorAll('[data-track-index]').forEach((row) => {
      row.addEventListener('click', (event) => {
        if (event.target.closest('button')) return;
        const index = Number(row.getAttribute('data-track-index'));
        const track = sortedTracks[index];
        if (track?.id) context.router?.go(`/tracks/${encodeURIComponent(track.id)}`);
      });
      row.addEventListener('contextmenu', (event) => {
        event.preventDefault();
        const index = Number(row.getAttribute('data-track-index'));
        const track = sortedTracks[index];
        showContextMenu(event, [
          { label: t('play', 'Play'), action: async () => context.player.replaceQueueFromTracks(sortedTracks, index, 'tracks', 'list', true) },
          { label: t('add_to_queue', 'Add to queue'), action: async () => context.player.appendTracks([track]) },
          { label: t('view', 'View'), action: async () => context.router?.go(`/tracks/${encodeURIComponent(track.id)}`) },
          {
            label: t('delete', 'Delete'),
            danger: true,
            action: async () => {
              if (!track?.id) return;
              await API.deleteTrack(track.id);
              await loadData();
            }
          }
        ]);
      });
    });

    hydrateTrackDurations(list, sortedTracks);
  }

  searchInput?.addEventListener('input', () => {
    searchTerm = String(searchInput.value || '').trim().toLowerCase();
    renderList();
  });

  duplicatesOnlyToggle?.addEventListener('change', () => {
    duplicatesOnly = Boolean(duplicatesOnlyToggle.checked);
    renderList();
  });

  sortSelect?.addEventListener('change', async () => {
    currentSort = String(sortSelect.value || 'name').trim().toLowerCase() || 'name';
    await loadData();
  });

  const refreshHandler = () => {
    loadData().catch(() => {});
  };
  window.addEventListener('soundhub:library-updated', refreshHandler);
  root.addEventListener(
    'DOMNodeRemoved',
    () => {
      window.removeEventListener('soundhub:library-updated', refreshHandler);
    },
    { once: true }
  );

  await loadData();
}

function filterTracks(tracks, { searchTerm, duplicatesOnly }) {
  const duplicates = duplicateTitleSet(tracks);
  return tracks.filter((track) => {
    const haystack = `${track.title} ${track.artist} ${track.albumTitle}`.toLowerCase();
    if (searchTerm && !haystack.includes(searchTerm)) {
      return false;
    }
    if (duplicatesOnly && !duplicates.has(normalizeDuplicateTitle(track.title))) {
      return false;
    }
    return true;
  });
}

function duplicateTitleSet(tracks) {
  const counts = new Map();
  tracks.forEach((track) => {
    const key = normalizeDuplicateTitle(track.title);
    if (!key) return;
    counts.set(key, (counts.get(key) || 0) + 1);
  });
  const duplicates = new Set();
  counts.forEach((count, key) => {
    if (count > 1) duplicates.add(key);
  });
  return duplicates;
}

function normalizeDuplicateTitle(title) {
  return String(title || '').trim().toLowerCase();
}

function normalizeAlbums(raw, artistMap) {
  return (Array.isArray(raw) ? raw : []).map((album) => ({
    id: String(album.id || album.ID || '').trim(),
    title: String(album.title || album.Title || '').trim(),
    artistId: String(album.artistId || album.ArtistID || '').trim(),
    year: Number(album.year || album.Year || 0) || 0,
    artist: artistMap.get(String(album.artistId || album.ArtistID || '').trim()) || '',
  }));
}

function normalizeTracks(raw, artistMap, albums) {
  const albumMap = new Map(albums.map((album) => [album.id, album]));
  return (Array.isArray(raw) ? raw : []).map((track) => {
    const albumId = String(track.albumId || track.AlbumID || '').trim();
    const album = albumMap.get(albumId);
    const artistId = String(track.artistId || track.ArtistID || '').trim();
    return {
      id: String(track.id || track.ID || '').trim(),
      title: String(track.title || track.Title || '').trim(),
      artist: resolveArtistName(track, artistMap),
      artistId,
      albumId,
      albumTitle: album?.title || '',
      year: Number(album?.year || 0) || 0,
      createdAt: parseDateValue(track.created_at || track.createdAt || track.CreatedAt),
      durationSeconds: normalizeDurationSeconds(track),
    };
  });
}

function sortTracks(items, sortBy) {
  const out = items.slice();
  switch (String(sortBy || '').toLowerCase()) {
    case 'created_at':
      out.sort((a, b) => Number(b.createdAt || 0) - Number(a.createdAt || 0));
      break;
    case 'year':
      out.sort((a, b) => {
        const ay = Number(a.year || 0);
        const by = Number(b.year || 0);
        if (ay <= 0 && by > 0) return 1;
        if (by <= 0 && ay > 0) return -1;
        if (ay !== by) return by - ay;
        return String(a.title || '').localeCompare(String(b.title || ''), undefined, { sensitivity: 'base' });
      });
      break;
    case 'duration':
      out.sort((a, b) => {
        const ad = Number(a.durationSeconds || 0);
        const bd = Number(b.durationSeconds || 0);
        if (ad <= 0 && bd > 0) return 1;
        if (bd <= 0 && ad > 0) return -1;
        if (ad !== bd) return bd - ad;
        return String(a.title || '').localeCompare(String(b.title || ''), undefined, { sensitivity: 'base' });
      });
      break;
    default:
      out.sort((a, b) => String(a.title || '').localeCompare(String(b.title || ''), undefined, { sensitivity: 'base' }));
      break;
  }
  return out;
}

function parseDateValue(value) {
  if (!value) return 0;
  const ts = Date.parse(String(value));
  return Number.isFinite(ts) ? ts : 0;
}

function toBackendTrackSort(sortBy) {
  return String(sortBy || '').toLowerCase() === 'created_at' ? 'created_at' : 'name';
}

function normalizeDurationSeconds(track) {
  const raw = Number(track.duration_seconds ?? track.durationSeconds ?? track.DurationSeconds ?? track.duration ?? track.Duration ?? 0);
  if (!Number.isFinite(raw) || raw <= 0) return 0;
  if (raw > 1000000) return Math.round(raw / 1e9);
  return Math.round(raw);
}

function formatDuration(seconds) {
  const safe = Math.max(0, Number.parseInt(String(seconds || 0), 10) || 0);
  const minutes = Math.floor(safe / 60);
  const sec = safe % 60;
  return `${minutes}:${String(sec).padStart(2, '0')}`;
}

function hydrateTrackDurations(container, tracks) {
  const byId = new Map((Array.isArray(tracks) ? tracks : []).map((track) => [String(track.id), track]));
  container.querySelectorAll('[data-track-id]').forEach((row) => {
    const trackId = String(row.getAttribute('data-track-id') || '').trim();
    const track = byId.get(trackId);
    if (!track?.id || Number(track.durationSeconds || 0) > 1) return;
    const target = row.querySelector('.sh-detail-track-duration');
    if (!target) return;
    resolveStreamDuration(track.id).then((seconds) => {
      if (seconds > 1) {
        target.textContent = formatDuration(seconds);
      }
    }).catch(() => {});
  });
}

const streamDurationCache = new Map();

function resolveStreamDuration(trackId) {
  if (streamDurationCache.has(trackId)) {
    return streamDurationCache.get(trackId);
  }
  const pending = new Promise((resolve, reject) => {
    const audio = new Audio();
    audio.preload = 'metadata';
    audio.src = API.streamUrl(trackId);
    const cleanup = () => {
      audio.removeAttribute('src');
      audio.load();
    };
    audio.addEventListener('loadedmetadata', () => {
      const seconds = Math.max(0, Math.round(Number(audio.duration || 0)));
      cleanup();
      resolve(seconds);
    }, { once: true });
    audio.addEventListener('error', () => {
      cleanup();
      reject(new Error('failed to probe duration'));
    }, { once: true });
  });
  streamDurationCache.set(trackId, pending);
  return pending;
}

function escapeHtml(value) {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

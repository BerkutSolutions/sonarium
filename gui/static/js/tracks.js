import { API } from './api.js';
import { t } from './i18n.js';
import { loadArtistNameMap, resolveArtistName } from './artist-map.js';
import { showContextMenu } from './context-menu.js';

export async function renderTracks(context, root) {
  const list = root.querySelector('#tracks-list');
  const searchInput = root.querySelector('#tracks-search');
  const duplicatesOnlyToggle = root.querySelector('#tracks-duplicates-only');

  let tracks = [];
  let albums = [];
  let artistMap = new Map();
  let searchTerm = '';
  let duplicatesOnly = false;

  async function loadData() {
    const [tracksRaw, albumsRaw, artists] = await Promise.all([
      API.getTracks({ limit: 500, offset: 0, sort: 'name' }),
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

    if (!visibleTracks.length) {
      list.innerHTML = `<p>${escapeHtml(t('empty_no_tracks', 'No tracks found.'))}</p>`;
      return;
    }

    list.innerHTML = visibleTracks
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
        context.player.replaceQueueFromTracks(visibleTracks, index, 'tracks', 'list', true);
      });
    });

    list.querySelectorAll('[data-track-index]').forEach((row) => {
      row.addEventListener('click', (event) => {
        if (event.target.closest('button')) return;
        const index = Number(row.getAttribute('data-track-index'));
        const track = visibleTracks[index];
        if (track?.id) context.router?.go(`/tracks/${encodeURIComponent(track.id)}`);
      });
      row.addEventListener('contextmenu', (event) => {
        event.preventDefault();
        const index = Number(row.getAttribute('data-track-index'));
        const track = visibleTracks[index];
        showContextMenu(event, [
          { label: t('play', 'Play'), action: async () => context.player.replaceQueueFromTracks(visibleTracks, index, 'tracks', 'list', true) },
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
  }

  searchInput?.addEventListener('input', () => {
    searchTerm = String(searchInput.value || '').trim().toLowerCase();
    renderList();
  });

  duplicatesOnlyToggle?.addEventListener('change', () => {
    duplicatesOnly = Boolean(duplicatesOnlyToggle.checked);
    renderList();
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
      durationSeconds: normalizeDurationSeconds(track),
    };
  });
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

function escapeHtml(value) {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

import { API } from './api.js';
import { t } from './i18n.js';
import { loadArtistNameMap, resolveArtistName } from './artist-map.js';

export async function renderGenres(context, root) {
  const grid = root.querySelector('#genres-grid');
  const titleEl = root.querySelector('#genres-active-title');
  const countEl = root.querySelector('#genres-active-count');
  const list = root.querySelector('#genres-tracks');

  const [tracksRaw, albumsRaw, artists] = await Promise.all([
    API.getTracks({ limit: 1000, offset: 0, sort: 'name' }),
    API.getAlbums({ limit: 1000, offset: 0, sort: 'year' }),
    loadArtistNameMap(),
  ]);

  const albums = normalizeAlbums(albumsRaw, artists);
  const tracks = normalizeTracks(tracksRaw, artists, albums);
  const groups = groupByGenre(tracks);
  const genres = Array.from(groups.entries())
    .map(([name, items]) => ({ name, items }))
    .sort((a, b) => {
      if (a.name === unknownGenreLabel() && b.name !== unknownGenreLabel()) return 1;
      if (b.name === unknownGenreLabel() && a.name !== unknownGenreLabel()) return -1;
      return a.name.localeCompare(b.name);
    });

  let activeGenre = genres[0]?.name || unknownGenreLabel();

  function renderGenreCards() {
    if (!genres.length) {
      grid.innerHTML = `<p class="sh-detail-empty">${escapeHtml(t('empty_no_tracks', 'No tracks found.'))}</p>`;
      list.innerHTML = '';
      countEl.textContent = '0';
      return;
    }

    grid.innerHTML = genres
      .map((genre) => `
        <button type="button" class="sh-genres-card${genre.name === activeGenre ? ' active' : ''}" data-genre="${escapeHtml(genre.name)}">
          <strong>${escapeHtml(genre.name)}</strong>
          <span>${escapeHtml(String(genre.items.length))} ${escapeHtml(t('tracks', 'Tracks'))}</span>
        </button>
      `)
      .join('');

    grid.querySelectorAll('[data-genre]').forEach((button) => {
      button.addEventListener('click', () => {
        activeGenre = button.getAttribute('data-genre') || unknownGenreLabel();
        renderGenreCards();
        renderTracksList();
      });
    });
  }

  function renderTracksList() {
    const visibleTracks = groups.get(activeGenre) || [];
    titleEl.textContent = activeGenre;
    countEl.textContent = String(visibleTracks.length);

    if (!visibleTracks.length) {
      list.innerHTML = `<p class="sh-detail-empty">${escapeHtml(t('empty_no_tracks', 'No tracks found.'))}</p>`;
      return;
    }

    list.innerHTML = visibleTracks
      .map((track, index) => `
        <article class="sh-detail-track-row sh-track-page-row" data-track-id="${escapeHtml(track.id)}" data-track-index="${index}">
          <span class="sh-detail-track-index-slot">
            <span class="sh-detail-track-index">${index + 1}</span>
            <button type="button" class="sh-detail-track-play" data-play-index="${index}" aria-label="${escapeHtml(t('play', 'Play'))}">&#9654;</button>
          </span>
          <img src="${track.albumId ? API.albumCoverThumbUrl(track.albumId, 64) : '/static/logo.png'}" alt="${escapeHtml(track.title)} ${escapeHtml(t('cover', 'cover'))}" class="sh-detail-track-cover" loading="lazy" />
          <span class="sh-detail-track-text">
            <strong>${escapeHtml(track.title)}</strong>
            <small>${escapeHtml(track.artist || t('unknown_artist', 'Unknown artist'))} - ${escapeHtml(track.albumTitle || t('albums', 'Albums'))}</small>
          </span>
          <span class="sh-detail-track-row-meta">
            <span class="sh-detail-track-duration">${escapeHtml(formatDuration(track.durationSeconds || 0))}</span>
          </span>
        </article>
      `)
      .join('');

    list.querySelectorAll('[data-play-index]').forEach((button) => {
      button.addEventListener('click', (event) => {
        event.preventDefault();
        event.stopPropagation();
        const index = Number(button.getAttribute('data-play-index'));
        context.player.replaceQueueFromTracks(visibleTracks, index, 'genres', activeGenre, true);
      });
    });

    list.querySelectorAll('[data-track-id]').forEach((row) => {
      row.addEventListener('click', (event) => {
        if (event.target.closest('button')) return;
        const trackId = row.getAttribute('data-track-id');
        if (trackId) context.router?.go(`/tracks/${encodeURIComponent(trackId)}`);
      });
    });
  }

  renderGenreCards();
  renderTracksList();
}

function groupByGenre(tracks) {
  const groups = new Map();
  tracks.forEach((track) => {
    const genres = splitGenres(track.genre);
    genres.forEach((genre) => {
      if (!groups.has(genre)) groups.set(genre, []);
      groups.get(genre).push(track);
    });
  });
  groups.forEach((items) => {
    items.sort((a, b) => {
      const artistCompare = a.artist.localeCompare(b.artist);
      if (artistCompare !== 0) return artistCompare;
      const albumCompare = a.albumTitle.localeCompare(b.albumTitle);
      if (albumCompare !== 0) return albumCompare;
      return a.title.localeCompare(b.title);
    });
  });
  return groups;
}

function splitGenres(value) {
  const raw = String(value || '').trim();
  if (!raw) return [unknownGenreLabel()];

  const normalized = raw
    .replaceAll('|', ';')
    .replaceAll(' / ', ';')
    .replaceAll('\\', ';');

  const parts = normalized
    .split(/[;,]/)
    .map((item) => normalizeGenre(item))
    .filter(Boolean);

  return parts.length ? Array.from(new Set(parts)) : [unknownGenreLabel()];
}

function normalizeGenre(value) {
  const clean = String(value || '').trim();
  return clean || unknownGenreLabel();
}

function unknownGenreLabel() {
  return t('unknown_genre', 'Unknown genre');
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
    return {
      id: String(track.id || track.ID || '').trim(),
      title: String(track.title || track.Title || '').trim(),
      artist: resolveArtistName(track, artistMap),
      artistId: String(track.artistId || track.ArtistID || '').trim(),
      albumId,
      albumTitle: album?.title || '',
      durationSeconds: normalizeDurationSeconds(track),
      genre: String(track.genre || track.Genre || '').trim(),
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

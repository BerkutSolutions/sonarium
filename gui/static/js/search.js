import { API } from './api.js';
import { t } from './i18n.js';
import { loadArtistNameMap, resolveArtistName } from './artist-map.js';

export async function renderSearch(context, root) {
  const form = root.querySelector('#search-form');
  const input = root.querySelector('#search-query');
  const results = root.querySelector('#search-results');
  const artistMap = await loadArtistNameMap();

  async function execute(query) {
    const data = await API.search(query, { limit: 30, offset: 0, sort: 'name' });
    const artists = Array.isArray(data?.artists) ? data.artists : [];
    const albums = normalizeAlbums(data?.albums || [], artistMap);
    const tracks = normalizeTracks(data?.tracks || [], artistMap);
    const queryHash = simpleHash(query);

    results.innerHTML = `
      <section class="sh-section">
        <h3>${escapeHtml(t('artists', 'Artists'))}</h3>
        <div class="sh-list">${renderEntityList(artists, (item) => item.name || item.Name, (item) => `/artists/${encodeURIComponent(item.id || item.ID || '')}`)}</div>
      </section>
      <section class="sh-section">
        <h3>${escapeHtml(t('albums', 'Albums'))}</h3>
        <div class="sh-search-album-grid">${renderAlbumCards(albums)}</div>
      </section>
      <section class="sh-section">
        <h3>${escapeHtml(t('tracks', 'Tracks'))}</h3>
        <div class="sh-detail-track-list sh-track-page-list">${renderTrackRows(tracks)}</div>
      </section>
    `;

    results.querySelectorAll('[data-album-play]').forEach((btn) => {
      btn.addEventListener('click', async (event) => {
        event.preventDefault();
        event.stopPropagation();
        const albumId = btn.getAttribute('data-album-play');
        if (!albumId) return;
        const albumTracksRaw = await API.getAlbumTracks(albumId, { limit: 500, offset: 0, sort: 'name' });
        const albumTracks = normalizeTracks(albumTracksRaw || [], artistMap);
        if (!albumTracks.length) return;
        context.player.replaceQueueFromTracks(albumTracks, 0, 'search-album', albumId, true);
      });
    });

    results.querySelectorAll('[data-play-index]').forEach((btn) => {
      btn.addEventListener('click', (event) => {
        event.preventDefault();
        event.stopPropagation();
        const index = Number(btn.getAttribute('data-play-index'));
        context.player.replaceQueueFromTracks(tracks, index, 'search', queryHash, true);
      });
    });

    results.querySelectorAll('[data-queue-index]').forEach((btn) => {
      btn.addEventListener('click', (event) => {
        event.preventDefault();
        event.stopPropagation();
        const index = Number(btn.getAttribute('data-queue-index'));
        context.player.appendTracks([tracks[index]]);
      });
    });

    results.querySelectorAll('[data-detail-link]').forEach((link) => {
      link.addEventListener('click', (event) => {
        if (event.target.closest('button')) return;
        event.preventDefault();
        const href = link.getAttribute('href');
        if (href) {
          context.router?.go(href);
        }
      });
    });
  }

  form.addEventListener('submit', async (event) => {
    event.preventDefault();
    const query = input.value.trim();
    if (!query) return;
    await execute(query);
  });
}

function normalizeTracks(raw, artistMap) {
  return (Array.isArray(raw) ? raw : []).map((track) => ({
    id: String(track.id || track.ID || '').trim(),
    title: String(track.title || track.Title || '').trim(),
    artist: resolveArtistName(track, artistMap),
    albumId: String(track.albumId || track.AlbumID || '').trim()
  }));
}

function normalizeAlbums(raw, artistMap) {
  return (Array.isArray(raw) ? raw : []).map((album) => ({
    id: String(album.id || album.ID || '').trim(),
    title: String(album.title || album.Title || '').trim(),
    year: Number(album.year || album.Year || 0) || 0,
    artistId: String(album.artistId || album.ArtistID || '').trim(),
    artist: artistMap.get(String(album.artistId || album.ArtistID || '').trim()) || ''
  }));
}

function renderEntityList(items, mapper, hrefMapper) {
  if (!items || !items.length) {
    return `<p>${escapeHtml(t('empty_no_results', 'No results'))}</p>`;
  }
  return items
    .map((item) => `<a class="sh-list-row" href="${escapeHtml(hrefMapper(item) || '#')}" data-detail-link><span>${escapeHtml(mapper(item) || '')}</span></a>`)
    .join('');
}

function renderAlbumCards(items) {
  if (!items.length) {
    return `<p>${escapeHtml(t('empty_no_results', 'No results'))}</p>`;
  }
  return items.map((album) => `
    <a class="sh-card sh-album-card" href="/albums/${encodeURIComponent(album.id)}" data-detail-link>
      <div class="sh-album-cover-wrap">
        <img src="${album.id ? API.albumCoverThumbUrl(album.id, 256) : '/static/logo.png'}" alt="${escapeHtml(album.title)} ${escapeHtml(t('cover', 'cover'))}" class="sh-album-cover" loading="lazy" />
        <div class="sh-album-cover-actions">
          <button type="button" class="sh-cover-play-btn" data-album-play="${escapeHtml(album.id)}" aria-label="${escapeHtml(t('play', 'Play'))}">&#9654;</button>
        </div>
      </div>
      <h3>${escapeHtml(album.title || t('unnamed', 'Unnamed'))}</h3>
      <div>${escapeHtml(t('year', 'Year'))}: ${escapeHtml(album.year || '-')}</div>
      <small>${escapeHtml(album.artist || t('unknown_artist', 'Unknown artist'))}</small>
    </a>
  `).join('');
}

function renderTrackRows(tracks) {
  if (!tracks.length) {
    return `<p>${escapeHtml(t('empty_no_results', 'No results'))}</p>`;
  }
  return tracks.map((track, idx) => `
    <article class="sh-detail-track-row sh-track-page-row" data-detail-link href="/tracks/${encodeURIComponent(track.id)}">
      <span class="sh-detail-track-index-slot">
        <span class="sh-detail-track-index">${idx + 1}</span>
        <button type="button" class="sh-detail-track-play" data-play-index="${idx}" aria-label="${escapeHtml(t('play', 'Play'))}">&#9654;</button>
      </span>
      <img src="${track.albumId ? API.albumCoverThumbUrl(track.albumId, 64) : '/static/logo.png'}" alt="${escapeHtml(track.title)} ${escapeHtml(t('cover', 'cover'))}" class="sh-detail-track-cover" loading="lazy" />
      <span class="sh-detail-track-text">
        <strong>${escapeHtml(track.title || t('unknown_title', 'Unknown title'))}</strong>
        <small>${escapeHtml(track.artist || t('unknown_artist', 'Unknown artist'))}</small>
      </span>
      <span class="sh-detail-track-row-meta">
        <div class="sh-actions">
          <button type="button" data-queue-index="${idx}">${escapeHtml(t('queue', 'Queue'))}</button>
        </div>
      </span>
    </article>
  `).join('');
}

function simpleHash(input) {
  let hash = 0;
  for (let i = 0; i < input.length; i += 1) {
    hash = ((hash << 5) - hash) + input.charCodeAt(i);
    hash |= 0;
  }
  return `q${Math.abs(hash)}`;
}

function escapeHtml(value) {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

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
    const albums = Array.isArray(data?.albums) ? data.albums : [];
    const tracks = normalizeTracks(data?.tracks || [], artistMap);
    const queryHash = simpleHash(query);

    results.innerHTML = `
      <section class="sh-section">
        <h3>${escapeHtml(t('artists', 'Artists'))}</h3>
        <div class="sh-list">${renderEntityList(artists, (item) => item.name || item.Name, (item) => `/artists/${encodeURIComponent(item.id || item.ID || '')}`)}</div>
      </section>
      <section class="sh-section">
        <h3>${escapeHtml(t('albums', 'Albums'))}</h3>
        <div class="sh-list">${renderEntityList(albums, (item) => item.title || item.Title, (item) => `/albums/${encodeURIComponent(item.id || item.ID || '')}`)}</div>
      </section>
      <section class="sh-section">
        <h3>${escapeHtml(t('tracks', 'Tracks'))}</h3>
        <div class="sh-list">${tracks.map((track, idx) => `
          <a class="sh-list-row" href="/tracks/${encodeURIComponent(track.id)}" data-detail-link>
            <span>${escapeHtml(track.title)}</span>
            <div class="sh-actions">
              <button type="button" data-play-index="${idx}">${escapeHtml(t('play', 'Play'))}</button>
              <button type="button" data-queue-index="${idx}">${escapeHtml(t('queue', 'Queue'))}</button>
            </div>
          </a>
        `).join('')}</div>
      </section>
    `;

    results.querySelectorAll('[data-play-index]').forEach((btn) => {
      btn.addEventListener('click', () => {
        const index = Number(btn.getAttribute('data-play-index'));
        context.player.replaceQueueFromTracks(tracks, index, 'search', queryHash, true);
      });
    });
    results.querySelectorAll('[data-queue-index]').forEach((btn) => {
      btn.addEventListener('click', () => {
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
    id: track.id || track.ID || '',
    title: track.title || track.Title || '',
    artist: resolveArtistName(track, artistMap),
    albumId: track.albumId || track.AlbumID || ''
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

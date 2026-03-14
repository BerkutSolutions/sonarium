import { API } from './api.js';
import { t } from './i18n.js';
import { loadArtistNameMap, resolveArtistName } from './artist-map.js';
import { showContextMenu } from './context-menu.js';

export async function renderAlbumDetail(context, root, params) {
  const albumId = String(params.albumId || '').trim();
  const [artistMap, artistsRaw, albumsRaw] = await Promise.all([
    loadArtistNameMap(),
    API.getArtists({ limit: 500, offset: 0, sort: 'name' }),
    API.getAlbums({ limit: 500, offset: 0, sort: 'year' })
  ]);
  const artists = normalizeArtists(artistsRaw);
  const albums = normalizeAlbums(albumsRaw, artistMap);
  const album = albums.find((item) => item.id === albumId);
  const [tracksRaw, favoriteTrackIDs] = await Promise.all([
    API.getAlbumTracks(albumId, { limit: 500, offset: 0, sort: 'name' }),
    loadFavoriteTrackIDs()
  ]);
  const tracks = normalizeTracks(tracksRaw, artistMap);
  if (!album) {
    root.innerHTML = renderNotFound(t('albums', 'Albums'));
    return;
  }
  renderDetailPage(root, {
    entityType: 'album',
    entityId: album.id,
    kind: t('album_contents', 'Album'),
    title: album.title,
    subtitle: album.artist || t('unknown_artist', 'Unknown artist'),
    meta: album.year ? String(album.year) : '',
    cover: coverMarkup(`/api/covers/album/${encodeURIComponent(album.id)}`, album.title, 'sh-detail-cover'),
    actions: renderActions([
      { id: 'play', label: t('play', 'Play') },
      { id: 'queue', label: t('add_to_queue', 'Add to queue') }
    ]),
    editForm: renderEntityEditForm({
      title: t('edit_album', 'Edit album'),
      fields: [
        { type: 'text', name: 'title', label: t('title', 'Title'), value: album.title, required: true },
        { type: 'select', name: 'artist_id', label: t('artist', 'Artist'), value: album.artistId, required: true, options: buildArtistOptions(artists) },
        { type: 'number', name: 'year', label: t('year', 'Year'), value: album.year ? String(album.year) : '', min: 0, max: 9999 }
      ]
    }),
    body: renderTrackList(tracks, { albumId: album.id, albumTitle: album.title }),
    infoBody: renderInfoList([
      { label: 'ID', value: album.id },
      { label: t('artist', 'Artist'), value: album.artist || '-' },
      { label: t('year', 'Year'), value: album.year ? String(album.year) : '-' },
      { label: t('tracks', 'Tracks'), value: String(tracks.length) }
    ])
  });
  bindDetailActions(root, context, {
    play: () => context.player.replaceQueueFromTracks(tracks, 0, 'album', album.id, true),
    queue: () => context.player.appendTracks(tracks)
  });
  bindTrackInteractions(root, context, tracks, {
    albumTitle: album.title,
    collectionType: 'album',
    collectionId: album.id,
    favoriteTrackIDs
  });
  hydrateTrackDurations(root, tracks);
  bindHistoryControls(root);
  bindDetailMenu(root, context, {
    entityType: 'album',
    entityId: album.id,
    canShare: true,
    onEdit: () => setDetailEditing(root, true),
    deleteAction: async () => {
      await API.deleteAlbum(album.id);
      await context.router.go('/albums');
    }
  });
  bindEditForm(root, async (form) => {
    await API.updateAlbum(album.id, {
      title: readFormValue(form, 'title'),
      artist_id: readFormValue(form, 'artist_id'),
      year: Number.parseInt(readFormValue(form, 'year'), 10) || 0
    });
    await context.router.refresh();
  });
}

export async function renderArtistDetail(context, root, params) {
  const artistId = String(params.artistId || '').trim();
  const [artistMap, artistsRaw, albumsRaw] = await Promise.all([
    loadArtistNameMap(),
    API.getArtists({ limit: 500, offset: 0, sort: 'name' }),
    API.getArtistAlbums(artistId, { limit: 500, offset: 0, sort: 'year' })
  ]);
  const artists = normalizeArtists(artistsRaw);
  const artist = artists.find((item) => item.id === artistId);
  const albums = normalizeAlbums(albumsRaw, artistMap);
  const artistTracks = await buildArtistQueue(artistId, artistMap, albums);
  if (!artist) {
    root.innerHTML = renderNotFound(t('artists', 'Artists'));
    return;
  }
  root.innerHTML = `
    <section class="sh-detail-page">
      <div class="sh-detail-frame">
        ${renderDetailHistory()}
        <div class="sh-detail-hero sh-detail-hero-artist">
          <div class="sh-detail-cover-wrap">
            ${coverMarkup(`/api/covers/artist/${encodeURIComponent(artist.id)}`, artist.name, 'sh-detail-cover')}
          </div>
          <div class="sh-detail-main">
            <div class="sh-detail-topbar">
              <div class="sh-detail-kicker">${escapeHtml(t('artist', 'Artist'))}</div>
              <div class="sh-detail-menu-slot">
                <button type="button" class="sh-detail-menu-btn" data-detail-menu-btn aria-label="More actions">&#8942;</button>
              </div>
            </div>
            <div class="sh-detail-static" data-detail-static>
              <h1 class="sh-detail-title">${escapeHtml(artist.name)}</h1>
              <div class="sh-detail-subtitle">${escapeHtml(t('albums_count', 'Albums'))}: ${albums.length}</div>
            </div>
            <div class="sh-detail-actions">
              ${renderActions([
                { id: 'play', label: t('play', 'Play') },
                { id: 'queue', label: t('add_to_queue', 'Add to queue') }
              ])}
            </div>
            ${renderEntityEditForm({
              title: t('edit_artist', 'Edit artist'),
              fields: [
                { type: 'text', name: 'name', label: t('artist', 'Artist'), value: artist.name, required: true },
                { type: 'file', name: 'cover', label: t('cover', 'Cover'), accept: 'image/*' }
              ]
            })}
            <div class="sh-detail-edit-state" data-detail-edit-state hidden>${escapeHtml(t('edit', 'Edit'))}</div>
          </div>
        </div>
        <section class="sh-detail-section">
          <h2>${escapeHtml(t('albums', 'Albums'))}</h2>
          <div class="sh-detail-card-grid">
            ${albums.map((album) => `
              <a class="sh-detail-card" href="/albums/${encodeURIComponent(album.id)}" data-detail-link>
                ${coverMarkup(API.albumCoverThumbUrl(album.id, 256), album.title, 'sh-detail-card-cover')}
                <strong>${escapeHtml(album.title)}</strong>
                <span>${escapeHtml(album.year ? String(album.year) : '')}</span>
              </a>
            `).join('')}
          </div>
        </section>
        <section class="sh-detail-section sh-detail-info-section" data-detail-info-section hidden>
          <h2>${escapeHtml(t('info', 'Information'))}</h2>
          ${renderInfoList([
            { label: 'ID', value: artist.id },
            { label: t('albums_count', 'Albums'), value: String(albums.length) }
          ])}
        </section>
      </div>
    </section>
  `;
  bindDetailLinks(root, context);
  bindHistoryControls(root);
  bindDetailActions(root, context, {
    play: () => context.player.replaceQueueFromTracks(artistTracks, 0, 'artist', artist.id, true),
    queue: () => context.player.appendTracks(artistTracks)
  });
  bindDetailMenu(root, context, {
    entityType: 'artist',
    entityId: artist.id,
    canShare: true,
    onEdit: () => setDetailEditing(root, true),
    deleteAction: async () => {
      await API.deleteArtist(artist.id);
      await context.router.go('/artists');
    }
  });
  bindEditForm(root, async (form) => {
    await API.updateArtist(artist.id, readFormValue(form, 'name'));
    const coverFile = form.elements.namedItem('cover')?.files?.[0];
    if (coverFile) {
      await API.uploadArtistCover(artist.id, coverFile);
    }
    await context.router.refresh();
  });
}

async function buildArtistQueue(artistId, artistMap, albums) {
  const albumList = Array.isArray(albums) ? albums : [];
  const seen = new Set();
  const tracks = [];
  for (const album of albumList) {
    const raw = await API.getAlbumTracks(album.id, { limit: 500, offset: 0, sort: 'name' });
    for (const track of normalizeTracks(raw, artistMap)) {
      if (!track?.id || seen.has(track.id)) continue;
      seen.add(track.id);
      tracks.push(track);
    }
  }
  if (tracks.length > 0) {
    return tracks;
  }
  const allTracksRaw = await API.getTracks({ limit: 5000, offset: 0, sort: 'name' });
  return normalizeTracks(allTracksRaw, artistMap).filter((track) => track.artistId === artistId);
}

export async function renderPlaylistDetail(context, root, params) {
  const playlistId = String(params.playlistId || '').trim();
  const artistMap = await loadArtistNameMap();
  const [playlists, favoriteTrackIDs, allTracksRaw] = await Promise.all([
    API.getPlaylists({ limit: 200, offset: 0, sort: 'name' }).then(normalizePlaylists),
    loadFavoriteTrackIDs(),
    API.getTracks({ limit: 3000, offset: 0, sort: 'name' })
  ]);
  let playlist = playlists.find((item) => item.id === playlistId);
  let tracks = [];
  const allTracks = normalizeTracks(allTracksRaw, artistMap);
  if (playlistId === '__favorites__') {
    const home = await API.getHome(500);
    playlist = {
      id: '__favorites__',
      name: t('favorites', 'Favorites'),
      system: true
    };
    tracks = normalizeTracks(home?.favorites?.tracks || [], artistMap);
  } else {
    const payload = await API.getPlaylist(playlistId, {
      share: new URLSearchParams(window.location.search).get('share') || ''
    });
    playlist = normalizePlaylist(payload?.playlist || playlist);
    tracks = normalizeTracks(payload?.tracks || [], artistMap);
    playlist.permissions = payload?.permissions || {};
  }
  if (!playlist) {
    root.innerHTML = renderNotFound(t('playlists', 'Playlists'));
    return;
  }
  renderDetailPage(root, {
    entityType: 'playlist',
    entityId: playlist.id,
    kind: t('playlist_contents', 'Playlist'),
    title: playlist.name,
    subtitle: `${tracks.length} ${t('tracks', 'Tracks').toLowerCase()}`,
    meta: '',
    cover: coverMarkup(playlistCoverSrc(playlist), playlist.name, 'sh-detail-cover'),
    actions: renderActions([
      { id: 'play', label: t('play', 'Play') },
      { id: 'queue', label: t('add_to_queue', 'Add to queue') }
    ]),
    editForm: playlist.system ? '' : renderEntityEditForm({
      title: t('edit_playlist', 'Edit playlist'),
      fields: [
        { type: 'text', name: 'name', label: t('title', 'Title'), value: playlist.name, required: true },
        { type: 'textarea', name: 'description', label: t('description', 'Description'), value: playlist.description || '', rows: 4 }
      ]
    }),
    body: `
      ${playlist.system ? '' : renderPlaylistTrackSearch()}
      ${renderTrackList(tracks)}
    `,
    infoBody: renderInfoList([
      { label: 'ID', value: playlist.id },
      { label: t('description', 'Description'), value: playlist.description || '-' },
      { label: t('tracks', 'Tracks'), value: String(tracks.length) }
    ])
  });
  bindDetailActions(root, context, {
    play: () => context.player.replaceQueueFromTracks(tracks, 0, 'playlist', playlist.id, true),
    queue: () => context.player.appendTracks(tracks)
  });
  bindTrackInteractions(root, context, tracks, {
    collectionType: 'playlist',
    collectionId: playlist.id,
    favoriteTrackIDs,
    allowRemoveFromCollection: !playlist.system
  });
  if (!playlist.system) {
    bindPlaylistTrackSearch(root, context, {
      playlist,
      tracks,
      allTracks
    });
  }
  bindHistoryControls(root);
  bindDetailMenu(root, context, {
    entityType: 'playlist',
    entityId: playlist.id,
    canEdit: !playlist.system && playlist.permissions?.can_edit !== false,
    canShare: !playlist.system && playlist.permissions?.can_share !== false,
    onEdit: playlist.system ? null : () => setDetailEditing(root, true),
    deleteAction: playlist.system ? null : async () => {
      await API.deletePlaylist(playlist.id);
      await context.router.go('/playlists');
    }
  });
  if (!playlist.system) {
    bindEditForm(root, async (form) => {
      await API.updatePlaylist(playlist.id, {
        name: readFormValue(form, 'name'),
        description: readFormValue(form, 'description')
      });
      await context.router.refresh();
    });
  }
}

export async function renderTrackDetail(context, root, params) {
  const trackId = String(params.trackId || '').trim();
  const [artistMap, artistsRaw, albumsRaw, trackRaw] = await Promise.all([
    loadArtistNameMap(),
    API.getArtists({ limit: 500, offset: 0, sort: 'name' }),
    API.getAlbums({ limit: 500, offset: 0, sort: 'year' }),
    API.getTrack(trackId)
  ]);
  const artists = normalizeArtists(artistsRaw);
  const albums = normalizeAlbums(albumsRaw, artistMap);
  const track = normalizeTrack(trackRaw, artistMap);
  const album = track?.albumId ? normalizeAlbum(await API.getAlbum(track.albumId), artistMap) : null;
  if (!track?.id) {
    root.innerHTML = renderNotFound(t('tracks', 'Tracks'));
    return;
  }
  renderDetailPage(root, {
    entityType: 'track',
    entityId: track.id,
    kind: t('track_info', 'Track'),
    title: track.title,
    subtitle: track.artist || t('unknown_artist', 'Unknown artist'),
    meta: formatDuration(track.durationSeconds || 0),
    cover: coverMarkup(track.albumId ? `/api/covers/album/${encodeURIComponent(track.albumId)}` : '/static/logo.png', track.title, 'sh-detail-cover'),
    actions: renderActions([
      { id: 'play', label: t('play', 'Play') },
      { id: 'queue', label: t('add_to_queue', 'Add to queue') },
      { id: 'album', label: t('album_contents', 'Album') }
    ]),
    editForm: renderEntityEditForm({
      title: t('edit_track', 'Edit track'),
      fields: [
        { type: 'text', name: 'title', label: t('title', 'Title'), value: track.title, required: true },
        { type: 'select', name: 'artist_id', label: t('artist', 'Artist'), value: track.artistId, required: true, options: buildArtistOptions(artists) },
        { type: 'select', name: 'album_id', label: t('album_contents', 'Album'), value: track.albumId, required: true, options: buildAlbumOptions(albums) }
      ]
    }),
    body: '',
    infoBody: renderInfoList([
      { label: 'ID', value: track.id },
      { label: t('artist', 'Artist'), value: track.artist || '-' },
      { label: t('album_contents', 'Album'), value: album?.title || '-' }
    ])
  });
  bindDetailActions(root, context, {
    play: () => context.player.playTrack(track),
    queue: () => context.player.appendTracks([track]),
    album: () => {
      if (track.albumId) context.router.go(`/albums/${encodeURIComponent(track.albumId)}`);
    }
  });
  hydrateSingleTrackDuration(root, track);
  bindHistoryControls(root);
  bindDetailMenu(root, context, {
    entityType: 'track',
    entityId: track.id,
    canShare: true,
    onEdit: () => setDetailEditing(root, true),
    deleteAction: async () => {
      const prepared = context.player?.prepareTrackDeletion?.(track.id);
      try {
        await API.deleteTrack(track.id);
        context.player?.handleTrackDeleted?.(track.id);
        window.dispatchEvent(new CustomEvent('soundhub:library-updated'));
        await context.router.go('/tracks');
      } catch (error) {
        if (prepared) {
          context.player?.recoverPreparedTrackDeletion?.(track.id);
        }
        throw error;
      }
    }
  });
  bindEditForm(root, async (form) => {
    await API.updateTrack(track.id, {
      title: readFormValue(form, 'title'),
      album_id: readFormValue(form, 'album_id'),
      artist_id: readFormValue(form, 'artist_id')
    });
    await context.router.refresh();
  });
}

function renderDetailPage(root, { entityType, entityId, kind, title, subtitle, meta, cover, actions, editForm = '', body, infoBody = '' }) {
  root.innerHTML = `
    <section class="sh-detail-page">
      <div class="sh-detail-frame" data-detail-entity-type="${escapeHtml(entityType || '')}" data-detail-entity-id="${escapeHtml(entityId || '')}">
        ${renderDetailHistory()}
        <div class="sh-detail-hero">
          <div class="sh-detail-cover-wrap">
            ${cover}
          </div>
          <div class="sh-detail-main">
            <div class="sh-detail-topbar">
              <div class="sh-detail-kicker">${escapeHtml(kind)}</div>
              <div class="sh-detail-menu-slot">
                <button type="button" class="sh-detail-menu-btn" data-detail-menu-btn aria-label="More actions">&#8942;</button>
              </div>
            </div>
            <div class="sh-detail-static" data-detail-static>
              <h1 class="sh-detail-title">${escapeHtml(title)}</h1>
              <div class="sh-detail-subtitle">${escapeHtml(subtitle || '')}</div>
              ${meta ? `<div class="sh-detail-meta" data-detail-meta>${escapeHtml(meta)}</div>` : ''}
              <div class="sh-detail-actions">${actions}</div>
            </div>
            ${editForm}
            <div class="sh-detail-edit-state" data-detail-edit-state hidden>${escapeHtml(t('edit', 'Edit'))}</div>
          </div>
        </div>
        <div class="sh-detail-body-layout">
          <div class="sh-detail-body-main">
            <section class="sh-detail-section">
              ${body}
            </section>
            <section class="sh-detail-section sh-detail-info-section" data-detail-info-section hidden>
              <h2>${escapeHtml(t('info', 'Information'))}</h2>
              ${infoBody}
            </section>
          </div>
          <aside class="sh-detail-track-panel" data-track-panel hidden></aside>
        </div>
      </div>
    </section>
  `;
}

function renderDetailHistory() {
  return `
    <div class="sh-detail-history">
      <button type="button" class="sh-detail-history-btn" data-history-nav="back" aria-label="Back">
        <span aria-hidden="true">&#8249;</span>
      </button>
      <button type="button" class="sh-detail-history-btn" data-history-nav="forward" aria-label="Forward">
        <span aria-hidden="true">&#8250;</span>
      </button>
    </div>
  `;
}

function renderActions(items) {
  return items.map((item) => `
    <button type="button" class="sh-detail-action${item.id === 'play' ? ' primary' : ''}" data-detail-action="${escapeHtml(item.id)}">
      ${escapeHtml(item.label)}
    </button>
  `).join('');
}

function renderTrackList(tracks, options = {}) {
  if (!tracks.length) {
    return `<p class="sh-detail-empty">${escapeHtml(t('empty_no_tracks', 'No tracks found.'))}</p>`;
  }
  const fallbackAlbumId = String(options.albumId || '').trim();
  const fallbackAlbumTitle = String(options.albumTitle || '').trim();
  return `
    <div class="sh-detail-track-list">
      ${tracks.map((track, index) => `
        <article
          class="sh-detail-track-row"
          data-track-row
          data-track-id="${escapeHtml(track.id)}"
          data-track-index="${index}"
          data-track-album-id="${escapeHtml(track.albumId || fallbackAlbumId)}"
          data-track-album-title="${escapeHtml(track.albumTitle || fallbackAlbumTitle)}"
        >
          <span class="sh-detail-track-index-slot">
            <span class="sh-detail-track-index">${index + 1}</span>
            <button type="button" class="sh-detail-track-play" data-track-play aria-label="${escapeHtml(t('play', 'Play'))}">&#9654;</button>
          </span>
          ${coverMarkup(track.albumId || fallbackAlbumId ? API.albumCoverThumbUrl(track.albumId || fallbackAlbumId, 64) : '/static/logo.png', track.title, 'sh-detail-track-cover')}
          <span class="sh-detail-track-text">
            <strong>${escapeHtml(track.title)}</strong>
            <small>${escapeHtml(track.artist || t('unknown_artist', 'Unknown artist'))}</small>
          </span>
          <span class="sh-detail-track-row-meta">
            <span class="sh-detail-track-duration" data-track-duration data-track-id="${escapeHtml(track.id)}">${escapeHtml(formatDuration(track.durationSeconds || 0))}</span>
            <button type="button" class="sh-detail-track-menu" data-track-menu aria-label="${escapeHtml(t('info', 'Information'))}">&#8942;</button>
          </span>
        </article>
      `).join('')}
    </div>
  `;
}

function renderPlaylistTrackSearch() {
  return `
    <div class="sh-detail-playlist-search">
      <input
        type="search"
        class="sh-detail-playlist-search-input"
        data-playlist-search-input
        placeholder="${escapeHtml(t('playlist_track_search_placeholder', 'Search tracks to add'))}"
      >
      <div class="sh-detail-playlist-search-results" data-playlist-search-results hidden></div>
    </div>
  `;
}

function renderInfoList(items) {
  return `
    <div class="sh-detail-info-list">
      ${items.map((item) => `
        <div><span>${escapeHtml(item.label)}</span><strong>${escapeHtml(item.value)}</strong></div>
      `).join('')}
    </div>
  `;
}

function renderEntityEditForm(config) {
  if (!config || !Array.isArray(config.fields) || !config.fields.length) {
    return '';
  }
  return `
    <form class="sh-detail-edit-form" data-detail-edit-form hidden>
      <div class="sh-detail-edit-title">${escapeHtml(config.title || t('edit', 'Edit'))}</div>
      <div class="sh-detail-edit-grid">
        ${config.fields.map(renderEditField).join('')}
      </div>
      <div class="sh-detail-edit-status" data-detail-edit-status hidden></div>
      <div class="sh-detail-edit-actions">
        <button type="submit" class="sh-detail-action primary">${escapeHtml(t('save', 'Save'))}</button>
        <button type="button" class="sh-detail-action" data-detail-edit-cancel>${escapeHtml(t('cancel', 'Cancel'))}</button>
      </div>
    </form>
  `;
}

function renderEditField(field) {
  const label = `
    <label class="sh-detail-edit-field">
      <span>${escapeHtml(field.label || '')}</span>
      ${renderEditControl(field)}
    </label>
  `;
  return label;
}

function renderEditControl(field) {
  const name = escapeHtml(field.name || '');
  const required = field.required ? ' required' : '';
  if (field.type === 'select') {
    return `
      <select name="${name}"${required}>
        ${(Array.isArray(field.options) ? field.options : []).map((option) => `
          <option value="${escapeHtml(option.value)}"${option.value === field.value ? ' selected' : ''}>${escapeHtml(option.label)}</option>
        `).join('')}
      </select>
    `;
  }
  if (field.type === 'textarea') {
    return `<textarea name="${name}" rows="${Number(field.rows || 4)}"${required}>${escapeHtml(field.value || '')}</textarea>`;
  }
  if (field.type === 'file') {
    return `<input type="file" name="${name}" accept="${escapeHtml(field.accept || '*/*')}"${required}>`;
  }
  const min = field.min !== undefined ? ` min="${escapeHtml(field.min)}"` : '';
  const max = field.max !== undefined ? ` max="${escapeHtml(field.max)}"` : '';
  return `<input type="${escapeHtml(field.type || 'text')}" name="${name}" value="${escapeHtml(field.value || '')}"${required}${min}${max}>`;
}

function bindDetailActions(root, context, handlers) {
  root.querySelectorAll('[data-detail-action]').forEach((button) => {
    button.addEventListener('click', () => {
      const action = button.getAttribute('data-detail-action') || '';
      handlers[action]?.();
    });
  });
}

function bindTrackInteractions(root, context, tracks, options = {}) {
  bindDetailLinks(root, context);
  const panel = root.querySelector('[data-track-panel]');
  const favoriteTrackIDs = options.favoriteTrackIDs instanceof Set ? options.favoriteTrackIDs : new Set();
  root.querySelectorAll('[data-track-row]').forEach((row, index) => {
    const track = tracks[index];
    if (!track) return;

    row.querySelector('[data-track-play]')?.addEventListener('click', (event) => {
      event.preventDefault();
      event.stopPropagation();
      context.player.playTrack(track);
    });

    row.querySelector('[data-track-menu]')?.addEventListener('click', (event) => {
      event.preventDefault();
      event.stopPropagation();
      showContextMenu(event, buildTrackRowMenuItems(root, context, track, {
        panel,
        albumId: row.getAttribute('data-track-album-id') || track.albumId || '',
        albumTitle: row.getAttribute('data-track-album-title') || track.albumTitle || options.albumTitle || '',
        collectionType: options.collectionType || '',
        collectionId: options.collectionId || '',
        favoriteTrackIDs,
        allowRemoveFromCollection: Boolean(options.allowRemoveFromCollection)
      }));
    });

    row.addEventListener('click', (event) => {
      if (event.target.closest('[data-track-play]') || event.target.closest('[data-track-menu]')) return;
      event.preventDefault();
      openTrackPanel(root, context, track, {
        panel,
        albumId: row.getAttribute('data-track-album-id') || track.albumId || '',
        albumTitle: row.getAttribute('data-track-album-title') || track.albumTitle || options.albumTitle || ''
      });
      root.querySelectorAll('[data-track-row].is-selected').forEach((item) => item.classList.remove('is-selected'));
      row.classList.add('is-selected');
    });
  });
}

function buildTrackRowMenuItems(root, context, track, options = {}) {
  const favoriteTrackIDs = options.favoriteTrackIDs instanceof Set ? options.favoriteTrackIDs : new Set();
  const isFavorite = favoriteTrackIDs.has(track.id);
  const items = [
    {
      label: t('play', 'Play'),
      action: async () => {
        context.player.playTrack(track);
      }
    },
    {
      label: t('add_to_queue', 'Add to queue'),
      action: async () => {
        context.player.appendTracks([track]);
      }
    },
    {
      label: isFavorite ? t('unlike', 'Dislike') : t('like', 'Like'),
      action: async () => {
        await API.toggleFavoriteTrack(track.id);
        if (isFavorite) {
          favoriteTrackIDs.delete(track.id);
        } else {
          favoriteTrackIDs.add(track.id);
        }
        if (options.collectionId === '__favorites__' && isFavorite) {
          await context.router.refresh();
        }
      }
    }
  ];
  if (options.allowRemoveFromCollection && options.collectionType === 'playlist' && options.collectionId) {
    items.push({
      label: t('remove_from_playlist', 'Remove from playlist'),
      action: async () => {
        await API.removeTrackFromPlaylist(options.collectionId, track.id);
        await context.router.refresh();
      }
    });
  }
  items.push(
    {
      label: t('delete', 'Delete'),
      danger: true,
      action: async () => {
        const prepared = context.player?.prepareTrackDeletion?.(track.id);
        try {
          await API.deleteTrack(track.id);
          context.player?.handleTrackDeleted?.(track.id);
          window.dispatchEvent(new CustomEvent('soundhub:library-updated'));
          await context.router.refresh();
        } catch (error) {
          if (prepared) {
            context.player?.recoverPreparedTrackDeletion?.(track.id);
          }
          throw error;
        }
      }
    },
    {
      label: t('info', 'Information'),
      action: async () => {
        openTrackPanel(root, context, track, options);
        const row = root.querySelector(`[data-track-row][data-track-id="${cssEscape(track.id)}"]`);
        root.querySelectorAll('[data-track-row].is-selected').forEach((item) => item.classList.remove('is-selected'));
        row?.classList.add('is-selected');
      }
    }
  );
  return items;
}

function openTrackPanel(root, context, track, options = {}) {
  const panel = options.panel || root.querySelector('[data-track-panel]');
  if (!panel || !track) return;
  const trackId = String(track.id || '').trim();
  if (!trackId) return;
  if (!panel.hidden && panel.dataset.trackId === trackId) {
    panel.hidden = true;
    panel.dataset.trackId = '';
    root.querySelector('.sh-detail-body-layout')?.classList.remove('has-track-panel');
    root.querySelectorAll('[data-track-row].is-selected').forEach((item) => item.classList.remove('is-selected'));
    return;
  }
  root.querySelector('.sh-detail-body-layout')?.classList.add('has-track-panel');
  const albumId = String(options.albumId || track.albumId || '').trim();
  const albumTitle = String(options.albumTitle || track.albumTitle || '').trim();
  panel.dataset.trackId = trackId;
  panel.hidden = false;
  panel.innerHTML = `
    <div class="sh-detail-track-panel-card">
      <button type="button" class="sh-detail-track-panel-close" data-track-panel-close aria-label="${escapeHtml(t('cancel', 'Close'))}">&times;</button>
      <div class="sh-detail-track-panel-kicker">${escapeHtml(t('track_info', 'Track'))}</div>
      <div class="sh-detail-track-panel-cover-wrap">
        ${coverMarkup(albumId ? `/api/covers/album/${encodeURIComponent(albumId)}` : '/static/logo.png', track.title, 'sh-detail-track-panel-cover')}
      </div>
      <h3 class="sh-detail-track-panel-title">${escapeHtml(track.title)}</h3>
      <div class="sh-detail-track-panel-subtitle">${escapeHtml(track.artist || t('unknown_artist', 'Unknown artist'))}</div>
      <div class="sh-detail-track-panel-meta">${escapeHtml(formatDuration(track.durationSeconds || 0))}</div>
      <div class="sh-detail-track-panel-actions">
        <button type="button" class="sh-detail-action primary" data-track-panel-action="play">${escapeHtml(t('play', 'Play'))}</button>
        <button type="button" class="sh-detail-action" data-track-panel-action="queue">${escapeHtml(t('add_to_queue', 'Add to queue'))}</button>
      </div>
      <div class="sh-detail-track-panel-block">
        <div class="sh-detail-track-panel-block-title">${escapeHtml(t('album_contents', 'Album'))}</div>
        <button type="button" class="sh-detail-track-panel-album"${albumId ? ` data-track-panel-open-album="${escapeHtml(albumId)}"` : ' disabled'}>
          ${coverMarkup(albumId ? API.albumCoverThumbUrl(albumId, 64) : '/static/logo.png', albumTitle || track.title, 'sh-detail-track-panel-album-cover')}
          <span>
            <strong>${escapeHtml(albumTitle || '-')}</strong>
            <small>${escapeHtml(albumId ? t('view', 'View') : t('unavailable', 'Unavailable'))}</small>
          </span>
        </button>
      </div>
      <button type="button" class="sh-detail-track-panel-open" data-track-panel-open-track>${escapeHtml(t('info', 'Information'))}</button>
    </div>
  `;
  panel.querySelector('[data-track-panel-action="play"]')?.addEventListener('click', () => {
    context.player.playTrack(track);
  });
  panel.querySelector('[data-track-panel-action="queue"]')?.addEventListener('click', () => {
    context.player.appendTracks([track]);
  });
  panel.querySelector('[data-track-panel-open-track]')?.addEventListener('click', () => {
    context.router.go(`/tracks/${encodeURIComponent(track.id)}`);
  });
  panel.querySelector('[data-track-panel-open-album]')?.addEventListener('click', (event) => {
    const targetAlbumId = event.currentTarget.getAttribute('data-track-panel-open-album');
    if (targetAlbumId) {
      context.router.go(`/albums/${encodeURIComponent(targetAlbumId)}`);
    }
  });
  panel.querySelector('[data-track-panel-close]')?.addEventListener('click', () => {
    panel.hidden = true;
    panel.dataset.trackId = '';
    root.querySelector('.sh-detail-body-layout')?.classList.remove('has-track-panel');
    root.querySelectorAll('[data-track-row].is-selected').forEach((item) => item.classList.remove('is-selected'));
  });
}

function bindDetailLinks(root, context) {
  root.querySelectorAll('[data-detail-link]').forEach((link) => {
    link.addEventListener('click', (event) => {
      event.preventDefault();
      const href = link.getAttribute('href');
      if (href) context.router.go(href);
    });
  });
}

function normalizeAlbums(response, artistMap) {
  return (Array.isArray(response) ? response : []).map((album) => normalizeAlbum(album, artistMap)).filter(Boolean);
}

function normalizeAlbum(album, artistMap) {
  if (!album) return null;
  const artistId = String(album.artistId || album.ArtistID || '').trim();
  return {
    id: String(album.id || album.ID || '').trim(),
    title: String(album.title || album.Title || '').trim(),
    year: Number(album.year || album.Year || 0) || 0,
    artistId,
    artist: artistMap.get(artistId) || '',
  };
}

function normalizeArtists(response) {
  return (Array.isArray(response) ? response : []).map((artist) => ({
    id: String(artist.id || artist.ID || '').trim(),
    name: String(artist.name || artist.Name || '').trim()
  }));
}

function normalizePlaylists(response) {
  return (Array.isArray(response) ? response : []).map((playlist) => normalizePlaylist(playlist)).filter(Boolean);
}

function normalizeTrack(track, artistMap) {
  if (!track) return null;
  return {
    id: String(track.id || track.ID || '').trim(),
    title: String(track.title || track.Title || '').trim(),
    artist: resolveArtistName(track, artistMap),
    artistId: String(track.artistId || track.ArtistID || '').trim(),
    albumId: String(track.albumId || track.AlbumID || '').trim(),
    albumTitle: String(track.albumTitle || track.AlbumTitle || '').trim(),
    durationSeconds: normalizeDurationSeconds(track)
  };
}

function normalizeTracks(response, artistMap) {
  return (Array.isArray(response) ? response : []).map((track) => normalizeTrack(track, artistMap)).filter(Boolean);
}

function normalizeDurationSeconds(track) {
  const raw = Number(track.duration_seconds ?? track.durationSeconds ?? track.DurationSeconds ?? track.duration ?? track.Duration ?? 0);
  if (!Number.isFinite(raw) || raw <= 0) return 0;
  if (raw > 1000000) return Math.round(raw / 1e9);
  return Math.round(raw);
}

function normalizeSearchText(value) {
  return String(value || '').trim().toLowerCase();
}

function trackSearchMatches(track, query) {
  const normalizedQuery = normalizeSearchText(query);
  if (!normalizedQuery) return false;
  return normalizeSearchText(track.title).includes(normalizedQuery)
    || normalizeSearchText(track.artist).includes(normalizedQuery)
    || normalizeSearchText(track.albumTitle).includes(normalizedQuery);
}

function formatDuration(seconds) {
  const safe = Math.max(0, Number.parseInt(String(seconds || 0), 10) || 0);
  const minutes = Math.floor(safe / 60);
  const sec = safe % 60;
  return `${minutes}:${String(sec).padStart(2, '0')}`;
}

function playlistCoverSrc(playlist) {
  return playlist.id ? `/static/logo.png` : '/static/logo.png';
}

function coverMarkup(src, alt, className) {
  return `<img src="${escapeHtml(src || '/static/logo.png')}" alt="${escapeHtml(alt || '')}" class="${escapeHtml(className)}" onerror="this.onerror=null;this.src='/static/logo.png';" />`;
}

function hydrateTrackDurations(root, tracks) {
  const byId = new Map(tracks.map((track) => [String(track.id), track]));
  root.querySelectorAll('[data-track-duration]').forEach((node) => {
    const trackId = String(node.getAttribute('data-track-id') || '').trim();
    const track = byId.get(trackId);
    if (!track?.id || Number(track.durationSeconds || 0) > 1) {
      return;
    }
    resolveStreamDuration(track.id).then((seconds) => {
      if (seconds > 1) {
        node.textContent = formatDuration(seconds);
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

function hydrateSingleTrackDuration(root, track) {
  const meta = root.querySelector('[data-detail-meta]');
  if (!meta || Number(track.durationSeconds || 0) > 1 || !track.id) {
    return;
  }
  resolveStreamDuration(track.id).then((seconds) => {
    if (seconds > 1) {
      meta.textContent = formatDuration(seconds);
    }
  }).catch(() => {});
}

function bindPlaylistTrackSearch(root, context, options) {
  const input = root.querySelector('[data-playlist-search-input]');
  const results = root.querySelector('[data-playlist-search-results]');
  if (!input || !results || !options?.playlist?.id) return;

  const playlistTracks = Array.isArray(options.tracks) ? options.tracks : [];
  const allTracks = Array.isArray(options.allTracks) ? options.allTracks : [];
  const playlistTrackIDs = new Set(playlistTracks.map((track) => track.id));

  const render = () => {
    const query = input.value.trim();
    if (!query) {
      results.hidden = true;
      results.innerHTML = '';
      return;
    }
    const inPlaylistMatches = playlistTracks.filter((track) => trackSearchMatches(track, query));
    const globalMatches = allTracks
      .filter((track) => !playlistTrackIDs.has(track.id))
      .filter((track) => trackSearchMatches(track, query))
      .slice(0, 25);

    if (!inPlaylistMatches.length && !globalMatches.length) {
      results.hidden = false;
      results.innerHTML = `<div class="sh-detail-playlist-search-empty">${escapeHtml(t('empty_no_results', 'No results'))}</div>`;
      return;
    }

    results.hidden = false;
    results.innerHTML = `
      ${inPlaylistMatches.length ? `
        <div class="sh-detail-playlist-search-group">
          <div class="sh-detail-playlist-search-title">${escapeHtml(t('playlist_search_in_playlist', 'Already in playlist'))}</div>
          ${inPlaylistMatches.map((track) => renderPlaylistSearchRow(track, true)).join('')}
        </div>
      ` : ''}
      ${globalMatches.length ? `
        <div class="sh-detail-playlist-search-group">
          <div class="sh-detail-playlist-search-title">${escapeHtml(t('playlist_search_global', 'Global matches'))}</div>
          ${globalMatches.map((track) => renderPlaylistSearchRow(track, false)).join('')}
        </div>
      ` : ''}
    `;

    results.querySelectorAll('[data-playlist-search-add]').forEach((button) => {
      button.addEventListener('click', async (event) => {
        event.preventDefault();
        event.stopPropagation();
        const trackId = String(button.getAttribute('data-playlist-search-add') || '').trim();
        if (!trackId) return;
        button.disabled = true;
        try {
          await API.addTrackToPlaylist(options.playlist.id, trackId, playlistTracks.length + 1);
          await context.router.refresh();
        } catch (error) {
          button.disabled = false;
          button.textContent = error?.message || t('save_failed', 'Failed to save changes');
        }
      });
    });

    results.querySelectorAll('[data-playlist-search-track]').forEach((node) => {
      node.addEventListener('click', (event) => {
        if (event.target.closest('[data-playlist-search-add]')) return;
        const trackId = String(node.getAttribute('data-playlist-search-track') || '').trim();
        const track = [...playlistTracks, ...allTracks].find((item) => item.id === trackId);
        if (track) {
          openTrackPanel(root, context, track, {
            albumId: track.albumId || '',
            albumTitle: track.albumTitle || ''
          });
        }
      });
    });
  };

  input.addEventListener('input', render);
  input.addEventListener('search', render);
}

function renderPlaylistSearchRow(track, inPlaylist) {
  return `
    <article class="sh-detail-playlist-search-row" data-playlist-search-track="${escapeHtml(track.id)}">
      ${coverMarkup(track.albumId ? API.albumCoverThumbUrl(track.albumId, 64) : '/static/logo.png', track.title, 'sh-detail-playlist-search-cover')}
      <span class="sh-detail-playlist-search-text">
        <strong>${escapeHtml(track.title)}</strong>
        <small>${escapeHtml(track.artist || t('unknown_artist', 'Unknown artist'))}</small>
      </span>
      ${inPlaylist ? `<span class="sh-detail-playlist-search-badge">${escapeHtml(t('playlist_search_in_playlist_badge', 'In playlist'))}</span>` : `
        <button type="button" class="sh-detail-playlist-search-add" data-playlist-search-add="${escapeHtml(track.id)}">
          ${escapeHtml(t('add_to_playlist', 'Add'))}
        </button>
      `}
    </article>
  `;
}

function bindHistoryControls(root) {
  const back = root.querySelector('[data-history-nav="back"]');
  const forward = root.querySelector('[data-history-nav="forward"]');
  if (!back || !forward) return;

  const update = () => {
    const currentIdx = Number(history.state?.__sh_idx ?? sessionStorage.getItem('sh-history-current') ?? 0) || 0;
    const maxIdx = Number(sessionStorage.getItem('sh-history-max') || '0') || 0;
    back.disabled = currentIdx <= 0;
    forward.disabled = currentIdx >= maxIdx;
  };

  back.addEventListener('click', () => {
    if (!back.disabled) history.back();
  });
  forward.addEventListener('click', () => {
    if (!forward.disabled) history.forward();
  });

  update();
  window.addEventListener('popstate', update, { once: true });
}

function bindDetailMenu(root, context, options) {
  const button = root.querySelector('[data-detail-menu-btn]');
  if (!button) return;
  button.addEventListener('click', (event) => {
    event.preventDefault();
    event.stopPropagation();
    const pinned = isPinned(options.entityType, options.entityId);
    const items = [
      {
        label: t('info', 'Information'),
        action: async () => {
          const section = root.querySelector('[data-detail-info-section]');
          if (!section) return;
          section.hidden = false;
          section.scrollIntoView({ behavior: 'smooth', block: 'start' });
        }
      },
      ...(options.canShare === false ? [] : [{
        label: t('share', 'Share'),
        action: async () => {
          await context.shareModal?.open?.(options.entityType, options.entityId);
        }
      }]),
      ...(options.canEdit === false ? [] : [{
        label: t('edit', 'Edit'),
        action: async () => {
          if (typeof options.onEdit === 'function') {
            await options.onEdit();
            return;
          }
          setDetailEditing(root, true);
        }
      }]),
      ...(typeof options.deleteAction === 'function' ? [{
        label: t('delete', 'Delete'),
        danger: true,
        action: options.deleteAction
      }] : []),
      {
        label: pinned ? t('unpin', 'Unpin') : t('pin', 'Pin'),
        action: async () => {
          togglePinned(options.entityType, options.entityId);
        }
      }
    ];
    showContextMenu(event, items);
  });
}

function pinStorageKey() {
  return 'soundhub-pinned-entities';
}

function readPinned() {
  try {
    return JSON.parse(localStorage.getItem(pinStorageKey()) || '[]');
  } catch {
    return [];
  }
}

function isPinned(type, id) {
  return readPinned().some((item) => item?.type === type && item?.id === id);
}

function togglePinned(type, id) {
  const current = readPinned();
  const next = current.some((item) => item?.type === type && item?.id === id)
    ? current.filter((item) => !(item?.type === type && item?.id === id))
    : [...current, { type, id }];
  localStorage.setItem(pinStorageKey(), JSON.stringify(next));
}

function setDetailEditing(root, enabled) {
  root.dataset.editing = enabled ? 'true' : 'false';
  const form = root.querySelector('[data-detail-edit-form]');
  const staticBlock = root.querySelector('[data-detail-static]');
  const state = root.querySelector('[data-detail-edit-state]');
  if (form) form.hidden = !enabled;
  if (staticBlock) staticBlock.hidden = enabled;
  if (state) state.hidden = !enabled;
}

function bindEditForm(root, onSubmit) {
  const form = root.querySelector('[data-detail-edit-form]');
  if (!form || typeof onSubmit !== 'function') {
    return;
  }
  const cancelButton = root.querySelector('[data-detail-edit-cancel]');
  const status = root.querySelector('[data-detail-edit-status]');

  form.addEventListener('submit', async (event) => {
    event.preventDefault();
    setEditStatus(status, '', false);
    form.classList.add('is-busy');
    try {
      await onSubmit(form);
    } catch (error) {
      setEditStatus(status, error?.message || 'Failed to save changes', true);
    } finally {
      form.classList.remove('is-busy');
    }
  });

  cancelButton?.addEventListener('click', () => {
    setEditStatus(status, '', false);
    setDetailEditing(root, false);
  });
}

function setEditStatus(node, message, isError) {
  if (!node) return;
  node.hidden = !message;
  node.textContent = message || '';
  node.dataset.state = isError ? 'error' : 'idle';
}

function readFormValue(form, name) {
  const field = form.elements.namedItem(name);
  return typeof field?.value === 'string' ? field.value.trim() : '';
}

async function loadFavoriteTrackIDs() {
  try {
    const home = await API.getHome(500);
    return new Set(
      (home?.favorites?.tracks || [])
        .map((track) => String(track?.id || track?.ID || '').trim())
        .filter(Boolean)
    );
  } catch {
    return new Set();
  }
}

function cssEscape(value) {
  if (typeof window.CSS?.escape === 'function') {
    return window.CSS.escape(String(value ?? ''));
  }
  return String(value ?? '').replace(/["\\]/g, '\\$&');
}

function buildArtistOptions(artists) {
  return (Array.isArray(artists) ? artists : []).map((artist) => ({
    value: artist.id,
    label: artist.name || t('unknown_artist', 'Unknown artist')
  }));
}

function buildAlbumOptions(albums) {
  return (Array.isArray(albums) ? albums : []).map((album) => ({
    value: album.id,
    label: album.artist ? `${album.title} - ${album.artist}` : album.title
  }));
}

function normalizePlaylist(playlist) {
  if (!playlist) return null;
  return {
    id: String(playlist.id || playlist.ID || '').trim(),
    name: String(playlist.name || playlist.Name || '').trim(),
    description: String(playlist.description || playlist.Description || '').trim(),
    system: Boolean(playlist.system || playlist.System),
    ownerUserId: String(playlist.owner_user_id || playlist.ownerUserId || playlist.OwnerUserID || '').trim(),
    accessRole: String(playlist.access_role || playlist.accessRole || playlist.AccessRole || '').trim()
  };
}

function renderNotFound(label) {
  return `<section class="sh-detail-page"><div class="sh-detail-frame"><p class="sh-detail-empty">${escapeHtml(label)}: not found</p></div></section>`;
}

function escapeHtml(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

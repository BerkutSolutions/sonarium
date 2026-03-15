export const API = {
  async getAuthStatus() {
    return request('/api/auth/status');
  },
  async login(payload) {
    return post('/api/auth/login', payload);
  },
  async register(payload) {
    return post('/api/auth/register', payload);
  },
  async logout() {
    return post('/api/auth/logout', {});
  },
  async getUsers() {
    return request('/api/auth/users');
  },
  async getShareableUsers() {
    return request('/api/auth/users/lookup');
  },
  async setUserActive(userId, active) {
    return post(`/api/auth/users/${encodeURIComponent(userId)}/active`, { active });
  },
  async deleteUser(userId) {
    return post(`/api/auth/users/${encodeURIComponent(userId)}/delete`, {});
  },
  async setRegistrationOpen(open) {
    return post('/api/auth/settings/registration', { open });
  },
  async updateProfile(payload) {
    return post('/api/auth/profile/update', payload);
  },
  async getProfile(userId) {
    return request(`/api/auth/profile/${encodeURIComponent(userId)}`);
  },
  async changePassword(payload) {
    return post('/api/auth/profile/password', payload);
  },
  async getArtists(params = {}) {
    return request('/api/artists', params);
  },
  async getArtistAlbums(artistId, params = {}) {
    return request(`/api/artists/${encodeURIComponent(artistId)}/albums`, params);
  },
  async getAlbums(params = {}) {
    return request('/api/albums', params);
  },
  async getAlbumTracks(albumId, params = {}) {
    return request(`/api/albums/${encodeURIComponent(albumId)}/tracks`, params);
  },
  async getAlbum(albumId) {
    return request(`/api/albums/${encodeURIComponent(albumId)}`);
  },
  async getTracks(params = {}) {
    return request('/api/tracks', params);
  },
  async getTrack(trackId) {
    return request(`/api/tracks/${encodeURIComponent(trackId)}`);
  },
  async getTrackWaveform(trackId) {
    return request(`/api/tracks/${encodeURIComponent(trackId)}/waveform`);
  },
  async getPlaylists(params = {}) {
    return request('/api/playlists', params);
  },
  async getPlaylist(playlistId, params = {}) {
    return request(`/api/playlists/${encodeURIComponent(playlistId)}`, params);
  },
  async createPlaylist(payload) {
    return post('/api/playlists', payload);
  },
  async updatePlaylist(playlistId, payload) {
    return post(`/api/playlists/${encodeURIComponent(playlistId)}/update`, payload);
  },
  async renamePlaylist(playlistId, name) {
    return post(`/api/playlists/${encodeURIComponent(playlistId)}/rename`, { name });
  },
  async deletePlaylist(playlistId) {
    const response = await fetch(`/api/playlists/${encodeURIComponent(playlistId)}`, { method: 'DELETE' });
    return readEnvelope(response);
  },
  async addTrackToPlaylist(playlistId, trackId, position) {
    return post(`/api/playlists/${encodeURIComponent(playlistId)}/tracks`, {
      track_id: trackId,
      position
    });
  },
  async removeTrackFromPlaylist(playlistId, trackId) {
    const response = await fetch(`/api/playlists/${encodeURIComponent(playlistId)}/tracks/${encodeURIComponent(trackId)}`, { method: 'DELETE' });
    return readEnvelope(response);
  },
  async search(query, params = {}) {
    return request('/api/search', { ...params, q: query });
  },
  async getHome(limit = 12) {
    return request('/api/library/home', { limit });
  },
  async getRandomAlbums(limit = 12) {
    return request('/api/library/random-albums', { limit });
  },
  async getArtistAlbumCounts() {
    return request('/api/library/artist-album-counts');
  },
  async scanLibrary() {
    return post('/api/library/scan', {});
  },
  async getLibraryScanStatus() {
    return request('/api/library/scan/status');
  },
  async uploadLibraryFile(file, { duplicatePolicy = 'keep', skipDuplicates = false } = {}) {
    const form = new FormData();
    form.append('file', file, file.name);
    form.append('skip_duplicates', skipDuplicates ? 'true' : 'false');
    form.append('duplicate_policy', String(duplicatePolicy || 'keep'));
    const response = await fetch('/api/library/upload', {
      method: 'POST',
      body: form
    });
    return readEnvelope(response);
  },
  uploadLibraryFileWithProgress(file, { onProgress, signal, duplicatePolicy = 'keep', skipDuplicates = false } = {}) {
    const form = new FormData();
    form.append('file', file, file.name);
    form.append('skip_duplicates', skipDuplicates ? 'true' : 'false');
    form.append('duplicate_policy', String(duplicatePolicy || 'keep'));
    return new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      xhr.open('POST', '/api/library/upload', true);
      xhr.responseType = 'json';
      xhr.upload.onprogress = (event) => {
        if (event.lengthComputable && typeof onProgress === 'function') {
          onProgress({
            loaded: event.loaded,
            total: event.total,
            percent: event.total > 0 ? Math.round((event.loaded / event.total) * 100) : 0
          });
        }
      };
      xhr.onload = () => {
        const payload = xhr.response || safeParseJSON(xhr.responseText);
        if (xhr.status >= 200 && xhr.status < 300 && !payload?.error) {
          resolve(payload?.data);
          return;
        }
        reject(new Error(payload?.error?.message || 'API request failed'));
      };
      xhr.onerror = () => reject(new Error('Network error'));
      xhr.onabort = () => reject(new Error('Upload cancelled'));
      if (signal) {
        if (signal.aborted) {
          xhr.abort();
          return;
        }
        signal.addEventListener('abort', () => xhr.abort(), { once: true });
      }
      xhr.send(form);
    });
  },
  async getSettings() {
    return request('/api/settings');
  },
  async checkUpdates() {
    return post('/api/settings/updates/check', {});
  },
  async setAutoCheckUpdates(enabled) {
    return post('/api/settings/updates/auto', { enabled });
  },
  async getStorageUsage() {
    return request('/api/settings/storage');
  },
  async getLibraryIntegrity() {
    return request('/api/library/integrity');
  },
  async deleteAllMusic() {
    return post('/api/settings/library/delete-all', {});
  },
  async setUploadConcurrency(value) {
    return post('/api/settings/upload-concurrency', { value });
  },
  async toggleFavoriteTrack(trackId) {
    return post(`/api/library/favorites/tracks/${encodeURIComponent(trackId)}/toggle`, {});
  },
  async deleteTrack(trackId) {
    return post(`/api/library/tracks/${encodeURIComponent(trackId)}/delete`, {});
  },
  async renameTrack(trackId, title) {
    return post(`/api/library/tracks/${encodeURIComponent(trackId)}/rename`, { title });
  },
  async updateTrack(trackId, payload) {
    return post(`/api/library/tracks/${encodeURIComponent(trackId)}/update`, payload);
  },
  async toggleFavoriteAlbum(albumId) {
    return post(`/api/library/favorites/albums/${encodeURIComponent(albumId)}/toggle`, {});
  },
  async deleteAlbum(albumId) {
    return post(`/api/library/albums/${encodeURIComponent(albumId)}/delete`, {});
  },
  async renameAlbum(albumId, title) {
    return post(`/api/library/albums/${encodeURIComponent(albumId)}/rename`, { title });
  },
  async createAlbum(payload) {
    return post('/api/library/albums/create', payload);
  },
  async updateAlbum(albumId, payload) {
    return post(`/api/library/albums/${encodeURIComponent(albumId)}/update`, payload);
  },
  async mergeAlbum(albumId, targetAlbumId) {
    return post(`/api/library/albums/${encodeURIComponent(albumId)}/merge`, { target_album_id: targetAlbumId });
  },
  async toggleFavoriteArtist(artistId) {
    return post(`/api/library/favorites/artists/${encodeURIComponent(artistId)}/toggle`, {});
  },
  async updateArtist(artistId, payload) {
    const body = typeof payload === 'string' ? { name: payload } : payload;
    return post(`/api/library/artists/${encodeURIComponent(artistId)}/update`, body);
  },
  async uploadArtistCover(artistId, file) {
    const form = new FormData();
    form.append('file', file, file.name);
    const response = await fetch(`/api/library/artists/${encodeURIComponent(artistId)}/cover`, {
      method: 'POST',
      body: form
    });
    return readEnvelope(response);
  },
  async deleteArtist(artistId) {
    return post(`/api/library/artists/${encodeURIComponent(artistId)}/delete`, {});
  },
  async getEntityShares(entityType, entityId) {
    return request(`/api/shares/${encodeURIComponent(entityType)}/${encodeURIComponent(entityId)}`);
  },
  async shareEntityWithUser(entityType, entityId, payload) {
    return post(`/api/shares/${encodeURIComponent(entityType)}/${encodeURIComponent(entityId)}/users`, payload);
  },
  async setEntityPublicShare(entityType, entityId, enabled) {
    return post(`/api/shares/${encodeURIComponent(entityType)}/${encodeURIComponent(entityId)}/public`, { enabled });
  },
  async deleteEntityShare(shareId) {
    const response = await fetch(`/api/shares/${encodeURIComponent(shareId)}`, { method: 'DELETE' });
    return readEnvelope(response);
  },
  async getReceivedShares(userId = '') {
    return request('/api/shares/received', userId ? { user_id: userId } : {});
  },
  async getPlayerState() {
    return request('/api/player/state');
  },
  async savePlayerState(payload) {
    return post('/api/player/state', payload);
  },
  async replacePlayerQueue(payload) {
    return post('/api/player/queue/replace', payload);
  },
  async appendPlayerQueue(payload) {
    return post('/api/player/queue/append', payload);
  },
  async removePlayerQueueItem(index) {
    return post('/api/player/queue/remove', { index });
  },
  async clearPlayerQueue() {
    return post('/api/player/queue/clear', {});
  },
  async movePlayerQueueItem(from, to) {
    return post('/api/player/queue/move', { from, to });
  },
  async setPlayerShuffle(enabled) {
    return post('/api/player/queue/shuffle', { enabled });
  },
  async playerPlayed(payload) {
    return post('/api/player/played', payload);
  },
  streamUrl(trackId) {
    return `/api/stream/${encodeURIComponent(trackId)}`;
  },
  placeholderCoverUrl(_label = '') {
    return '/static/logo.png';
  },
  albumCoverThumbUrl(albumId, size = 256) {
    return `/api/covers/album/${encodeURIComponent(albumId)}/thumb/${encodeURIComponent(size)}`;
  }
};

let unauthorizedNotified = false;

async function request(path, query = {}) {
  const url = new URL(path, window.location.origin);
  Object.entries(query).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') {
      url.searchParams.set(key, String(value));
    }
  });

  const response = await fetch(url.toString());
  return readEnvelope(response);
}

async function post(path, payload) {
  const response = await fetch(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  });
  return readEnvelope(response);
}

async function readEnvelope(response) {
  const text = await response.text();
  const payload = safeParseJSON(text) || {};
  if (response.status === 401) {
    notifyUnauthorized();
  } else if (response.ok) {
    unauthorizedNotified = false;
  }
  if (!response.ok || payload.error) {
    throw new Error(payload?.error?.message || 'API request failed');
  }
  return payload.data;
}

function notifyUnauthorized() {
  if (unauthorizedNotified) return;
  unauthorizedNotified = true;
  window.dispatchEvent(new CustomEvent('soundhub:session-expired'));
}

function safeParseJSON(value) {
  try {
    return JSON.parse(value);
  } catch {
    return null;
  }
}

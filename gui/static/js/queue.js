export class QueueModel {
  constructor() {
    this.items = [];
    this.position = -1;
    this.contextType = '';
    this.contextID = '';
    this.shuffleEnabled = false;
    this._originalOrder = null;
  }

  replace(items, startIndex = 0, contextType = '', contextID = '') {
    this.items = compactItems(items);
    this.contextType = contextType || '';
    this.contextID = contextID || '';
    this._originalOrder = null;
    if (!this.items.length) {
      this.position = -1;
      return;
    }
    this.position = clamp(startIndex, 0, this.items.length - 1);
  }

  append(items) {
    this.items.push(...compactItems(items));
    if (this.position === -1 && this.items.length > 0) {
      this.position = 0;
    }
  }

  remove(index) {
    if (index < 0 || index >= this.items.length) return false;
    this.items.splice(index, 1);
    if (!this.items.length) {
      this.position = -1;
      return true;
    }
    if (index < this.position) {
      this.position -= 1;
    } else if (index === this.position && this.position >= this.items.length) {
      this.position = this.items.length - 1;
    }
    return true;
  }

  clear() {
    this.items = [];
    this.position = -1;
    this.contextType = '';
    this.contextID = '';
    this._originalOrder = null;
    this.shuffleEnabled = false;
  }

  move(from, to) {
    if (from < 0 || from >= this.items.length || to < 0 || to >= this.items.length) return false;
    if (from === to) return true;
    const [item] = this.items.splice(from, 1);
    this.items.splice(to, 0, item);
    this._reindexByTrackID(this.current()?.track_id || '');
    return true;
  }

  current() {
    if (this.position < 0 || this.position >= this.items.length) return null;
    return this.items[this.position];
  }

  setPosition(index) {
    if (index < 0 || index >= this.items.length) return false;
    this.position = index;
    return true;
  }

  nextIndex(repeatMode) {
    if (!this.items.length) return -1;
    if (this.position < 0 || this.position >= this.items.length) {
      this.position = 0;
    }
    if (repeatMode === 'one') return this.position;
    const next = this.position + 1;
    if (next < this.items.length) return next;
    if (repeatMode === 'all') return 0;
    return -1;
  }

  previousIndex(repeatMode) {
    if (!this.items.length) return -1;
    if (this.position < 0 || this.position >= this.items.length) {
      this.position = 0;
    }
    if (repeatMode === 'one') return this.position;
    const prev = this.position - 1;
    if (prev >= 0) return prev;
    if (repeatMode === 'all') return this.items.length - 1;
    return -1;
  }

  setShuffle(enabled) {
    if (this.shuffleEnabled === enabled) return;
    this.shuffleEnabled = enabled;
    const current = this.current();
    if (!current) return;
    const currentID = current.track_id;
    if (enabled) {
      this._originalOrder = this.items.map((item) => ({ ...item }));
      this._shuffleItems();
      this._reindexByTrackID(currentID);
      return;
    }
    if (!this._originalOrder) return;
    const restored = this._originalOrder.map((item) => ({ ...item }));
    this.items = restored;
    this._originalOrder = null;
    this._reindexByTrackID(currentID);
  }

  snapshot() {
    return {
      queue: this.items.map((item) => ({ ...item })),
      queue_position: this.position,
      context_type: this.contextType,
      context_id: this.contextID
    };
  }

  restore(snapshot) {
    this.items = compactItems(snapshot?.queue || []);
    this.position = Number.isInteger(snapshot?.queue_position) ? snapshot.queue_position : -1;
    this.contextType = snapshot?.context_type || '';
    this.contextID = snapshot?.context_id || '';
    this.shuffleEnabled = Boolean(snapshot?.shuffle_enabled);
    this._originalOrder = null;
    if (this.position >= this.items.length) {
      this.position = this.items.length ? this.items.length - 1 : -1;
    }
  }

  _shuffleItems() {
    for (let i = this.items.length - 1; i > 0; i -= 1) {
      const j = Math.floor(Math.random() * (i + 1));
      const tmp = this.items[i];
      this.items[i] = this.items[j];
      this.items[j] = tmp;
    }
  }

  _reindexByTrackID(trackID) {
    if (!trackID) return;
    const idx = this.items.findIndex((item) => item.track_id === trackID);
    if (idx >= 0) this.position = idx;
  }
}

function compactItems(items) {
  const compact = [];
  for (const raw of Array.isArray(items) ? items : []) {
    const trackID = String(raw.track_id || raw.id || raw.ID || '').trim();
    if (!trackID) continue;
    compact.push({
      track_id: trackID,
      title: String(raw.title || raw.Title || '').trim(),
      artist: String(raw.artist || raw.Artist || raw.artist_name || raw.artistName || '').trim(),
      artist_id: String(raw.artist_id || raw.artistId || raw.ArtistID || '').trim(),
      duration: toInt(raw.duration || raw.Duration || 0),
      cover_ref: String(raw.cover_ref || raw.album_id || raw.albumId || raw.AlbumID || '').trim()
    });
  }
  return compact;
}

function toInt(value) {
  const parsed = Number.parseInt(String(value), 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 0;
}

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value));
}

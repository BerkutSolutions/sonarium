const STORAGE_KEY = 'soundhub.player.state.v2';
const MAX_PERSISTED_QUEUE = 1000;
const WINDOW_SIZE = 400;

export function loadPlaybackState() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw);
    return sanitizeState(parsed);
  } catch (_) {
    return null;
  }
}

export function savePlaybackState(state) {
  const sanitized = sanitizeState(state);
  if (!sanitized) return;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(sanitized));
  } catch (_) {
    // storage quota can be exceeded on very large queues; ignore and continue
  }
}

export function clearPlaybackState() {
  try {
    localStorage.removeItem(STORAGE_KEY);
  } catch (_) {
    // best-effort cleanup
  }
}

export function sanitizeState(raw) {
  if (!raw || typeof raw !== 'object') return null;

  const queue = compactQueue(raw.queue || []);
  const safeQueue = trimQueueWindow(queue, Number(raw.queue_position || -1));
  const queuePosition = safeQueue.position;

  return {
    current_track_id: String(raw.current_track_id || '').trim(),
    queue: safeQueue.queue,
    queue_position: queuePosition,
    is_playing: Boolean(raw.is_playing),
    shuffle_enabled: Boolean(raw.shuffle_enabled),
    repeat_mode: normalizeRepeatMode(raw.repeat_mode),
    volume: clamp(Number(raw.volume ?? 0.5), 0, 1),
    current_time_seconds: 0,
    context_type: String(raw.context_type || '').trim(),
    context_id: String(raw.context_id || '').trim()
  };
}

function compactQueue(items) {
  const out = [];
  for (const item of Array.isArray(items) ? items : []) {
    const trackID = String(item.track_id || item.id || item.ID || '').trim();
    if (!trackID) continue;
    out.push({
      track_id: trackID,
      title: String(item.title || item.Title || '').trim(),
      artist: String(item.artist || item.Artist || '').trim(),
      artist_id: String(item.artist_id || item.artistId || item.ArtistID || '').trim(),
      duration: Math.max(0, Number.parseInt(String(item.duration || item.Duration || 0), 10) || 0),
      cover_ref: String(item.cover_ref || item.coverRef || item.album_id || item.albumId || '').trim()
    });
  }
  return out;
}

function trimQueueWindow(queue, position) {
  if (queue.length <= MAX_PERSISTED_QUEUE) {
    return {
      queue,
      position: clamp(position, queue.length ? 0 : -1, queue.length ? queue.length - 1 : -1)
    };
  }
  const safePosition = clamp(position, 0, queue.length - 1);
  const start = clamp(safePosition - Math.floor(WINDOW_SIZE / 2), 0, queue.length - WINDOW_SIZE);
  const end = Math.min(queue.length, start + WINDOW_SIZE);
  return {
    queue: queue.slice(start, end),
    position: safePosition - start
  };
}

function normalizeRepeatMode(value) {
  if (value === 'one' || value === 'all') return value;
  return 'off';
}

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value));
}

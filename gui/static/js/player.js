import { API } from './api.js';
import { QueueModel } from './queue.js';
import { loadPlaybackState, savePlaybackState, sanitizeState } from './playback-state.js';
import { applyTranslations, t } from './i18n.js';
import { loadArtistNameMap } from './artist-map.js';

export class Player {
  constructor(root, options = {}) {
    this.root = root;
    this.auth = options.auth || null;
    this.audio = root.querySelector('#audio-player');
    this.cover = root.querySelector('#player-cover');
    this.title = root.querySelector('#player-title');
    this.artist = root.querySelector('#player-artist');
    this.meta = root.querySelector('.sh-player-meta');
    this.surface = root.querySelector('.sh-player-surface');
    this.main = root.querySelector('.sh-player-main');
    this.progress = root.querySelector('#player-progress');
    this.progressWrap = root.querySelector('.sh-player-progress-frame');
    this.progressOverlay = root.querySelector('.sh-player-progress-overlay');
    this.progressDot = root.querySelector('#player-progress-dot');
    this.volume = root.querySelector('#player-volume');
    this.currentTime = root.querySelector('#player-time-current');
    this.duration = root.querySelector('#player-time-duration');
    this.waveformCanvas = root.querySelector('#player-waveform');
    this.queueList = root.querySelector('#player-queue-list');
    this.queuePanel = root.querySelector('#player-queue-panel');
    this.queueToggle = root.querySelector('#player-queue-toggle');
    this.repeatButton = root.querySelector('#player-repeat');
    this.shuffleButton = root.querySelector('#player-shuffle');
    this.toggleButton = root.querySelector('#player-toggle');
    this.clearQueueButton = root.querySelector('#player-queue-clear');

    this.queue = new QueueModel();
    this.isPlaying = false;
    this.repeatMode = 'off';
    this._syncTimer = null;
    this._artistMapPromise = null;
    this._audioContext = null;
    this._analyser = null;
    this._sourceNode = null;
    this._visualizerFrame = 0;
    this._progressFrame = 0;
    this._restoredCurrentTime = 0;
    this._progressAnchorTime = 0;
    this._progressAnchorPerf = 0;
    this._dragQueueIndex = -1;
    this._queueOutsideClickHandler = (event) => {
      if (!this.queuePanel.classList.contains('open')) return;
      if (event.target.closest('#player-queue-panel')) return;
      if (event.target.closest('#player-queue-toggle')) return;
      this.queuePanel.classList.remove('open');
      this.queueToggle.classList.remove('active');
    };
    this._progressHoverOn = () => {
      this.root.classList.add('progress-hover');
      window.requestAnimationFrame(() => this._renderProgress());
    };
    this._progressHoverOff = () => this.root.classList.remove('progress-hover');

    this.audio.volume = 0.5;

    this._bindEvents();
    this._applyInitialState();
    this._render();
    this.onLanguageChanged();
  }

  async _applyInitialState() {
    const local = loadPlaybackState();
    if (local) {
      const sanitizedLocal = this._isAuthenticated()
        ? await this._sanitizeStateAgainstLibrary(local)
        : local;
      this._applyState(sanitizedLocal, false, { hydrateMedia: this._isAuthenticated() });
    }
    if (!this._isAuthenticated()) {
      if (!local) {
        this.volume.value = '0.5';
        this._setVolumeVisual(0.5);
      }
      return;
    }
    try {
      const remote = await API.getPlayerState();
      if (remote && Array.isArray(remote.queue) && remote.queue.length > 0 && !local) {
        const sanitizedRemote = await this._sanitizeStateAgainstLibrary(remote);
        this._applyState(sanitizedRemote, false);
      }
    } catch (_) {
      // endpoint is best-effort for this stage
    }
    if (!local) {
      this.volume.value = '0.5';
      this._setVolumeVisual(0.5);
    }
  }

  _bindEvents() {
    document.addEventListener('click', this._queueOutsideClickHandler);
    this.progressWrap?.addEventListener('mouseenter', this._progressHoverOn);
    this.progressWrap?.addEventListener('mouseleave', this._progressHoverOff);
    this.root.addEventListener('mouseenter', () => {
      window.requestAnimationFrame(() => this._renderProgress());
    });
    this.progress?.addEventListener('focus', this._progressHoverOn);
    this.progress?.addEventListener('blur', this._progressHoverOff);

    this.toggleButton.addEventListener('click', () => {
      if (this.audio.paused) {
        this.audio.play().catch(() => {});
        return;
      }
      this.audio.pause();
    });
    this.root.querySelector('#player-prev').addEventListener('click', () => this.previous());
    this.root.querySelector('#player-next').addEventListener('click', () => this.next());
    this.shuffleButton.addEventListener('click', () => this.toggleShuffle());
    this.repeatButton.addEventListener('click', () => this.cycleRepeatMode());
    this.queueToggle.addEventListener('click', () => this.toggleQueuePanel());
    this.clearQueueButton.addEventListener('click', () => this.clearQueue());

    this.volume.addEventListener('input', () => {
      const nextVolume = clamp(Number(this.volume.value), 0, 1);
      this.audio.volume = nextVolume;
      this._setVolumeVisual(nextVolume);
      this._scheduleSync();
    });

    this.progress.addEventListener('input', () => {
      if (!this.audio.duration) return;
      const nextTime = (Number(this.progress.value) / 10000) * this.audio.duration;
      this.audio.currentTime = nextTime;
      this._restoredCurrentTime = nextTime;
      this._syncProgressAnchor(nextTime);
      this.currentTime.textContent = formatTime(Math.floor(nextTime));
      this._setProgressVisual(this.audio.duration ? (nextTime / this.audio.duration) : 0);
      this._scheduleSync();
    });

    this.audio.addEventListener('play', () => {
      this.isPlaying = true;
      this._restoredCurrentTime = Math.max(0, Number(this.audio.currentTime || this._restoredCurrentTime || 0));
      this._syncProgressAnchor(this._restoredCurrentTime);
      this._startVisualizer();
      this._startProgressLoop();
      this._renderControls();
      this._scheduleSync();
    });
    this.audio.addEventListener('pause', () => {
      this.isPlaying = false;
      this._restoredCurrentTime = Math.max(0, Number(this.audio.currentTime || this._restoredCurrentTime || 0));
      this._syncProgressAnchor(this._restoredCurrentTime);
      this._stopVisualizer();
      this._stopProgressLoop();
      this._renderProgress();
      this._renderControls();
      this._scheduleSync();
    });
    this.audio.addEventListener('timeupdate', () => {
      this._restoredCurrentTime = Math.max(0, Number(this.audio.currentTime || this._restoredCurrentTime || 0));
      this._syncProgressAnchor(this._restoredCurrentTime);
      this._renderProgress();
    });
    this.audio.addEventListener('loadedmetadata', () => {
      if (this._restoredCurrentTime > 0) {
        try {
          this.audio.currentTime = clamp(this._restoredCurrentTime, 0, Math.max(0, Number(this.audio.duration || 0)));
        } catch (_) {
          // best-effort restore
        }
      }
      this._syncProgressAnchor(Math.max(0, Number(this.audio.currentTime || this._restoredCurrentTime || 0)));
      this.duration.textContent = formatTime(Math.floor(this.audio.duration || 0));
      this._renderProgress();
    });
    this.audio.addEventListener('seeking', () => {
      this._syncProgressAnchor(Math.max(0, Number(this.audio.currentTime || this._restoredCurrentTime || 0)));
    });
    this.audio.addEventListener('seeked', () => {
      this._restoredCurrentTime = Math.max(0, Number(this.audio.currentTime || this._restoredCurrentTime || 0));
      this._syncProgressAnchor(this._restoredCurrentTime);
      this._renderProgress();
    });
    this.audio.addEventListener('ended', () => {
      this._stopProgressLoop();
      this.next(true);
    });
    this.audio.addEventListener('error', () => {
      this._handleMissingCurrentTrack(this.queue.current()?.track_id || '');
    });

    window.addEventListener('resize', () => this._renderProgress());
    this.meta?.addEventListener('click', (event) => {
      if (event.target.closest('button')) return;
      this._navigateToCurrentTrack();
    });
  }

  replaceQueueFromTracks(tracks, startIndex, contextType, contextID, autoplay = true) {
    const queueItems = compactTracks(tracks);
    this.queue.replace(queueItems, startIndex, contextType, contextID);
    this.queue.setShuffle(false);
    this.shuffleButton.classList.remove('active');
    this.repeatMode = this.repeatMode || 'off';
    this._pushQueueReplace();
    if (autoplay) {
      this.playAt(this.queue.position);
    } else {
      this._render();
    }
  }

  appendTracks(tracks) {
    const queueItems = compactTracks(tracks);
    if (!queueItems.length) return;
    this.queue.append(queueItems);
    this._pushQueueAppend(queueItems);
    this._render();
    this._scheduleSync();
  }

  addNext(track) {
    const queueItems = compactTracks([track]);
    if (!queueItems.length) return;
    const insertAt = this._currentIndex();
    if (insertAt < 0 || insertAt >= this.queue.items.length - 1) {
      this.appendTracks(queueItems);
      return;
    }
    this.queue.items.splice(insertAt + 1, 0, queueItems[0]);
    this._pushQueueReplace();
    this._renderQueue();
    this._scheduleSync();
  }

  setQueue(tracks, startIndex = 0) {
    this.replaceQueueFromTracks(tracks, startIndex, 'tracks', 'list', true);
  }

  playTrack(track) {
    this.replaceQueueFromTracks([track], 0, 'track', compactTracks([track])[0]?.track_id || '', true);
  }

  playAt(index) {
    if (!this.queue.setPosition(index)) return;
    const current = this.queue.current();
    if (!current) return;

    this.title.textContent = current.title || t('unknown_title', 'Unknown title');
    this.artist.textContent = current.artist || t('unknown_artist', 'Unknown artist');
    this.cover.src = current.cover_ref ? API.albumCoverThumbUrl(current.cover_ref, 256) : '/static/logo.png';
    this.cover.onerror = () => {
      this.cover.src = '/static/logo.png';
    };

    this.audio.pause();
    this._restoredCurrentTime = 0;
    this._syncProgressAnchor(0);
    try {
      this.audio.currentTime = 0;
    } catch (_) {
      // metadata may not be ready yet
    }
    this.audio.src = API.streamUrl(current.track_id);
    this.audio.load();
    this._recordPlayed(current.track_id);
    this._resolveArtistName(current);
    this._hydrateCurrentMetadata(current);
    this.audio.play().catch(() => {});
    this._renderProgress();
    this._renderQueue();
    this._scheduleSync();
  }

  previous() {
    if (this.audio.currentTime > 3) {
      this.audio.currentTime = 0;
      this._scheduleSync();
      return;
    }
    const prevIndex = this._previousIndex();
    if (prevIndex < 0) return;
    this.playAt(prevIndex);
  }

  next(fromEnded = false) {
    const nextIndex = this._nextIndex();
    if (nextIndex < 0) return;
    if (fromEnded && nextIndex === this.queue.position && this.repeatMode !== 'one') return;
    this.playAt(nextIndex);
  }

  toggleShuffle() {
    const next = !this.queue.shuffleEnabled;
    this.queue.setShuffle(next);
    this.shuffleButton.classList.toggle('active', next);
    this._pushShuffle(next);
    this._renderQueue();
    this._scheduleSync();
  }

  cycleRepeatMode() {
    this.repeatMode = nextRepeatMode(this.repeatMode);
    this.repeatButton.dataset.mode = this.repeatMode;
    this.repeatButton.setAttribute('title', formatRepeatLabel(this.repeatMode));
    this._scheduleSync();
  }

  toggleQueuePanel() {
    const open = !this.queuePanel.classList.contains('open');
    this.queuePanel.classList.toggle('open', open);
    this.queueToggle.classList.toggle('active', open);
  }

  removeQueueItem(index) {
    const ok = this.queue.remove(index);
    if (!ok) return;
    this._pushQueueRemove(index);
    this._render();
    this._scheduleSync();
  }

  clearQueue() {
    this.queue.clear();
    this.audio.pause();
    this.audio.removeAttribute('src');
    this.audio.load();
    this._pushQueueClear();
    this._render();
    this._scheduleSync();
  }

  _render() {
    const current = this.queue.current();
    if (current) {
      this.title.textContent = current.title || t('unknown_title', 'Unknown title');
      this.artist.textContent = current.artist || t('unknown_artist', 'Unknown artist');
      this.cover.src = current.cover_ref ? API.albumCoverThumbUrl(current.cover_ref, 256) : '/static/logo.png';
      this.cover.onerror = () => {
        this.cover.src = '/static/logo.png';
      };
      this.meta?.classList.add('is-clickable');
    } else {
      this.title.textContent = t('player_no_track', 'No track selected');
      this.artist.textContent = '-';
      this.cover.src = '/static/logo.png';
      this.meta?.classList.remove('is-clickable');
    }
    this.volume.value = String(this.audio.volume || 1);
    this._setVolumeVisual(Number(this.audio.volume || 1));
    this._renderProgress();
    this.repeatButton.dataset.mode = this.repeatMode;
    this.repeatButton.setAttribute('title', formatRepeatLabel(this.repeatMode));
    this.shuffleButton.classList.toggle('active', this.queue.shuffleEnabled);
    this._renderQueue();
    this._drawLiveSpectrum(this.audio.paused || this.audio.ended);
    this._renderControls();
    applyTranslations(this.root);
  }

  _renderControls() {
    this.toggleButton.classList.toggle('is-playing', this.isPlaying);
    this.toggleButton.setAttribute('aria-label', this.isPlaying ? t('player_pause', 'Pause') : t('player_play', 'Play'));
    this.toggleButton.setAttribute('data-i18n-aria-label', this.isPlaying ? 'player_pause' : 'player_play');
  }

  _renderQueue() {
    const currentID = this.queue.current()?.track_id || '';
    if (!this.queue.items.length) {
      this.queueList.innerHTML = `<li class="sh-queue-empty">${escapeHtml(t('player_queue_empty', 'Queue is empty'))}</li>`;
      return;
    }

    this._renderQueueEnhanced(currentID);
    return;

    this.queueList.innerHTML = this.queue.items.map((item, index) => `
      <li class="sh-queue-item ${item.track_id === currentID ? 'active' : ''}">
        <button type="button" class="sh-queue-play" data-action="play" data-index="${index}">
          <span class="sh-queue-title">${escapeHtml(item.title || t('unknown_title', 'Unknown title'))}</span>
          <span class="sh-queue-artist">${escapeHtml(item.artist || t('unknown_artist', 'Unknown artist'))}</span>
        </button>
        <div class="sh-queue-controls">
          <button type="button" class="sh-queue-move" data-action="up" data-index="${index}" aria-label="${escapeHtml(t('move_up', 'Move up'))}">↑</button>
          <button type="button" class="sh-queue-move" data-action="down" data-index="${index}" aria-label="${escapeHtml(t('move_down', 'Move down'))}">↓</button>
          <button type="button" class="sh-queue-remove" data-action="remove" data-index="${index}" aria-label="${escapeHtml(t('remove_from_queue', 'Remove from queue'))}">x</button>
        </div>
      </li>
    `).join('');

    this.queueList.querySelectorAll('[data-action="play"]').forEach((button) => {
      button.addEventListener('click', () => {
        const index = Number(button.getAttribute('data-index'));
        this.playAt(index);
      });
    });
    this.queueList.querySelectorAll('[data-action="remove"]').forEach((button) => {
      button.addEventListener('click', () => {
        const index = Number(button.getAttribute('data-index'));
        this.removeQueueItem(index);
      });
    });
    this.queueList.querySelectorAll('[data-action="up"]').forEach((button) => {
      button.addEventListener('click', () => {
        const index = Number(button.getAttribute('data-index'));
        this.moveQueueItem(index, index - 1);
      });
    });
    this.queueList.querySelectorAll('[data-action="down"]').forEach((button) => {
      button.addEventListener('click', () => {
        const index = Number(button.getAttribute('data-index'));
        this.moveQueueItem(index, index + 1);
      });
    });
  }

  _renderQueueEnhanced(currentID) {
    this.queueList.innerHTML = this.queue.items.map((item, index) => `
      <li class="sh-queue-item ${item.track_id === currentID ? 'active' : ''}" draggable="true" data-queue-index="${index}">
        <button type="button" class="sh-queue-drag" data-action="drag" data-index="${index}" aria-label="${escapeHtml(t('reorder', 'Reorder'))}">
          <svg viewBox="0 0 16 16" aria-hidden="true">
            <circle cx="5" cy="4" r="1.1"></circle>
            <circle cx="11" cy="4" r="1.1"></circle>
            <circle cx="5" cy="8" r="1.1"></circle>
            <circle cx="11" cy="8" r="1.1"></circle>
            <circle cx="5" cy="12" r="1.1"></circle>
            <circle cx="11" cy="12" r="1.1"></circle>
          </svg>
        </button>
        <button type="button" class="sh-queue-play" data-action="play" data-index="${index}">
          <span class="sh-queue-title">${escapeHtml(item.title || t('unknown_title', 'Unknown title'))}</span>
          <span class="sh-queue-artist">${escapeHtml(item.artist || t('unknown_artist', 'Unknown artist'))}</span>
        </button>
        <div class="sh-queue-controls">
          <button type="button" class="sh-queue-remove" data-action="remove" data-index="${index}" aria-label="${escapeHtml(t('remove_from_queue', 'Remove from queue'))}">×</button>
        </div>
      </li>
    `).join('');

    this.queueList.querySelectorAll('[data-action="play"]').forEach((button) => {
      button.addEventListener('click', () => {
        const index = Number(button.getAttribute('data-index'));
        this.playAt(index);
      });
    });

    this.queueList.querySelectorAll('[data-action="remove"]').forEach((button) => {
      button.addEventListener('click', () => {
        const index = Number(button.getAttribute('data-index'));
        this.removeQueueItem(index);
      });
    });

    this.queueList.querySelectorAll('.sh-queue-item').forEach((item) => {
      item.addEventListener('dragstart', (event) => {
        const index = Number(item.getAttribute('data-queue-index'));
        if (!Number.isInteger(index) || index < 0) return;
        this._dragQueueIndex = index;
        item.classList.add('dragging');
        if (event.dataTransfer) {
          event.dataTransfer.effectAllowed = 'move';
          event.dataTransfer.setData('text/plain', String(index));
        }
      });
      item.addEventListener('dragend', () => {
        this._dragQueueIndex = -1;
        this.queueList.querySelectorAll('.sh-queue-item').forEach((row) => row.classList.remove('dragging', 'drag-target'));
      });
      item.addEventListener('dragover', (event) => {
        event.preventDefault();
        if (!item.classList.contains('dragging')) {
          this.queueList.querySelectorAll('.sh-queue-item').forEach((row) => row.classList.remove('drag-target'));
          item.classList.add('drag-target');
        }
      });
      item.addEventListener('dragleave', () => {
        item.classList.remove('drag-target');
      });
      item.addEventListener('drop', (event) => {
        event.preventDefault();
        const toIndex = Number(item.getAttribute('data-queue-index'));
        const fromIndex = this._dragQueueIndex;
        item.classList.remove('drag-target');
        if (!Number.isInteger(fromIndex) || fromIndex < 0 || !Number.isInteger(toIndex) || toIndex < 0 || fromIndex === toIndex) return;
        this.moveQueueItem(fromIndex, toIndex);
      });
    });
  }

  moveQueueItem(from, to) {
    if (to < 0 || to >= this.queue.items.length) return;
    const ok = this.queue.move(from, to);
    if (!ok) return;
    this._pushQueueMove(from, to);
    this._render();
    this._scheduleSync();
  }

  _currentIndex() {
    if (!this.queue.items.length) return -1;
    if (this.queue.position >= 0 && this.queue.position < this.queue.items.length) {
      return this.queue.position;
    }
    const currentID = this.queue.current()?.track_id || '';
    if (currentID) {
      const found = this.queue.items.findIndex((item) => item.track_id === currentID);
      if (found >= 0) {
        this.queue.position = found;
        return found;
      }
    }
    this.queue.position = 0;
    return 0;
  }

  _nextIndex() {
    const index = this._currentIndex();
    if (index < 0) return -1;
    if (this.repeatMode === 'one') return index;
    const next = index + 1;
    if (next < this.queue.items.length) return next;
    return this.repeatMode === 'all' ? 0 : -1;
  }

  _previousIndex() {
    const index = this._currentIndex();
    if (index < 0) return -1;
    if (this.repeatMode === 'one') return index;
    const prev = index - 1;
    if (prev >= 0) return prev;
    return this.repeatMode === 'all' ? this.queue.items.length - 1 : -1;
  }

  _stateSnapshot() {
    const snapshot = this.queue.snapshot();
    const current = this.queue.current();
    return sanitizeState({
      current_track_id: current?.track_id || '',
      queue: snapshot.queue,
      queue_position: snapshot.queue_position,
      is_playing: this.isPlaying,
      shuffle_enabled: this.queue.shuffleEnabled,
      repeat_mode: this.repeatMode,
      volume: Number(this.audio.volume || 1),
      current_time_seconds: 0,
      context_type: snapshot.context_type,
      context_id: snapshot.context_id
    });
  }

  _applyState(state, autoplay, options = {}) {
    const hydrateMedia = options.hydrateMedia ?? true;
    this.queue.restore(state);
    if (this.queue.position < 0 && state.current_track_id) {
      const existing = this.queue.items.findIndex((item) => item.track_id === state.current_track_id);
      if (existing >= 0) {
        this.queue.position = existing;
      } else {
        this.queue.items.unshift({
          track_id: String(state.current_track_id),
          title: '',
          artist: '',
          artist_id: '',
          duration: 0,
          cover_ref: ''
        });
        this.queue.position = 0;
      }
    }
    this.repeatMode = state.repeat_mode || 'off';
    this.audio.volume = clamp(Number(state.volume ?? 1), 0, 1);
    this._setVolumeVisual(this.audio.volume);
    this.audio.currentTime = 0;
    this._restoredCurrentTime = 0;
    if (autoplay && state.is_playing && this.queue.position >= 0) {
      this.playAt(this.queue.position);
    } else {
      const current = this.queue.current();
      if (current) {
        this.title.textContent = current.title || t('unknown_title', 'Unknown title');
        this.artist.textContent = current.artist || t('unknown_artist', 'Unknown artist');
        if (hydrateMedia && this._isAuthenticated()) {
          this.cover.src = current.cover_ref ? API.albumCoverThumbUrl(current.cover_ref, 256) : '/static/logo.png';
          this.audio.src = API.streamUrl(current.track_id);
          this.audio.load();
          this._resolveArtistName(current);
          this._hydrateCurrentMetadata(current);
        } else {
          this.cover.src = '/static/logo.png';
          this.audio.removeAttribute('src');
          this.audio.load();
        }
      }
      this._render();
    }
  }

  onAuthChanged(status) {
    if (!status?.authenticated) {
      this.audio.pause();
      this.audio.removeAttribute('src');
      this.audio.load();
      this.cover.src = '/static/logo.png';
      this._render();
      return;
    }
    const local = loadPlaybackState();
    if (local) {
      this._sanitizeStateAgainstLibrary(local)
        .then((sanitized) => {
          this._applyState(sanitized, false, { hydrateMedia: true });
        })
        .catch(() => {
          this._applyState(local, false, { hydrateMedia: true });
        });
      return;
    }
    void this._applyInitialState();
  }

  async _sanitizeStateAgainstLibrary(state) {
    const snapshot = sanitizeState(state || {});
    const rawQueue = Array.isArray(snapshot.queue) ? snapshot.queue : [];
    if (!rawQueue.length && !snapshot.current_track_id) {
      return snapshot;
    }
    try {
      const libraryTracks = await API.getTracks({ limit: 5000, offset: 0, sort: 'name' });
      const validTrackIDs = new Set((Array.isArray(libraryTracks) ? libraryTracks : [])
        .map((track) => String(track?.id || track?.ID || '').trim())
        .filter(Boolean));
      const queue = rawQueue.filter((item) => validTrackIDs.has(String(item?.track_id || '').trim()));
      const currentTrackID = validTrackIDs.has(String(snapshot.current_track_id || '').trim())
        ? String(snapshot.current_track_id || '').trim()
        : (queue[0]?.track_id || '');
      const queuePosition = currentTrackID
        ? Math.max(0, queue.findIndex((item) => item.track_id === currentTrackID))
        : -1;
      const nextState = sanitizeState({
        ...snapshot,
        current_track_id: currentTrackID,
        queue,
        queue_position: queuePosition,
        is_playing: currentTrackID ? Boolean(snapshot.is_playing) : false,
        current_time_seconds: 0
      });
      if (JSON.stringify(nextState) !== JSON.stringify(snapshot)) {
        savePlaybackState(nextState);
      }
      return nextState;
    } catch (_) {
      return snapshot;
    }
  }

  onLanguageChanged() {
    this.repeatButton.setAttribute('title', formatRepeatLabel(this.repeatMode));
    const current = this.queue.current();
    if (!current) {
      this.title.textContent = t('player_no_track', 'No track selected');
    }
    this._renderQueue();
    applyTranslations(this.root);
  }

  _scheduleSync() {
    const state = this._stateSnapshot();
    savePlaybackState(state);
    if (!this._isAuthenticated()) {
      return;
    }
    if (this._syncTimer) {
      clearTimeout(this._syncTimer);
    }
    this._syncTimer = setTimeout(async () => {
      try {
        await API.savePlayerState(state);
      } catch (_) {
        // endpoint sync is best-effort
      }
    }, 300);
  }

  async _pushQueueReplace() {
    if (!this._isAuthenticated()) return;
    const snapshot = this.queue.snapshot();
    try {
      await API.replacePlayerQueue({
        queue: snapshot.queue,
        queue_position: snapshot.queue_position,
        context_type: snapshot.context_type,
        context_id: snapshot.context_id
      });
    } catch (_) {}
  }

  async _pushQueueAppend(items) {
    if (!this._isAuthenticated()) return;
    try {
      await API.appendPlayerQueue({ items });
    } catch (_) {}
  }

  async _pushQueueRemove(index) {
    if (!this._isAuthenticated()) return;
    try {
      await API.removePlayerQueueItem(index);
    } catch (_) {}
  }

  async _pushQueueClear() {
    if (!this._isAuthenticated()) return;
    try {
      await API.clearPlayerQueue();
    } catch (_) {}
  }

  async _pushQueueMove(from, to) {
    if (!this._isAuthenticated()) return;
    try {
      await API.movePlayerQueueItem(from, to);
    } catch (_) {}
  }

  async _pushShuffle(enabled) {
    if (!this._isAuthenticated()) return;
    try {
      await API.setPlayerShuffle(enabled);
    } catch (_) {}
  }

  async _recordPlayed(trackID) {
    if (!this._isAuthenticated()) return;
    if (!trackID) return;
    const snapshot = this.queue.snapshot();
    try {
      await API.playerPlayed({
        track_id: trackID,
        position_seconds: 0,
        context_type: snapshot.context_type || '',
        context_id: snapshot.context_id || ''
      });
    } catch (_) {}
  }

  async _resolveArtistName(current) {
    if (!current || current.artist) return;
    if (!current.artist_id) return;
    if (!this._artistMapPromise) {
      this._artistMapPromise = loadArtistNameMap().catch(() => new Map());
    }
    const map = await this._artistMapPromise;
    const name = map.get(current.artist_id) || '';
    if (!name) return;
    current.artist = name;
    if (this.queue.current()?.track_id === current.track_id) {
      this.artist.textContent = name;
      this._renderQueue();
    }
  }

  async _hydrateCurrentMetadata(current) {
    if (!current?.track_id) return;
    if (current.title && current.artist) return;
    try {
      const track = await API.getTrack(current.track_id);
      if (!track) return;
      if (!current.title) {
        current.title = String(track.title || track.Title || '').trim();
        if (this.queue.current()?.track_id === current.track_id && current.title) {
          this.title.textContent = current.title;
        }
      }
      if (!current.artist_id) {
        current.artist_id = String(track.artistId || track.ArtistID || '').trim();
      }
      if (!current.artist && current.artist_id) {
        await this._resolveArtistName(current);
      }
      this._renderQueue();
    } catch (_) {
      this._handleMissingCurrentTrack(current.track_id);
    }
  }

  _handleMissingCurrentTrack(trackID) {
    if (!trackID) return;
    const index = this.queue.items.findIndex((item) => item.track_id === trackID);
    if (index < 0) return;
    const wasCurrent = index === this.queue.position;
    this.queue.remove(index);
    this.audio.pause();
    this.audio.removeAttribute('src');
    this.audio.load();
    this._restoredCurrentTime = 0;

    if (wasCurrent) {
      const current = this.queue.current();
      if (current) {
        this.title.textContent = current.title || t('unknown_title', 'Unknown title');
        this.artist.textContent = current.artist || t('unknown_artist', 'Unknown artist');
        this.cover.src = current.cover_ref ? API.albumCoverThumbUrl(current.cover_ref, 256) : '/static/logo.png';
        if (this._isAuthenticated()) {
          this._resolveArtistName(current);
          this._hydrateCurrentMetadata(current);
        }
      } else {
        this.title.textContent = t('player_no_track', 'No track selected');
        this.artist.textContent = '-';
        this.cover.src = '/static/logo.png';
      }
    }

    this._render();
    this._scheduleSync();
  }

  _isAuthenticated() {
    return typeof this.auth?.isAuthenticated === 'function' ? this.auth.isAuthenticated() : true;
  }

  _navigateToCurrentTrack() {
    const trackId = String(this.queue.current()?.track_id || '').trim();
    if (!trackId) return;
    window.dispatchEvent(new CustomEvent('soundhub:navigate', { detail: { path: `/tracks/${encodeURIComponent(trackId)}` } }));
  }

  _startVisualizer() {
    if (!this.waveformCanvas) return;
    if (this._visualizerFrame) return;
    if (typeof window.AudioContext === 'undefined' && typeof window.webkitAudioContext === 'undefined') {
      return;
    }
    try {
      if (!this._audioContext) {
        const Ctx = window.AudioContext || window.webkitAudioContext;
        this._audioContext = new Ctx();
        this._analyser = this._audioContext.createAnalyser();
        this._analyser.fftSize = 256;
        this._analyser.smoothingTimeConstant = 0.78;
        this._sourceNode = this._audioContext.createMediaElementSource(this.audio);
        this._sourceNode.connect(this._analyser);
        this._analyser.connect(this._audioContext.destination);
      }
      if (this._audioContext.state === 'suspended') {
        this._audioContext.resume().catch(() => {});
      }
      const draw = () => {
        this._drawLiveSpectrum();
        if (!this.audio.paused && !this.audio.ended) {
          this._visualizerFrame = window.requestAnimationFrame(draw);
          return;
        }
        this._visualizerFrame = 0;
      };
      this._visualizerFrame = window.requestAnimationFrame(draw);
    } catch (_) {
      this._drawLiveSpectrum(true);
    }
  }

  _stopVisualizer() {
    if (this._visualizerFrame) {
      window.cancelAnimationFrame(this._visualizerFrame);
      this._visualizerFrame = 0;
    }
    this._drawLiveSpectrum(true);
  }

  _startProgressLoop() {
    if (this._progressFrame) return;
    const tick = () => {
      this._renderProgress();
      if (!this.audio.paused && !this.audio.ended) {
        this._progressFrame = window.requestAnimationFrame(tick);
        return;
      }
      this._progressFrame = 0;
    };
    this._progressFrame = window.requestAnimationFrame(tick);
  }

  _stopProgressLoop() {
    if (this._progressFrame) {
      window.cancelAnimationFrame(this._progressFrame);
      this._progressFrame = 0;
    }
    this._renderProgress();
  }

  _drawLiveSpectrum(empty = false) {
    if (!this.waveformCanvas) return;
    const canvas = this.waveformCanvas;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;
    const width = canvas.width;
    const height = canvas.height;
    ctx.clearRect(0, 0, width, height);
    if (empty || !this._analyser) return;

    const bins = this._analyser.frequencyBinCount;
    const data = new Uint8Array(bins);
    this._analyser.getByteFrequencyData(data);

    const bars = Math.min(72, bins);
    const step = Math.max(1, Math.floor(bins / bars));
    const barWidth = Math.max(2, Math.floor(width / bars) - 1);
    for (let i = 0; i < bars; i += 1) {
      const value = data[i * step] || 0;
      const amp = value / 255;
      const barHeight = Math.max(2, Math.floor(amp * (height - 8)));
      const x = i * (barWidth + 1);
      const y = height - barHeight - 3;
      const color = `rgba(${38 + Math.floor(amp * 24)}, ${62 + Math.floor(amp * 52)}, ${118 + Math.floor(amp * 76)}, 0.86)`;
      ctx.fillStyle = color;
      ctx.fillRect(x, y, barWidth, barHeight);
    }
  }

  _setProgressVisual(fraction) {
    const safe = clamp(Number(fraction || 0), 0, 1);
    const pct = `${(safe * 100).toFixed(3)}%`;
    this.progress.style.setProperty('--progress-pct', pct);
    const width = Math.max(1, this.progress.clientWidth || 1);
    const dotX = width * safe;
    const chipX = clamp(dotX, 34, Math.max(34, width - 34));
    this.progress.style.setProperty('--progress-chip-x', `${chipX}px`);
    this.progress.style.setProperty('--progress-dot-x', `${dotX}px`);
    this.progressWrap?.style.setProperty('--progress-pct', pct);
    this.progressWrap?.style.setProperty('--progress-chip-x', `${chipX}px`);
    this.progressWrap?.style.setProperty('--progress-dot-x', `${dotX}px`);
    this.progressOverlay?.style.setProperty('--progress-pct', pct);
    this.progressOverlay?.style.setProperty('--progress-chip-x', `${chipX}px`);
    this.progressOverlay?.style.setProperty('--progress-dot-x', `${dotX}px`);
    const durationChipWidth = 56;
    const currentChipWidth = 68;
    const gap = Math.max(0, width - chipX);
    const overlapStart = durationChipWidth + currentChipWidth + 18;
    const overlapEnd = durationChipWidth + 12;
    const fade = gap <= overlapStart
      ? clamp((overlapStart - gap) / Math.max(1, overlapStart - overlapEnd), 0, 1)
      : 0;
    this.progressOverlay?.style.setProperty('--duration-fade', `${fade.toFixed(3)}`);
    this.progressDot?.style.setProperty('--progress-dot-x', `${dotX}px`);
    this.surface?.style.setProperty('--progress-pct', pct);
    this.main?.style.setProperty('--progress-pct', pct);
  }

  _setVolumeVisual(value) {
    const safe = clamp(Number(value || 0), 0, 1);
    this.volume.style.setProperty('--volume-pct', `${Math.round(safe * 100)}%`);
  }

  _syncProgressAnchor(seconds = 0) {
    this._progressAnchorTime = Math.max(0, Number(seconds || 0));
    this._progressAnchorPerf = performance.now();
  }

  _currentPlaybackSeconds() {
    const base = Math.max(0, Number(this._progressAnchorTime || this._restoredCurrentTime || this.audio.currentTime || 0));
    if (this.audio.paused || this.audio.ended) {
      return base;
    }
    const elapsed = Math.max(0, performance.now() - Number(this._progressAnchorPerf || 0));
    return base + (elapsed / 1000) * Math.max(0, Number(this.audio.playbackRate || 1));
  }

  _renderProgress() {
    const durationSeconds = this._currentDurationSeconds();
    const currentSeconds = Math.max(0, this._currentPlaybackSeconds());
    if (!durationSeconds || !Number.isFinite(durationSeconds)) {
      this.progress.value = '0';
      this.currentTime.textContent = formatTime(Math.floor(currentSeconds));
      this.duration.textContent = '0:00';
      this._setProgressVisual(0);
      return;
    }
    const fraction = clamp(currentSeconds / durationSeconds, 0, 1);
    this.progress.value = String(Math.floor(fraction * 10000));
    this._setProgressVisual(fraction);
    this.currentTime.textContent = formatTime(Math.floor(currentSeconds));
    this.duration.textContent = formatTime(Math.floor(durationSeconds));
  }

  _currentDurationSeconds() {
    if (Number.isFinite(this.audio.duration) && this.audio.duration > 0) {
      return Number(this.audio.duration);
    }
    return Math.max(0, Number(this.queue.current()?.duration || 0));
  }
}

function compactTracks(tracks) {
  const compact = [];
  for (const track of Array.isArray(tracks) ? tracks : []) {
    const trackID = String(track.track_id || track.id || track.ID || '').trim();
    if (!trackID) continue;
    compact.push({
      track_id: trackID,
      title: String(track.title || track.Title || '').trim(),
      artist: String(track.artist || track.Artist || track.artist_name || track.artistName || '').trim(),
      artist_id: String(track.artist_id || track.artistId || track.ArtistID || '').trim(),
      duration: toInt(track.duration || track.Duration || 0),
      cover_ref: String(track.cover_ref || track.coverRef || track.album_id || track.albumId || track.AlbumID || '').trim()
    });
  }
  return compact;
}

function nextRepeatMode(mode) {
  switch (mode) {
    case 'off':
      return 'all';
    case 'all':
      return 'one';
    default:
      return 'off';
  }
}

function formatRepeatLabel(mode) {
  return `${t('player_repeat', 'Repeat')}: ${t(`repeat_${mode}`, mode)}`;
}

function formatTime(seconds) {
  const safe = Number.isFinite(seconds) ? Math.max(0, seconds) : 0;
  const minutes = Math.floor(safe / 60);
  const sec = safe % 60;
  return `${minutes}:${String(sec).padStart(2, '0')}`;
}

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value));
}

function toInt(value) {
  const parsed = Number.parseInt(String(value), 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 0;
}

function escapeHtml(value) {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

import { createRouter } from './router.js';
import { Player } from './player.js';
import { initI18n, t } from './i18n.js';
import { renderHome } from './home.js';
import { renderArtists } from './artists.js';
import { renderAlbums } from './albums.js';
import { renderTracks } from './tracks.js';
import { renderGenres } from './genres.js';
import { renderPlaylists } from './playlists.js';
import { renderSearch } from './search.js';
import { renderLibrary } from './library.js';
import { renderSettings } from './settings.js';
import { renderUsers } from './users.js';
import { renderProfile } from './profile.js';
import { renderAlbumDetail, renderArtistDetail, renderPlaylistDetail, renderTrackDetail } from './detail-pages.js';
import { createAuthManager } from './auth.js';
import { createShareModal } from './share-modal.js';
import { API } from './api.js';

const SIDEBAR_COLLAPSED_KEY = 'sonarium.sidebar.collapsed';
const MOBILE_SIDEBAR_MEDIA = '(max-width: 900px)';

async function init() {
  await initI18n(document.getElementById('lang-select'));
  const langSelect = document.getElementById('lang-select');
  if (langSelect && langSelect.options.length >= 2) {
    langSelect.options[0].textContent = 'EN';
    langSelect.options[1].textContent = 'RU';
  }

  const auth = createAuthManager();
  const authStatus = await auth.init();
  const sidebarVersion = document.getElementById('sidebar-version');
  const sidebarToggle = document.getElementById('sidebar-toggle');
  const sidebarBrand = document.querySelector('.sh-sidebar .sh-brand');
  const mobileNavToggle = document.getElementById('mobile-nav-toggle');
  const mobileSidebarBackdrop = document.getElementById('mobile-sidebar-backdrop');
  initSidebarCollapse(sidebarToggle, sidebarBrand, mobileNavToggle, mobileSidebarBackdrop);
  const playerRoot = document.getElementById('player-root');
  const playerHtml = await fetch('/static/player.html').then((res) => res.text());
  playerRoot.innerHTML = playerHtml;

  const player = new Player(playerRoot, { auth });
  const shareModal = createShareModal();

  const context = { player, auth, shareModal };
  const controllers = {
    home: renderHome,
    artists: renderArtists,
    albums: renderAlbums,
    tracks: renderTracks,
    genres: renderGenres,
    playlists: renderPlaylists,
    search: renderSearch,
    library: renderLibrary,
    settings: renderSettings,
    users: renderUsers,
    profile: renderProfile,
    albumDetail: renderAlbumDetail,
    artistDetail: renderArtistDetail,
    playlistDetail: renderPlaylistDetail,
    trackDetail: renderTrackDetail
  };

  const router = createRouter({
    contentEl: document.getElementById('page-content'),
    titleEl: document.getElementById('page-title'),
    navEl: document.getElementById('sidebar-nav'),
    controllers,
    context
  });
  context.router = router;

  if (authStatus?.authenticated) {
    await syncSidebarVersion(sidebarVersion);
    await router.start();
  }

  window.addEventListener('soundhub:lang-changed', async () => {
    if (!auth.isAuthenticated()) return;
    await router.refresh();
    if (typeof player.onLanguageChanged === 'function') {
      player.onLanguageChanged();
    }
  });

  window.addEventListener('soundhub:auth-changed', async () => {
    if (typeof player.onAuthChanged === 'function') {
      player.onAuthChanged(auth.getStatus?.());
    }
    if (auth.isAuthenticated()) {
      await syncSidebarVersion(sidebarVersion);
      await router.start();
      return;
    }
    if (sidebarVersion) {
      sidebarVersion.hidden = true;
    }
    document.getElementById('page-content').innerHTML = '';
  });

  window.addEventListener('soundhub:open-profile', async () => {
    if (!auth.isAuthenticated()) return;
    await router.go('/profile');
  });

  window.addEventListener('soundhub:navigate', async (event) => {
    const path = event?.detail?.path;
    if (!path || !auth.isAuthenticated()) return;
    await router.go(path);
  });
}

function initSidebarCollapse(toggleButton, brand, mobileToggleButton, mobileBackdrop) {
  if (!toggleButton) return;
  const mobileMedia = window.matchMedia(MOBILE_SIDEBAR_MEDIA);
  const setMobileNav = (open) => {
    document.body.classList.toggle('mobile-nav-open', open);
    if (mobileBackdrop) {
      mobileBackdrop.hidden = !open;
    }
    if (mobileToggleButton) {
      mobileToggleButton.setAttribute('aria-expanded', open ? 'true' : 'false');
    }
  };
  const closeMobileNav = () => setMobileNav(false);
  const toggleMobileNav = () => setMobileNav(!document.body.classList.contains('mobile-nav-open'));
  const syncMobileMode = () => {
    document.body.classList.toggle('mobile-layout', mobileMedia.matches);
    if (!mobileMedia.matches) {
      closeMobileNav();
    }
  };
  const apply = (collapsed) => {
    document.body.classList.toggle('sidebar-collapsed', collapsed);
    toggleButton.setAttribute('aria-label', collapsed ? t('sidebar_expand', 'Expand sidebar') : t('sidebar_collapse', 'Collapse sidebar'));
  };
  const toggle = () => {
    if (mobileMedia.matches) {
      toggleMobileNav();
      return;
    }
    const next = !document.body.classList.contains('sidebar-collapsed');
    apply(next);
    window.localStorage.setItem(SIDEBAR_COLLAPSED_KEY, next ? '1' : '0');
  };
  const stored = window.localStorage.getItem(SIDEBAR_COLLAPSED_KEY);
  apply(stored === '1');
  syncMobileMode();
  toggleButton.addEventListener('click', toggle);
  mobileToggleButton?.addEventListener('click', toggleMobileNav);
  mobileBackdrop?.addEventListener('click', closeMobileNav);
  brand?.addEventListener('click', (event) => {
    if (event.target.closest('#sidebar-toggle')) return;
    toggle();
  });
  document.getElementById('sidebar-nav')?.addEventListener('click', (event) => {
    if (!mobileMedia.matches) return;
    if (!event.target.closest('a[href]')) return;
    closeMobileNav();
  });
  window.addEventListener('resize', syncMobileMode);
  window.addEventListener('orientationchange', syncMobileMode);
  mobileMedia.addEventListener?.('change', syncMobileMode);
  window.addEventListener('soundhub:lang-changed', () => {
    apply(document.body.classList.contains('sidebar-collapsed'));
  });
  window.addEventListener('soundhub:navigate', closeMobileNav);
}

async function syncSidebarVersion(sidebarVersion) {
  if (!sidebarVersion) return;
  try {
    const settings = await API.getSettings();
    const version = String(settings?.version || '').trim();
    if (!version) {
      sidebarVersion.hidden = true;
      return;
    }
    sidebarVersion.textContent = `v${version.replace(/^v/i, '')}`;
    sidebarVersion.hidden = false;
  } catch {
    sidebarVersion.hidden = true;
  }
}

document.addEventListener('DOMContentLoaded', () => {
  init().catch((error) => {
    console.error('failed to initialize app', error);
  });
});

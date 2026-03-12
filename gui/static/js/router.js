import { applyTranslations, t } from './i18n.js';

const routes = [
  { pattern: /^\/$/, page: 'home', titleKey: 'home' },
  { pattern: /^\/artists$/, page: 'artists', titleKey: 'artists' },
  { pattern: /^\/artists\/([^/]+)$/, page: null, controllerKey: 'artistDetail', titleKey: 'artist', detail: true, params: ['artistId'] },
  { pattern: /^\/albums$/, page: 'albums', titleKey: 'albums' },
  { pattern: /^\/albums\/([^/]+)$/, page: null, controllerKey: 'albumDetail', titleKey: 'albums', detail: true, params: ['albumId'] },
  { pattern: /^\/tracks$/, page: 'tracks', titleKey: 'tracks' },
  { pattern: /^\/tracks\/([^/]+)$/, page: null, controllerKey: 'trackDetail', titleKey: 'tracks', detail: true, params: ['trackId'] },
  { pattern: /^\/genres$/, page: 'genres', titleKey: 'genres' },
  { pattern: /^\/playlists$/, page: 'playlists', titleKey: 'playlists' },
  { pattern: /^\/playlists\/([^/]+)$/, page: null, controllerKey: 'playlistDetail', titleKey: 'playlists', detail: true, params: ['playlistId'] },
  { pattern: /^\/search$/, page: 'search', titleKey: 'search' },
  { pattern: /^\/library$/, page: 'library', titleKey: 'library' },
  { pattern: /^\/settings$/, page: 'settings', titleKey: 'settings' },
  { pattern: /^\/users$/, page: 'users', titleKey: 'users' },
  { pattern: /^\/profile$/, page: 'profile', titleKey: 'profile' },
  { pattern: /^\/profile\/([^/]+)$/, page: 'profile', titleKey: 'profile', params: ['userId'] }
];

export function createRouter({ contentEl, titleEl, navEl, controllers, context }) {
  async function renderPath(path, replace = false) {
    const route = resolveRoute(path);
    if (replace) {
      history.replaceState(normalizeState(history.state), '', path);
    }
    syncHistoryState();

    const pageTitle = t(route.titleKey, route.page || route.controllerKey || 'page');
    titleEl.textContent = pageTitle;
    document.title = pageTitle ? `${pageTitle} · Sonarium` : 'Sonarium';
    highlight(path, route);
    document.querySelector('.sh-content')?.classList.toggle('detail-mode', Boolean(route.detail));

    if (route.page) {
      const pageHtml = await fetch(`/static/${route.page}.html`).then((res) => res.text());
      contentEl.innerHTML = pageHtml;
      applyTranslations(contentEl);
    } else {
      contentEl.innerHTML = '';
    }

    const controller = controllers[route.controllerKey || route.page];
    if (controller) {
      await controller(context, contentEl, route.routeParams || {});
    }
    applyTranslations(contentEl);
  }

  function highlight(path, route) {
    navEl.querySelectorAll('[data-route]').forEach((a) => {
      if (a.getAttribute('data-route') === navPath(route, path)) {
        a.classList.add('active');
      } else {
        a.classList.remove('active');
      }
    });
  }

  navEl.addEventListener('click', (event) => {
    const link = event.target.closest('[data-route]');
    if (!link) return;
    event.preventDefault();
    const path = link.getAttribute('data-route');
    history.pushState({}, '', path);
    renderPath(path);
  });

  window.addEventListener('popstate', () => {
    renderPath(window.location.pathname || '/', true);
  });

  return {
    start() {
      if (!history.state || typeof history.state.__sh_idx !== 'number') {
        history.replaceState({ __sh_idx: 0 }, '', window.location.pathname || '/');
      }
      syncHistoryState();
      return renderPath(window.location.pathname || '/', true);
    },
    go(path) {
      const nextIdx = getHistoryIndex() + 1;
      history.pushState({ __sh_idx: nextIdx }, '', path);
      setHistorySession(nextIdx);
      return renderPath(path);
    },
    refresh() {
      return renderPath(window.location.pathname || '/', true);
    }
  };
}

function normalizeState(state) {
  if (state && typeof state.__sh_idx === 'number') {
    return state;
  }
  return { __sh_idx: getHistoryIndex() };
}

function getHistoryIndex() {
  return Number(history.state?.__sh_idx ?? sessionStorage.getItem('sh-history-current') ?? 0) || 0;
}

function setHistorySession(idx) {
  const safeIdx = Math.max(0, Number(idx) || 0);
  sessionStorage.setItem('sh-history-current', String(safeIdx));
  const maxIdx = Number(sessionStorage.getItem('sh-history-max') || '0') || 0;
  sessionStorage.setItem('sh-history-max', String(Math.max(maxIdx, safeIdx)));
}

function syncHistoryState() {
  setHistorySession(getHistoryIndex());
}

function resolveRoute(path) {
  for (const route of routes) {
    const match = path.match(route.pattern);
    if (!match) continue;
    const routeParams = {};
    for (let i = 0; i < (route.params || []).length; i += 1) {
      routeParams[route.params[i]] = decodeURIComponent(match[i + 1] || '');
    }
    return { ...route, routeParams };
  }
  return routes[0];
}

function navPath(route, fallbackPath) {
  if (!route?.detail) return fallbackPath;
  if (route.controllerKey === 'albumDetail') return '/albums';
  if (route.controllerKey === 'artistDetail') return '/artists';
  if (route.controllerKey === 'playlistDetail') return '/playlists';
  if (route.controllerKey === 'trackDetail') return '/tracks';
  return fallbackPath;
}

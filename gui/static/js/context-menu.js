let menuEl = null;
let hideBound = false;

export function showContextMenu(event, items) {
  hideContextMenu();
  if (!event || !Array.isArray(items) || !items.length) return;

  menuEl = document.createElement('div');
  menuEl.className = 'sh-context-menu';
  menuEl.setAttribute('role', 'menu');
  menuEl.innerHTML = items.map((item, idx) => `
    <button type="button" role="menuitem" data-idx="${idx}" ${item?.danger ? 'class="danger"' : ''}>
      ${escapeHtml(item?.label || '')}
    </button>
  `).join('');
  document.body.appendChild(menuEl);

  const rect = menuEl.getBoundingClientRect();
  const padding = 8;
  const x = Math.max(padding, Math.min(event.clientX, window.innerWidth - rect.width - padding));
  const y = Math.max(padding, Math.min(event.clientY, window.innerHeight - rect.height - padding));
  menuEl.style.left = `${x}px`;
  menuEl.style.top = `${y}px`;

  menuEl.querySelectorAll('button').forEach((button) => {
    button.addEventListener('click', async (clickEvent) => {
      clickEvent.stopPropagation();
      const idx = Number(button.getAttribute('data-idx'));
      hideContextMenu();
      const action = items[idx]?.action;
      if (typeof action === 'function') {
        await action();
      }
    });
  });

  bindHideListeners();
}

export function hideContextMenu() {
  if (menuEl) {
    menuEl.remove();
    menuEl = null;
  }
  unbindHideListeners();
}

function bindHideListeners() {
  if (hideBound) return;
  hideBound = true;
  document.addEventListener('click', onGlobalClick, true);
  document.addEventListener('contextmenu', onGlobalContextMenu, true);
  document.addEventListener('scroll', onGlobalScroll, true);
  window.addEventListener('resize', onGlobalResize, true);
  window.addEventListener('keydown', onGlobalKeyDown, true);
}

function unbindHideListeners() {
  if (!hideBound) return;
  hideBound = false;
  document.removeEventListener('click', onGlobalClick, true);
  document.removeEventListener('contextmenu', onGlobalContextMenu, true);
  document.removeEventListener('scroll', onGlobalScroll, true);
  window.removeEventListener('resize', onGlobalResize, true);
  window.removeEventListener('keydown', onGlobalKeyDown, true);
}

function onGlobalClick(event) {
  if (!menuEl) return;
  if (event.target && event.target.closest('.sh-context-menu')) return;
  hideContextMenu();
}

function onGlobalContextMenu(event) {
  if (!menuEl) return;
  if (event.target && event.target.closest('.sh-context-menu')) return;
  hideContextMenu();
}

function onGlobalScroll() {
  hideContextMenu();
}

function onGlobalResize() {
  hideContextMenu();
}

function onGlobalKeyDown(event) {
  if (event.key === 'Escape') {
    hideContextMenu();
  }
}

function escapeHtml(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

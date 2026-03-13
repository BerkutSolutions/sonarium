import { API } from './api.js';
import { t } from './i18n.js';

export function createAuthManager() {
  const overlay = document.getElementById('auth-overlay');
  const loginView = document.getElementById('auth-login-view');
  const setupView = document.getElementById('auth-setup-view');
  const loginForm = document.getElementById('auth-login-form');
  const registerForm = document.getElementById('auth-register-form');
  const loginError = document.getElementById('auth-login-error');
  const registerError = document.getElementById('auth-register-error');
  const registerToggle = document.getElementById('auth-open-register');
  const backToLogin = document.getElementById('auth-back-login');
  const profileBtn = document.getElementById('profile-btn');
  const logoutBtn = document.getElementById('logout-btn');
  const usersLink = document.getElementById('nav-users');

  let status = null;
  let idleTimer = null;
  let countdownTimer = null;
  let currentUser = null;

  function bind() {
    loginForm?.addEventListener('submit', async (event) => {
      event.preventDefault();
      setMessage(loginError, '');
      try {
        const form = new FormData(loginForm);
        await API.login({
          username: String(form.get('username') || ''),
          password: String(form.get('password') || '')
        });
        status = await API.getAuthStatus();
        applyStatus(status);
        window.dispatchEvent(new CustomEvent('soundhub:auth-changed', { detail: status }));
      } catch (error) {
        setMessage(loginError, error.message || t('auth_login_failed', 'Login failed'));
      }
    });

    registerForm?.addEventListener('submit', async (event) => {
      event.preventDefault();
      setMessage(registerError, '');
      try {
        const form = new FormData(registerForm);
        const password = String(form.get('password') || '');
        const passwordRepeat = String(form.get('password_repeat') || '');
        if (password !== passwordRepeat) {
          throw new Error(t('passwords_no_match', 'Passwords do not match'));
        }
        await API.register({
          username: String(form.get('username') || ''),
          display_name: String(form.get('display_name') || ''),
          password
        });
        status = await API.getAuthStatus();
        applyStatus(status);
        window.dispatchEvent(new CustomEvent('soundhub:auth-changed', { detail: status }));
      } catch (error) {
        setMessage(registerError, error.message || t('auth_register_failed', 'Registration failed'));
      }
    });

    registerToggle?.addEventListener('click', () => {
      loginView.hidden = true;
      setupView.hidden = false;
    });
    backToLogin?.addEventListener('click', () => {
      loginView.hidden = false;
      setupView.hidden = true;
    });
    logoutBtn?.addEventListener('click', () => {
      logout().catch(() => {});
    });
    profileBtn?.addEventListener('click', () => {
      if (!status?.authenticated) return;
      window.dispatchEvent(new CustomEvent('soundhub:open-profile'));
    });
    ['mousemove', 'keydown', 'mousedown', 'touchstart'].forEach((eventName) => {
      window.addEventListener(eventName, () => refreshIdleTimer());
    });
    profileBtn?.addEventListener('mouseenter', showProfileCard);
    profileBtn?.addEventListener('mouseleave', hideProfileCard);
  }

  async function init() {
    bind();
    try {
      status = await API.getAuthStatus();
    } catch {
      status = { authenticated: false, registration_open: false, setup_required: false };
    }
    applyStatus(status);
    return status;
  }

  function applyStatus(nextStatus) {
    status = nextStatus || {};
    currentUser = status.user || null;
    document.body.classList.remove('auth-pending');
    document.body.classList.toggle('auth-locked', !status.authenticated);
    if (overlay) overlay.hidden = Boolean(status.authenticated);
    if (loginView) loginView.hidden = Boolean(status.setup_required);
    if (setupView) setupView.hidden = !status.setup_required;
    if (registerToggle) registerToggle.hidden = !status.registration_open;
    if (backToLogin) backToLogin.hidden = status.setup_required;
    if (usersLink) {
      usersLink.hidden = currentUser?.role !== 'admin';
    }
    refreshIdleTimer();
    renderSessionRemaining();
  }

  function refreshIdleTimer() {
    if (idleTimer) window.clearTimeout(idleTimer);
    if (!status?.authenticated) return;
    const timeoutMs = Math.max(1000, Number(status.session_idle_timeout_seconds || 0) * 1000);
    idleTimer = window.setTimeout(() => {
      logout(true).catch(() => {});
    }, timeoutMs);
  }

  function renderSessionRemaining() {
    if (countdownTimer) window.clearInterval(countdownTimer);
    if (!status?.authenticated || !status.session_expires_at) return;
    const update = () => {
      const expiresAt = new Date(status.session_expires_at).getTime();
      if (expiresAt <= Date.now()) return;
    };
    update();
    countdownTimer = window.setInterval(update, 30000);
  }

  async function logout(skipRequest = false) {
    if (!skipRequest) {
      try {
        await API.logout();
      } catch {}
    }
    status = await API.getAuthStatus();
    applyStatus(status);
    window.dispatchEvent(new CustomEvent('soundhub:auth-changed', { detail: status }));
  }

  function isAuthenticated() {
    return Boolean(status?.authenticated);
  }

  function showProfileCard() {
    if (!profileBtn || !currentUser) return;
    hideProfileCard();
    const rect = profileBtn.getBoundingClientRect();
    const card = document.createElement('div');
    card.className = 'profile-hover-card';
    card.id = 'profile-hover-card';
    card.innerHTML = `
      <div><strong>${escapeHtml(currentUser.display_name)}</strong></div>
      <div>@${escapeHtml(currentUser.username)}</div>
      <div>${escapeHtml(currentUser.role)}</div>
    `;
    card.style.top = `${rect.top - 8}px`;
    card.style.left = `${rect.right + 10}px`;
    document.body.appendChild(card);
  }

  function hideProfileCard() {
    document.getElementById('profile-hover-card')?.remove();
  }

  return {
    init,
    isAuthenticated,
    logout,
    async refresh() {
      status = await API.getAuthStatus();
      applyStatus(status);
      return status;
    },
    getStatus() {
      return status;
    }
  };
}

function setMessage(node, message) {
  if (!node) return;
  node.hidden = !message;
  node.textContent = message || '';
}

function escapeHtml(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;');
}

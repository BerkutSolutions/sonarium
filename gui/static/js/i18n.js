const STORAGE_KEY = 'soundhub_lang';

let dictionary = {};
let currentLang = 'en';

export async function initI18n(selectEl) {
  const saved = localStorage.getItem(STORAGE_KEY);
  currentLang = saved || 'en';
  await setLanguage(currentLang);
  if (selectEl) {
    selectEl.value = currentLang;
    selectEl.addEventListener('change', async () => {
      await setLanguage(selectEl.value);
    });
  }
}

export async function setLanguage(lang) {
  currentLang = lang === 'ru' ? 'ru' : 'en';
  localStorage.setItem(STORAGE_KEY, currentLang);
  try {
    const [base, extra] = await Promise.all([
      fetch(`/static/i18n/${currentLang}.json`).then((response) => response.json()),
      fetch(`/static/i18n/${currentLang}.extra.json`).then((response) => response.ok ? response.json() : {})
    ]);
    dictionary = { ...(base || {}), ...(extra || {}) };
  } catch (_) {
    dictionary = {};
  }
  applyTranslations();
  window.dispatchEvent(new CustomEvent('soundhub:lang-changed', { detail: { lang: currentLang } }));
}

export function t(key, fallback = '') {
  return dictionary[key] || fallback || key;
}

export function applyTranslations(root = document) {
  root.querySelectorAll('[data-i18n]').forEach((el) => {
    const key = el.getAttribute('data-i18n');
    const value = t(key, el.textContent || key);
    el.textContent = value;
  });
  root.querySelectorAll('[data-i18n-placeholder]').forEach((el) => {
    const key = el.getAttribute('data-i18n-placeholder');
    const value = t(key, el.getAttribute('placeholder') || key);
    el.setAttribute('placeholder', value);
  });
  root.querySelectorAll('[data-i18n-aria-label]').forEach((el) => {
    const key = el.getAttribute('data-i18n-aria-label');
    const value = t(key, el.getAttribute('aria-label') || key);
    el.setAttribute('aria-label', value);
  });
  root.querySelectorAll('[data-i18n-title]').forEach((el) => {
    const key = el.getAttribute('data-i18n-title');
    const value = t(key, el.getAttribute('title') || key);
    el.setAttribute('title', value);
  });
}

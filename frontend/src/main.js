import './style.css';
import './app.css';

import torLogo from './assets/images/tor.webp';

import * as App from '../wailsjs/go/main/App';
import { EventsOn } from '../wailsjs/runtime/runtime';

const $ = (id) => document.getElementById(id);

const hasBackend = () => !!(window.go && window.go.main && window.go.main.App);
const hasRuntime = () => !!window.runtime;

async function call(fn, ...args) {
  if (!hasBackend()) return undefined;
  try { return await fn(...args); } catch (e) { console.warn('backend call failed:', e); return undefined; }
}

let state = {
  vpnStatus: 'DISCONNECTED',
  servers: [],
  selectedServer: null,
  selected: { ip: '', name: '' },
  selectedCountry: 'All',
  sortBy: 'score',
  search: '',
  settings: null,
  torPort: 9050,
  torActive: false,
  torStarting: false,
  activeProfile: '',
  profiles: [],
  wgStatus: 'DISCONNECTED',
  wgProfile: '',
  wgProfiles: [],
  wgStats: null,
};

/* ── Toast ─────────────────────────────────────────────────── */
function toast(msg, kind) {
  const t = $('toast');
  if (!t || !msg) return;
  t.textContent = msg;
  t.classList.remove('hidden');
  t.style.borderLeftColor = kind === 'negative' ? '#e33e3f' : kind === 'positive' ? '#2fbf71' : '#4da3f7';
  clearTimeout(t._timer);
  t._timer = setTimeout(() => t.classList.add('hidden'), 3500);
}

/* ── Логи ──────────────────────────────────────────────────── */
function appendLog(boxId, msg) {
  const box = $(boxId);
  if (!box) return;
  const line = document.createElement('div');
  line.textContent = msg;
  box.appendChild(line);
  box.scrollTop = box.scrollHeight;
}

/* ── Навігація ─────────────────────────────────────────────── */
function showPage(name) {
  document.querySelectorAll('.nav-item, .nav-mini').forEach(b => b.classList.toggle('active', b.dataset.page === name));
  document.querySelectorAll('.page').forEach(p => p.classList.toggle('active', p.id === 'page-' + name));
  if (name === 'logs') $('vpn-logs').scrollTop = $('vpn-logs').scrollHeight;
}

document.querySelectorAll('.nav-item, .nav-mini').forEach(btn => {
  btn.addEventListener('click', () => showPage(btn.dataset.page));
});

$('btn-collapse').addEventListener('click', () => {
  document.querySelector('.shell').classList.toggle('collapsed');
});

/* ── Головна: хмаринка та кнопка ───────────────────────────── */
const HERO = {
  DISCONNECTED: { cls: 'state-disconnected', title: "Від'єднано", sub: 'Ваш Інтернет не є приватним.', btn: 'Підключити' },
  CONNECTING:   { cls: 'state-connecting',   title: 'Підключення...', sub: 'Встановлюємо захищений тунель.', btn: 'Скасувати' },
  CONNECTED:    { cls: 'state-connected',    title: 'Підключено', sub: 'Ваш Інтернет є приватним.', btn: 'Відключити' },
};

function updateHero() {
  const h = HERO[state.vpnStatus] || HERO.DISCONNECTED;
  const icon = $('cloud-icon');
  icon.classList.remove('state-disconnected', 'state-connecting', 'state-connected');
  icon.classList.add(h.cls);
  $('status-title').textContent = h.title;
  $('status-subtitle').textContent = h.sub;
  $('btn-connect').textContent = h.btn;
}

async function refreshVPNState() {
  const s = await call(App.GetVPNState);
  if (s) state.vpnStatus = s;
  updateHero();
}

$('btn-connect').addEventListener('click', async () => {
  if (!hasBackend()) { toast('Демо-режим у браузері', 'info'); return; }
  if (state.vpnStatus === 'DISCONNECTED') await App.ToggleVPN();
  else await App.DisconnectVPN();
});

/* ── Режим ─────────────────────────────────────────────────── */
$('warp-mode').addEventListener('change', async (e) => {
  const v = e.target.value;
  state.settings = await call(App.GetSettings) || state.settings;
  if (state.settings) state.settings.protocol = v;
  await call(App.UpdateSetting, 'protocol', v);
  $('pr-proto').textContent = v;
  toast('Режим: ' + e.target.selectedOptions[0].textContent, 'info');
});

/* ── IP ────────────────────────────────────────────────────── */
async function refreshIP() {
  const info = await call(App.GetIPInfo);
  if (!info) return;
  $('pr-ip').textContent = info.ip || '—';
  $('pr-loc').textContent = (info.city ? info.city + ', ' : '') + (info.country || '—');
  $('pr-isp').textContent = info.isp || '—';
}

$('btn-refresh-ip').addEventListener('click', refreshIP);

/* ── Обраний сервер ────────────────────────────────────────── */
function renderSelection() {
  const bar = $('selection-bar');
  const homeLine = $('home-server-line');
  if (state.activeProfile) {
    // активний власний .ovpn профіль — показуємо так само, як обраний сервер
    bar.classList.remove('hidden');
    $('selection-name').textContent = state.activeProfile;
    $('selection-ip').textContent = 'власний .ovpn профіль';
    $('sel-switch').checked = true;
    homeLine.innerHTML =
      'Сервер: <b>' + escapeHtml(state.activeProfile) + '</b> (власний .ovpn)' +
      '<label class="bar-switch" title="Вимкнути — прибрати профіль і повернути авто-підбір">' +
      '<input type="checkbox" id="home-sel-switch" class="switch" checked/></label>';
    $('home-sel-switch').addEventListener('change', onSelSwitch);
  } else if (state.selected.ip) {
    bar.classList.remove('hidden');
    $('selection-name').textContent = state.selected.name || '—';
    $('selection-ip').textContent = state.selected.ip;
    $('sel-switch').checked = true;
    homeLine.innerHTML =
      'Сервер: <b>' + escapeHtml(state.selected.name || state.selected.ip) + '</b>' +
      '<label class="bar-switch" title="Вимкнути — прибрати обраний сервер і повернути авто-підбір">' +
      '<input type="checkbox" id="home-sel-switch" class="switch" checked/></label>';
    $('home-sel-switch').addEventListener('change', onSelSwitch);
  } else {
    bar.classList.add('hidden');
    $('sel-switch').checked = false;
    homeLine.innerHTML = 'Сервер: автоматичний підбір (VPN Gate)';
  }
}

function onSelSwitch(e) {
  if (e.target.checked) {
    // увімкнули вручну, але сервер не обрано — повертаємо і підказуємо
    e.target.checked = false;
    if (state.selected.ip) { e.target.checked = true; return; }
    toast('Спочатку оберіть сервер зі списку на сторінці «Підключення»', 'info');
    return;
  }
  clearSelectedServer();
}

function escapeHtml(s) {
  return String(s).replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}

async function clearSelectedServer() {
  await call(App.ClearSelectedServer);
  // активний .ovpn профіль теж скидаємо — перемикач вимкнуто вручну
  if (state.activeProfile) {
    await call(App.DeselectCustomProfile);
    state.activeProfile = '';
    $('pr-profile').textContent = 'Авто (VPN Gate)';
  }
  state.selected = { ip: '', name: '' };
  state.selectedServer = null;
  renderSelection();
  renderServers();
  loadProfiles();
  toast('Вибір скинуто — активний авто-підбір (VPN Gate)', 'info');
}

$('sel-switch').addEventListener('change', onSelSwitch);

/* ── Сервери ───────────────────────────────────────────────── */
function pingClass(ping) {
  if (ping <= 120) return 'ping-good';
  if (ping <= 320) return 'ping-mid';
  return 'ping-bad';
}

function renderServers() {
  const list = $('servers-list');
  list.innerHTML = '';
  let filtered = state.servers;
  if (state.search) {
    const q = state.search.toLowerCase();
    filtered = filtered.filter(s =>
      (s.CountryLong || '').toLowerCase().includes(q) ||
      (s.IP || '').toLowerCase().includes(q) ||
      (s.HostName || '').toLowerCase().includes(q));
  }
  if (state.selectedCountry && state.selectedCountry !== 'All') {
    filtered = filtered.filter(s => s.CountryLong === state.selectedCountry);
  }
  const sorted = [...filtered].sort((a, b) => {
    if (state.sortBy === 'ping') return (a.Ping || 0) - (b.Ping || 0);
    if (state.sortBy === 'speed') return (b.Speed || 0) - (a.Speed || 0);
    return (b.Score || 0) - (a.Score || 0);
  });
  // Власні .ovpn профілі — у тому ж списку, що й сервери VPN Gate
  // (показуються лише без активного фільтра країн)
  const profiles = (state.selectedCountry || 'All') === 'All'
    ? state.profiles.filter(n => !state.search || n.toLowerCase().includes(state.search.toLowerCase()))
    : [];
  $('servers-count').textContent = 'Знайдено: ' + (sorted.length + profiles.length) + ' серверів';

  profiles.forEach(name => {
    const card = document.createElement('div');
    const selected = state.activeProfile === name;
    card.className = 'server-card profile-server' + (selected ? ' selected' : '');
    card.innerHTML = `
      <div class="sc-host">${escapeHtml(name)}
        <span class="sc-type">власний .ovpn</span>
      </div>
      <div class="sc-meta">Підключення через власний конфіг OpenVPN</div>
    `;
    card.addEventListener('click', () => selectProfile(name));
    list.appendChild(card);
  });

  const countries = [...new Set(state.servers.map(s => s.CountryLong).filter(Boolean))].sort();
  const sel = $('country-filter');
  const prev = sel.value;
  sel.innerHTML = '<option value="All">Всі країни</option>';
  countries.forEach(c => { const o = document.createElement('option'); o.value = c; o.textContent = c; sel.appendChild(o); });
  if (prev) sel.value = prev;

  sorted.slice(0, 200).forEach(s => {
    const card = document.createElement('div');
    const selected = state.selectedServer && state.selectedServer.IP === s.IP;
    card.className = 'server-card' + (selected ? ' selected' : '');
    card.innerHTML = `
      <div class="sc-host">${s.HostName || s.IP}
        <span class="sc-ping ${pingClass(s.Ping || 0)}">${s.Ping || 0} ms</span>
      </div>
      <div class="sc-meta">${s.CountryLong || '—'} • ${s.IP || '—'}</div>
      <div class="sc-meta">Швидкість: ${Math.round((s.Speed || 0) / 1000000)} Mbps • Рейтинг: ${s.Score || 0}</div>
    `;
    card.addEventListener('click', async () => {
      state.selectedServer = s;
      state.selected = { ip: s.IP, name: s.HostName || s.IP };
      // вибір сервера та власного .ovpn профілю взаємовиключні
      await call(App.SelectServer, s.IP, s.HostName || s.IP);
      if (state.activeProfile) {
        state.activeProfile = '';
        $('pr-profile').textContent = 'Авто (VPN Gate)';
        loadProfiles();
      }
      renderSelection();
      renderServers();
      toast('Обрано сервер: ' + (s.HostName || s.IP) + ' — підключення буде до нього', 'positive');
    });
    list.appendChild(card);
  });
}

async function loadServers() {
  const cached = await call(App.GetServers);
  if (cached && cached.length) { state.servers = cached; renderServers(); }
  call(App.RefreshServers);
}

$('btn-refresh-servers').addEventListener('click', () => { call(App.RefreshServers); toast('Оновлення...', 'info'); });
$('server-search').addEventListener('input', (e) => { state.search = e.target.value; renderServers(); });
$('country-filter').addEventListener('change', (e) => { state.selectedCountry = e.target.value; renderServers(); });
$('sort-by').addEventListener('change', (e) => { state.sortBy = e.target.value; renderServers(); });

/* ── Профілі .ovpn ─────────────────────────────────────────── */
async function selectProfile(name) {
  await call(App.SelectCustomProfile, name);
  state.activeProfile = name;
  state.selectedServer = null;
  state.selected = { ip: '', name: '' };
  $('pr-profile').textContent = name;
  renderSelection();
  renderServers();
  loadProfiles();
  toast('Профіль ' + name + ' обрано — підключення буде через нього', 'positive');
}

async function deselectActiveProfile() {
  await call(App.DeselectCustomProfile);
  state.activeProfile = '';
  $('pr-profile').textContent = 'Авто (VPN Gate)';
  renderSelection();
  renderServers();
  loadProfiles();
  toast('Профіль відмінено — знову авто-підбір VPN Gate', 'info');
}

async function deleteProfile(name) {
  const ok = await call(App.DeleteProfile, name);
  if (ok) {
    toast('Профіль ' + name + ' видалено', 'positive');
    // Бекенд сам скидає активний профіль у settings.json —
    // тут лише синхронізуємо стан інтерфейсу
    if (state.activeProfile === name) {
      state.activeProfile = '';
      $('pr-profile').textContent = 'Авто (VPN Gate)';
      renderSelection();
    }
    loadProfiles();
  } else toast('Не вдалося видалити профіль ' + name, 'negative');
}

async function loadProfiles() {
  const names = await call(App.ListProfiles) || [];
  state.profiles = names;
  const box = $('profiles-list');
  box.innerHTML = '';
  if (!names.length) {
    box.innerHTML = '<div class="muted">Поки немає збережених конфігів</div>';
  }
  names.forEach(name => {
    const active = state.activeProfile === name;
    const item = document.createElement('div');
    item.className = 'profile-item' + (active ? ' active' : '');
    item.innerHTML = `<span>${escapeHtml(name)}</span><span class="pi-actions">
      <label class="bar-switch" title="${active
        ? 'Скасувати профіль — повернутися до авто-підбору VPN Gate'
        : 'Вибрати цей профіль — підключення буде через нього'}">
        <input type="checkbox" class="switch" ${active ? 'checked' : ''}/>
      </label>
      <button data-del="${escapeHtml(name)}">Видалити</button>
    </span>`;
    const sw = item.querySelector('.switch');
    sw.addEventListener('change', () => {
      if (sw.checked) selectProfile(name);
      else if (state.activeProfile === name) deselectActiveProfile();
      else sw.checked = false;
    });
    box.appendChild(item);
  });
  box.querySelectorAll('button[data-del]').forEach(b => {
    b.addEventListener('click', () => deleteProfile(b.dataset.del));
  });
  // Профілі тепер також показуються у списку серверів — оновлюємо його
  renderServers();
}

$('btn-upload-ovpn').addEventListener('click', async () => {
  const saved = await call(App.SaveProfileDialog);
  if (saved) {
    state.activeProfile = saved;
    $('pr-profile').textContent = saved;
    state.selectedServer = null;
    state.selected = { ip: '', name: '' };
    renderSelection();
    loadProfiles();
    toast('Завантажено: ' + saved + ' — профіль активний', 'positive');
  }
});

/* ── Tor ───────────────────────────────────────────────────── */
const TOR_HERO = {
  stopped:  { cls: 'state-stopped',    title: 'Tor зупинено',  sub: 'SOCKS5-проксі для анонімізації трафіку.', btn: 'Запустити Tor' },
  starting: { cls: 'state-connecting', title: 'Запуск Tor...', sub: "Налаштовуємо мости та встановлюємо з'єднання...", btn: 'Запуск...' },
  active:   { cls: 'state-active',     title: 'Tor активний',  sub: 'SOCKS5-проксі готовий до роботи.', btn: 'Зупинити Tor' },
};

function updateTorHero() {
  const mode = state.torActive ? 'active' : (state.torStarting ? 'starting' : 'stopped');
  const h = TOR_HERO[mode];
  const logo = $('tor-logo');
  if (logo) {
    logo.classList.remove('state-stopped', 'state-connecting', 'state-active');
    logo.classList.add(h.cls);
  }
  const title = $('tor-title');
  const subtitle = $('tor-subtitle');
  if (title) title.textContent = h.title;
  if (subtitle) subtitle.textContent = state.torActive
    ? 'Локальна адреса: 127.0.0.1:' + state.torPort
    : h.sub;
  const btn = $('btn-toggle-tor');
  if (btn) {
    btn.textContent = h.btn;
    btn.disabled = state.torStarting && !state.torActive;
  }
}

async function refreshTor() {
  const st = await call(App.GetTorState);
  if (!st) return;
  state.torActive = !!st.active;
  state.torStarting = false;
  state.torPort = st.port || 9050;
  $('tor-port').textContent = state.torPort;
  $('tor-port-input').value = state.torPort;
  updateTorHero();
}

/* Зберігає власні мости Tor (переключатель + список рядків) */
async function saveTorBridges(showToast) {
  const enabled = $('tor-bridges-enabled').checked;
  const lines = $('tor-bridges').value;
  await call(App.UpdateSetting, 'tor_bridges_enabled', String(enabled));
  await call(App.UpdateSetting, 'tor_bridges', lines);
  if (showToast) {
    if (state.torActive) toast('Мости збережено — зупиніть і запустіть Tor знову, щоб застосувати', 'info');
    else toast('Налаштування мостів збережено', 'positive');
  }
}

$('btn-toggle-tor').addEventListener('click', async () => {
  if (state.torStarting) return;
  if (state.torActive) { await call(App.ToggleTor); return; }
  // Перед запуском зберігаємо порт та мости — Tor читає їх при старті
  const port = parseInt($('tor-port-input').value) || 9050;
  await call(App.UpdateSetting, 'tor_socks_port', String(port));
  await saveTorBridges(false);
  state.torStarting = true;
  updateTorHero();
  await call(App.ToggleTor);
});

$('tor-port-input').addEventListener('change', async (e) => {
  const p = parseInt(e.target.value);
  if (p < 1024 || p > 65535) { e.target.value = 9050; toast('Порт має бути у діапазоні 1024–65535', 'negative'); return; }
  await call(App.UpdateSetting, 'tor_socks_port', String(p));
});

$('tor-bridges-enabled').addEventListener('change', async (e) => {
  $('tor-bridges').classList.toggle('hidden', !e.target.checked);
  await saveTorBridges(true);
});
$('tor-bridges').addEventListener('change', () => saveTorBridges(true));
$('btn-apply-tor').addEventListener('click', () => saveTorBridges(true));

$('btn-copy-tor-logs').addEventListener('click', async () => {
  const logs = await call(App.GetTorLogs) || [];
  await navigator.clipboard.writeText(logs.join('\n'));
  toast('Логи Tor скопійовано', 'positive');
});
$('btn-clear-tor-logs').addEventListener('click', async () => { await call(App.ClearTorLogs); $('tor-logs').innerHTML = ''; });

/* ── WireGuard ─────────────────────────────────────────────── */
const WG_HERO = {
  DISCONNECTED: { cls: 'state-stopped',    title: 'WireGuard відключено',  sub: 'Криптографічний тунель на базі wireguard-go.', btn: 'Підключити WireGuard' },
  CONNECTING:   { cls: 'state-connecting', title: 'Підключення...',        sub: 'Створюємо Wintun адаптер та чекаємо хендшейк.', btn: 'Скасувати' },
  CONNECTED:    { cls: 'state-connected',  title: 'WireGuard підключено',  sub: 'Тунель активний — весь трафік зашифровано.', btn: 'Відключити' },
};

function humanBytes(n) {
  if (!n || n <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let i = 0;
  while (n >= 1024 && i < units.length - 1) { n /= 1024; i++; }
  return (i === 0 ? String(Math.round(n)) : n.toFixed(1)) + ' ' + units[i];
}

function updateWGHero() {
  const st = state.wgStatus || 'DISCONNECTED';
  const h = WG_HERO[st] || WG_HERO.DISCONNECTED;
  const shield = $('wg-shield');
  if (shield) {
    shield.classList.remove('state-stopped', 'state-connecting', 'state-connected');
    shield.classList.add(h.cls);
  }
  const title = $('wg-title');
  if (title) title.textContent = h.title;
  const subtitle = $('wg-subtitle');
  if (subtitle) {
    subtitle.textContent = st === 'CONNECTED' && state.wgStats && state.wgStats.endpoint
      ? 'Тунель активний: ' + state.wgStats.endpoint
      : h.sub;
  }
  const btn = $('btn-toggle-wg');
  if (btn) btn.textContent = h.btn;
  const line = $('wg-profile-line');
  if (line) {
    if (state.wgProfile) {
      line.innerHTML = 'Профіль: <b>' + escapeHtml(state.wgProfile) + '</b>' +
        (state.wgStats && state.wgStats.address ? ' • Адреса тунелю: <b>' + escapeHtml(state.wgStats.address) + '</b>' : '');
    } else {
      line.innerHTML = 'Профіль: <b>не обрано</b> — імпортуйте .conf файл нижче';
    }
  }
}

function updateWGStats(st) {
  if (!st) return;
  const $v = (id) => $(id);
  $v('wg-address').textContent = st.address || '—';
  $v('wg-endpoint').textContent = st.endpoint || '—';
  $v('wg-dns').textContent = (st.dns && st.dns.length) ? st.dns.join(', ') : '—';
  const hs = $v('wg-handshake');
  if (st.status !== 'CONNECTED') hs.textContent = '—';
  else if (st.last_handshake < 0) hs.textContent = 'очікується...';
  else hs.textContent = st.last_handshake < 3 ? 'щойно' : Math.round(st.last_handshake) + ' с тому';
  $v('wg-rx').textContent = humanBytes(st.rx);
  $v('wg-tx').textContent = humanBytes(st.tx);
}

function wgInfoText(info) {
  if (!info || !info.ok) return '';
  const parts = [];
  if (info.endpoint) parts.push('Endpoint: ' + info.endpoint);
  if (info.addresses && info.addresses.length) parts.push('Адреса: ' + info.addresses.join(', '));
  parts.push(info.peers === 1 ? '1 пір' : (info.peers || 0) + ' пірів');
  return parts.join(' • ');
}

async function fillWGInfo(name) {
  const info = await call(App.GetWireGuardProfileInfo, name);
  const el = document.querySelector('[data-wg-info="' + CSS.escape(name) + '"]');
  if (el) el.textContent = wgInfoText(info);
}

function renderWGProfiles() {
  const box = $('wg-profiles-list');
  box.innerHTML = '';
  if (!state.wgProfiles.length) {
    box.innerHTML = '<div class="muted">Поки немає .conf профілів</div>';
  }
  state.wgProfiles.forEach(name => {
    const active = state.wgProfile === name;
    const item = document.createElement('div');
    item.className = 'profile-item' + (active ? ' active' : '');
    item.innerHTML = `<div class="wg-prof-main">
      <span>${escapeHtml(name)}</span>
      <span class="wg-prof-info" data-wg-info="${escapeHtml(name)}"></span>
    </div>
    <span class="pi-actions">
      <label class="bar-switch" title="${active
        ? 'Скасувати профіль WireGuard'
        : 'Вибрати цей профіль — підключення буде через нього'}">
        <input type="checkbox" class="switch" ${active ? 'checked' : ''}/>
      </label>
      <button data-wg-del="${escapeHtml(name)}">Видалити</button>
    </span>`;
    const sw = item.querySelector('.switch');
    sw.addEventListener('change', () => {
      if (sw.checked) selectWGProfile(name);
      else if (state.wgProfile === name) deselectWGProfile(name);
      else sw.checked = false;
    });
    box.appendChild(item);
    fillWGInfo(name);
  });
  box.querySelectorAll('button[data-wg-del]').forEach(b => {
    b.addEventListener('click', () => deleteWGProfile(b.dataset.wgDel));
  });
}

async function selectWGProfile(name) {
  await call(App.SelectWireGuardProfile, name);
  state.wgProfile = name;
  renderWGProfiles();
  updateWGHero();
  toast('Профіль ' + name + ' обрано — підключення буде через нього', 'positive');
}

async function deselectWGProfile(name) {
  await call(App.DeselectWireGuardProfile, name);
  if (state.wgProfile === name) state.wgProfile = '';
  renderWGProfiles();
  updateWGHero();
  toast('Профіль відмінено', 'info');
}

async function deleteWGProfile(name) {
  const ok = await call(App.DeleteWireGuardProfile, name);
  if (ok) {
    toast('Профіль ' + name + ' видалено', 'positive');
    if (state.wgProfile === name) state.wgProfile = '';
    await loadWGProfiles();
    updateWGHero();
  } else toast('Не вдалося видалити профіль ' + name, 'negative');
}

async function loadWGProfiles() {
  const names = await call(App.ListWireGuardProfiles) || [];
  state.wgProfiles = names;
  renderWGProfiles();
}

async function refreshWG() {
  const st = await call(App.GetWireGuardState);
  if (st) { state.wgStatus = st.status || 'DISCONNECTED'; state.wgStats = st; }
  updateWGHero();
  updateWGStats(state.wgStats);
}

$('btn-toggle-wg').addEventListener('click', async () => {
  if (!hasBackend()) { toast('Демо-режим у браузері', 'info'); return; }
  await call(App.ToggleWireGuard);
});

$('btn-import-wg').addEventListener('click', async () => {
  const saved = await call(App.ImportWireGuardProfileDialog);
  if (saved) {
    await call(App.SelectWireGuardProfile, saved);
    state.wgProfile = saved;
    await loadWGProfiles();
    updateWGHero();
    toast('Завантажено: ' + saved + ' — профіль активний', 'positive');
  }
});

$('btn-copy-wg-logs').addEventListener('click', async () => {
  const logs = await call(App.GetWireGuardLogs) || [];
  await navigator.clipboard.writeText(logs.join('\n'));
  toast('Логи WireGuard скопійовано', 'positive');
});
$('btn-clear-wg-logs').addEventListener('click', async () => { await call(App.ClearWireGuardLogs); $('wg-logs').innerHTML = ''; });

/* ── Налаштування (автозбереження) ─────────────────────────── */
async function saveBool(key, checked) {
  await call(App.UpdateSetting, key, String(!!checked));
}

$('s-autostart').addEventListener('change', async (e) => {
  // autostart зберігається через SaveSettings (цілісний об'єкт)
  const s = state.settings;
  if (s) { s.autostart = e.target.checked; await call(App.SaveSettings, s); }
  else await saveBool('autostart', e.target.checked);
  toast('Налаштування збережено', 'positive');
});

['s-autoconnect', 's-notifications', 's-killswitch'].forEach(id => {
  $(id).addEventListener('change', async (e) => {
    const key = { 's-autoconnect': 'autoconnect', 's-notifications': 'notifications', 's-killswitch': 'kill_switch' }[id];
    await saveBool(key, e.target.checked);
    if (id === 's-notifications') {
      toast(e.target.checked ? 'Спливаючі сповіщення увімкнено' : 'Спливаючі сповіщення вимкнено (помилки показуватимуться)', 'info');
    } else if (id === 's-killswitch') {
      toast(e.target.checked
        ? 'Kill Switch активується при підключенні VPN — трафік поза тунелем блокується'
        : 'Kill Switch вимкнено — політику брандмауера відновлено', 'info');
    } else {
      toast('Налаштування збережено', 'positive');
    }
  });
});

async function applyDNSSettings() {
  await call(App.UpdateSetting, 'dns_leak_protection', String($('dns-leak').checked));
  await call(App.UpdateSetting, 'block_ads', String($('block-ads').checked));
  await call(App.UpdateSetting, 'dns_resolver', $('dns-resolver').value);
  await call(App.UpdateSetting, 'custom_dns', $('custom-dns').value);
  if (state.vpnStatus === 'CONNECTED' || state.vpnStatus === 'CONNECTING') {
    toast('Налаштування DNS збережено — застосуються після перепідключення VPN', 'info');
  } else {
    toast('Налаштування DNS збережено', 'positive');
  }
}

$('dns-leak').addEventListener('change', applyDNSSettings);
$('block-ads').addEventListener('change', applyDNSSettings);
$('dns-resolver').addEventListener('change', applyDNSSettings);
$('custom-dns').addEventListener('change', applyDNSSettings);

/* ── Логи VPN ──────────────────────────────────────────────── */
$('btn-copy-logs').addEventListener('click', async () => {
  const logs = await call(App.GetVPNLogs) || [];
  await navigator.clipboard.writeText(logs.join('\n'));
  toast('Логи скопійовано в буфер обміну', 'positive');
});
$('btn-clear-logs').addEventListener('click', async () => { await call(App.ClearVPNLogs); $('vpn-logs').innerHTML = ''; });

/* ── Адмін-банер ───────────────────────────────────────────── */
async function checkAdmin() {
  const isAdmin = await call(App.IsAdmin);
  const banner = $('admin-banner');
  const val = $('pr-admin');
  if (isAdmin === undefined) {
    banner.classList.add('hidden');
    if (val) val.textContent = '—';
    return;
  }
  // показуємо стан прав адміністратора у полі на вкладці «Профіль»
  if (val) {
    val.innerHTML = isAdmin
      ? '<span class="val-ok">✓ Запущено від адміністратора</span>'
      : '<span class="val-no">✗ Не запущено від імені адміністратора</span>';
  }
  if (!isAdmin) {
    banner.textContent = 'Потрібні права адміністратора Windows — натисніть, щоб перезапустити з правами адміна';
    banner.classList.remove('hidden');
    banner.onclick = async () => { await call(App.Elevate); };
  } else {
    banner.classList.add('hidden');
  }
}

/* ── Події з бекенду ───────────────────────────────────────── */
function bindEvents() {
  if (!hasRuntime()) return;
  EventsOn('vpn:log', (msg) => appendLog('vpn-logs', msg));
  EventsOn('vpn:state', (status) => {
    state.vpnStatus = status || 'DISCONNECTED';
    updateHero();
    refreshTor();
    // бейдж на сторінці профілю
    const badge = $('pr-state');
    const st = state.vpnStatus;
    badge.textContent = st === 'CONNECTED' ? 'ПІДКЛЮЧЕНО' : st === 'CONNECTING' ? 'ПІДКЛЮЧЕННЯ...' : "ВІД'ЄДНАНО";
    badge.className = 'badge ' + (st === 'CONNECTED' ? 'active' : st === 'CONNECTING' ? 'connecting' : 'stopped');
    if (st === 'DISCONNECTED') refreshIP();
  });
  EventsOn('vpn:ipinfo', (info) => {
    if (info) {
      $('pr-ip').textContent = info.ip || '—';
      $('pr-loc').textContent = (info.city ? info.city + ', ' : '') + (info.country || '—');
      $('pr-isp').textContent = info.isp || '—';
    }
  });
  EventsOn('vpn:notification', (n) => { if (n && n.message) toast(n.message, n.kind); });
  EventsOn('tor:log', (msg) => appendLog('tor-logs', msg));
  EventsOn('tor:state', () => refreshTor());
  EventsOn('wg:log', (msg) => appendLog('wg-logs', msg));
  EventsOn('wg:stats', (st) => {
    if (!st) return;
    state.wgStatus = st.status || 'DISCONNECTED';
    state.wgStats = st;
    updateWGHero();
    updateWGStats(st);
  });
  EventsOn('servers:refreshed', (servers) => {
    if (servers && servers.length) {
      state.servers = servers;
      renderServers();
      toast('Оновлено: знайдено ' + servers.length + ' активних серверів', 'positive');
    } else toast('Не вдалося завантажити список серверів!', 'negative');
  });
}

/* ── Ініціалізація ─────────────────────────────────────────── */
async function init() {
  // Логотипи Tor (бічна панель + hero)
  $('nav-tor-logo').src = torLogo;
  $('tor-logo').src = torLogo;

  bindEvents();
  state.settings = await call(App.GetSettings);
  const s = state.settings;
  if (s) {
    $('s-autostart').checked = !!s.autostart;
    $('s-autoconnect').checked = !!s.autoconnect;
    $('s-notifications').checked = !!s.notifications;
    $('s-killswitch').checked = !!s.kill_switch;
    $('dns-leak').checked = !!s.dns_leak_protection;
    $('block-ads').checked = !!s.block_ads;
    $('dns-resolver').value = s.dns_resolver || 'Cloudflare (DoH)';    $('custom-dns').value = s.custom_dns || '';
    $('tor-port-input').value = s.tor_socks_port || 9050;
    // Власні мости Tor
    $('tor-bridges-enabled').checked = !!s.tor_bridges_enabled;
    $('tor-bridges').value = s.tor_bridges || '';
    $('tor-bridges').classList.toggle('hidden', !s.tor_bridges_enabled);
    // WireGuard
    state.wgProfile = s.wg_profile || '';
    $('warp-mode').value = ['UDP', 'TCP', 'Auto'].includes(s.protocol) ? s.protocol : 'UDP';
    state.activeProfile = s.custom_ovpn_name || '';
    if (state.activeProfile) $('pr-profile').textContent = state.activeProfile;
    $('pr-proto').textContent = $('warp-mode').value;
    state.selected = {
      ip: s.selected_server_ip || '',
      name: s.selected_server_name || '',
    };
  }
  renderSelection();
  await refreshVPNState();
  // бейдж стану
  const st = state.vpnStatus;
  const badge = $('pr-state');
  badge.textContent = st === 'CONNECTED' ? 'ПІДКЛЮЧЕНО' : st === 'CONNECTING' ? 'ПІДКЛЮЧЕННЯ...' : "ВІД'ЄДНАНО";
  badge.className = 'badge ' + (st === 'CONNECTED' ? 'active' : st === 'CONNECTING' ? 'connecting' : 'stopped');

  await checkAdmin();
  await refreshIP();
  await loadServers();
  await loadProfiles();
  updateTorHero();
  await refreshTor();
  await refreshWG();
  await loadWGProfiles();
  const vlogs = await call(App.GetVPNLogs) || [];
  vlogs.forEach(m => appendLog('vpn-logs', m));
  const tlogs = await call(App.GetTorLogs) || [];
  tlogs.forEach(m => appendLog('tor-logs', m));
  const wglogs = await call(App.GetWireGuardLogs) || [];
  wglogs.forEach(m => appendLog('wg-logs', m));
}

init();

"use strict";

/* Braise — front-end vanilla, embarqué dans le binaire.
   Aucun build, aucune dépendance. */

const $ = (sel, root = document) => root.querySelector(sel);
const $$ = (sel, root = document) => Array.from(root.querySelectorAll(sel));

const COLORS = ["#f97316", "#fbbf24", "#34d399", "#38bdf8", "#a78bfa", "#f472b6", "#f87171", "#2dd4bf"];
const LOCALE = "fr-FR";

const ICONS = {
  dir: '<svg viewBox="0 0 24 24" fill="none"><path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7Z" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round"/></svg>',
  img: '<svg viewBox="0 0 24 24" fill="none"><rect x="3" y="4" width="18" height="16" rx="2" stroke="currentColor" stroke-width="1.7"/><circle cx="8.5" cy="9.5" r="1.5" stroke="currentColor" stroke-width="1.5"/><path d="m5 17 4.5-4.5L13 16l2.5-2.5L21 19" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"/></svg>',
  vid: '<svg viewBox="0 0 24 24" fill="none"><rect x="3" y="5" width="13" height="14" rx="2" stroke="currentColor" stroke-width="1.7"/><path d="m16 10 5-3v10l-5-3" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round"/></svg>',
  aud: '<svg viewBox="0 0 24 24" fill="none"><path d="M9 18V6l10-2v12" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round"/><circle cx="6" cy="18" r="3" stroke="currentColor" stroke-width="1.7"/><circle cx="16" cy="16" r="3" stroke="currentColor" stroke-width="1.7"/></svg>',
  pdf: '<svg viewBox="0 0 24 24" fill="none"><path d="M6 2h8l4 4v16H6V2Z" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round"/><path d="M14 2v5h5M8.5 13h1.2a1.3 1.3 0 0 1 0 2.6H8.5V13Zm0 2.6V18m3.5-5h1a2 2 0 0 1 0 5h-1v-5Zm5 0h2.5m-2.5 0v5m0-2.6h2" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round"/></svg>',
  code: '<svg viewBox="0 0 24 24" fill="none"><path d="m8 8-4 4 4 4m8-8 4 4-4 4M14 5l-4 14" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"/></svg>',
  zip: '<svg viewBox="0 0 24 24" fill="none"><path d="M6 2h12v20H6V2Z" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round"/><path d="M12 2v6m0 0v3m0 0v3m-1.5 3h3v4h-3v-4Z" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round"/></svg>',
  file: '<svg viewBox="0 0 24 24" fill="none"><path d="M6 2h8l4 4v16H6V2Z" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round"/><path d="M14 2v5h5M9 12h6M9 16h6" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>',
  download: '<svg viewBox="0 0 24 24" fill="none"><path d="M12 4v12m0 0 5-5m-5 5-5-5M5 20h14" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/></svg>',
  trash: '<svg viewBox="0 0 24 24" fill="none"><path d="M4 7h16M9 7V4h6v3m-8 0 1 14h8l1-14M10 11v6m4-6v6" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/></svg>',
  folder: '<svg viewBox="0 0 24 24" fill="none"><path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7Z" stroke="currentColor" stroke-width="1.7" stroke-linejoin="round"/></svg>',
};

const IMAGE_EXT = new Set(["png", "jpg", "jpeg", "gif", "webp", "avif", "bmp", "svg", "heic", "heif", "ico", "tif", "tiff"]);
const VIDEO_EXT = new Set(["mp4", "webm", "mkv", "mov", "avi", "m4v", "ogv"]);
const AUDIO_EXT = new Set(["mp3", "wav", "ogg", "oga", "m4a", "flac", "opus", "aac", "wma"]);
const CODE_EXT = new Set(["js", "mjs", "ts", "tsx", "jsx", "go", "py", "rb", "rs", "c", "h", "cpp", "hpp", "cs", "java", "kt", "php", "swift", "sh", "bash", "zsh", "ps1", "sql", "html", "htm", "css", "scss", "json", "yaml", "yml", "toml", "xml", "ini", "conf", "env", "dockerfile"]);
const ZIP_EXT = new Set(["zip", "rar", "7z", "tar", "gz", "bz2", "xz", "zst", "iso", "dmg"]);
const TEXT_EXT = new Set(["txt", "md", "markdown", "log", "csv", "tsv", "rtf", "srt", "vtt", "nfo"]);

const state = {
  info: null,
  identity: null,
  entries: [],
  path: "",
  query: "",
  messages: [],
  peers: [],
  lastEventId: 0,
  lastDay: null,
  lastAuthor: null,
  lastAt: 0,
  attachment: null,
  es: null,
  sending: false,
  unread: 0,
  notify: false,
  atBottom: true,
  typing: new Map(),
  view: "chat",
  booted: false,
};

/* ---------- utilitaires ---------- */

function storageGet(key) {
  try { return localStorage.getItem(key); } catch { return null; }
}
function storageSet(key, value) {
  try { localStorage.setItem(key, value); } catch { /* mode privé */ }
}

function newId() {
  try {
    if (window.crypto && typeof window.crypto.randomUUID === "function") {
      return window.crypto.randomUUID();
    }
  } catch { /* contexte non sécurisé */ }
  const bytes = new Uint8Array(16);
  if (window.crypto && window.crypto.getRandomValues) {
    window.crypto.getRandomValues(bytes);
  } else {
    for (let i = 0; i < bytes.length; i++) bytes[i] = Math.floor(Math.random() * 256);
  }
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}

function randomName() {
  return "Invité " + String(Math.floor(10 + Math.random() * 90));
}

function initial(name) {
  const chars = Array.from((name || "?").trim());
  return (chars[0] || "?").toUpperCase();
}

const numberFmt = new Intl.NumberFormat(LOCALE, { maximumFractionDigits: 1 });
const clockFmt = new Intl.DateTimeFormat(LOCALE, { hour: "2-digit", minute: "2-digit" });
const dateFmt = new Intl.DateTimeFormat(LOCALE, { day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit" });
const longDateFmt = new Intl.DateTimeFormat(LOCALE, { weekday: "long", day: "numeric", month: "long" });

function fmtSize(n) {
  if (!n || n < 0) return "0 o";
  const units = ["o", "Ko", "Mo", "Go", "To"];
  let v = n, i = 0;
  while (v >= 1024 && i < units.length - 1) { v /= 1024; i++; }
  return `${numberFmt.format(v)} ${units[i]}`;
}

function fmtClock(ts) {
  const d = new Date(ts);
  return Number.isNaN(d.getTime()) ? "" : clockFmt.format(d);
}

function fmtDate(ts) {
  const d = new Date(ts);
  return Number.isNaN(d.getTime()) ? "" : dateFmt.format(d);
}

function sameDay(a, b) {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
}

function dayKey(ts) {
  const d = new Date(ts);
  return Number.isNaN(d.getTime()) ? "?" : d.toDateString();
}

function dayLabel(ts) {
  const d = new Date(ts);
  const today = new Date();
  const yesterday = new Date();
  yesterday.setDate(today.getDate() - 1);
  if (sameDay(d, today)) return "Aujourd'hui";
  if (sameDay(d, yesterday)) return "Hier";
  return longDateFmt.format(d);
}

function baseName(p) {
  return String(p || "").split("/").filter(Boolean).pop() || "";
}

function fileKind(entry) {
  if (entry.dir) return "dir";
  const name = String(entry.name || "").toLowerCase();
  const ext = name.includes(".") ? name.split(".").pop() : "";
  const mime = String(entry.mime || "");
  if (mime.startsWith("image/") || IMAGE_EXT.has(ext)) return "img";
  if (mime.startsWith("video/") || VIDEO_EXT.has(ext)) return "vid";
  if (mime.startsWith("audio/") || AUDIO_EXT.has(ext)) return "aud";
  if (mime === "application/pdf" || ext === "pdf") return "pdf";
  if (mime.startsWith("text/") || TEXT_EXT.has(ext)) return "file";
  if (CODE_EXT.has(ext)) return "code";
  if (ZIP_EXT.has(ext)) return "zip";
  return "file";
}

function isTextEntry(entry) {
  const mime = String(entry.mime || "");
  if (mime.startsWith("text/") || mime === "application/json" || mime === "application/xml" || mime === "application/javascript") return true;
  const name = String(entry.name || "").toLowerCase();
  const ext = name.includes(".") ? name.split(".").pop() : "";
  return TEXT_EXT.has(ext) || CODE_EXT.has(ext);
}

function iconHTML(kind) {
  return ICONS[kind] || ICONS.file;
}

function toast(message, { error = false, timeout = 4200 } = {}) {
  const host = $("#toasts");
  const el = document.createElement("div");
  el.className = "toast" + (error ? " error" : "");
  const dot = document.createElement("span");
  dot.className = "dot";
  const text = document.createElement("span");
  text.textContent = message;
  el.append(dot, text);
  host.appendChild(el);
  if (timeout) {
    setTimeout(() => {
      el.style.transition = "opacity .18s, transform .18s";
      el.style.opacity = "0";
      el.style.transform = "translateY(4px)";
      setTimeout(() => el.remove(), 200);
    }, timeout);
  }
}

function isMobile() {
  return window.matchMedia("(max-width: 900px)").matches;
}

/* ---------- API ---------- */

async function api(path, { method = "GET", json, body, headers } = {}) {
  const h = new Headers(headers || {});
  h.set("Accept", "application/json");
  if (state.identity) h.set("X-Braise-Peer", state.identity.id);
  const opts = { method, headers: h, cache: "no-store" };
  if (json !== undefined) {
    h.set("Content-Type", "application/json");
    opts.body = JSON.stringify(json);
  } else if (body !== undefined) {
    opts.body = body;
  }
  const res = await fetch(path, opts);
  const ct = res.headers.get("content-type") || "";
  let data = null;
  try {
    data = ct.includes("application/json") ? await res.json() : await res.text();
  } catch { /* réponse vide */ }
  if (!res.ok) {
    const msg = data && typeof data === "object" && data.error ? data.error : `Erreur ${res.status}`;
    throw new Error(msg);
  }
  return data;
}

function downloadURL(path) {
  return "/api/download?path=" + encodeURIComponent(path);
}
function rawURL(path) {
  return "/api/raw?path=" + encodeURIComponent(path);
}

function triggerDownload(entry) {
  const a = document.createElement("a");
  a.href = downloadURL(entry.path);
  a.download = entry.name || "";
  document.body.appendChild(a);
  a.click();
  a.remove();
}

/* ---------- identité ---------- */

function loadIdentity() {
  const raw = storageGet("braise.identity");
  if (!raw) return null;
  try {
    const v = JSON.parse(raw);
    if (v && typeof v.id === "string" && /^[A-Za-z0-9_-]{6,64}$/.test(v.id) && typeof v.name === "string" && v.name.trim()) {
      return { id: v.id, name: v.name.slice(0, 48), color: /^#[0-9a-fA-F]{6}$/.test(v.color) ? v.color : COLORS[0] };
    }
  } catch { /* corrompu */ }
  return null;
}

function saveIdentity() {
  if (state.identity) storageSet("braise.identity", JSON.stringify(state.identity));
}

function identityDialog(existing) {
  return new Promise((resolve) => {
    const dlg = $("#identity-dialog");
    const form = $("#identity-form");
    const input = $("#identity-name");
    const swatches = $("#swatches");
    let selected = existing?.color || COLORS[Math.floor(Math.random() * COLORS.length)];

    input.value = existing?.name || randomName();
    swatches.innerHTML = "";
    for (const color of COLORS) {
      const b = document.createElement("button");
      b.type = "button";
      b.className = "swatch" + (color === selected ? " active" : "");
      b.style.background = color;
      b.setAttribute("role", "radio");
      b.setAttribute("aria-checked", color === selected ? "true" : "false");
      b.setAttribute("aria-label", "Couleur " + color);
      b.addEventListener("click", () => {
        selected = color;
        $$(".swatch", swatches).forEach((x) => {
          const on = x === b;
          x.classList.toggle("active", on);
          x.setAttribute("aria-checked", on ? "true" : "false");
        });
      });
      swatches.appendChild(b);
    }

    const submit = (e) => {
      e.preventDefault();
      const name = input.value.trim().slice(0, 48) || randomName();
      form.removeEventListener("submit", submit);
      dlg.close();
      resolve({ id: existing?.id || newId(), name, color: selected });
    };
    form.addEventListener("submit", submit);
    if (!dlg.open) dlg.showModal();
    setTimeout(() => { input.focus(); input.select(); }, 60);
  });
}

function applyIdentity() {
  if (!state.identity) return;
  const av = $("#me-avatar");
  av.textContent = initial(state.identity.name);
  av.style.background = state.identity.color;
  av.style.color = "#0a0c11";
  $("#me-btn").title = `${state.identity.name} — changer de profil`;
}

/* ---------- démarrage ---------- */

async function init() {
  bindUI();
  state.notify = storageGet("braise.notify") === "1" && "Notification" in window && Notification.permission === "granted";
  updateNotifyButton();

  let identity = loadIdentity();
  if (!identity) {
    identity = await identityDialog(null);
  }
  state.identity = identity;
  saveIdentity();
  applyIdentity();

  await loadInfo();
  await Promise.allSettled([loadMessages(), loadFiles()]);
  connectSSE();
  startPresence();
  setView(isMobile() ? "files" : "chat");
  state.booted = true;
}

/* ---------- infos ---------- */

async function loadInfo() {
  try {
    const info = await api("/api/info");
    state.info = info;
    $("#space-name").textContent = info.rootName || info.name || "espace local";
    $("#version-line").textContent = `Braise ${info.version} · ${info.root}`;
    $("#share-hint").textContent = "Scannez avec l'appareil photo d'un téléphone connecté au même Wi-Fi.";
    applyReadOnly();
    updateTitle();
  } catch (e) {
    toast("Impossible de joindre le serveur Braise.", { error: true });
  }
}

function applyReadOnly() {
  const ro = !!state.info?.readOnly;
  $("#readonly-badge").hidden = !ro;
  $("#upload-label").hidden = ro;
  $("#mkdir-btn").hidden = ro;
  $("#fab-upload").hidden = ro;
  const attach = $(".composer .attach");
  if (attach) attach.hidden = ro;
  if (ro) clearAttachment();
  updateSend();
}

/* ---------- messages ---------- */

async function loadMessages() {
  try {
    const data = await api("/api/messages?limit=200");
    state.messages = Array.isArray(data.messages) ? data.messages : [];
    state.lastEventId = Number(data.lastEventId) || 0;
    renderMessages();
    updateChatSub();
  } catch (e) {
    toast("Historique indisponible : " + e.message, { error: true });
  }
}

function messageElement(m, { lead }) {
  const el = document.createElement("div");
  el.className = "msg" + (lead ? " lead" : " grouped") + (m.authorId === state.identity?.id ? " self" : "");

  if (lead) {
    const av = document.createElement("span");
    av.className = "avatar";
    av.textContent = initial(m.author);
    av.style.background = m.color || COLORS[0];
    el.appendChild(av);
  } else {
    const gutter = document.createElement("div");
    gutter.className = "msg-gutter";
    el.appendChild(gutter);
  }

  const body = document.createElement("div");
  body.className = "msg-body";

  if (lead) {
    const head = document.createElement("div");
    head.className = "msg-head";
    const author = document.createElement("span");
    author.className = "msg-author";
    author.textContent = m.author || "Invité";
    author.style.color = m.color || COLORS[0];
    const time = document.createElement("time");
    time.className = "msg-time";
    time.dateTime = m.at || "";
    time.textContent = fmtClock(m.at);
    head.append(author, time);
    body.appendChild(head);
  }

  if (m.text) {
    const text = document.createElement("div");
    text.className = "msg-text";
    linkify(text, m.text);
    body.appendChild(text);
  }

  if (m.file && m.file.path) {
    body.appendChild(attachmentElement(m.file));
  }

  el.appendChild(body);
  return el;
}

function attachmentElement(file) {
  const el = document.createElement("div");
  el.className = "attachment";
  el.tabIndex = 0;
  el.setAttribute("role", "button");
  el.title = "Ouvrir " + file.name;

  const kind = fileKind({ name: file.name, mime: file.mime, dir: false });
  const icon = document.createElement("span");
  icon.className = "file-icon " + kind;
  icon.innerHTML = iconHTML(kind);

  const info = document.createElement("span");
  info.className = "attachment-info";
  const name = document.createElement("span");
  name.className = "attachment-name";
  name.textContent = file.name;
  const size = document.createElement("span");
  size.className = "attachment-size";
  size.textContent = fmtSize(file.size) + (file.mime ? " · " + file.mime.split(";")[0] : "");
  info.append(name, size);
  el.append(icon, info);

  const open = () => openPreview(file);
  el.addEventListener("click", open);
  el.addEventListener("keydown", (e) => {
    if (e.key === "Enter" || e.key === " ") { e.preventDefault(); open(); }
  });
  return el;
}

function linkify(el, text) {
  const re = /\bhttps?:\/\/[^\s<>"']+/g;
  let last = 0, m;
  while ((m = re.exec(text)) !== null) {
    if (m.index > last) el.appendChild(document.createTextNode(text.slice(last, m.index)));
    let url = m[0];
    const trailing = url.match(/[),.;:!?]+$/);
    if (trailing) url = url.slice(0, -trailing[0].length);
    const a = document.createElement("a");
    a.href = url;
    a.textContent = url;
    a.target = "_blank";
    a.rel = "noopener noreferrer";
    el.appendChild(a);
    if (trailing) el.appendChild(document.createTextNode(trailing[0]));
    last = m.index + m[0].length;
  }
  if (last < text.length) el.appendChild(document.createTextNode(text.slice(last)));
}

function renderMessages() {
  const host = $("#messages");
  host.innerHTML = "";
  state.lastDay = null;
  state.lastAuthor = null;
  state.lastAt = 0;
  if (!state.messages.length) {
    host.appendChild(emptyState(ICONS.folder, "Aucun message", "Lancez la discussion : un bonjour, un fichier, une blague."));
    return;
  }
  for (const m of state.messages) appendMessageNode(m);
  requestAnimationFrame(() => { host.scrollTop = host.scrollHeight; });
}

function appendMessageNode(m) {
  const host = $("#messages");
  const placeholder = host.querySelector(".empty");
  if (placeholder) host.innerHTML = "";

  const ts = Date.parse(m.at) || Date.now();
  const dk = dayKey(ts);
  if (dk !== state.lastDay) {
    const sep = document.createElement("div");
    sep.className = "date-sep";
    sep.textContent = dayLabel(ts);
    host.appendChild(sep);
    state.lastDay = dk;
    state.lastAuthor = null;
  }
  const lead = state.lastAuthor !== m.authorId || (ts - state.lastAt > 5 * 60 * 1000);
  host.appendChild(messageElement(m, { lead }));
  state.lastAuthor = m.authorId;
  state.lastAt = ts;
}

function ingestMessage(m, { eventId } = {}) {
  if (!m || typeof m.id !== "number") return false;
  if (state.messages.some((x) => x.id === m.id)) return false;
  const atBottom = isAtBottom();
  state.messages.push(m);
  if (eventId && eventId > state.lastEventId) state.lastEventId = eventId;

  if (state.messages.length > 800) {
    state.messages.splice(0, 200);
    renderMessages();
  } else {
    appendMessageNode(m);
  }
  updateChatSub();

  if (atBottom) {
    scrollToBottom(true);
  } else {
    showJump(true);
  }

  const mine = m.authorId === state.identity?.id;
  if (!mine) {
    const chatVisible = !isMobile() || state.view === "chat";
    const away = document.hidden || !document.hasFocus();
    if (away || !chatVisible || !atBottom) bumpUnread();
    notifyMessage(m, { chatVisible, away });
  }
  return true;
}

function bumpUnread() {
  state.unread++;
  updateUnreadUI();
}

function clearUnread() {
  if (!state.unread) return;
  state.unread = 0;
  updateUnreadUI();
}

function updateUnreadUI() {
  const badge = $("#chat-badge");
  badge.hidden = state.unread === 0;
  badge.textContent = state.unread > 99 ? "99+" : String(state.unread);
  updateTitle();
}

function updateTitle() {
  const name = state.info?.name || "Braise";
  document.title = (state.unread ? `(${state.unread}) ` : "") + name;
}

function updateChatSub() {
  const n = state.peers.length;
  const people = n <= 1 ? `${n} personne connectée` : `${n} personnes connectées`;
  const messages = state.messages.length;
  $("#chat-sub").textContent = `${people} · ${messages} message${messages > 1 ? "s" : ""}`;
}

/* ---------- présence / saisie ---------- */

function renderPeers() {
  const host = $("#peers");
  host.innerHTML = "";
  const others = state.peers.filter((p) => p.id !== state.identity?.id);
  for (const p of others.slice(0, 5)) {
    const el = document.createElement("span");
    el.className = "peer";
    el.style.background = p.color || COLORS[0];
    el.textContent = initial(p.name);
    el.title = p.name;
    host.appendChild(el);
  }
  if (others.length > 5) {
    const more = document.createElement("span");
    more.className = "peer peer-more";
    more.textContent = "+" + (others.length - 5);
    more.title = others.slice(5).map((p) => p.name).join(", ");
    host.appendChild(more);
  }
  host.title = others.length ? others.map((p) => p.name).join(", ") : "Personne d'autre pour l'instant";
  updateChatSub();
}

function handleTyping(t) {
  if (!t || !t.id || t.id === state.identity?.id) return;
  state.typing.set(t.id, { name: t.name || "Quelqu'un", until: Date.now() + 3500 });
  renderTyping();
}

function renderTyping() {
  const now = Date.now();
  for (const [id, v] of state.typing) if (v.until < now) state.typing.delete(id);
  const names = [...state.typing.values()].map((v) => v.name);
  const el = $("#typing");
  if (!names.length) {
    el.textContent = "";
  } else if (names.length === 1) {
    el.textContent = `${names[0]} écrit…`;
  } else if (names.length === 2) {
    el.textContent = `${names[0]} et ${names[1]} écrivent…`;
  } else {
    el.textContent = `${names.slice(0, 2).join(", ")} et ${names.length - 2} autre${names.length > 3 ? "s" : ""} écrivent…`;
  }
}

async function sendPresence() {
  if (!state.identity) return;
  try {
    const data = await api("/api/presence", {
      method: "POST",
      json: { id: state.identity.id, name: state.identity.name, color: state.identity.color },
    });
    if (data && Array.isArray(data.peers)) {
      state.peers = data.peers;
      renderPeers();
    }
  } catch { /* le SSE signalera l'état */ }
}

function startPresence() {
  sendPresence();
  setInterval(sendPresence, 20000);
  window.addEventListener("online", sendPresence);
  document.addEventListener("visibilitychange", () => {
    if (!document.hidden) {
      sendPresence();
      if (state.view === "chat" && state.atBottom) clearUnread();
    }
  });
  window.addEventListener("pagehide", () => {
    try {
      if (!state.identity) return;
      const blob = new Blob([JSON.stringify({ id: state.identity.id })], { type: "application/json" });
      navigator.sendBeacon("/api/presence/leave", blob);
    } catch { /* ignore */ }
  });
}

let lastTypingSent = 0;
function sendTyping() {
  if (!state.identity) return;
  const now = Date.now();
  if (now - lastTypingSent < 1800) return;
  lastTypingSent = now;
  api("/api/typing", { method: "POST", json: { id: state.identity.id, name: state.identity.name } }).catch(() => {});
}

/* ---------- SSE ---------- */

function noteEvent(e) {
  const id = Number(e.lastEventId);
  if (Number.isFinite(id) && id > state.lastEventId) state.lastEventId = id;
}

function connectSSE() {
  if (state.es) state.es.close();
  setConn("connecting");
  const es = new EventSource(`/api/events?after=${state.lastEventId}`);
  state.es = es;

  es.addEventListener("hello", () => setConn("online"));
  es.onopen = () => setConn("online");
  es.onerror = () => setConn("offline");

  es.addEventListener("message", (e) => {
    noteEvent(e);
    let m;
    try { m = JSON.parse(e.data); } catch { return; }
    ingestMessage(m, { eventId: Number(e.lastEventId) });
  });

  es.addEventListener("file", (e) => {
    noteEvent(e);
    let ev;
    try { ev = JSON.parse(e.data); } catch { return; }
    handleFileEvent(ev);
  });

  es.addEventListener("presence", (e) => {
    try {
      const peers = JSON.parse(e.data);
      if (Array.isArray(peers)) {
        state.peers = peers;
        renderPeers();
      }
    } catch { /* ignore */ }
  });

  es.addEventListener("typing", (e) => {
    try { handleTyping(JSON.parse(e.data)); } catch { /* ignore */ }
  });
}

function setConn(status) {
  const el = $("#conn");
  el.dataset.state = status;
  $("#conn-label").textContent = status === "online" ? "connecté" : status === "offline" ? "hors ligne" : "connexion";
}

function handleFileEvent(ev) {
  if (!ev) return;
  const mine = ev.actor && ev.actor === state.identity?.id;
  if (!mine) {
    const name = baseName(ev.path) || ev.path;
    if (ev.action === "deleted") toast(`« ${name} » a été supprimé`);
    else if (ev.action === "mkdir") toast(`Dossier « ${name} » créé`);
    else if (ev.action === "added") toast(`« ${name} » vient d'être déposé`);
  }
  const current = state.path;
  if (ev.action === "deleted" && (current === ev.path || current.startsWith(ev.path + "/"))) {
    openDir(ev.parent || "");
    return;
  }
  if (ev.parent === current || ev.path === current || current.startsWith(ev.path + "/")) {
    scheduleFilesRefresh();
  }
}

let filesTimer = 0;
function scheduleFilesRefresh() {
  clearTimeout(filesTimer);
  filesTimer = setTimeout(loadFiles, 180);
}

/* ---------- fichiers ---------- */

async function loadFiles() {
  const requested = state.path;
  try {
    const data = await api("/api/files?path=" + encodeURIComponent(requested));
    if (requested !== state.path) return;
    state.entries = Array.isArray(data.entries) ? data.entries : [];
    renderCrumbs();
    renderFiles();
  } catch (e) {
    toast("Dossier indisponible : " + e.message, { error: true });
  }
}

function openDir(path) {
  state.path = path || "";
  state.query = "";
  $("#file-search").value = "";
  loadFiles();
}

function renderCrumbs() {
  const host = $("#crumbs");
  host.innerHTML = "";
  const rootName = state.info?.rootName || "Racine";
  const rootBtn = document.createElement("button");
  rootBtn.type = "button";
  rootBtn.className = "crumb" + (state.path ? "" : " current");
  rootBtn.textContent = rootName;
  rootBtn.title = rootName;
  rootBtn.addEventListener("click", () => openDir(""));
  host.appendChild(rootBtn);

  if (!state.path) return;
  const parts = state.path.split("/");
  let acc = "";
  parts.forEach((part) => {
    acc = acc ? acc + "/" + part : part;
    const sep = document.createElement("span");
    sep.className = "crumb-sep";
    sep.textContent = "/";
    host.appendChild(sep);
    const btn = document.createElement("button");
    btn.type = "button";
    btn.className = "crumb";
    btn.textContent = part;
    btn.title = part;
    btn.addEventListener("click", () => openDir(acc));
    host.appendChild(btn);
  });
  const last = host.lastElementChild;
  if (last) last.classList.add("current");
  host.scrollLeft = host.scrollWidth;
}

function emptyState(icon, title, subtitle) {
  const el = document.createElement("div");
  el.className = "empty";
  el.innerHTML = icon;
  const strong = document.createElement("strong");
  strong.textContent = title;
  const span = document.createElement("span");
  span.textContent = subtitle;
  el.append(strong, span);
  return el;
}

function renderFiles() {
  const host = $("#file-list");
  host.innerHTML = "";
  const q = state.query.trim().toLowerCase();
  const entries = q ? state.entries.filter((e) => e.name.toLowerCase().includes(q)) : state.entries;

  if (!entries.length) {
    host.appendChild(q
      ? emptyState(ICONS.file, "Aucun résultat", "Aucun fichier ne correspond à « " + state.query + " ».")
      : emptyState(ICONS.folder, "Dossier vide", state.info?.readOnly ? "Ce dossier est vide." : "Déposez des fichiers ici ou utilisez le bouton d'envoi."));
  } else {
    for (const entry of entries) host.appendChild(fileRow(entry));
  }
  const n = entries.length;
  $("#file-count").textContent = `${n} élément${n > 1 ? "s" : ""}${q ? " trouvé" + (n > 1 ? "s" : "") : ""}`;
}

function fileRow(entry) {
  const row = document.createElement("div");
  row.className = "file-row";
  row.setAttribute("role", "listitem");
  row.tabIndex = 0;
  row.title = entry.name;

  const kind = fileKind(entry);
  const icon = document.createElement("span");
  icon.className = "file-icon " + kind;
  icon.innerHTML = iconHTML(kind);

  const main = document.createElement("div");
  main.className = "file-main";
  const name = document.createElement("div");
  name.className = "file-name";
  name.textContent = entry.name;
  const meta = document.createElement("div");
  meta.className = "file-meta";
  const size = document.createElement("span");
  size.textContent = entry.dir ? "dossier" : fmtSize(entry.size);
  const date = document.createElement("span");
  date.textContent = fmtDate(entry.modTime);
  meta.append(size, date);
  main.append(name, meta);

  const actions = document.createElement("div");
  actions.className = "file-actions";
  if (!entry.dir) {
    const dl = document.createElement("button");
    dl.type = "button";
    dl.className = "icon-btn";
    dl.title = "Télécharger";
    dl.setAttribute("aria-label", "Télécharger " + entry.name);
    dl.innerHTML = ICONS.download;
    dl.addEventListener("click", (e) => { e.stopPropagation(); triggerDownload(entry); });
    actions.appendChild(dl);
  }
  if (!state.info?.readOnly) {
    const del = document.createElement("button");
    del.type = "button";
    del.className = "icon-btn danger";
    del.title = "Mettre à la corbeille";
    del.setAttribute("aria-label", "Supprimer " + entry.name);
    del.innerHTML = ICONS.trash;
    del.addEventListener("click", async (e) => {
      e.stopPropagation();
      if (!confirm(`Déplacer « ${entry.name} » dans la corbeille ?`)) return;
      try {
        await api("/api/delete", { method: "POST", json: { path: entry.path } });
        toast(`« ${entry.name} » mis à la corbeille`);
        loadFiles();
      } catch (err) {
        toast(err.message, { error: true });
      }
    });
    actions.appendChild(del);
  }

  const activate = () => {
    if (entry.dir) openDir(entry.path);
    else openPreview(entry);
  };
  row.addEventListener("click", activate);
  row.addEventListener("keydown", (e) => {
    if (e.key === "Enter" || e.key === " ") { e.preventDefault(); activate(); }
  });

  row.append(icon, main, actions);
  return row;
}

/* ---------- envoi ---------- */

function uploadFiles(files, attach = false) {
  const list = Array.from(files || []);
  if (!list.length) return;
  if (state.info?.readOnly) {
    toast("Cet espace est en lecture seule.", { error: true });
    return;
  }
  const fd = new FormData();
  for (const f of list) fd.append("files", f, f.name);

  const status = $("#upload-status");
  const bar = $("#upload-bar");
  const label = $("#upload-text");
  status.hidden = false;
  bar.style.width = "0%";
  label.textContent = `Envoi de ${list.length} fichier${list.length > 1 ? "s" : ""}… 0 %`;

  const xhr = new XMLHttpRequest();
  xhr.open("POST", "/api/upload?path=" + encodeURIComponent(state.path));
  xhr.setRequestHeader("X-Braise-Peer", state.identity.id);
  xhr.upload.addEventListener("progress", (e) => {
    if (!e.lengthComputable) return;
    const pct = Math.round((e.loaded / e.total) * 100);
    bar.style.width = pct + "%";
    label.textContent = `Envoi… ${pct} %`;
  });
  xhr.addEventListener("load", () => {
    status.hidden = true;
    let data = null;
    try { data = JSON.parse(xhr.responseText); } catch { /* ignore */ }
    if (xhr.status >= 200 && xhr.status < 300) {
      const n = data?.files?.length || list.length;
      toast(`${n} fichier${n > 1 ? "s" : ""} déposé${n > 1 ? "s" : ""}`);
      if (attach && data?.files?.[0]) setAttachment(data.files[0]);
      loadFiles();
    } else {
      toast(data?.error || "Échec de l'envoi", { error: true });
    }
  });
  xhr.addEventListener("error", () => {
    status.hidden = true;
    toast("Échec de l'envoi : le serveur ne répond pas.", { error: true });
  });
  xhr.send(fd);
}

function setAttachment(file) {
  state.attachment = file;
  const chip = $("#attach-chip");
  chip.hidden = false;
  $("#attach-name").textContent = file.name;
  const kind = fileKind({ name: file.name, mime: file.mime, dir: false });
  $(".attach-icon").innerHTML = iconHTML(kind);
  updateSend();
}

function clearAttachment() {
  state.attachment = null;
  $("#attach-chip").hidden = true;
  $("#attach-name").textContent = "";
  updateSend();
}

/* ---------- composer ---------- */

function autoSize() {
  const input = $("#input");
  input.style.height = "auto";
  input.style.height = Math.min(input.scrollHeight, 150) + "px";
}

function updateSend() {
  const input = $("#input");
  const has = !!input.value.trim() || !!state.attachment;
  $("#send-btn").disabled = !has || state.sending;
}

async function sendMessage(e) {
  if (e) e.preventDefault();
  if (state.sending) return;
  const input = $("#input");
  const text = input.value.trim();
  const file = state.attachment;
  if (!text && !file) return;
  state.sending = true;
  updateSend();
  try {
    const payload = {
      authorId: state.identity.id,
      author: state.identity.name,
      color: state.identity.color,
      text,
      file: file ? { name: file.name, path: file.path, size: file.size, mime: file.mime } : null,
    };
    const data = await api("/api/messages", { method: "POST", json: payload });
    if (data?.message) ingestMessage(data.message, { eventId: data.eventId });
    input.value = "";
    autoSize();
    clearAttachment();
    scrollToBottom(true);
  } catch (err) {
    toast(err.message || "Message non envoyé", { error: true });
  } finally {
    state.sending = false;
    updateSend();
  }
}

/* ---------- notifications ---------- */

function updateNotifyButton() {
  const btn = $("#notify-btn");
  const granted = "Notification" in window && Notification.permission === "granted";
  const active = state.notify && granted;
  btn.classList.toggle("active", active);
  btn.title = active ? "Notifications activées" : "Activer les notifications";
  btn.setAttribute("aria-pressed", active ? "true" : "false");
}

async function toggleNotifications() {
  if (!("Notification" in window)) {
    toast("Ce navigateur ne gère pas les notifications système.");
    return;
  }
  if (!window.isSecureContext) {
    toast("Les notifications système demandent HTTPS ou localhost. Ici, Braise garde les alertes dans l'onglet, avec badge et son.");
    return;
  }
  if (Notification.permission === "denied") {
    toast("Les notifications sont bloquées dans les réglages du navigateur.");
    return;
  }
  if (Notification.permission === "default") {
    const permission = await Notification.requestPermission();
    if (permission !== "granted") {
      toast("Notifications refusées.");
      return;
    }
  }
  state.notify = !state.notify;
  storageSet("braise.notify", state.notify ? "1" : "0");
  updateNotifyButton();
  toast(state.notify ? "Notifications activées" : "Notifications désactivées");
}

function notifyMessage(m, { chatVisible, away }) {
  const preview = m.text ? m.text.slice(0, 160) : m.file ? "Pièce jointe : " + m.file.name : "Nouveau message";
  if (state.notify && away && "Notification" in window && Notification.permission === "granted") {
    try {
      const n = new Notification(`${m.author} · Braise`, {
        body: preview,
        tag: "braise-" + m.id,
        icon: "/favicon.svg",
        silent: false,
      });
      n.onclick = () => { window.focus(); n.close(); };
    } catch { /* certains mobiles refusent */ }
  }
  if (!chatVisible) toast(`${m.author} : ${preview}`);
  if (state.notify) {
    beep();
    try { navigator.vibrate?.(70); } catch { /* pas de vibreur */ }
  }
}

let audioCtx = null;
function beep() {
  try {
    const Ctx = window.AudioContext || window.webkitAudioContext;
    if (!Ctx) return;
    audioCtx ||= new Ctx();
    if (audioCtx.state === "suspended") audioCtx.resume();
    const now = audioCtx.currentTime;
    const osc = audioCtx.createOscillator();
    const gain = audioCtx.createGain();
    osc.type = "sine";
    osc.frequency.setValueAtTime(720, now);
    osc.frequency.exponentialRampToValueAtTime(980, now + 0.09);
    gain.gain.setValueAtTime(0.0001, now);
    gain.gain.exponentialRampToValueAtTime(0.055, now + 0.012);
    gain.gain.exponentialRampToValueAtTime(0.0001, now + 0.15);
    osc.connect(gain).connect(audioCtx.destination);
    osc.start(now);
    osc.stop(now + 0.16);
  } catch { /* audio bloqué */ }
}

/* ---------- aperçu ---------- */

function openPreview(entry) {
  const overlay = $("#preview");
  const body = $("#preview-body");
  overlay.hidden = false;
  $("#preview-name").textContent = entry.name || "Fichier";
  const meta = [];
  if (entry.size) meta.push(fmtSize(entry.size));
  if (entry.modTime) meta.push(fmtDate(entry.modTime));
  $("#preview-meta").textContent = meta.join(" · ");
  const dl = $("#preview-download");
  dl.href = downloadURL(entry.path);
  dl.setAttribute("download", entry.name || "");
  body.innerHTML = "";

  const mime = String(entry.mime || "");
  const kind = fileKind({ ...entry, dir: false });

  if (kind === "img") {
    const img = document.createElement("img");
    img.alt = entry.name || "";
    img.src = rawURL(entry.path);
    img.addEventListener("error", () => body.replaceChildren(previewFallback(entry, "Impossible d'afficher cette image.")));
    body.appendChild(img);
  } else if (kind === "vid") {
    const video = document.createElement("video");
    video.controls = true;
    video.autoplay = false;
    video.src = rawURL(entry.path);
    body.appendChild(video);
  } else if (kind === "aud") {
    const audio = document.createElement("audio");
    audio.controls = true;
    audio.autoplay = false;
    audio.src = rawURL(entry.path);
    body.appendChild(audio);
  } else if (mime === "application/pdf" || String(entry.name || "").toLowerCase().endsWith(".pdf")) {
    const iframe = document.createElement("iframe");
    iframe.src = rawURL(entry.path);
    iframe.title = entry.name || "PDF";
    body.appendChild(iframe);
  } else if (isTextEntry(entry)) {
    if (entry.size > 2 * 1024 * 1024) {
      body.appendChild(previewFallback(entry, "Fichier trop volumineux pour un aperçu texte (2 Mo max)."));
    } else {
      const pre = document.createElement("pre");
      pre.textContent = "Chargement…";
      body.appendChild(pre);
      fetch(rawURL(entry.path), { cache: "no-store" })
        .then((r) => { if (!r.ok) throw new Error("HTTP " + r.status); return r.text(); })
        .then((text) => { pre.textContent = text; })
        .catch((err) => { pre.textContent = "Aperçu impossible : " + err.message; });
    }
  } else {
    body.appendChild(previewFallback(entry, "Aucun aperçu disponible pour ce format."));
  }
  $("#preview-close").focus();
}

function previewFallback(entry, message) {
  const wrap = document.createElement("div");
  wrap.className = "preview-fallback";
  const kind = fileKind({ ...entry, dir: false });
  const icon = document.createElement("div");
  icon.className = "file-icon " + kind;
  icon.style.margin = "0 auto 14px";
  icon.style.width = "54px";
  icon.style.height = "54px";
  icon.innerHTML = iconHTML(kind);
  const p = document.createElement("p");
  p.textContent = message;
  const btn = document.createElement("button");
  btn.type = "button";
  btn.className = "btn primary";
  btn.textContent = "Télécharger le fichier";
  btn.addEventListener("click", () => triggerDownload(entry));
  wrap.append(icon, p, btn);
  return wrap;
}

function closePreview() {
  const overlay = $("#preview");
  overlay.hidden = true;
  $("#preview-body").innerHTML = "";
}

/* ---------- partage / QR ---------- */

function openShare() {
  const base = state.info?.urls?.[0] || location.origin;
  let u;
  try { u = new URL(base); } catch { u = new URL(location.origin); }
  const host = u.hostname;
  const port = u.port || (u.protocol === "https:" ? "443" : "80");
  const dav = `${u.protocol}//${u.host}/dav/`;

  $("#qr-img").src = "/api/qr?url=" + encodeURIComponent(base);
  $("#share-url").textContent = base;
  $("#mount-win").textContent = "net use Z: " + "\\\\" + host + "@" + port + "\\dav";
  $("#mount-mac").textContent = dav;
  $("#mount-linux").textContent = `gio mount "dav://${u.host}/dav"`;
  $("#share-dialog").showModal();
}

async function copyText(text) {
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text);
      toast("Copié dans le presse-papiers");
      return;
    }
  } catch { /* fallback */ }
  const ta = document.createElement("textarea");
  ta.value = text;
  ta.style.position = "fixed";
  ta.style.opacity = "0";
  document.body.appendChild(ta);
  ta.select();
  try {
    document.execCommand("copy");
    toast("Copié dans le presse-papiers");
  } catch {
    toast("Copie impossible", { error: true });
  }
  ta.remove();
}

/* ---------- vue mobile ---------- */

function setView(view) {
  state.view = view === "chat" ? "chat" : "files";
  const app = $("#app");
  app.dataset.view = state.view;
  $$(".mobile-tabs button").forEach((b) => {
    const on = b.dataset.view === state.view;
    b.classList.toggle("active", on);
    b.setAttribute("aria-selected", on ? "true" : "false");
  });
  if (state.view === "chat") {
    clearUnread();
    scrollToBottom(false);
  }
}

/* ---------- scroll ---------- */

function isAtBottom() {
  const el = $("#messages");
  return el.scrollHeight - el.scrollTop - el.clientHeight < 90;
}

function scrollToBottom(smooth) {
  const el = $("#messages");
  requestAnimationFrame(() => {
    el.scrollTo({ top: el.scrollHeight, behavior: smooth ? "smooth" : "auto" });
  });
  showJump(false);
}

function showJump(show) {
  $("#jump-btn").hidden = !show;
}

/* ---------- UI ---------- */

function bindUI() {
  $("#refresh-btn").addEventListener("click", loadFiles);
  $("#mkdir-btn").addEventListener("click", createFolder);
  $("#fab-upload").addEventListener("click", () => $("#file-input").click());

  $("#file-input").addEventListener("change", (e) => {
    const files = Array.from(e.target.files || []);
    e.target.value = "";
    uploadFiles(files, false);
  });
  $("#attach-input").addEventListener("change", (e) => {
    const files = Array.from(e.target.files || []);
    e.target.value = "";
    uploadFiles(files, true);
  });
  $("#attach-remove").addEventListener("click", clearAttachment);

  $("#file-search").addEventListener("input", (e) => {
    state.query = e.target.value;
    renderFiles();
  });

  $("#composer").addEventListener("submit", sendMessage);
  const input = $("#input");
  input.addEventListener("input", () => {
    autoSize();
    updateSend();
    if (input.value.trim()) sendTyping();
  });
  input.addEventListener("keydown", (e) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  });

  $("#jump-btn").addEventListener("click", () => {
    scrollToBottom(true);
    clearUnread();
  });

  $("#me-btn").addEventListener("click", async () => {
    const identity = await identityDialog(state.identity);
    state.identity = identity;
    saveIdentity();
    applyIdentity();
    sendPresence();
    toast("Profil mis à jour");
  });

  $("#notify-btn").addEventListener("click", toggleNotifications);
  $("#share-btn").addEventListener("click", openShare);
  $("#help-btn").addEventListener("click", () => { $("#help-dialog").showModal(); });

  $("#preview-close").addEventListener("click", closePreview);
  $("#preview").addEventListener("click", (e) => { if (e.target === $("#preview")) closePreview(); });

  $$(".mobile-tabs button").forEach((b) => b.addEventListener("click", () => setView(b.dataset.view)));

  $("#copy-url").addEventListener("click", () => copyText($("#share-url").textContent));
  $$("[data-copy]").forEach((b) => {
    b.addEventListener("click", () => {
      const el = document.getElementById(b.dataset.copy);
      if (el) copyText(el.textContent);
    });
  });
  $$("[data-close]").forEach((b) => b.addEventListener("click", () => b.closest("dialog")?.close()));

  for (const dlg of $$("dialog.dialog")) {
    dlg.addEventListener("click", (e) => {
      if (e.target !== dlg) return;
      const r = dlg.getBoundingClientRect();
      const outside = e.clientX < r.left || e.clientX > r.right || e.clientY < r.top || e.clientY > r.bottom;
      if (outside) dlg.close();
    });
  }
  $("#identity-dialog").addEventListener("cancel", (e) => e.preventDefault());

  $("#messages").addEventListener("scroll", () => {
    state.atBottom = isAtBottom();
    showJump(!state.atBottom);
    if (state.atBottom && (!isMobile() || state.view === "chat")) clearUnread();
  }, { passive: true });

  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape" && !$("#preview").hidden) closePreview();
  });

  document.addEventListener("paste", (e) => {
    if (state.info?.readOnly) return;
    const files = Array.from(e.clipboardData?.files || []);
    if (!files.length) return;
    e.preventDefault();
    uploadFiles(files, true);
  });

  let dragDepth = 0;
  const hasFiles = (e) => Array.from(e.dataTransfer?.types || []).includes("Files");
  window.addEventListener("dragenter", (e) => {
    if (!hasFiles(e)) return;
    e.preventDefault();
    dragDepth++;
    $("#drop-target").textContent = state.path ? "dans « " + state.path + " »" : "à la racine";
    $("#drop-overlay").hidden = false;
  });
  window.addEventListener("dragover", (e) => {
    if (!hasFiles(e)) return;
    e.preventDefault();
    e.dataTransfer.dropEffect = "copy";
  });
  window.addEventListener("dragleave", (e) => {
    if (!hasFiles(e)) return;
    dragDepth = Math.max(0, dragDepth - 1);
    if (dragDepth === 0) $("#drop-overlay").hidden = true;
  });
  window.addEventListener("drop", (e) => {
    if (!hasFiles(e) && !e.dataTransfer?.files?.length) return;
    e.preventDefault();
    dragDepth = 0;
    $("#drop-overlay").hidden = true;
    uploadFiles(e.dataTransfer.files, false);
  });

  window.addEventListener("focus", () => {
    if ((!isMobile() || state.view === "chat") && isAtBottom()) clearUnread();
  });

  setInterval(renderTyping, 1000);
  autoSize();
  updateSend();
}

async function createFolder() {
  if (state.info?.readOnly) return;
  const name = prompt("Nom du nouveau dossier :");
  if (!name) return;
  try {
    await api("/api/mkdir", { method: "POST", json: { path: state.path, name } });
    toast(`Dossier « ${name.trim()} » créé`);
    loadFiles();
  } catch (e) {
    toast(e.message, { error: true });
  }
}

/* ---------- go ---------- */

document.addEventListener("DOMContentLoaded", init);

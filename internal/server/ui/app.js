"use strict";

const $ = (id) => document.getElementById(id);
const token = new URLSearchParams(location.search).get("token") || "";
const withToken = (p) => (token ? p + (p.includes("?") ? "&" : "?") + "token=" + encodeURIComponent(token) : p);
const api = async (p) => {
  const res = await fetch(withToken(p));
  const body = await res.json();
  if (!res.ok) throw new Error(body.error || res.statusText);
  return body;
};
const esc = (s) => String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
const svgNS = "http://www.w3.org/2000/svg";
const el = (tag, attrs = {}) => { const n = document.createElementNS(svgNS, tag); for (const [k, v] of Object.entries(attrs)) n.setAttribute(k, v); return n; };
const LOCALE = "en-GB";
const clock = (iso) => new Date(iso).toLocaleTimeString(LOCALE, { hour: "2-digit", minute: "2-digit" });
const when = (iso) => new Date(iso).toLocaleString(LOCALE, { day: "numeric", month: "short", hour: "2-digit", minute: "2-digit" });
const plural = (n, one, many) => n.toLocaleString(LOCALE) + " " + (n === 1 ? one : many);
const reduced = matchMedia("(prefers-reduced-motion: reduce)").matches;

let summary = null;
let rulesBySeverity = {};

async function refresh() {
  try {
    summary = await api("/api/v1/summary");
  } catch (err) {
    $("lede-sub").textContent = "Could not read the record: " + err.message;
    return;
  }
  route();
}

function show(page) {
  document.querySelectorAll(".page").forEach((p) => { p.hidden = p.id !== "page-" + page; });
  const nav = page === "source" ? "sources" : page;
  document.querySelectorAll("nav a").forEach((a) => a.classList.toggle("on", a.dataset.page === nav));
}

function route() {
  const [, page = "overview", ...rest] = location.hash.split("/");
  const pages = ["overview", "live", "sources", "source", "credentials", "paths", "behaviours", "sensors", "about"];
  const target = pages.includes(page) ? page : "overview";
  show(target);
  if (!summary && target !== "about") return;
  const render = { overview: renderOverview, live: renderLive, sources: renderSources, source: () => renderSource(decodeURIComponent(rest.join("/"))),
    credentials: renderCredentials, paths: renderPaths, behaviours: renderBehaviours, sensors: renderSensors }[target];
  if (render) render();
}

function sevOf(tags) {
  return Math.max(1, ...(tags || []).map((t) => rulesBySeverity[t] || 1));
}

function tagChips(tags) {
  return `<span class="tags">${(tags || []).map((t) => `<span class="tag s${rulesBySeverity[t] || 1}">${esc(t)}</span>`).join("")}</span>`;
}

/* ---------- overview ---------- */

function renderOverview() {
  const s = summary;
  const logins = s.sources.reduce((n, x) => n + x.logins, 0);
  const alerts = s.by_severity["3"] || 0;
  const busiest = s.hourly.indexOf(Math.max(...s.hourly));
  const busiestHour = (new Date().getHours() - (23 - busiest) + 24) % 24;
  $("h-overview").textContent = s.events
    ? `${plural(s.events, "knock", "knocks")} from ${plural(s.sources.length, "source", "sources")}`
    : "Nothing has reached the sensors yet";
  $("lede-sub").textContent = s.events
    ? `${plural(logins, "login attempt", "login attempts")} and ${plural(alerts, "alert", "alerts")} since this time yesterday. The busiest hour started at ${String(busiestHour).padStart(2, "0")}:00.`
    : "Sensors are listening. Events will appear here as soon as something connects.";
  drawDial(s.hourly);
  rankList($("top-sources"), s.sources.slice(0, 6).map((x) => ({ value: x.remote, n: x.events, href: "#/source/" + encodeURIComponent(x.remote) })), false, true);
  rankList($("top-pairs"), s.pairs.slice(0, 6));
  rankList($("top-tags"), s.tags.slice(0, 6));
  loadFeed();
}

function drawDial(hourly) {
  const svg = $("dial");
  svg.replaceChildren();
  const cx = 210, cy = 210, inner = 74, span = 112;
  const max = Math.max(1, ...hourly);
  const nowHour = new Date().getHours();
  const total = hourly.reduce((a, b) => a + b, 0);
  svg.append(el("circle", { cx, cy, r: inner + span + 6, fill: "none", stroke: "#3d3860", "stroke-width": 1 }));
  svg.append(el("circle", { cx, cy, r: inner - 8, fill: "#24213a", stroke: "#3d3860" }));
  const wedge = (h, r1, r2) => {
    const a0 = ((h / 24) * 360 - 90 + 1.2) * Math.PI / 180, a1 = (((h + 1) / 24) * 360 - 90 - 1.2) * Math.PI / 180;
    const p = (r, a) => `${cx + r * Math.cos(a)} ${cy + r * Math.sin(a)}`;
    return `M ${p(r1, a0)} L ${p(r2, a0)} A ${r2} ${r2} 0 0 1 ${p(r2, a1)} L ${p(r1, a1)} A ${r1} ${r1} 0 0 0 ${p(r1, a0)} Z`;
  };
  hourly.forEach((n, i) => {
    const hour = (nowHour - (23 - i) + 24) % 24;
    svg.append(el("path", { d: wedge(hour, inner, inner + span), fill: "rgba(233,228,245,.04)" }));
    if (n > 0) {
      const ratio = n / max;
      const fill = ratio > 0.66 ? "#ff6b6b" : ratio > 0.25 ? "#ffb547" : "#6fd3c7";
      const p = el("path", { d: wedge(hour, inner, inner + 8 + ratio * (span - 8)), fill, opacity: 0.35 + ratio * 0.65 });
      const tip = el("title"); tip.textContent = `${String(hour).padStart(2, "0")}:00, ${plural(n, "event", "events")}`;
      p.append(tip); svg.append(p);
    }
    if (hour === nowHour) svg.append(el("path", { d: wedge(hour, inner - 2, inner + span + 2), fill: "none", stroke: "#e9e4f5", "stroke-width": 1.5 }));
  });
  for (let h = 0; h < 24; h += 3) {
    const a = ((h / 24) * 360 - 90) * Math.PI / 180, r = inner + span + 22;
    const t = el("text", { x: cx + r * Math.cos(a), y: cy + r * Math.sin(a) + 4, "text-anchor": "middle", "font-size": 13, fill: "#a9a2c4", "font-family": "DIN Alternate, Avenir Next Condensed, sans-serif" });
    t.textContent = String(h).padStart(2, "0"); svg.append(t);
  }
  const big = el("text", { x: cx, y: cy + 8, "text-anchor": "middle", "font-size": 46, "font-weight": 600, fill: "#e9e4f5", "font-family": "DIN Alternate, Avenir Next Condensed, sans-serif" });
  big.textContent = total.toLocaleString(LOCALE);
  const small = el("text", { x: cx, y: cy + 30, "text-anchor": "middle", "font-size": 13, fill: "#a9a2c4" });
  small.textContent = "in 24 hours";
  svg.append(big, small);
  $("dial-caption").textContent = "Each slice is an hour of the day, reaching further out the busier it was. The outlined slice is the current hour.";
}

function rankList(ol, rows, bars = false, link = false) {
  const max = Math.max(1, ...rows.map((r) => r.n));
  ol.innerHTML = rows.length ? rows.map((r) => {
    const label = link && r.href ? `<a class="v" href="${r.href}">${esc(r.value || "(empty)")}</a>` : `<span class="v">${esc(r.value || "(empty)")}</span>`;
    return `<li>${label}<span class="n">${r.n.toLocaleString(LOCALE)}</span>${bars ? `<span class="bar"><i style="width:${(r.n / max) * 100}%"></i></span>` : ""}</li>`;
  }).join("") : `<li class="empty">Nothing recorded yet.</li>`;
}

function feedItem(e, fresh) {
  const li = document.createElement("li");
  li.className = "sev-" + e.severity + (fresh ? " fresh" : "");
  li.innerHTML = `<time datetime="${e.time}">${clock(e.time)}</time><span class="sens">${esc(e.sensor)}</span>` +
    `<span class="what">${esc(e.summary)}<span class="from">${esc(e.remote)}</span></span>`;
  return li;
}

async function loadFeed() {
  const events = await api("/api/v1/events?limit=14");
  $("feed").replaceChildren(...events.map((e) => feedItem(e, false)));
}

/* pages filled in later */
let liveFilter = { sev: 0, sensor: "" };

function logRow(e, fresh) {
  const tr = document.createElement("tr");
  tr.className = "go" + (fresh ? " fresh" : "");
  tr.dataset.remote = e.remote;
  tr.innerHTML = `<td class="t">${when(e.time)}</td><td><span class="sens sev-${e.severity}">${esc(e.sensor)}</span></td>` +
    `<td class="ip">${esc(e.remote)}</td><td>${esc(e.summary)}</td><td>${tagChips(e.tags)}</td>`;
  return tr;
}

function matchesLive(e) {
  return e.severity >= liveFilter.sev && (!liveFilter.sensor || e.sensor === liveFilter.sensor);
}

async function renderLive() {
  const q = `/api/v1/events?limit=200&severity=${liveFilter.sev}` + (liveFilter.sensor ? "&sensor=" + liveFilter.sensor : "");
  const events = await api(q);
  const body = $("log");
  body.replaceChildren(...events.map((e) => logRow(e, false)));
  if (!events.length) body.innerHTML = `<tr><td colspan="5" class="empty">No events match these filters yet.</td></tr>`;
}

// One stream feeds every page that shows live activity.
let staleTimer = null;
function onLive(e) {
  if (!$("page-overview").hidden) {
    const feed = $("feed");
    feed.prepend(feedItem(e, !reduced));
    while (feed.children.length > 14) feed.lastChild.remove();
  }
  if (!$("page-live").hidden && matchesLive(e)) {
    const body = $("log");
    if (body.querySelector(".empty")) body.replaceChildren();
    body.prepend(logRow(e, !reduced));
    while (body.children.length > 200) body.lastChild.remove();
  }
  const door = document.querySelector(`.door[data-sensor="${e.sensor}"]`);
  if (door && !reduced) { door.classList.remove("knock"); void door.offsetWidth; door.classList.add("knock"); }
  clearTimeout(staleTimer);
  staleTimer = setTimeout(async () => {
    summary = await api("/api/v1/summary").catch(() => summary);
    if (!$("page-overview").hidden) renderOverviewNumbers();
  }, 4000);
}

function renderOverviewNumbers() {
  const feed = $("feed").cloneNode(true);
  renderOverview();
  $("feed").replaceWith(feed);
}

function openStream() {
  const es = new EventSource(withToken("/api/v1/stream"));
  es.onmessage = (m) => { try { onLive(JSON.parse(m.data)); } catch { /* ignore a malformed frame */ } };
  es.onerror = () => { $("watch").classList.remove("on"); };
  es.onopen = () => { $("watch").classList.add("on"); };
}
function topTags(tagCounts, n) {
  return Object.entries(tagCounts || {}).sort((a, b) => b[1] - a[1]).slice(0, n).map(([t]) => t);
}

function span(first, last) {
  const mins = Math.round((new Date(last) - new Date(first)) / 60000);
  if (mins < 1) return "a single moment";
  if (mins < 60) return plural(mins, "minute", "minutes");
  return plural(Math.round(mins / 60), "hour", "hours");
}

function renderSources() {
  const rows = summary.sources;
  $("sources").innerHTML = rows.length ? rows.map((x) => `
    <tr class="go" tabindex="0" data-remote="${esc(x.remote)}">
      <td class="ip">${esc(x.remote)}</td><td>${x.events}</td><td>${x.logins || ""}</td>
      <td>${x.sensors.map((n) => `<span class="sens">${esc(n)}</span>`).join(" ")}</td>
      <td class="t">${span(x.first, x.last)}</td><td>${tagChips(topTags(x.tags, 3))}</td>
    </tr>`).join("") : `<tr><td colspan="6" class="empty">No source has reached a sensor yet.</td></tr>`;
}

async function renderSource(remote) {
  $("h-source").textContent = remote;
  $("source-sub").textContent = "Loading this source";
  let data;
  try {
    data = await api("/api/v1/sources/" + encodeURIComponent(remote));
  } catch (err) {
    $("source-sub").textContent = err.message + ". It may have aged out of memory.";
    ["source-tags", "source-pairs", "source-events"].forEach((id) => { $(id).innerHTML = ""; });
    $("strip").replaceChildren();
    return;
  }
  const src = data.source;
  $("source-sub").textContent = `${plural(src.events, "event", "events")} on ${src.sensors.join(", ")} between ${when(src.first)} and ${when(src.last)}.`;
  drawStrip(data.events);
  $("source-tags").innerHTML = Object.entries(src.tags).sort((a, b) => b[1] - a[1])
    .map(([t, n]) => `<li><span class="tag s${rulesBySeverity[t] || 1}">${esc(t)}</span><b>${n}</b></li>`).join("") || `<li>No recognised behaviour.</li>`;
  rankList($("source-pairs"), data.summary.pairs.slice(0, 8));
  $("source-events").innerHTML = data.events.slice(0, 120).map((e) =>
    `<tr><td class="t">${when(e.time)}</td><td><span class="sens sev-${e.severity}">${esc(e.sensor)}</span></td><td>${esc(e.summary)}</td><td>${tagChips(e.tags)}</td></tr>`).join("");
}

function drawStrip(events) {
  const svg = $("strip");
  svg.replaceChildren();
  const W = svg.clientWidth || 900, H = 90, pad = 10, mid = 42;
  svg.setAttribute("viewBox", `0 0 ${W} ${H}`);
  const end = Date.now(), start = end - 24 * 3600e3;
  const x = (t) => pad + ((new Date(t) - start) / (end - start)) * (W - pad * 2);
  svg.append(el("line", { x1: pad, x2: W - pad, y1: mid, y2: mid, stroke: "#4a4468", "stroke-width": 2 }));
  for (let h = 0; h <= 24; h += 6) {
    const t = new Date(start + h * 3600e3);
    const tx = pad + (h / 24) * (W - pad * 2);
    svg.append(el("line", { x1: tx, x2: tx, y1: mid + 14, y2: mid + 20, stroke: "#4a4468" }));
    const label = el("text", { x: tx, y: H - 6, "text-anchor": h === 0 ? "start" : h === 24 ? "end" : "middle", "font-size": 11, fill: "#a9a2c4" });
    label.textContent = h === 24 ? "now" : clock(t.toISOString());
    svg.append(label);
  }
  const color = { 1: "#6fd3c7", 2: "#ffb547", 3: "#ff6b6b" };
  for (const e of events) {
    if (new Date(e.time) < start) continue;
    const h = 8 + e.severity * 7;
    svg.append(el("line", { x1: x(e.time), x2: x(e.time), y1: mid - h, y2: mid + 4, stroke: color[e.severity] || color[1], "stroke-width": 2, "stroke-linecap": "round", opacity: 0.85 }));
  }
}
function renderCredentials() {
  rankList($("pairs"), summary.pairs, true);
  rankList($("users"), summary.usernames, true);
  rankList($("passwords"), summary.passwords, true);
}

function renderPaths() {
  rankList($("paths"), summary.paths, true);
  rankList($("clients"), summary.clients, true);
}

function renderBehaviours() {
  const counts = Object.fromEntries(summary.tags.map((t) => [t.value, t.n]));
  $("rules").innerHTML = (window.lureRules || []).map((r) => `
    <article class="rule s${r.severity}"><h3>${esc(r.tag)}</h3><p>${esc(r.description)}</p>
    <span class="count" title="matches today">${(counts[r.tag] || 0).toLocaleString(LOCALE)}</span></article>`).join("");
}

const sensorWords = {
  http: "A web server with a stock admin login page. Records every request and any login form posted to it.",
  ssh: "An OpenSSH version banner. Records the software the client announces, then hangs up.",
  telnet: "A device login prompt. Records each username and password tried, and refuses them all.",
  raw: "A port dressed as another service. Records the first bytes a client sends.",
};

function renderSensors() {
  const listeners = (window.lureStatus && window.lureStatus.listeners) || [];
  const counts = summary.by_sensor || {};
  const names = listeners.length ? listeners.map((l) => ({ name: l.spec.sensor, addr: l.addr, label: l.spec.label }))
    : Object.keys(counts).map((n) => ({ name: n, addr: "not running here" }));
  $("doors").innerHTML = names.length ? names.map((d) => `
    <article class="door" data-sensor="${esc(d.name)}">
      <h3>${esc(d.name)}${d.label ? ` <span class="lbl">as ${esc(d.label)}</span>` : ""}</h3>
      <span class="addr">${esc(d.addr)}</span>
      <p class="lbl">${esc(sensorWords[d.name] || "")}</p>
      <span class="big">${(counts[d.name] || 0).toLocaleString(LOCALE)}</span><span class="lbl">events in 24 hours</span>
    </article>`).join("") : `<p class="empty">No sensors are running.</p>`;
}

async function init() {
  window.addEventListener("hashchange", route);
  window.addEventListener("resize", () => { if (!$("page-source").hidden) route(); });
  document.querySelectorAll("#sev-seg button").forEach((b) => b.addEventListener("click", () => {
    liveFilter.sev = Number(b.dataset.sev);
    document.querySelectorAll("#sev-seg button").forEach((x) => x.classList.toggle("on", x === b));
    renderLive();
  }));
  $("sensor-filter").addEventListener("change", (e) => { liveFilter.sensor = e.target.value; renderLive(); });
  const openSource = (ev) => {
    const tr = ev.target.closest("tr[data-remote]");
    if (tr && (ev.type === "click" || ev.key === "Enter")) location.hash = "#/source/" + encodeURIComponent(tr.dataset.remote);
  };
  for (const id of ["log", "sources"]) { $(id).addEventListener("click", openSource); $(id).addEventListener("keydown", openSource); }
  try {
    const [st, rules] = await Promise.all([api("/api/v1/status"), api("/api/v1/rules")]);
    rules.forEach((r) => { rulesBySeverity[r.tag] = r.severity; });
    window.lureRules = rules;
    window.lureStatus = st;
    $("build").textContent = `Lure ${st.version} on ${st.platform}`;
    const n = (st.listeners || []).length;
    $("watch").classList.add("on");
    $("watch").querySelector("span").textContent = n ? `${plural(n, "sensor", "sensors")} on watch` : "Panel only, no sensors";
  } catch (err) {
    $("watch").querySelector("span").textContent = "Sensors unreachable";
  }
  await refresh();
  openStream();
}

init();

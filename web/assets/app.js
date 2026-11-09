// WaveTrack dashboard. Every value from the server reaches the DOM through textContent.

const ICONS = "/assets/icons.svg";
const view = document.getElementById("view");
const dialog = document.getElementById("dialog");
let refreshTimer = null;
let lastStatus = null;

// MARK: tiny DOM helpers

function h(tag, props = {}, ...children) {
  const el = document.createElement(tag);
  for (const [key, value] of Object.entries(props || {})) {
    if (value === null || value === undefined || value === false) continue;
    if (key === "class") el.className = value;
    else if (key === "text") el.textContent = value;
    else if (key.startsWith("on")) el.addEventListener(key.slice(2), value);
    else if (key === "vars") for (const [k, v] of Object.entries(value)) el.style.setProperty(k, v);
    else el.setAttribute(key, value === true ? "" : value);
  }
  append(el, children);
  return el;
}

function append(el, children) {
  for (const child of children.flat(Infinity)) {
    if (child === null || child === undefined || child === false) continue;
    el.append(child instanceof Node ? child : document.createTextNode(String(child)));
  }
}

function icon(name, cls = "icon") {
  const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.setAttribute("class", cls);
  svg.setAttribute("aria-hidden", "true");
  const use = document.createElementNS("http://www.w3.org/2000/svg", "use");
  use.setAttribute("href", `${ICONS}#${name}`);
  svg.append(use);
  return svg;
}

function toast(message) {
  const el = document.getElementById("toast");
  el.textContent = message;
  el.classList.add("show");
  clearTimeout(toast.timer);
  toast.timer = setTimeout(() => el.classList.remove("show"), 2600);
}

// MARK: formatting

const timeFmt = new Intl.DateTimeFormat(undefined, { hour: "2-digit", minute: "2-digit" });
const secondsFmt = new Intl.DateTimeFormat(undefined, { hour: "2-digit", minute: "2-digit", second: "2-digit" });
const stampFmt = new Intl.DateTimeFormat(undefined, { day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit", second: "2-digit" });
const dayFmt = new Intl.DateTimeFormat(undefined, { weekday: "long", day: "numeric", month: "long" });
const shortDay = new Intl.DateTimeFormat(undefined, { weekday: "short" });

const at = (iso) => timeFmt.format(new Date(iso));

function duration(seconds) {
  const minutes = Math.round(seconds / 60);
  if (minutes < 1) return "<1m";
  if (minutes < 60) return `${minutes}m`;
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  return m ? `${h}h ${String(m).padStart(2, "0")}m` : `${h}h`;
}

function ago(iso, now) {
  const minutes = Math.round((now - new Date(iso)) / 60000);
  if (minutes < 1) return "just now";
  if (minutes < 60) return `${minutes} min ago`;
  return `at ${at(iso)}`;
}

function initials(name) {
  const parts = name.trim().split(/\s+/).filter((w) => /^\p{L}/u.test(w));
  return ((parts[0]?.[0] || "") + (parts.length > 1 ? parts.at(-1)[0] : "")).toUpperCase();
}

// Avatar hues: teal, blue, steel, green, cyan, amber, rust and olive. A name always gets
// the same one; purples and pinks are left out on purpose.
const HUES = [174, 205, 214, 148, 190, 34, 16, 82];

function hue(name) {
  let n = 0;
  for (const c of name) n = (n * 31 + c.codePointAt(0)) >>> 0;
  return HUES[n % HUES.length];
}

function avatar(name, { present = null, small = false } = {}) {
  return h("span", { class: `avatar${small ? " small" : ""}${present === false ? " away" : ""}`, vars: { "--hue": hue(name) }, "aria-hidden": "true" },
    initials(name), present ? h("span", { class: "dot" }) : null);
}

const KIND_ICON = { phone: "phone", computer: "computer", network: "network", iot: "iot" };

// MARK: API

class Unauthorized extends Error {}

async function api(path, { method = "GET", body } = {}) {
  const res = await fetch(path, {
    method,
    headers: body === undefined ? {} : { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
    credentials: "same-origin",
  });
  if (res.status === 401) throw new Unauthorized();
  if (res.status === 204) return null;
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || `Request failed (${res.status})`);
  return data;
}

// MARK: theme

const THEMES = ["light", "dark", "classic", "code", "contrast"];

function storedTheme() {
  try { return localStorage.getItem("wavetrack-theme"); } catch { return null; }
}

// An empty theme follows the system's light or dark appearance.
function applyTheme(theme) {
  const valid = THEMES.includes(theme) ? theme : "";
  if (valid) document.documentElement.dataset.theme = valid;
  else delete document.documentElement.dataset.theme;
  document.getElementById("theme").value = valid;
}

document.getElementById("theme").addEventListener("change", (event) => {
  try { localStorage.setItem("wavetrack-theme", event.target.value); } catch {}
  applyTheme(event.target.value);
});

// MARK: sign in

async function signIn(token) {
  await api("/api/login", { method: "POST", body: { token } });
}

document.getElementById("signin-form").addEventListener("submit", async (event) => {
  event.preventDefault();
  const error = document.getElementById("signin-error");
  error.hidden = true;
  try {
    await signIn(document.getElementById("token").value);
    start();
  } catch (e) {
    error.textContent = e instanceof Unauthorized ? "That access token isn’t right." : e.message;
    error.hidden = false;
  }
});

document.getElementById("signout").addEventListener("click", async () => {
  await api("/api/logout", { method: "POST", body: {} }).catch(() => {});
  showSignIn();
});

function showSignIn() {
  clearInterval(refreshTimer);
  clearInterval(statusTimer);
  document.getElementById("app").hidden = true;
  document.getElementById("signin").hidden = false;
  document.getElementById("token").focus();
}

// MARK: router

const routes = { now: renderNow, timeline: renderTimeline, history: renderHistory, people: renderPeople, nearby: renderNearby, audit: renderAudit };

function currentRoute() {
  const [name, ...rest] = location.hash.replace(/^#\/?/, "").split("/");
  return { name: routes[name] ? name : "now", arg: rest.join("/") };
}

async function route() {
  clearInterval(refreshTimer);
  const { name, arg } = currentRoute();
  for (const link of document.querySelectorAll(".nav a")) {
    link.toggleAttribute("aria-current", link.getAttribute("href") === `#/${name}`);
    if (link.hasAttribute("aria-current")) link.setAttribute("aria-current", "page");
  }
  try {
    await routes[name](arg);
  } catch (e) {
    if (e instanceof Unauthorized) return showSignIn();
    view.replaceChildren(h("div", { class: "card empty" }, h("h3", { text: "Something went wrong" }), h("p", { text: e.message })));
  }
}

function every(seconds, fn) {
  clearInterval(refreshTimer);
  refreshTimer = setInterval(() => fn().catch((e) => { if (e instanceof Unauthorized) showSignIn(); }), seconds * 1000);
}

// The status bar: what WaveTrack is capturing, who's in, and when the page last refreshed.
function updateStatus(data) {
  lastStatus = data;
  const capture = document.getElementById("sb-capture");
  capture.querySelector(".label").textContent = data.capture.kind === "demo"
    ? "Demo office · simulated capture"
    : `Capturing on ${data.capture.interface} · ${data.capture.mode} mode`;
  capture.querySelector(".pulse").classList.remove("off");
  if (data.present !== undefined) {
    document.getElementById("sb-presence").textContent = `${data.present} of ${data.people} in`;
  }
  document.getElementById("sb-version").textContent = `WaveTrack ${data.version}`;
  if (data.now) document.getElementById("sb-updated").textContent = `Updated ${secondsFmt.format(new Date(data.now))}`;
}

function statusLost() {
  const capture = document.getElementById("sb-capture");
  capture.querySelector(".label").textContent = "Disconnected from WaveTrack";
  capture.querySelector(".pulse").classList.add("off");
}

let statusTimer = null;
function watchStatus() {
  clearInterval(statusTimer);
  const poll = () => api("/api/status").then(updateStatus).catch((e) => { if (!(e instanceof Unauthorized)) statusLost(); });
  poll();
  statusTimer = setInterval(poll, 30000);
}

function pageHead(title, subtitle, ...actions) {
  return h("div", { class: "page-head" },
    h("div", {}, h("h1", { text: title }), subtitle ? h("p", { text: subtitle }) : null),
    actions.length ? h("div", { class: "actions" }, actions) : null);
}

// MARK: Now

async function renderNow() {
  const draw = async () => {
    const data = await api("/api/now");
    updateStatus({ ...data, present: data.people.filter((p) => p.present).length, people: data.people.length });
    const now = new Date(data.now);
    const here = data.people.filter((p) => p.present);
    const away = data.people.filter((p) => !p.present);
    const arrivedToday = data.people.filter((p) => p.first_today);
    const firsts = arrivedToday.map((p) => new Date(p.first_today)).map((d) => d.getHours() * 60 + d.getMinutes());
    const avgArrival = firsts.length ? firsts.reduce((a, b) => a + b, 0) / firsts.length : null;
    const longest = here.reduce((best, p) => (!best || p.today_seconds > best.today_seconds ? p : best), null);

    const teams = {};
    for (const p of here) teams[p.team || "No team"] = (teams[p.team || "No team"] || 0) + 1;

    view.replaceChildren(
      pageHead(dayFmt.format(now), `Updated ${at(data.now)} · refreshes every 10 seconds`),
      hero(here.length, data.people.length, teams, data.people.length === 0),
      h("div", { class: "stats" },
        stat("Arrived today", arrivedToday.length, `of ${data.people.length} people`),
        stat("Average arrival", avgArrival === null ? "—" : `${String(Math.floor(avgArrival / 60)).padStart(2, "0")}:${String(Math.round(avgArrival % 60)).padStart(2, "0")}`,
          arrivedToday.length ? "first sighting of the day" : "nobody yet"),
        stat("Longest today", longest ? duration(longest.today_seconds) : "—", longest ? longest.name : "nobody here"),
        stat("Unregistered nearby", data.nearby, "devices seen recently", "#/nearby")),
      h("div", { class: "columns" },
        h("div", { class: "card" },
          h("div", { class: "card-head" }, h("h2", { text: `Here now · ${here.length}` })),
          h("div", { class: "card-body" }, here.length
            ? h("div", { class: "here-grid" }, here.sort((a, b) => a.name.localeCompare(b.name)).map((p) => personCard(p)))
            : emptyState("people", "Nobody’s here yet", "People appear as soon as one of their devices shows up on the network."))),
        h("div", { class: "card" },
          h("div", { class: "card-head" }, h("h2", { text: "Today" })),
          h("div", { class: "card-body" }, data.events.length
            ? h("ul", { class: "feed" }, data.events.slice(0, 12).map((e) =>
              h("li", {}, h("span", { class: `mark ${e.kind}` }), h("span", {}, h("strong", { text: e.name }), e.kind === "arrived" ? " arrived" : " left"),
                h("time", { datetime: e.at, text: at(e.at) }))))
            : h("p", { class: "faint", text: "Arrivals and departures since WaveTrack started appear here." }),
          away.length ? h("div", {},
            h("h2", { class: "faint", text: `Away · ${away.length}` }),
            h("div", { class: "away-list" }, away.map((p) => h("div", { class: "away-row" },
              avatar(p.name, { present: false, small: true }),
              h("span", { class: "name", text: p.name }),
              h("span", { class: "when", text: p.last_today ? `left ${at(p.last_today)}` : "not in today" }))))) : null))));
  };
  await draw();
  every(10, draw);
}

function hero(present, total, teams, empty) {
  const r = 84;
  const circumference = 2 * Math.PI * r;
  const fraction = total ? present / total : 0;
  const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.setAttribute("class", "ring");
  svg.setAttribute("viewBox", "0 0 200 200");
  svg.setAttribute("role", "img");
  svg.setAttribute("aria-label", `${present} of ${total} people are in`);
  const circle = (cls, radius, extra = {}) => {
    const c = document.createElementNS("http://www.w3.org/2000/svg", "circle");
    c.setAttribute("class", cls);
    c.setAttribute("cx", "100");
    c.setAttribute("cy", "100");
    c.setAttribute("r", radius);
    for (const [k, v] of Object.entries(extra)) c.setAttribute(k, v);
    return c;
  };
  svg.append(circle("wave", 97), circle("track", r),
    circle("fill", r, { "stroke-dasharray": circumference, "stroke-dashoffset": circumference * (1 - fraction), transform: "rotate(-90 100 100)" }));
  const text = (cls, y, value) => {
    const t = document.createElementNS("http://www.w3.org/2000/svg", "text");
    t.setAttribute("class", cls);
    t.setAttribute("x", "100");
    t.setAttribute("y", y);
    t.setAttribute("text-anchor", "middle");
    t.textContent = value;
    return t;
  };
  svg.append(text("", "112", String(present)), text("sub", "136", `OF ${total} IN`));

  const headline = empty ? "Add the people you want to see here."
    : present === 0 ? "The office is empty."
      : present === total ? "Everyone’s in."
        : `${present} ${present === 1 ? "person is" : "people are"} in the office.`;
  const sub = empty ? "Invite people with a QR code, or add their devices yourself from the People page."
    : "Presence comes from phones and laptops on the office Wi-Fi. Nobody is tracked outside it.";

  return h("section", { class: "card hero" }, svg,
    h("div", {}, h("h2", { text: headline }), h("p", { text: sub }),
      empty ? h("div", { class: "teams" }, h("a", { class: "button primary", href: "#/people" }, icon("plus"), "Add people"))
        : h("div", { class: "teams" }, Object.entries(teams).sort((a, b) => b[1] - a[1]).map(([team, n]) =>
          h("span", { class: "chip" }, h("strong", { text: String(n) }), team)))));
}

function stat(label, value, detail, href) {
  const inner = [h("div", { class: "label", text: label }), h("div", { class: "value num", text: String(value) }), h("div", { class: "detail", text: detail })];
  return href ? h("a", { class: "card stat", href }, inner) : h("div", { class: "card stat" }, inner);
}

function personCard(p) {
  const share = Math.min(1, p.today_seconds / (8 * 3600));
  return h("div", { class: "person-card" },
    avatar(p.name, { present: true }),
    h("div", {},
      h("div", { class: "name", text: p.name }),
      h("div", { class: "meta", text: `${p.team ? p.team + " · " : ""}since ${at(p.since)}` }),
      h("div", { class: "meta num", text: `${duration(p.today_seconds)} today` }),
      h("div", { class: "bar", title: "Share of an 8-hour day" }, h("span", { vars: { width: `${share * 100}%` } }))));
}

function emptyState(iconName, title, message) {
  return h("div", { class: "empty" }, icon(iconName), h("h3", { text: title }), h("p", { text: message }));
}

// MARK: Timeline

function isoDate(d) {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

async function renderTimeline(arg) {
  const draw = async () => {
    const data = await api(`/api/timeline${arg ? `?date=${encodeURIComponent(arg)}` : ""}`);
    const day = new Date(`${data.date}T00:00:00`);
    const now = new Date(data.now);
    const isToday = isoDate(now) === data.date;

    // Fit the scale to the day's activity, from 07:00 to 20:00 at least.
    let startHour = 7, endHour = 20;
    for (const p of data.people) for (const iv of p.intervals) {
      startHour = Math.min(startHour, new Date(iv.start).getHours());
      endHour = Math.max(endHour, Math.ceil((new Date(iv.end) - day) / 3600000));
    }
    endHour = Math.min(24, endHour);
    const span = (endHour - startHour) * 3600000;
    const pos = (date) => Math.max(0, Math.min(1, (new Date(date) - day - startHour * 3600000) / span));

    const ticks = [];
    for (let hour = startHour; hour <= endHour; hour += (endHour - startHour > 14 ? 2 : 1)) ticks.push(hour);
    const prev = new Date(day); prev.setDate(prev.getDate() - 1);
    const next = new Date(day); next.setDate(next.getDate() + 1);

    const groups = {};
    for (const p of data.people) (groups[p.team || "No team"] ||= []).push(p);

    const track = (p) => {
      const el = h("div", { class: "tl-track" },
        ticks.map((t) => h("span", { class: "grid", vars: { left: `${((t - startHour) / (endHour - startHour)) * 100}%` } })),
        p.intervals.map((iv) => h("span", {
          class: `tl-bar${iv.ongoing ? " ongoing" : ""}${p.archived ? " archived" : ""}`,
          title: `${p.name}: ${at(iv.start)} – ${iv.ongoing ? "now" : at(iv.end)}`,
          vars: { left: `${pos(iv.start) * 100}%`, width: `${(pos(iv.end) - pos(iv.start)) * 100}%` },
        })),
        isToday ? h("span", { class: "tl-now", vars: { left: `${pos(now) * 100}%` }, title: `Now, ${at(now)}` }) : null);
      return el;
    };
    const total = (p) => p.intervals.reduce((sum, iv) => sum + (new Date(iv.end) - new Date(iv.start)) / 1000, 0);

    view.replaceChildren(
      pageHead("Timeline", "Each bar is a stay, from the first sighting to the last.",
        h("div", { class: "date-nav" },
          h("a", { class: "icon-button", href: `#/timeline/${isoDate(prev)}`, "aria-label": "Previous day" }, icon("left")),
          h("span", { class: "label", text: isToday ? `Today, ${dayFmt.format(day)}` : dayFmt.format(day) }),
          h("a", { class: "icon-button", href: `#/timeline/${isoDate(next)}`, "aria-label": "Next day", hidden: isToday || null }, icon("right")),
          isToday ? null : h("a", { class: "button small", href: "#/timeline" }, "Today"))),
      h("div", { class: "card timeline" },
        h("div", { class: "tl-axis" }, h("span"),
          h("div", { class: "scale" }, ticks.map((t) => h("span", { class: "tick", vars: { left: `${((t - startHour) / (endHour - startHour)) * 100}%` }, text: `${String(t % 24).padStart(2, "0")}:00` })),
            isToday ? h("span", { class: "tl-now-label", vars: { left: `${pos(now) * 100}%` }, text: at(now) }) : null),
          h("span", { class: "tl-total faint", text: "Total" })),
        data.people.length === 0 ? emptyState("timeline", "No one to show", "Add people to see their days here.") : null,
        Object.keys(groups).sort().map((team) => [
          h("div", { class: "tl-group", text: team }),
          groups[team].map((p) => h("div", { class: "tl-row" },
            h("div", { class: "who" }, avatar(p.name, { small: true, present: p.intervals.length ? null : false }), h("span", { text: p.name })),
            track(p),
            h("span", { class: "tl-total", text: total(p) ? duration(total(p)) : "—" }))),
        ])));
  };
  await draw();
  if (!arg) every(30, draw);
}

// MARK: History

async function renderHistory(arg) {
  const days = [7, 14, 30].includes(Number(arg)) ? Number(arg) : 7;
  const data = await api(`/api/history?days=${days}`);
  const full = 8 * 3600;

  const cell = (d) => {
    if (!d.seconds) return h("div", { class: "cell", text: "·" });
    const level = Math.round(18 + 70 * Math.min(1, d.seconds / full));
    return h("div", {
      class: `cell on${level > 60 ? " strong" : ""}`,
      vars: { "--level": `${level}%` },
      title: `${d.date}: ${at(d.first)} – ${at(d.last)}`,
      text: duration(d.seconds),
    });
  };

  const header = data.dates.map((iso) => {
    const d = new Date(`${iso}T00:00:00`);
    const weekend = d.getDay() === 0 || d.getDay() === 6;
    return h("th", { class: weekend ? "weekend" : null }, shortDay.format(d), h("small", { text: String(d.getDate()) }));
  });

  const toggle = h("div", { class: "segmented", role: "group", "aria-label": "Range" },
    [7, 14, 30].map((n) => h("button", { type: "button", "aria-pressed": String(n === days), onclick: () => { location.hash = `#/history/${n}`; }, text: `${n} days` })));

  view.replaceChildren(
    pageHead("History", "Hours on site per day. A full cell is an 8-hour day.",
      toggle,
      h("a", { class: "button", href: `/api/export.csv?days=${days}`, download: "" }, icon("download"), "Export CSV")),
    h("div", { class: "card" },
      h("div", { class: "card-body" },
        data.people.length === 0 ? emptyState("history", "No history yet", "Visits appear here once people are registered and seen.") :
          h("div", { class: "scroll" },
            h("table", { class: "matrix" },
              h("thead", {}, h("tr", {}, h("th", { class: "who", text: "" }), header, h("th", { class: "total", text: "Total" }))),
              h("tbody", {}, data.people.map((p) => h("tr", {},
                h("td", { class: "who" }, h("div", {}, avatar(p.name, { small: true }), h("span", { text: p.name }), p.archived ? h("span", { class: "badge", text: "archived" }) : null)),
                p.days.map((d) => h("td", {}, cell(d))),
                h("td", { class: "total num", text: p.total_seconds ? duration(p.total_seconds) : "—" })))))),
        h("div", { class: "legend" }, "Less", [18, 40, 62, 88].map((l) => h("i", { vars: { "--level": `${l}%` } })), "More"))));
}

// MARK: People

async function renderPeople() {
  const { people } = await api("/api/people");
  view.replaceChildren(
    pageHead("People", "Everyone WaveTrack recognizes, and the devices that identify them.",
      h("button", { class: "button", type: "button", onclick: openInvite }, icon("qr"), "Invite with QR code"),
      h("button", { class: "button primary", type: "button", onclick: () => openPerson() }, icon("plus"), "Add person")),
    h("div", { class: "card" },
      people.length === 0
        ? emptyState("people", "No one yet", "Invite people with a QR code so they can register their own phone, or add them yourself.")
        : h("table", { class: "table" },
          h("thead", {}, h("tr", {}, h("th", { text: "Person" }), h("th", { text: "Devices" }), h("th", { text: "Added" }), h("th"))),
          h("tbody", {}, people.map((p) => h("tr", {},
            h("td", {}, h("div", { class: "who" }, avatar(p.name),
              h("div", {}, h("strong", { text: p.name }), h("small", { text: p.team || "No team" })))),
            h("td", {}, h("div", { class: "devices" },
              p.devices.map((d) => deviceChip(d)),
              h("button", { class: "button ghost small", type: "button", onclick: () => openDevice(p) }, icon("plus"), "Device"))),
            h("td", {}, h("span", { class: "faint", text: new Date(p.created_at).toLocaleDateString() }),
              p.consented_at ? [" ", h("span", { class: "badge consent", title: "Registered their own device and agreed to tracking" }, "self-registered")] : null),
            h("td", { class: "end" },
              h("button", { class: "icon-button", type: "button", "aria-label": `Edit ${p.name}`, onclick: () => openPerson(p) }, icon("edit")),
              h("button", { class: "icon-button", type: "button", "aria-label": `Remove ${p.name}`, onclick: () => openRemove(p) }, icon("trash")))))))));
}

function deviceChip(d) {
  return h("span", { class: "device", title: `${d.vendor || "Unknown maker"}${d.last_seen ? ` · last seen ${new Date(d.last_seen).toLocaleString()}` : ""}` },
    icon(KIND_ICON[d.kind] || "unknown"), d.label || d.vendor || "Device", h("span", { class: "mono", text: d.mac }),
    h("button", { type: "button", "aria-label": `Remove ${d.mac}`, onclick: () => removeDevice(d) }, icon("close")));
}

async function removeDevice(d) {
  if (!confirm(`Remove ${d.label || d.mac}? Its owner won’t be detected through it anymore.`)) return;
  await api(`/api/devices/${encodeURIComponent(d.mac)}`, { method: "DELETE" });
  toast("Device removed");
  route();
}

function openDialog(title, subtitle, body, footer) {
  dialog.replaceChildren(
    h("div", { class: "dialog-head" },
      h("div", {}, h("h2", { text: title }), subtitle ? h("p", { text: subtitle }) : null),
      h("button", { class: "icon-button", type: "button", "aria-label": "Close", onclick: () => dialog.close() }, icon("close"))),
    h("div", { class: "dialog-body" }, body),
    footer ? h("div", { class: "dialog-foot" }, footer) : null);
  dialog.showModal();
}

function errorBox() {
  return h("p", { class: "error", hidden: true });
}

function showError(box, e) {
  box.textContent = e.message;
  box.hidden = false;
}

function openPerson(person = null) {
  const name = h("input", { class: "input", id: "p-name", required: true, value: person?.name || "", autocomplete: "off" });
  const team = h("input", { class: "input", id: "p-team", value: person?.team || "", autocomplete: "off", placeholder: "Design, Engineering…" });
  const mac = person ? null : h("input", { class: "input mono", id: "p-mac", placeholder: "a4:83:e7:21:5c:10", autocomplete: "off" });
  const label = person ? null : h("input", { class: "input", id: "p-label", placeholder: "iPhone", autocomplete: "off" });
  const error = errorBox();

  const save = async () => {
    try {
      if (person) {
        await api(`/api/people/${person.id}`, { method: "PATCH", body: { name: name.value, team: team.value } });
      } else {
        const devices = mac.value.trim() ? [{ mac: mac.value, label: label.value }] : [];
        await api("/api/people", { method: "POST", body: { name: name.value, team: team.value, devices } });
      }
      dialog.close();
      toast(person ? "Saved" : `${name.value} added`);
      route();
    } catch (e) { showError(error, e); }
  };

  openDialog(person ? "Edit person" : "Add a person",
    person ? null : "Or send them a QR invite so they can register their own phone.",
    [
      h("div", { class: "field" }, h("label", { for: "p-name", text: "Name" }), name),
      h("div", { class: "field" }, h("label", { for: "p-team", text: "Team" }), team),
      person ? null : h("div", { class: "field" }, h("label", { for: "p-mac", text: "Device MAC address" }), mac,
        h("span", { class: "hint", text: "Optional. On iPhone: Settings › Wi-Fi › ⓘ › Wi-Fi Address. You can also pick it from Nearby." })),
      person ? null : h("div", { class: "field" }, h("label", { for: "p-label", text: "Device name" }), label),
      error,
    ],
    [h("button", { class: "button", type: "button", onclick: () => dialog.close() }, "Cancel"),
      h("button", { class: "button primary", type: "button", onclick: save }, person ? "Save" : "Add person")]);
  name.focus();
}

function openDevice(person) {
  const mac = h("input", { class: "input mono", id: "d-mac", placeholder: "a4:83:e7:21:5c:10", autocomplete: "off" });
  const label = h("input", { class: "input", id: "d-label", placeholder: "Laptop", autocomplete: "off" });
  const error = errorBox();
  const save = async () => {
    try {
      await api(`/api/people/${person.id}/devices`, { method: "POST", body: { mac: mac.value, label: label.value } });
      dialog.close();
      toast("Device added");
      route();
    } catch (e) { showError(error, e); }
  };
  openDialog(`Add a device for ${person.name}`, "Any of their devices being seen counts as them being here.", [
    h("div", { class: "field" }, h("label", { for: "d-mac", text: "MAC address" }), mac),
    h("div", { class: "field" }, h("label", { for: "d-label", text: "Device name" }), label),
    error,
  ], [h("button", { class: "button", type: "button", onclick: () => dialog.close() }, "Cancel"),
    h("button", { class: "button primary", type: "button", onclick: save }, "Add device")]);
  mac.focus();
}

function openRemove(person) {
  const error = errorBox();
  const option = (value, title, text, checked) => h("label", {},
    h("input", { type: "radio", name: "mode", value, checked }), h("div", {}, h("strong", { text: title }), h("span", { text })));
  const choices = h("div", { class: "choice" },
    option("archive", "Anonymize", "Replace their name and forget their devices, but keep past hours so totals stay accurate.", true),
    option("erase", "Erase everything", "Delete the person, their devices and every visit. Use this for data-deletion requests."));
  const remove = async () => {
    const mode = choices.querySelector("input:checked").value;
    try {
      await api(`/api/people/${person.id}?mode=${mode}`, { method: "DELETE" });
      dialog.close();
      toast(mode === "erase" ? `${person.name} erased` : `${person.name} anonymized`);
      route();
    } catch (e) { showError(error, e); }
  };
  openDialog(`Remove ${person.name}?`, "Neither option can be undone.", [choices, error],
    [h("button", { class: "button", type: "button", onclick: () => dialog.close() }, "Cancel"),
      h("button", { class: "button danger solid", type: "button", onclick: remove }, "Remove")]);
}

async function openInvite() {
  const invite = await api("/api/invites", { method: "POST", body: {} });
  const link = h("input", { class: "input mono", readonly: true, value: invite.url, "aria-label": "Invite link" });
  openDialog("Invite with a QR code", "Valid for one person, for 24 hours.", [
    h("div", { class: "qr" },
      h("img", { src: invite.qr, alt: "QR code for the invite link" }),
      h("div", { class: "link" }, link,
        h("button", { class: "button", type: "button", onclick: async () => { await navigator.clipboard.writeText(invite.url); toast("Link copied"); } }, icon("copy"), "Copy"))),
    h("ol", { class: "steps" },
      h("li", { text: "Connect the phone to the office Wi-Fi." }),
      h("li", { text: "Scan the code with the camera and open the link." }),
      h("li", { text: "They enter their name and agree. Their phone is recognized from then on." })),
  ], [h("button", { class: "button primary", type: "button", onclick: () => { dialog.close(); route(); } }, "Done")]);
}

// MARK: Nearby

async function renderNearby() {
  const draw = async () => {
    const [{ devices, now: serverNow }, { people }] = await Promise.all([api("/api/nearby"), api("/api/people")]);
    const now = new Date(serverNow);
    const bars = (signal) => {
      const level = !signal ? 0 : signal > -55 ? 4 : signal > -65 ? 3 : signal > -75 ? 2 : 1;
      return h("span", { class: "signal", title: signal ? `${signal} dBm` : "Signal not measured in managed mode" },
        [1, 2, 3, 4].map((i) => h("i", { class: i <= level ? "on" : null })));
    };
    view.replaceChildren(
      pageHead("Nearby", "Devices seen recently that don’t belong to anyone. They’re kept in memory only and never saved."),
      h("div", { class: "card" },
        devices.length === 0
          ? emptyState("nearby", "Nothing unregistered nearby", "Unknown devices appear here for a while after they’re seen.")
          : h("table", { class: "table" },
            h("thead", {}, h("tr", {}, h("th", { text: "Device" }), h("th", { text: "Signal" }), h("th", { text: "Seen" }), h("th"))),
            h("tbody", {}, devices.map((d) => h("tr", {},
              h("td", {}, h("div", { class: "who" },
                h("span", { class: "kind" }, icon(KIND_ICON[d.kind] || "unknown")),
                h("div", {}, h("strong", {}, d.private ? "Unknown maker" : d.vendor, d.private ? [" ", h("span", { class: "badge private", title: "A private Wi-Fi address. It stays the same on this network, so it still works for presence." }, "private address")] : null),
                  h("small", { class: "mono", text: d.mac })))),
              h("td", {}, bars(d.signal)),
              h("td", { class: "faint", text: `${ago(d.last_seen, now)} · first ${at(d.first_seen)}` }),
              h("td", { class: "end" }, assignControl(d, people))))))));
  };
  await draw();
  every(10, draw);
}

function assignControl(device, people) {
  if (!people.length) return h("a", { class: "button small", href: "#/people" }, "Add people first");
  const select = h("select", { class: "input", "aria-label": `Assign ${device.mac} to` },
    h("option", { value: "", text: "Assign to…" }),
    people.map((p) => h("option", { value: p.id, text: p.name })));
  select.addEventListener("change", async () => {
    if (!select.value) return;
    const person = people.find((p) => String(p.id) === select.value);
    try {
      await api(`/api/people/${select.value}/devices`, { method: "POST", body: { mac: device.mac, label: device.vendor === "Private address" ? "Device" : device.vendor } });
      toast(`Assigned to ${person.name}`);
      route();
    } catch (e) { toast(e.message); select.value = ""; }
  });
  return select;
}

// MARK: Audit

const AUDIT_ACTIONS = {
  "session.sign_in": ["Signed in"],
  "session.sign_in_failed": ["Failed sign-in", "warn"],
  "session.sign_out": ["Signed out"],
  "person.create": ["Added person"],
  "person.update": ["Edited person"],
  "person.archive": ["Anonymized person", "warn"],
  "person.erase": ["Erased person", "danger"],
  "device.add": ["Added device"],
  "device.remove": ["Removed device"],
  "invite.create": ["Created invite"],
  "invite.redeem": ["Registered by invite"],
  "export.csv": ["Exported attendance"],
  "audit.export": ["Exported audit log"],
  "token.reset": ["Reset admin token", "danger"],
  "retention.visits": ["Deleted old visits"],
};

const AUDIT_GROUPS = [["", "All"], ["session.", "Sign-ins"], ["person.", "People"], ["device.", "Devices"], ["invite.", "Invites"], ["export.", "Exports"]];

function auditActor(actor) {
  if (actor.startsWith("admin:")) return h("span", { class: "actor" }, "Administrator", h("small", { text: `session ${actor.slice(6)}` }));
  const names = { admin: "Administrator", invite: "Invite link", system: "WaveTrack", cli: "Command line", anonymous: "Not signed in" };
  return h("span", { class: "actor", text: names[actor] || actor });
}

// Entries name people by number only; show the current name next to it when they still exist.
function auditTarget(target, names) {
  const match = /^person #(\d+)$/.exec(target);
  if (!match) return target ? h("span", { class: target.includes(":") ? "mono" : null, text: target }) : h("span", { class: "faint", text: "—" });
  const name = names.get(Number(match[1]));
  return name ? h("span", {}, name, h("span", { class: "faint", text: ` · #${match[1]}` }))
    : h("span", { class: "faint", text: `#${match[1]} · no longer listed` });
}

async function renderAudit(arg) {
  const group = AUDIT_GROUPS.some(([key]) => key === arg) ? arg : "";
  const query = (before) => `/api/audit?limit=100${group ? `&action=${encodeURIComponent(group)}` : ""}${before ? `&before=${before}` : ""}`;
  const [{ entries, more }, { people }] = await Promise.all([api(query()), api("/api/people")]);
  const names = new Map(people.map((p) => [p.id, p.name]));

  const row = (e) => {
    const [label, tone] = AUDIT_ACTIONS[e.action] || [e.action];
    return h("tr", {},
      h("td", { class: "when" }, h("time", { datetime: e.at, text: stampFmt.format(new Date(e.at)) })),
      h("td", {}, auditActor(e.actor)),
      h("td", {}, h("span", { class: `tag${tone ? " " + tone : ""}`, title: e.action, text: label })),
      h("td", {}, auditTarget(e.target, names)),
      h("td", { class: "faint", text: e.detail || "" }),
      h("td", { class: "mono faint", text: e.ip || "" }));
  };

  const body = h("tbody", {}, entries.map(row));
  let last = entries.at(-1)?.id;
  const moreButton = h("button", { class: "button", type: "button", text: "Load older entries" });
  const footer = h("div", { class: "more", hidden: !more });
  footer.append(moreButton);
  moreButton.addEventListener("click", async () => {
    const page = await api(query(last));
    body.append(...page.entries.map(row));
    last = page.entries.at(-1)?.id;
    footer.hidden = !page.more;
  });

  const filters = h("div", { class: "segmented", role: "group", "aria-label": "Show" },
    AUDIT_GROUPS.map(([key, label]) => h("button", { type: "button", "aria-pressed": String(key === group), onclick: () => { location.hash = `#/audit/${key}`; }, text: label })));

  view.replaceChildren(
    pageHead("Audit log", "Every administrative action: who did it, from where and when. People appear by number, so erasing someone leaves no trace of them here.",
      filters,
      h("a", { class: "button", href: "/api/audit.csv", download: "" }, icon("download"), "Export CSV")),
    h("div", { class: "card" },
      entries.length === 0
        ? emptyState("shield", "Nothing recorded yet", "Sign-ins, changes to people and devices, invites and exports appear here.")
        : h("div", { class: "scroll" }, h("table", { class: "table audit-table" },
          h("thead", {}, h("tr", {}, ["Time", "Who", "Action", "Target", "Detail", "Address"].map((t) => h("th", { text: t })))),
          body)),
      footer));
}

// MARK: start

async function start() {
  try {
    await api("/api/me");
  } catch (e) {
    if (e instanceof Unauthorized) return showSignIn();
  }
  document.getElementById("signin").hidden = true;
  document.getElementById("app").hidden = false;
  watchStatus();
  if (!location.hash || location.hash.startsWith("#token=")) history.replaceState(null, "", "#/now");
  route();
}

// ?theme=<name> pins a theme, which documentation screenshots use.
const pinnedTheme = new URLSearchParams(location.search).get("theme");
applyTheme(THEMES.includes(pinnedTheme) ? pinnedTheme : storedTheme());
window.addEventListener("hashchange", route);

// A sign-in link carries the token in the fragment, which never reaches server logs.
if (location.hash.startsWith("#token=")) {
  const token = decodeURIComponent(location.hash.slice(7));
  history.replaceState(null, "", "#/now");
  signIn(token).catch(() => {}).finally(start);
} else {
  start();
}

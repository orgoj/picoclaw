package adminapi

import "net/http"

func RegisterDashboardRoutes(r routeRegistrar) {
	if r == nil {
		return
	}
	r.HandleFunc("/dashboard", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(dashboardHTML))
	})
	r.HandleFunc("/dashboard/app.js", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		_, _ = w.Write([]byte(dashboardJS))
	})
}

const dashboardHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>PicoClaw Dashboard</title>
  <style>
    :root {
      --bg:#f8fafc; --bg2:#eef2ff; --panel:#ffffff; --border:#cbd5e1;
      --muted:#475569; --text:#0f172a; --acc:#16a34a; --warn:#d97706; --bad:#dc2626;
      --input:#f8fafc; --head:#0f172a;
    }
    @media (prefers-color-scheme: dark) {
      :root {
        --bg:#0b1220; --bg2:#0f172a; --panel:#111827; --border:#334155;
        --muted:#94a3b8; --text:#e5e7eb; --acc:#22c55e; --warn:#f59e0b; --bad:#ef4444;
        --input:#0b1220; --head:#cbd5e1;
      }
    }
    body[data-theme="light"] {
      --bg:#f8fafc; --bg2:#eef2ff; --panel:#ffffff; --border:#cbd5e1;
      --muted:#475569; --text:#0f172a; --acc:#16a34a; --warn:#d97706; --bad:#dc2626;
      --input:#f8fafc; --head:#0f172a;
    }
    body[data-theme="dark"] {
      --bg:#0b1220; --bg2:#0f172a; --panel:#111827; --border:#334155;
      --muted:#94a3b8; --text:#e5e7eb; --acc:#22c55e; --warn:#f59e0b; --bad:#ef4444;
      --input:#0b1220; --head:#cbd5e1;
    }
    body {
      margin:0;
      font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
      background:linear-gradient(180deg,var(--bg),var(--bg2));
      color:var(--text);
      min-height:100vh;
    }
    .toolbar {
      display:flex; gap:10px; align-items:center; justify-content:flex-end;
      padding:8px 14px; border-bottom:1px solid var(--border);
      background:color-mix(in srgb, var(--panel) 90%, transparent);
      position:sticky; top:0; z-index:1;
    }
    .toolbar .grow { flex:1; }
    .modal {
      position:fixed; inset:0; background:rgba(0,0,0,0.45);
      display:none; align-items:center; justify-content:center; z-index:20;
      padding:16px; box-sizing:border-box;
    }
    .modal.show { display:flex; }
    .modalCard {
      width:min(1100px, 100%); height:min(90vh, 100%);
      background:var(--panel); border:1px solid var(--border); border-radius:10px;
      display:grid; grid-template-rows:auto 1fr; gap:10px; padding:10px;
      box-sizing:border-box;
    }
    .modalBody {
      display:grid; grid-template-columns: 1fr 1fr; gap:10px; min-height:0;
    }
    .modalBody pre { margin:0; }
    @media (max-width: 980px) {
      .modalBody { grid-template-columns: 1fr; }
    }
    .wrap {
      padding:14px;
      display:grid;
      grid-template-columns: 1fr 1fr;
      grid-template-rows: minmax(0, 1fr) minmax(0, 1fr);
      gap:12px;
      min-height:calc(100vh - 54px);
      box-sizing:border-box;
    }
    .card {
      background:var(--panel);
      border:1px solid var(--border);
      border-radius:10px;
      padding:10px;
      display:flex;
      flex-direction:column;
      min-height:0;
    }
    h2 { margin:0 0 8px; font-size:14px; color:var(--head); }
    input,textarea,button { font:inherit; }
    input,textarea,select { width:100%; background:var(--input); color:var(--text); border:1px solid var(--border); border-radius:8px; padding:8px; box-sizing:border-box; }
    textarea { min-height:80px; resize:vertical; }
    button { background:var(--input); color:var(--text); border:1px solid var(--border); border-radius:8px; padding:6px 10px; cursor:pointer; }
    button:hover { border-color:var(--muted); }
    .row { display:flex; gap:8px; align-items:center; margin:6px 0; }
    .grow { flex:1; }
    .small { font-size:12px; color:var(--muted); }
    .tableWrap { overflow:auto; flex:1; min-height:0; border:1px solid var(--border); border-radius:8px; }
    table { width:100%; border-collapse:collapse; font-size:12px; }
    td,th { border-top:1px solid var(--border); padding:6px; text-align:left; vertical-align:top; }
    .ok { color:var(--acc); } .warn { color:var(--warn); } .bad { color:var(--bad); }
    pre {
      white-space:pre-wrap; word-break:break-word; background:var(--input);
      border:1px solid var(--border); border-radius:8px; padding:8px; margin:0;
      overflow:auto; flex:1; min-height:0;
    }
    @media (max-width: 980px) {
      .wrap {
        grid-template-columns: 1fr;
        grid-template-rows: auto;
        min-height:auto;
      }
      .card { min-height:320px; }
    }
  </style>
</head>
<body>
  <div class="toolbar">
    <span class="grow small">PicoClaw Dashboard</span>
    <span id="sseState" class="small warn">SSE connecting...</span>
    <button id="showRuntimeBtn">Runtime info</button>
    <label class="small" for="themeMode">Theme</label>
    <select id="themeMode" style="width:auto">
      <option value="system">System</option>
      <option value="light">Light</option>
      <option value="dark">Dark</option>
    </select>
  </div>
  <div id="runtimeModal" class="modal">
    <div class="modalCard">
      <div class="row">
        <h2 class="grow" style="margin:0">Runtime Inspector</h2>
        <button id="refreshRuntimeBtn">Refresh</button>
        <button id="closeRuntimeBtn">Close</button>
      </div>
      <div class="modalBody">
        <pre id="runtimeSummaryBox"></pre>
        <pre id="runtimeConfigBox"></pre>
      </div>
    </div>
  </div>
  <div class="wrap">
    <section class="card">
      <h2>Main Message</h2>
      <div class="row"><input id="sessionKey" class="grow" placeholder="session_key (e.g. telegram:7221629441)" value="telegram:7221629441"></div>
      <div class="row"><textarea id="content" placeholder="Type message..."></textarea></div>
      <div class="row">
        <label class="small"><input id="urgent" type="checkbox"> urgent</label>
        <button id="sendBtn">Send</button>
      </div>
      <div id="sendOut" class="small"></div>
    </section>

    <section class="card">
      <h2>Inbound Queue</h2>
      <div class="row manual-controls"><button id="refreshInbound">Refresh</button><span class="small">Editable pending queue</span></div>
      <div class="tableWrap"><table id="inboundTable"></table></div>
    </section>

    <section class="card">
      <h2>Session History</h2>
      <div class="row manual-controls"><button id="refreshHistory">Refresh</button></div>
      <pre id="historyBox"></pre>
    </section>

    <section class="card">
      <h2>Subagents</h2>
      <div class="row manual-controls"><button id="refreshSubagents">Refresh</button><span class="small">List + detail</span></div>
      <div class="tableWrap"><table id="subagentTable"></table></div>
      <pre id="subagentDetail"></pre>
    </section>
  </div>
  <script src="/dashboard/app.js"></script>
</body>
</html>
`

const dashboardJS = `
const $ = (id) => document.getElementById(id);
let es = null;

async function jfetch(url, opts={}) {
  const r = await fetch(url, { headers: { "Content-Type": "application/json" }, ...opts });
  const j = await r.json().catch(() => ({}));
  if (!r.ok) throw new Error(j.error || ("HTTP "+r.status));
  return j;
}

async function sendMessage() {
  const body = {
    session_key: $("sessionKey").value.trim(),
    content: $("content").value,
    urgent: $("urgent").checked,
  };
  try {
    const res = await jfetch("/api/v1/main/message", { method:"POST", body: JSON.stringify(body) });
    $("sendOut").textContent = JSON.stringify(res);
    $("content").value = "";
  } catch (e) {
    $("sendOut").textContent = "ERROR: "+e.message;
  }
}

function setManualControlsVisible(visible) {
  const rows = document.querySelectorAll(".manual-controls");
  for (const row of rows) {
    row.style.display = visible ? "flex" : "none";
  }
}

function renderInbound(items) {
  const t = $("inboundTable");
  let html = "<tr><th>ID</th><th>Session</th><th>Content</th><th>Actions</th></tr>";
  for (const it of (items || [])) {
    const id = it.id;
    const m = it.message || {};
    html += "<tr>";
    html += "<td>"+id+"</td>";
    html += "<td>"+(m.session_key||"")+"</td>";
    html += "<td><textarea id='e_"+id+"' style='min-height:54px'>"+(m.content||"")+"</textarea></td>";
    html += "<td>";
    html += "<button onclick='patchItem(\""+id+"\")'>save</button> ";
    html += "<button onclick='moveTop(\""+id+"\")'>top</button> ";
    html += "<button onclick='delItem(\""+id+"\")'>del</button>";
    html += "</td></tr>";
  }
  t.innerHTML = html;
}

function renderHistory(items) {
  const box = $("historyBox");
  box.textContent = (items || []).map((m,i) => "["+i+"] "+m.role+": "+(m.content||"")).join("\\n\\n");
  box.scrollTop = box.scrollHeight;
}

function renderSubagents(items) {
  const t = $("subagentTable");
  const detail = $("subagentDetail");
  const ordered = (items || []).slice().reverse();
  let html = "<tr><th>ID</th><th>Status</th><th>Label</th><th>Pending</th><th></th></tr>";
  for (const it of ordered) {
    html += "<tr>";
    html += "<td>"+it.id+"</td>";
    html += "<td>"+it.status+"</td>";
    html += "<td>"+(it.label||"")+"</td>";
    html += "<td>"+(it.pending||0)+"</td>";
    html += "<td><button onclick='loadSubagent(\""+it.id+"\")'>view</button></td>";
    html += "</tr>";
  }
  t.innerHTML = html;
  if (!ordered.length) detail.textContent = "No subagents.";
}

async function refreshInbound() {
  try {
    const res = await jfetch("/api/v1/inbound");
    renderInbound(res.items || []);
  } catch (e) {
    $("inboundTable").innerHTML = "<tr><td class='bad'>"+e.message+"</td></tr>";
  }
}

async function patchItem(id) {
  const el = $("e_"+id);
  const content = el ? el.value : "";
  await jfetch("/api/v1/inbound/"+id, { method:"PATCH", body: JSON.stringify({ content }) });
  await refreshInbound();
}

async function moveTop(id) {
  await jfetch("/api/v1/inbound/"+id+"/move", { method:"POST", body: JSON.stringify({ index: 0 }) });
  await refreshInbound();
}

async function delItem(id) {
  await jfetch("/api/v1/inbound/"+id, { method:"DELETE" });
  await refreshInbound();
}

async function refreshHistory() {
  const key = $("sessionKey").value.trim();
  const box = $("historyBox");
  if (!key) { box.textContent = "session_key required"; return; }
  try {
    const res = await jfetch("/api/v1/history?session_key="+encodeURIComponent(key));
    renderHistory(res.items || []);
  } catch (e) {
    box.textContent = "ERROR: "+e.message;
  }
}

async function refreshSubagents() {
  try {
    const res = await jfetch("/api/v1/subagents");
    renderSubagents(res.items || []);
  } catch (e) {
    $("subagentTable").innerHTML = "<tr><td class='bad'>"+e.message+"</td></tr>";
  }
}

async function loadSubagent(id) {
  const detail = $("subagentDetail");
  try {
    const res = await jfetch("/api/v1/subagents/"+id);
    detail.textContent = JSON.stringify(res, null, 2);
  } catch (e) {
    detail.textContent = "ERROR: "+e.message;
  }
}

function connectEvents() {
  if (es) {
    es.close();
    es = null;
  }
  const key = $("sessionKey").value.trim();
  if (!key) return;

  setSSEStatus("SSE connecting...", "warn");
  es = new EventSource("/api/v1/events?session_key="+encodeURIComponent(key));
  es.onopen = () => {
    setSSEStatus("SSE connected", "ok");
    setManualControlsVisible(false);
  };
  es.addEventListener("snapshot", (ev) => {
    try {
      const data = JSON.parse(ev.data);
      renderInbound(data.inbound || []);
      renderHistory(data.history || []);
      renderSubagents(data.subagents || []);
      refreshRuntime(true);
    } catch (e) {
      $("sendOut").textContent = "SSE parse error: "+e.message;
    }
  });
  es.onerror = () => {
    setSSEStatus("SSE disconnected, retrying...", "warn");
    setManualControlsVisible(true);
  };
}

function fmtObj(o) {
  return JSON.stringify(o, null, 2);
}

function buildRuntimeSummary(rt) {
  const lines = [];
  const ver = rt.version || {};
  lines.push("Version");
  lines.push("- app: " + (ver.app || ""));
  lines.push("- go: " + (ver.go || ""));
  lines.push("");

  const model = rt.model || {};
  lines.push("Model");
  lines.push("- provider: " + (model.provider || ""));
  lines.push("- model: " + (model.model || ""));
  lines.push("");

  const agent = rt.agent || {};
  const tools = agent.tools || {};
  const skills = agent.skills || {};
  const agents = agent.agents || {};
  lines.push("Capabilities");
  lines.push("- tools: " + (tools.count || 0));
  lines.push("- skills: " + (skills.available || 0) + "/" + (skills.total || 0));
  lines.push("- named agents: " + (agents.count || 0));
  lines.push("");

  const controls = rt.controls || {};
  lines.push("Controls");
  lines.push("- prefix: " + (controls.prefix || "+"));
  lines.push("- commands: " + (controls.count || 0));
  if (Array.isArray(controls.commands)) {
    for (const c of controls.commands) lines.push("  - " + c);
  }
  lines.push("");

  const ch = rt.channels || {};
  lines.push("Channels");
  lines.push("- enabled: " + (ch.enabled_count || 0));
  if (Array.isArray(ch.enabled) && ch.enabled.length) lines.push("- list: " + ch.enabled.join(", "));
  lines.push("");

  const sub = rt.subagents || {};
  const iq = rt.inbound_queue || {};
  lines.push("Queues");
  lines.push("- inbound queue: " + (iq.count || 0));
  lines.push("- subagent running: " + (sub.running_count || 0));
  lines.push("- subagent recent: " + (sub.recent_count || 0));
  lines.push("- subagent msg queue: " + (sub.queue_count || 0));
  return lines.join("\\n");
}

async function refreshRuntime(silent) {
  try {
    const res = await jfetch("/api/v1/runtime");
    const rt = (res && res.runtime) || {};
    $("runtimeSummaryBox").textContent = buildRuntimeSummary(rt);
    const cfg = rt.config || {};
    const top = {
      path: cfg.path || "",
      available: !!cfg.available,
      error: cfg.error || "",
    };
    $("runtimeConfigBox").textContent = "Config Meta\\n" + fmtObj(top) + "\\n\\nSanitized Config\\n" + fmtObj(cfg.sanitized || {});
  } catch (e) {
    if (!silent) $("runtimeSummaryBox").textContent = "ERROR: " + e.message;
  }
}

function showRuntime() {
  const modal = $("runtimeModal");
  if (!modal) return;
  modal.classList.add("show");
  refreshRuntime(false);
}

function closeRuntime() {
  const modal = $("runtimeModal");
  if (!modal) return;
  modal.classList.remove("show");
}

function setSSEStatus(text, klass) {
  const el = $("sseState");
  if (!el) return;
  el.textContent = text;
  el.className = "small " + (klass || "");
}

function applyTheme(mode) {
  const m = mode || "system";
  if (m === "light" || m === "dark") {
    document.body.setAttribute("data-theme", m);
  } else {
    document.body.removeAttribute("data-theme");
  }
}

function initTheme() {
  const sel = $("themeMode");
  if (!sel) return;
  const saved = localStorage.getItem("pc_dashboard_theme") || "system";
  sel.value = saved;
  applyTheme(saved);
  sel.addEventListener("change", () => {
    const mode = sel.value || "system";
    localStorage.setItem("pc_dashboard_theme", mode);
    applyTheme(mode);
  });
  const media = window.matchMedia ? window.matchMedia("(prefers-color-scheme: dark)") : null;
  if (media && typeof media.addEventListener === "function") {
    media.addEventListener("change", () => {
      if ((localStorage.getItem("pc_dashboard_theme") || "system") === "system") applyTheme("system");
    });
  }
}

const sendBtn = $("sendBtn");
if (sendBtn) sendBtn.addEventListener("click", sendMessage);
const contentBox = $("content");
if (contentBox) {
  contentBox.addEventListener("keydown", (ev) => {
    if (ev.key === "Enter" && ev.ctrlKey) {
      ev.preventDefault();
      sendMessage();
    }
  });
}
const refreshInboundBtn = $("refreshInbound");
if (refreshInboundBtn) refreshInboundBtn.addEventListener("click", refreshInbound);
const refreshHistoryBtn = $("refreshHistory");
if (refreshHistoryBtn) refreshHistoryBtn.addEventListener("click", refreshHistory);
const refreshSubagentsBtn = $("refreshSubagents");
if (refreshSubagentsBtn) refreshSubagentsBtn.addEventListener("click", refreshSubagents);
const sessionKey = $("sessionKey");
if (sessionKey) sessionKey.addEventListener("change", connectEvents);
const showRuntimeBtn = $("showRuntimeBtn");
if (showRuntimeBtn) showRuntimeBtn.addEventListener("click", showRuntime);
const closeRuntimeBtn = $("closeRuntimeBtn");
if (closeRuntimeBtn) closeRuntimeBtn.addEventListener("click", closeRuntime);
const refreshRuntimeBtn = $("refreshRuntimeBtn");
if (refreshRuntimeBtn) refreshRuntimeBtn.addEventListener("click", () => refreshRuntime(false));
const runtimeModal = $("runtimeModal");
if (runtimeModal) runtimeModal.addEventListener("click", (ev) => {
  if (ev.target === runtimeModal) closeRuntime();
});

window.patchItem = patchItem;
window.moveTop = moveTop;
window.delItem = delItem;
window.loadSubagent = loadSubagent;

initTheme();
setManualControlsVisible(true);
refreshInbound();
refreshHistory();
refreshSubagents();
refreshRuntime(true);
connectEvents();
`

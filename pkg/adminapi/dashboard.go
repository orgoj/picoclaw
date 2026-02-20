package adminapi

import "net/http"

func RegisterDashboardRoutes(r routeRegistrar) {
	if r == nil {
		return
	}
	r.HandleFunc("/dashboard", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		_, _ = w.Write([]byte(dashboardHTML))
	})
	r.HandleFunc("/dashboard/app.js", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
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
    html, body { height: 100%; }
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
      height:100vh;
      height:100dvh;
      display:flex;
      flex-direction:column;
      overflow:hidden;
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
    :root { --left-col: 62%; }
    .app {
      padding:14px;
      display:grid;
      grid-template-columns: minmax(340px, var(--left-col)) 8px minmax(360px, 1fr);
      gap:8px;
      height:calc(100vh - 54px);
      height:calc(100dvh - 54px);
      box-sizing:border-box;
      min-height:0;
      overflow:hidden;
    }
    .splitter {
      border-radius:8px;
      background:var(--border);
      cursor:col-resize;
      user-select:none;
    }
    .splitter:hover { filter:brightness(1.15); }
    .card {
      background:var(--panel);
      border:1px solid var(--border);
      border-radius:10px;
      padding:10px;
      display:flex;
      flex-direction:column;
      min-height:0;
      overflow:hidden;
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
    tr.clickable { cursor:pointer; }
    tr.clickable:hover { background: color-mix(in srgb, var(--input) 70%, var(--border)); }
    tr.selected { background: color-mix(in srgb, var(--acc) 20%, var(--input)); }
    tr.agent-running { background: color-mix(in srgb, var(--acc) 15%, transparent); }
    tr.agent-stopped { background: color-mix(in srgb, var(--muted) 22%, transparent); }
    .ok { color:var(--acc); } .warn { color:var(--warn); } .bad { color:var(--bad); }
    pre {
      white-space:pre-wrap; word-break:break-word; background:var(--input);
      border:1px solid var(--border); border-radius:8px; padding:8px; margin:0;
      overflow:auto; flex:1; min-height:0;
    }
    .rightCol {
      display:flex;
      flex-direction:column;
      gap:6px;
      min-height:0;
    }
    .rightCard { flex:1 1 0; min-height:120px; }
    .rowSplitter {
      height:8px;
      border-radius:8px;
      background:var(--border);
      cursor:row-resize;
      user-select:none;
      flex:0 0 8px;
    }
    .rowSplitter:hover { filter:brightness(1.15); }
    .leftCol { min-height:0; }
    .historyMeta { margin-bottom:8px; }
    @media (max-width: 980px) {
      body { overflow:auto; height:auto; }
      .app {
        grid-template-columns: 1fr;
        height:auto;
        overflow:visible;
      }
      .splitter { display:none; }
      .rowSplitter { display:none; }
      .rightCol { gap:8px; }
      .rightCard, .card { min-height:320px; }
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
  <div class="app">
    <section class="card leftCol">
      <h2>History</h2>
      <div id="historyMeta" class="small historyMeta"></div>
      <pre id="historyBox"></pre>
    </section>

    <div id="colSplitter" class="splitter" title="Drag to resize"></div>

    <div class="rightCol">
      <section class="card rightCard">
        <h2>Agents</h2>
        <div class="row">
          <div class="small grow">Click agent to filter history</div>
          <button id="clearAgentBtn" type="button">Clear filter</button>
        </div>
        <div class="tableWrap"><table id="subagentTable"></table></div>
      </section>
      <div class="rowSplitter" title="Drag to resize"></div>

      <section class="card rightCard">
        <h2>Sessions</h2>
        <div class="small">Click session to switch history stream</div>
        <div class="tableWrap"><table id="sessionTable"></table></div>
      </section>
      <div class="rowSplitter" title="Drag to resize"></div>

      <section class="card rightCard">
        <h2>Queue</h2>
        <div class="small">Editable inbound queue</div>
        <div class="tableWrap"><table id="inboundTable"></table></div>
      </section>
      <div class="rowSplitter" title="Drag to resize"></div>

      <section class="card rightCard">
        <h2>Message</h2>
        <div class="row"><input id="sessionKey" class="grow" placeholder="session_key (e.g. telegram:7221629441)"></div>
        <div class="row"><textarea id="content" placeholder="Type message..."></textarea></div>
        <div class="row">
          <label class="small" for="messageMode">Mode</label>
          <select id="messageMode" style="width:auto">
            <option value="normal">Queue</option>
            <option value="inject">Inject</option>
            <option value="first">Force First</option>
            <option value="append">Append</option>
            <option value="delete">Delete Last</option>
          </select>
          <button id="sendBtn">Send</button>
        </div>
        <div id="commandHint" class="small"></div>
        <div id="sendOut" class="small"></div>
      </section>
    </div>
  </div>
  <script src="/dashboard/app.js"></script>
</body>
</html>
`

const dashboardJS = `
const $ = (id) => document.getElementById(id);
let es = null;
let reconnectTimer = null;
const state = {
  selectedSession: "",
  selectedAgentID: "",
  selectedAgentName: "",
  history: [],
  sessions: [],
  subagents: [],
  historyLimit: 1000,
  controlPrefix: "+",
  hasKillControl: true,
};

async function jfetch(url, opts={}) {
  const r = await fetch(url, { headers: { "Content-Type": "application/json" }, ...opts });
  const j = await r.json().catch(() => ({}));
  if (!r.ok) throw new Error(j.error || ("HTTP "+r.status));
  return j;
}

async function sendMessage() {
  const btn = $("sendBtn");
  try {
    setButtonBusy(btn, true, "Sending...");
    const mode = (($("messageMode") && $("messageMode").value) || "normal").trim();
    const input = $("content").value;
    let content = "";
    if (mode === "normal") {
      content = input;
    } else if (mode === "inject") {
      if (!String(input).trim()) throw new Error("MESSAGE required for inject");
      content = state.controlPrefix + "inject " + input;
    } else if (mode === "first") {
      if (!String(input).trim()) throw new Error("MESSAGE required for force first");
      content = state.controlPrefix + "first " + input;
    } else if (mode === "append") {
      if (!String(input).trim()) throw new Error("MESSAGE required for append");
      content = state.controlPrefix + state.controlPrefix + input;
    } else if (mode === "delete") {
      if (!window.confirm("Delete last queued message for this session?")) return;
      content = state.controlPrefix + "delete";
    } else {
      throw new Error("Unknown mode: " + mode);
    }
    const res = await sendMainContent(content);
    setSendOut(describeMainResponse(mode, res), false);
    if (mode !== "delete") $("content").value = "";
  } catch (e) {
    setSendOut("ERROR: "+e.message, true);
  } finally {
    setButtonBusy(btn, false, "Send");
  }
}

async function sendMainContent(content) {
  const sessionKey = $("sessionKey").value.trim();
  if (!sessionKey) throw new Error("session_key is required");
  return jfetch("/api/v1/main/message", {
    method:"POST",
    body: JSON.stringify({ session_key: sessionKey, content }),
  });
}

function escapeHTML(v) {
  return String(v || "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
}

function renderInbound(items) {
  const t = $("inboundTable");
  const rows = (items || []);
  let html = "<tr><th>#</th><th>ID</th><th>Session</th><th>Content</th><th></th></tr>";
  for (let i = 0; i < rows.length; i++) {
    const it = rows[i];
    const id = it.id;
    const m = it.message || {};
    html += "<tr>";
    html += "<td>"+i+"</td>";
    html += "<td>"+escapeHTML(id)+"</td>";
    html += "<td>"+escapeHTML(m.session_key||"")+"</td>";
    html += "<td><textarea id='e_"+id+"' style='min-height:54px'>"+escapeHTML(m.content||"")+"</textarea></td>";
    html += "<td>";
    html += "<button onclick='moveItem(\""+id+"\",0,this,event)'>top</button> ";
    if (i > 0) html += "<button onclick='moveItem(\""+id+"\","+(i-1)+",this,event)'>up</button> ";
    if (i < rows.length - 1) html += "<button onclick='moveItem(\""+id+"\","+(i+1)+",this,event)'>down</button> ";
    html += "<button onclick='moveItem(\""+id+"\","+(rows.length - 1)+",this,event)'>bottom</button> ";
    html += "<button onclick='patchItem(\""+id+"\",this,event)'>save</button> ";
    html += "<button onclick='delItem(\""+id+"\",this,event)'>del</button>";
    html += "</td></tr>";
  }
  t.innerHTML = html;
}

function matchSelectedAgent(m) {
  if (!state.selectedAgentID && !state.selectedAgentName) return true;
  const content = String((m && m.content) || "").toLowerCase();
  const role = String((m && m.role) || "").toLowerCase();
  const c = role + "\n" + content;
  if (state.selectedAgentID) {
    const id = state.selectedAgentID.toLowerCase();
    if (c.includes(id) || c.includes("subagent:"+id) || c.includes("subagent-"+id)) return true;
  }
  if (state.selectedAgentName && c.includes(state.selectedAgentName.toLowerCase())) return true;
  return false;
}

function renderHistory(items) {
  const box = $("historyBox");
  const all = (items || []);
  const filtered = all.filter(matchSelectedAgent);
  const shown = filtered.length > state.historyLimit ? filtered.slice(filtered.length - state.historyLimit) : filtered;
  const base = Math.max(0, filtered.length - shown.length);
  const nearBottom = (box.scrollHeight - (box.scrollTop + box.clientHeight)) <= 40;
  const activeAgentFilter = !!(state.selectedAgentID || state.selectedAgentName);
  if (shown.length === 0 && activeAgentFilter) {
    box.textContent = "No messages for selected agent in this session yet.";
  } else {
    box.textContent = shown.map((m,i) => "["+(base+i)+"] "+m.role+": "+(m.content||"")).join("\n\n");
  }
  if (nearBottom || !activeAgentFilter) box.scrollTop = box.scrollHeight;
  const meta = $("historyMeta");
  if (meta) {
    let tag = state.selectedSession ? "session="+state.selectedSession : "session=(none)";
    if (state.selectedAgentID || state.selectedAgentName) tag += ", agent="+(state.selectedAgentName || state.selectedAgentID);
    meta.textContent = "showing " + shown.length + " / " + filtered.length + " messages (limit " + state.historyLimit + "), " + tag;
  }
}

function renderSubagents(items) {
  const t = $("subagentTable");
  const ordered = (items || []).slice().reverse();
  let html = "<tr><th>ID</th><th>Status</th><th>Agent</th><th>Label</th><th>Pending</th><th></th></tr>";
  for (const it of ordered) {
    const running = it.status === "running" || it.status === "pending";
    const cls = running ? "agent-running" : "agent-stopped";
    const selected = state.selectedAgentID === it.id ? " selected" : "";
    html += "<tr class='clickable "+cls+selected+"' onclick='selectAgent(\""+it.id+"\",\""+escapeAttr(it.name||"")+"\")'>";
    html += "<td>"+escapeHTML(it.id)+"</td>";
    html += "<td>"+escapeHTML(it.status)+"</td>";
    html += "<td>"+escapeHTML(it.name||"")+"</td>";
    html += "<td>"+escapeHTML(it.label||"")+"</td>";
    html += "<td>"+(it.pending||0)+"</td>";
    html += "<td>";
    if (running && state.hasKillControl) {
      html += "<button onclick='killSubagent(\""+it.id+"\",this,event)'>KILL</button>";
    }
    html += "</td>";
    html += "</tr>";
  }
  t.innerHTML = html;
}

function renderSessions(items) {
  const t = $("sessionTable");
  const rows = (items || []);
  let html = "<tr><th>Session</th><th>Msgs</th><th>Updated</th></tr>";
  for (const it of rows) {
    const selected = state.selectedSession === it.key ? " selected" : "";
    html += "<tr class='clickable"+selected+"' onclick='selectSession(\""+escapeAttr(it.key||"")+"\")'>";
    html += "<td>"+escapeHTML(it.key||"")+"</td>";
    html += "<td>"+(it.messages||0)+"</td>";
    html += "<td>"+fmtTs(it.updated)+"</td>";
    html += "</tr>";
  }
  t.innerHTML = html;
}

async function refreshInbound() {
  try {
    const res = await jfetch("/api/v1/inbound");
    renderInbound(res.items || []);
  } catch (e) {
    $("inboundTable").innerHTML = "<tr><td class='bad'>"+e.message+"</td></tr>";
  }
}

async function patchItem(id, btn, ev) {
  if (ev && typeof ev.stopPropagation === "function") ev.stopPropagation();
  const el = $("e_"+id);
  const content = el ? el.value : "";
  try {
    setButtonBusy(btn, true, "saving...");
    await jfetch("/api/v1/inbound/"+id, { method:"PATCH", body: JSON.stringify({ content }) });
    await refreshInbound();
    setSendOut("Queue item saved.", false);
  } catch (e) {
    setSendOut("ERROR: " + e.message, true);
  } finally {
    setButtonBusy(btn, false, "save");
  }
}

async function delItem(id, btn, ev) {
  if (ev && typeof ev.stopPropagation === "function") ev.stopPropagation();
  if (!window.confirm("Delete this queued item?")) return;
  try {
    setButtonBusy(btn, true, "deleting...");
    await jfetch("/api/v1/inbound/"+id, { method:"DELETE" });
    await refreshInbound();
    setSendOut("Queue item deleted.", false);
  } catch (e) {
    setSendOut("ERROR: " + e.message, true);
  } finally {
    setButtonBusy(btn, false, "del");
  }
}

async function moveItem(id, index, btn, ev) {
  if (ev && typeof ev.stopPropagation === "function") ev.stopPropagation();
  const old = btn && btn.textContent ? btn.textContent : "move";
  try {
    setButtonBusy(btn, true, "moving...");
    await jfetch("/api/v1/inbound/"+id+"/move", { method:"POST", body: JSON.stringify({ index }) });
    await refreshInbound();
  } catch (e) {
    setSendOut("ERROR: " + e.message, true);
  } finally {
    setButtonBusy(btn, false, old);
  }
}

async function killSubagent(id, btn, ev) {
  if (ev && typeof ev.stopPropagation === "function") ev.stopPropagation();
  if (!id) return;
  if (!window.confirm("Cancel subagent task '" + id + "'?")) return;
  try {
    setButtonBusy(btn, true, "killing...");
    const res = await sendMainContent(state.controlPrefix + "kill " + id);
    setSendOut(describeMainResponse("kill", res), false);
  } catch (e) {
    setSendOut("ERROR: " + e.message, true);
  } finally {
    setButtonBusy(btn, false, "KILL");
  }
}

async function refreshHistory() {
  const key = state.selectedSession || $("sessionKey").value.trim();
  const box = $("historyBox");
  if (!key) { box.textContent = "session_key required"; return; }
  try {
    const res = await jfetch("/api/v1/history?session_key="+encodeURIComponent(key));
    state.selectedSession = key;
    $("sessionKey").value = key;
    state.history = res.items || [];
    renderHistory(state.history);
  } catch (e) {
    box.textContent = "ERROR: "+e.message;
  }
}

async function refreshSubagents() {
  try {
    const res = await jfetch("/api/v1/subagents");
    state.subagents = res.items || [];
    renderSubagents(state.subagents);
  } catch (e) {
    $("subagentTable").innerHTML = "<tr><td class='bad'>"+e.message+"</td></tr>";
  }
}

async function refreshSessions() {
  try {
    const res = await jfetch("/api/v1/sessions?limit=200");
    state.sessions = res.items || [];
    renderSessions(state.sessions);
    if (!state.selectedSession && state.sessions.length > 0) {
      state.selectedSession = state.sessions[0].key || "";
      $("sessionKey").value = state.selectedSession;
      connectEvents();
    }
  } catch (e) {
    $("sessionTable").innerHTML = "<tr><td class='bad'>"+e.message+"</td></tr>";
  }
}

function connectEvents() {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  if (es) {
    es.close();
    es = null;
  }
  const key = state.selectedSession || $("sessionKey").value.trim();
  if (!key) return;
  state.selectedSession = key;
  $("sessionKey").value = key;

  setSSEStatus("SSE connecting...", "warn");
  es = new EventSource("/api/v1/events?session_key="+encodeURIComponent(key));
  es.onopen = () => {
    setSSEStatus("SSE connected", "ok");
    renderSessions(state.sessions);
  };
  es.addEventListener("snapshot", (ev) => {
    try {
      const data = JSON.parse(ev.data);
      renderInbound(data.inbound || []);
      state.history = data.history || [];
      renderHistory(state.history);
      state.subagents = data.subagents || [];
      renderSubagents(state.subagents);
      if (Array.isArray(data.sessions)) {
        state.sessions = data.sessions;
        renderSessions(state.sessions);
      }
      refreshRuntime(true);
    } catch (e) {
      $("sendOut").textContent = "SSE parse error: "+e.message;
    }
  });
  es.onerror = () => {
    setSSEStatus("SSE disconnected, retrying...", "warn");
    // Native EventSource reconnects automatically, but some gateways close streams
    // permanently. If closed, rebuild connection explicitly.
    if (es && es.readyState === EventSource.CLOSED) {
      if (reconnectTimer) clearTimeout(reconnectTimer);
      reconnectTimer = setTimeout(() => connectEvents(), 1200);
    }
  };
}

function selectSession(key) {
  state.selectedSession = key || "";
  state.selectedAgentID = "";
  state.selectedAgentName = "";
  $("sessionKey").value = state.selectedSession;
  renderSubagents(state.subagents);
  renderSessions(state.sessions);
  connectEvents();
  refreshHistory();
}

function selectAgent(id, name) {
  state.selectedAgentID = id || "";
  state.selectedAgentName = name || "";
  renderSubagents(state.subagents);
  renderHistory(state.history);
}

function clearAgentFilter() {
  state.selectedAgentID = "";
  state.selectedAgentName = "";
  renderSubagents(state.subagents);
  renderHistory(state.history);
}

function fmtTs(v) {
  const n = Number(v || 0);
  if (!n) return "";
  const d = new Date(n);
  if (isNaN(d.getTime())) return "";
  return d.toLocaleString();
}

function escapeAttr(v) {
  return String(v || "").replace(/"/g, "&quot;");
}

function initSplitter() {
  const splitter = $("colSplitter");
  const app = document.querySelector(".app");
  if (!splitter || !app) return;
  const savedLeft = localStorage.getItem("pc_dashboard_left_col_pct");
  if (savedLeft) {
    const v = Number(savedLeft);
    if (!Number.isNaN(v) && v >= 35 && v <= 80) {
      document.documentElement.style.setProperty("--left-col", v.toFixed(2) + "%");
    }
  }
  let dragging = false;
  const onMove = (ev) => {
    if (!dragging) return;
    const rect = app.getBoundingClientRect();
    if (!rect.width) return;
    const x = ev.clientX - rect.left;
    const p = Math.max(35, Math.min(80, (x / rect.width) * 100));
    document.documentElement.style.setProperty("--left-col", p.toFixed(2) + "%");
    localStorage.setItem("pc_dashboard_left_col_pct", p.toFixed(2));
  };
  splitter.addEventListener("pointerdown", (ev) => {
    dragging = true;
    splitter.setPointerCapture(ev.pointerId);
  });
  splitter.addEventListener("pointerup", () => { dragging = false; });
  splitter.addEventListener("pointercancel", () => { dragging = false; });
  splitter.addEventListener("pointermove", onMove);
}

function initRightRowSplitters() {
  const container = document.querySelector(".rightCol");
  if (!container) return;
  const cards = Array.from(container.querySelectorAll(".rightCard"));
  if (cards.length !== 4) return;

  const savedRaw = localStorage.getItem("pc_dashboard_right_heights_px");
  if (savedRaw) {
    try {
      const vals = JSON.parse(savedRaw);
      if (Array.isArray(vals) && vals.length === 4) {
        for (let i = 0; i < 4; i++) {
          const h = Number(vals[i]);
          if (!Number.isNaN(h) && h >= 120) cards[i].style.flex = "0 0 " + Math.round(h) + "px";
        }
      }
    } catch (_) {}
  }

  const saveHeights = () => {
    const heights = cards.map((c) => Math.round(c.getBoundingClientRect().height));
    localStorage.setItem("pc_dashboard_right_heights_px", JSON.stringify(heights));
  };

  const splitters = Array.from(container.querySelectorAll(".rowSplitter"));
  for (const splitter of splitters) {
    splitter.addEventListener("pointerdown", (ev) => {
      const prev = splitter.previousElementSibling;
      const next = splitter.nextElementSibling;
      if (!prev || !next) return;
      const min = 120;
      const startY = ev.clientY;
      const startPrev = prev.getBoundingClientRect().height;
      const startNext = next.getBoundingClientRect().height;
      splitter.setPointerCapture(ev.pointerId);

      const onMove = (mv) => {
        const dy = mv.clientY - startY;
        let newPrev = startPrev + dy;
        let newNext = startNext - dy;
        if (newPrev < min) {
          newNext -= (min - newPrev);
          newPrev = min;
        }
        if (newNext < min) {
          newPrev -= (min - newNext);
          newNext = min;
        }
        prev.style.flex = "0 0 " + Math.round(newPrev) + "px";
        next.style.flex = "0 0 " + Math.round(newNext) + "px";
      };

      const onUp = () => {
        splitter.removeEventListener("pointermove", onMove);
        splitter.removeEventListener("pointerup", onUp);
        splitter.removeEventListener("pointercancel", onUp);
        saveHeights();
      };

      splitter.addEventListener("pointermove", onMove);
      splitter.addEventListener("pointerup", onUp);
      splitter.addEventListener("pointercancel", onUp);
    });
  }
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
  return lines.join("\n");
}

async function refreshRuntime(silent) {
  try {
    const res = await jfetch("/api/v1/runtime");
    const rt = (res && res.runtime) || {};
    const controls = rt.controls || {};
    const prefix = String(controls.prefix || "+").trim();
    if (prefix) state.controlPrefix = prefix;
    const cmds = Array.isArray(controls.commands) ? controls.commands : [];
    state.hasKillControl = cmds.some((c) => String(c).indexOf(state.controlPrefix + "kill ") === 0);
    $("runtimeSummaryBox").textContent = buildRuntimeSummary(rt);
    const cfg = rt.config || {};
    const top = {
      path: cfg.path || "",
      available: !!cfg.available,
      error: cfg.error || "",
    };
    $("runtimeConfigBox").textContent = "Config Meta\n" + fmtObj(top) + "\n\nSanitized Config\n" + fmtObj(cfg.sanitized || {});
    updateMessageModeUI();
    renderSubagents(state.subagents);
  } catch (e) {
    if (!silent) $("runtimeSummaryBox").textContent = "ERROR: " + e.message;
  }
}

function updateMessageModeUI() {
  const modeEl = $("messageMode");
  const content = $("content");
  const hint = $("commandHint");
  if (!modeEl || !content || !hint) return;
  const mode = modeEl.value || "normal";
  const p = state.controlPrefix || "+";
  let text = "";
  if (mode === "normal") {
    text = "Queue normal message";
    content.placeholder = "Type message...";
    content.disabled = false;
  } else if (mode === "inject") {
    text = "Control command: " + p + "inject MESSAGE";
    content.placeholder = "MESSAGE for inject";
    content.disabled = false;
  } else if (mode === "first") {
    text = "Control command: " + p + "first MESSAGE";
    content.placeholder = "MESSAGE for force-first";
    content.disabled = false;
  } else if (mode === "append") {
    text = "Control command: " + p + p + "MESSAGE";
    content.placeholder = "MESSAGE to append to last queued message";
    content.disabled = false;
  } else if (mode === "delete") {
    text = "Control command: " + p + "delete";
    content.placeholder = "No message body needed for delete";
    content.disabled = true;
  }
  hint.textContent = text;
}

function setButtonBusy(btn, busy, label) {
  if (!btn) return;
  btn.disabled = !!busy;
  if (typeof label === "string") btn.textContent = label;
}

function setSendOut(text, isError) {
  const out = $("sendOut");
  if (!out) return;
  out.textContent = String(text || "");
  out.className = "small " + (isError ? "bad" : "ok");
}

function describeMainResponse(mode, res) {
  const r = res || {};
  if (r.injected) return "Injected into active run.";
  if (r.queued) return "Queued. Queue ID: " + (r.id || "(unknown)");
  if (mode === "delete") return "Delete command sent.";
  if (mode === "kill") return "Kill command sent.";
  return "Sent.";
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
const messageMode = $("messageMode");
if (messageMode) messageMode.addEventListener("change", updateMessageModeUI);
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
const sessionKey = $("sessionKey");
if (sessionKey) sessionKey.addEventListener("change", () => selectSession(sessionKey.value.trim()));
const clearAgentBtn = $("clearAgentBtn");
if (clearAgentBtn) clearAgentBtn.addEventListener("click", clearAgentFilter);
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
window.delItem = delItem;
window.moveItem = moveItem;
window.selectSession = selectSession;
window.selectAgent = selectAgent;
window.clearAgentFilter = clearAgentFilter;
window.killSubagent = killSubagent;

initTheme();
initSplitter();
initRightRowSplitters();
updateMessageModeUI();
refreshInbound();
refreshSubagents();
refreshSessions();
refreshHistory();
refreshRuntime(true);
connectEvents();
`

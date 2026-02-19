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
    :root { --bg:#0f172a; --panel:#111827; --muted:#94a3b8; --text:#e5e7eb; --acc:#22c55e; --warn:#f59e0b; --bad:#ef4444; }
    body { margin:0; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; background:linear-gradient(180deg,#0b1220,#0f172a); color:var(--text); }
    .wrap { padding:14px; display:grid; grid-template-columns: 1fr 1fr; gap:12px; }
    .card { background:var(--panel); border:1px solid #1f2937; border-radius:10px; padding:10px; }
    h2 { margin:0 0 8px; font-size:14px; color:#cbd5e1; }
    input,textarea,button { font:inherit; }
    input,textarea { width:100%; background:#0b1220; color:var(--text); border:1px solid #334155; border-radius:8px; padding:8px; box-sizing:border-box; }
    textarea { min-height:80px; resize:vertical; }
    button { background:#1e293b; color:var(--text); border:1px solid #334155; border-radius:8px; padding:6px 10px; cursor:pointer; }
    button:hover { border-color:#64748b; }
    .row { display:flex; gap:8px; align-items:center; margin:6px 0; }
    .grow { flex:1; }
    .small { font-size:12px; color:var(--muted); }
    table { width:100%; border-collapse:collapse; font-size:12px; }
    td,th { border-top:1px solid #1f2937; padding:6px; text-align:left; vertical-align:top; }
    .ok { color:var(--acc); } .warn { color:var(--warn); } .bad { color:var(--bad); }
    pre { white-space:pre-wrap; word-break:break-word; background:#0b1220; border:1px solid #1f2937; border-radius:8px; padding:8px; margin:0; max-height:260px; overflow:auto; }
    @media (max-width: 980px) { .wrap { grid-template-columns: 1fr; } }
  </style>
</head>
<body>
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
      <div class="row"><button id="refreshInbound">Refresh</button><span class="small">Editable pending queue</span></div>
      <table id="inboundTable"></table>
    </section>

    <section class="card">
      <h2>Session History</h2>
      <div class="row"><button id="refreshHistory">Refresh</button></div>
      <pre id="historyBox"></pre>
    </section>

    <section class="card">
      <h2>Subagents</h2>
      <div class="row"><button id="refreshSubagents">Refresh</button><span class="small">List + detail</span></div>
      <table id="subagentTable"></table>
      <pre id="subagentDetail"></pre>
    </section>
  </div>
  <script src="/dashboard/app.js"></script>
</body>
</html>
`

const dashboardJS = `
const $ = (id) => document.getElementById(id);

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
    await refreshInbound();
    await refreshHistory();
  } catch (e) {
    $("sendOut").textContent = "ERROR: "+e.message;
  }
}

async function refreshInbound() {
  const t = $("inboundTable");
  try {
    const res = await jfetch("/api/v1/inbound");
    const items = res.items || [];
    let html = "<tr><th>ID</th><th>Session</th><th>Content</th><th>Actions</th></tr>";
    for (const it of items) {
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
  } catch (e) {
    t.innerHTML = "<tr><td class='bad'>"+e.message+"</td></tr>";
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
    const items = res.items || [];
    box.textContent = items.map((m,i) => "["+i+"] "+m.role+": "+(m.content||"")).join("\\n\\n");
  } catch (e) {
    box.textContent = "ERROR: "+e.message;
  }
}

async function refreshSubagents() {
  const t = $("subagentTable");
  const detail = $("subagentDetail");
  try {
    const res = await jfetch("/api/v1/subagents");
    const items = res.items || [];
    let html = "<tr><th>ID</th><th>Status</th><th>Label</th><th>Pending</th><th></th></tr>";
    for (const it of items) {
      html += "<tr>";
      html += "<td>"+it.id+"</td>";
      html += "<td>"+it.status+"</td>";
      html += "<td>"+(it.label||"")+"</td>";
      html += "<td>"+(it.pending||0)+"</td>";
      html += "<td><button onclick='loadSubagent(\""+it.id+"\")'>view</button></td>";
      html += "</tr>";
    }
    t.innerHTML = html;
    if (items.length === 0) detail.textContent = "No subagents.";
  } catch (e) {
    t.innerHTML = "<tr><td class='bad'>"+e.message+"</td></tr>";
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

$("sendBtn").addEventListener("click", sendMessage);
$("refreshInbound").addEventListener("click", refreshInbound);
$("refreshHistory").addEventListener("click", refreshHistory);
$("refreshSubagents").addEventListener("click", refreshSubagents);

window.patchItem = patchItem;
window.moveTop = moveTop;
window.delItem = delItem;
window.loadSubagent = loadSubagent;

refreshInbound();
refreshHistory();
refreshSubagents();
setInterval(refreshInbound, 3000);
setInterval(refreshHistory, 5000);
setInterval(refreshSubagents, 5000);
`

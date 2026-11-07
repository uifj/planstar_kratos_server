package server

import "net/http"

// adminConsole 返回运营后台单页（举报处理台 + 企划审核）。
// 页面只依赖同源管理端 JSON API，不引入外部静态资源。
func adminConsole(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(adminConsoleHTML))
}

const adminConsoleHTML = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>企划星 · 运营后台</title>
<style>
  :root { --bg:#0f1216; --panel:#171c22; --line:#252d37; --text:#e6edf3; --muted:#8b98a5;
          --accent:#4f8cff; --ok:#2ea043; --warn:#d29922; --danger:#f85149; }
  * { box-sizing: border-box; }
  body { margin:0; background:var(--bg); color:var(--text);
         font: 14px/1.6 -apple-system, "Segoe UI", "Microsoft YaHei", sans-serif; }
  header { padding:20px 28px; border-bottom:1px solid var(--line); display:flex; align-items:baseline; gap:12px; }
  header h1 { font-size:18px; margin:0; }
  header span { color:var(--muted); font-size:12px; }
  main { max-width:1180px; margin:0 auto; padding:24px 20px 60px; display:grid; gap:22px; }
  section { background:var(--panel); border:1px solid var(--line); border-radius:10px; padding:18px 20px; }
  section h2 { margin:0 0 14px; font-size:15px; font-weight:600; }
  .row { display:flex; flex-wrap:wrap; gap:10px; align-items:center; margin-bottom:14px; }
  label { color:var(--muted); font-size:12px; }
  select, input { background:#0d1117; color:var(--text); border:1px solid var(--line);
                  border-radius:6px; padding:6px 9px; font-size:13px; }
  input.note { width:210px; }
  button { background:#212a35; color:var(--text); border:1px solid var(--line);
           border-radius:6px; padding:6px 12px; font-size:13px; cursor:pointer; }
  button:hover { border-color:#3a4756; }
  button.primary { background:var(--accent); border-color:var(--accent); color:#fff; }
  button.ok { background:rgba(46,160,67,.16); border-color:var(--ok); color:#7ee787; }
  button.danger { background:rgba(248,81,73,.14); border-color:var(--danger); color:#ff9d97; }
  table { width:100%; border-collapse:collapse; font-size:13px; }
  th, td { text-align:left; padding:9px 8px; border-bottom:1px solid var(--line); vertical-align:top; }
  th { color:var(--muted); font-weight:500; font-size:12px; }
  td.desc { max-width:260px; color:#c9d4df; }
  .badge { display:inline-block; padding:1px 8px; border-radius:999px; font-size:12px; border:1px solid var(--line); }
  .badge.pending { color:var(--warn); border-color:rgba(210,153,34,.5); }
  .badge.accepted { color:#7ee787; border-color:rgba(46,160,67,.5); }
  .badge.rejected { color:#ff9d97; border-color:rgba(248,81,73,.5); }
  .muted { color:var(--muted); font-size:12px; }
  .error { color:#ff9d97; font-size:13px; min-height:20px; }
  .chips { display:flex; flex-wrap:wrap; gap:6px; margin:8px 0 0; }
  .chip { padding:2px 9px; border-radius:999px; font-size:12px;
          background:rgba(248,81,73,.12); border:1px solid rgba(248,81,73,.45); color:#ff9d97; }
  .chip.clean { background:rgba(46,160,67,.12); border-color:rgba(46,160,67,.45); color:#7ee787; }
  .project { margin-top:12px; padding:12px 14px; background:#12171d; border:1px solid var(--line); border-radius:8px; }
  .project .title { font-size:15px; font-weight:600; margin-bottom:4px; }
  .empty { color:var(--muted); padding:12px 0; }
</style>
</head>
<body>
<header>
  <h1>企划星 · 运营后台</h1>
  <span>仅内网管理端口可用 · 数据实时读取</span>
</header>
<main>
  <section>
    <h2>举报处理台</h2>
    <div class="row">
      <label for="reportStatus">状态</label>
      <select id="reportStatus">
        <option value="">全部</option>
        <option value="pending">待处理</option>
        <option value="accepted">举报成立</option>
        <option value="rejected">举报不成立</option>
      </select>
      <label for="reportLimit">条数</label>
      <input id="reportLimit" type="number" min="1" max="200" value="50" style="width:80px">
      <button class="primary" id="reportRefresh">刷新</button>
      <span class="muted" id="reportSummary"></span>
      <span class="error" id="reportError"></span>
    </div>
    <div id="reportList" class="empty">加载中…</div>
  </section>

  <section>
    <h2>企划审核（发布闸门）</h2>
    <div class="row">
      <label for="projectId">企划 ID</label>
      <input id="projectId" type="number" min="1" placeholder="例如 4001" style="width:130px">
      <button class="primary" id="projectLoad">查询</button>
      <span class="muted">通过审核前会强制校验品类黑名单</span>
    </div>
    <div id="projectError" class="error"></div>
    <div id="projectPanel" class="empty">输入企划 ID 查看审核状态</div>
  </section>
</main>
<script>
(function () {
  "use strict";

  var STATUS_LABEL = { pending: "待处理", accepted: "举报成立", rejected: "举报不成立" };
  var PUBLISH_LABEL = { 0: "草稿", 1: "已发布", 2: "审核中", 3: "已驳回" };

  function el(tag, className, text) {
    var node = document.createElement(tag);
    if (className) node.className = className;
    if (text !== undefined && text !== null) node.textContent = String(text);
    return node;
  }

  async function api(path, options) {
    var res = await fetch(path, options);
    var data = null;
    try { data = await res.json(); } catch (e) { data = null; }
    if (!res.ok) {
      var msg = (data && data.message) || ("请求失败：HTTP " + res.status);
      if (data && data.reasons && data.reasons.length) {
        msg += "（" + data.reasons.map(function (r) { return r.rule + ":" + r.keyword; }).join("，") + "）";
      }
      throw new Error(msg);
    }
    return data;
  }

  function fmtTime(value) {
    if (!value) return "—";
    var d = new Date(value);
    return isNaN(d.getTime()) ? String(value) : d.toLocaleString("zh-CN", { hour12: false });
  }

  function badge(status) {
    var node = el("span", "badge " + status, STATUS_LABEL[status] || status);
    return node;
  }

  async function loadReports() {
    var status = document.getElementById("reportStatus").value;
    var limit = document.getElementById("reportLimit").value || "50";
    var query = new URLSearchParams({ limit: limit });
    if (status) query.set("status", status);
    var errorBox = document.getElementById("reportError");
    errorBox.textContent = "";
    try {
      var data = await api("/admin/reports?" + query.toString());
      renderReports(data.items || [], data.total || 0);
    } catch (err) {
      errorBox.textContent = err.message;
      document.getElementById("reportList").textContent = "加载失败";
    }
  }

  function renderReports(items, total) {
    document.getElementById("reportSummary").textContent = "共 " + total + " 条";
    var host = document.getElementById("reportList");
    host.textContent = "";
    if (!items.length) {
      host.className = "empty";
      host.textContent = "当前筛选下没有举报单";
      return;
    }
    host.className = "";
    var table = el("table");
    var head = el("tr");
    ["举报ID", "对象", "企划", "原因", "补充说明", "状态", "处理结论", "创建时间", "操作"]
      .forEach(function (name) { head.appendChild(el("th", "", name)); });
    var thead = el("thead");
    thead.appendChild(head);
    table.appendChild(thead);

    var tbody = el("tbody");
    items.forEach(function (item) {
      var tr = el("tr");
      tr.appendChild(el("td", "", item.reportId));
      tr.appendChild(el("td", "", item.targetType + " #" + item.targetId));
      tr.appendChild(el("td", "", item.projectId || "—"));
      tr.appendChild(el("td", "", item.reasonCode));
      tr.appendChild(el("td", "desc", item.description || "—"));

      var statusCell = el("td");
      statusCell.appendChild(badge(item.status));
      tr.appendChild(statusCell);
      tr.appendChild(el("td", "desc", item.handleNote || "—"));
      tr.appendChild(el("td", "muted", fmtTime(item.gmtCreate)));

      var actions = el("td");
      var note = el("input", "note");
      note.placeholder = "处理结论（可留空）";
      actions.appendChild(note);
      var accept = el("button", "ok", "成立");
      accept.onclick = function () { handleReport(item.reportId, "accepted", note.value); };
      var reject = el("button", "danger", "不成立");
      reject.style.marginLeft = "6px";
      reject.onclick = function () { handleReport(item.reportId, "rejected", note.value); };
      actions.appendChild(accept);
      actions.appendChild(reject);
      tr.appendChild(actions);

      tbody.appendChild(tr);
    });
    table.appendChild(tbody);
    host.appendChild(table);
  }

  async function handleReport(id, status, note) {
    var errorBox = document.getElementById("reportError");
    errorBox.textContent = "";
    try {
      await api("/admin/reports/" + encodeURIComponent(id) + "/handle", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ status: status, note: note || "" })
      });
      await loadReports();
    } catch (err) {
      errorBox.textContent = err.message;
    }
  }

  function projectIdValue() {
    var raw = document.getElementById("projectId").value.trim();
    return raw ? raw : "";
  }

  async function loadProject() {
    var id = projectIdValue();
    var errorBox = document.getElementById("projectError");
    errorBox.textContent = "";
    if (!id) {
      errorBox.textContent = "请先输入企划 ID";
      return;
    }
    try {
      var mod = await api("/admin/projects/" + encodeURIComponent(id));
      renderProject(mod);
    } catch (err) {
      errorBox.textContent = err.message;
      document.getElementById("projectPanel").className = "empty";
      document.getElementById("projectPanel").textContent = "查询失败";
    }
  }

  function renderProject(mod) {
    var host = document.getElementById("projectPanel");
    host.textContent = "";
    host.className = "project";

    host.appendChild(el("div", "title", mod.title || ("企划 " + mod.projectId)));
    host.appendChild(el("div", "muted",
      "ID " + mod.projectId + " · 品类 " + (mod.categoryName || mod.categoryId) +
      " · 筹资状态 " + (mod.fundingStatus || "—") +
      " · 发布状态 " + (PUBLISH_LABEL[mod.publishStatus] || mod.publishStatus)));

    var chips = el("div", "chips");
    if (mod.hits && mod.hits.length) {
      mod.hits.forEach(function (hit) {
        chips.appendChild(el("span", "chip", hit.rule + " · " + hit.keyword));
      });
    } else {
      chips.appendChild(el("span", "chip clean", "未命中黑名单"));
    }
    host.appendChild(chips);

    var actions = el("div", "row");
    actions.style.marginTop = "12px";
    actions.style.marginBottom = "0";
    var approve = el("button", "ok", "通过并发布");
    approve.disabled = !!(mod.hits && mod.hits.length);
    approve.onclick = function () { reviewProject("approve"); };
    var reject = el("button", "danger", "驳回");
    reject.onclick = function () { reviewProject("reject"); };
    actions.appendChild(approve);
    actions.appendChild(reject);
    if (mod.hits && mod.hits.length) {
      actions.appendChild(el("span", "muted", "命中黑名单，通过按钮已禁用"));
    }
    host.appendChild(actions);
  }

  async function reviewProject(decision) {
    var id = projectIdValue();
    var errorBox = document.getElementById("projectError");
    errorBox.textContent = "";
    if (!id) {
      errorBox.textContent = "请先输入企划 ID";
      return;
    }
    try {
      var mod = await api("/admin/projects/" + encodeURIComponent(id) + "/review", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ decision: decision })
      });
      renderProject(mod);
    } catch (err) {
      errorBox.textContent = err.message;
    }
  }

  document.getElementById("reportRefresh").onclick = loadReports;
  document.getElementById("reportStatus").onchange = loadReports;
  document.getElementById("projectLoad").onclick = loadProject;
  document.getElementById("projectId").addEventListener("keydown", function (e) {
    if (e.key === "Enter") loadProject();
  });

  loadReports();
})();
</script>
</body>
</html>
`

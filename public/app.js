// Sharing Vision frontend — same-origin: the Go backend serves these
// static files, so API calls hit the same host (no CORS, no hardcoded port).
const API_BASE = "";

// ---------- helpers ----------
let toastTimer = null;
function toast(msg, isError = false) {
  const t = document.getElementById("toast");
  const icon = document.getElementById("toast-icon");
  document.getElementById("toast-msg").textContent = msg;
  icon.textContent = isError ? "⚠️" : "✅";
  t.classList.remove("hidden", "bg-gray-900", "bg-red-600", "bg-green-600");
  t.classList.add(isError ? "bg-red-600" : "bg-green-600");
  // slide-in
  requestAnimationFrame(() => {
    t.classList.remove("opacity-0", "-translate-y-4");
  });
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => {
    t.classList.add("opacity-0", "-translate-y-4");
    setTimeout(() => t.classList.add("hidden"), 300);
  }, 3000);
}

async function api(path, options = {}) {
  const res = await fetch(API_BASE + path, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });
  if (!res.ok) {
    let detail = "";
    try { detail = (await res.json()).message || ""; } catch (_) {}
    throw new Error(detail || ("HTTP " + res.status));
  }
  const text = await res.text();
  return text ? JSON.parse(text) : null;
}

const STATUS_LABEL = { publish: "Published", draft: "Drafts", thrash: "Trashed" };

// ---------- view switching ----------
function showView(id) {
  document.querySelectorAll(".view").forEach(v => v.classList.add("hidden"));
  document.getElementById("view-" + id).classList.remove("hidden");
  document.querySelectorAll(".nav-btn").forEach(b =>
    b.classList.toggle("active", b.dataset.view === id.replace("edit", "posts") || b.dataset.view === id));
}

document.querySelectorAll(".nav-btn").forEach(btn => {
  btn.addEventListener("click", () => {
    document.querySelectorAll(".nav-btn").forEach(b => {
      b.classList.remove("active", "bg-blue-600", "text-white");
      b.classList.add("bg-white");
    });
    btn.classList.add("active", "bg-blue-600", "text-white");
    btn.classList.remove("bg-white");
    const v = btn.dataset.view;
    if (v === "posts") loadPosts(currentStatus);
    if (v === "preview") loadPreview(1);
    showView(v);
  });
});

// ---------- ALL POSTS (tabs) ----------
let currentStatus = "publish";
const POSTS_PER_PAGE = 10;
let postsPage = 1;

document.querySelectorAll(".tab").forEach(tab => {
  tab.addEventListener("click", () => {
    document.querySelectorAll(".tab").forEach(t => {
      t.classList.remove("active", "bg-blue-600", "text-white");
      t.classList.add("bg-white");
    });
    tab.classList.add("active", "bg-blue-600", "text-white");
    tab.classList.remove("bg-white");
    currentStatus = tab.dataset.status;
    postsPage = 1; // reset to first page when switching tabs
    loadPosts(currentStatus);
  });
});

async function loadPosts(status) {
  const body = document.getElementById("posts-body");
  body.innerHTML = '<tr><td colspan="3" class="text-center text-slate-400 p-8">Loading…</td></tr>';
  try {
    let rows, total;
    if (status === "thrash") {
      // Trashed tab: fetch the whole trash table (small), client-paginate.
      const all = await api("/trash/1000/0");
      rows = all || [];
      total = rows.length;
      const start = (postsPage - 1) * POSTS_PER_PAGE;
      rows = rows.slice(start, start + POSTS_PER_PAGE);
    } else {
      // Published / Drafts: server-side pagination by status.
      const offset = (postsPage - 1) * POSTS_PER_PAGE;
      rows = await api("/article/status/" + status + "/" + POSTS_PER_PAGE + "/" + offset);
      rows = rows || [];
      // We can't get a cheap total count here without another call, so we
      // infer "more pages" from whether we got a full page back.
      total = offset + rows.length + (rows.length === POSTS_PER_PAGE ? 1 : 0);
    }
    const totalPages = Math.max(1, Math.ceil(total / POSTS_PER_PAGE));
    if (rows.length === 0) {
      body.innerHTML = '<tr><td colspan="3" class="text-center text-slate-400 p-8">No ' +
        (STATUS_LABEL[status] || status) + ' articles.</td></tr>';
    } else {
      body.innerHTML = "";
      rows.forEach(a => {
        const tr = document.createElement("tr");
        if (status === "thrash") {
          tr.innerHTML = `
            <td class="p-4 align-middle font-medium text-slate-800">${escapeHtml(a.title)}</td>
            <td class="p-4 align-middle text-slate-600">${escapeHtml(a.category)}</td>
            <td class="p-4 align-middle whitespace-nowrap">
              <button class="icon-btn icon-restore" title="Restore" data-restore="${a.id}">↩️</button>
              <button class="icon-btn icon-delete" title="Delete Permanently" data-delete="${a.id}">❌</button>
            </td>`;
        } else {
          tr.innerHTML = `
            <td class="p-4 align-middle font-medium text-slate-800">${escapeHtml(a.title)}</td>
            <td class="p-4 align-middle text-slate-600">${escapeHtml(a.category)}</td>
            <td class="p-4 align-middle whitespace-nowrap">
              <button class="icon-btn icon-edit" title="Edit" data-edit="${a.id}">✏️</button>
              <button class="icon-btn icon-thrash" title="Move to Trash" data-thrash="${a.id}">🗑️</button>
            </td>`;
        }
        body.appendChild(tr);
      });
      body.querySelectorAll("[data-edit]").forEach(b =>
        b.addEventListener("click", () => openEdit(b.dataset.edit)));
      body.querySelectorAll("[data-thrash]").forEach(b =>
        b.addEventListener("click", () => moveToTrash(b.dataset.thrash)));
      body.querySelectorAll("[data-restore]").forEach(b =>
        b.addEventListener("click", () => restoreFromTrash(b.dataset.restore)));
      body.querySelectorAll("[data-delete]").forEach(b =>
        b.addEventListener("click", () => deletePermanently(b.dataset.delete)));
    }
    // pager
    document.getElementById("posts-page-info").textContent = "Page " + postsPage + " / " + totalPages;
    document.getElementById("posts-prev").disabled = postsPage <= 1;
    document.getElementById("posts-next").disabled = postsPage >= totalPages;
  } catch (e) {
    body.innerHTML = '<tr><td colspan="3" class="text-center text-red-500 p-8">Error: ' + escapeHtml(e.message) + '</td></tr>';
  }
}

document.getElementById("posts-prev").addEventListener("click", () => {
  if (postsPage > 1) { postsPage--; loadPosts(currentStatus); }
});
document.getElementById("posts-next").addEventListener("click", () => {
  postsPage++; loadPosts(currentStatus);
});

async function moveToTrash(id) {
  try {
    await api("/article/" + id + "/thrash", { method: "POST" });
    toast("Article moved to Trash");
    loadPosts(currentStatus);
  } catch (e) { toast("Failed: " + e.message, true); }
}

async function restoreFromTrash(id) {
  try {
    await api("/trash/" + id + "/restore", { method: "POST" });
    toast("Restored to Articles (as Draft)");
    loadPosts(currentStatus);
  } catch (e) { toast("Failed: " + e.message, true); }
}

async function deletePermanently(id) {
  if (!confirm("Delete this article permanently? This cannot be undone.")) return;
  try {
    await api("/trash/" + id, { method: "DELETE" });
    toast("Permanently deleted");
    loadPosts(currentStatus);
  } catch (e) { toast("Failed: " + e.message, true); }
}

// ---------- ADD NEW ----------
document.querySelectorAll("#add-form [data-action]").forEach(btn => {
  btn.addEventListener("click", async () => {
    const f = document.getElementById("add-form");
    const payload = {
      title: f.title.value.trim(),
      content: f.content.value.trim(),
      category: f.category.value.trim(),
      status: btn.dataset.action, // publish | draft
    };
    try {
      await api("/article/", { method: "POST", body: JSON.stringify(payload) });
      toast("Saved as " + btn.dataset.action);
      f.reset();
      currentStatus = payload.status;
      syncTabHighlight(currentStatus);
      await loadPosts(currentStatus);
      showView("posts");
    } catch (e) { toast("Failed: " + e.message, true); }
  });
});

// ---------- EDIT ----------
let editFromStatus = "draft";

async function openEdit(id) {
  try {
    const a = await api("/article/" + id); // GET /article/:id (note: backend param is :a)
    const f = document.getElementById("edit-form");
    f.id.value = a.id;
    f.title.value = a.title;
    f.content.value = a.content;
    f.category.value = a.category;
    editFromStatus = a.status; // remember the tab we came from, so we can return to it
    showView("edit");
  } catch (e) { toast("Failed to load: " + e.message, true); }
}

document.querySelectorAll("#edit-form [data-action]").forEach(btn => {
  btn.addEventListener("click", async () => {
    const f = document.getElementById("edit-form");
    const id = f.id.value;
    const payload = {
      title: f.title.value.trim(),
      content: f.content.value.trim(),
      category: f.category.value.trim(),
      status: btn.dataset.action,
    };
    try {
      await api("/article/" + id, { method: "PUT", body: JSON.stringify(payload) });
      toast("Updated as " + btn.dataset.action);
      // Return to the tab the user was viewing before editing, and refresh it.
      // If the status changed, also refresh the destination tab so the move is visible.
      currentStatus = editFromStatus;
      syncTabHighlight(currentStatus);
      await loadPosts(currentStatus);
      if (payload.status !== editFromStatus) {
        currentStatus = payload.status;
        syncTabHighlight(currentStatus);
        await loadPosts(currentStatus);
      }
      showView("posts");
    } catch (e) { toast("Failed: " + e.message, true); }
  });
});

// Keeps the tab buttons' highlight in sync with the status being shown.
function syncTabHighlight(status) {
  document.querySelectorAll(".tab").forEach(t => {
    const on = t.dataset.status === status;
    t.classList.toggle("active", on);
    t.classList.toggle("bg-blue-600", on);
    t.classList.toggle("text-white", on);
    t.classList.toggle("bg-white", !on);
  });
}

// ---------- PREVIEW (published, paginated) ----------
const PER_PAGE = 5;
let previewPage = 1;

async function loadPreview(page) {
  const box = document.getElementById("preview-list");
  box.innerHTML = '<p class="muted">Loading…</p>';
  try {
    const all = await api("/article/1000/0");
    const published = (all || [])
      .filter(a => a.status === "publish")
      .sort((a, b) => b.id - a.id); // descending by id (newest first)
    const totalPages = Math.max(1, Math.ceil(published.length / PER_PAGE));
    previewPage = Math.min(Math.max(1, page), totalPages);
    const start = (previewPage - 1) * PER_PAGE;
    const slice = published.slice(start, start + PER_PAGE);

    if (slice.length === 0) {
      box.innerHTML = '<p class="text-center text-slate-400 py-10">No published articles yet.</p>';
    } else {
      box.innerHTML = slice.map(a => `
        <article class="bg-white border border-slate-200 rounded-lg p-5 shadow-sm hover:shadow-md transition">
          <div class="flex items-center gap-2 mb-2">
            <span class="inline-block text-xs font-semibold uppercase tracking-wide text-blue-700 bg-blue-50 border border-blue-100 rounded-full px-3 py-1">${escapeHtml(a.category)}</span>
            <span class="text-xs text-slate-400">#${a.id}</span>
          </div>
          <h3 class="text-lg font-bold text-slate-800 mb-2">${escapeHtml(a.title)}</h3>
          <p class="text-slate-600 leading-relaxed whitespace-pre-line">${escapeHtml(a.content)}</p>
        </article>`).join("");
    }
    document.getElementById("page-info").textContent =
      "Page " + previewPage + " / " + totalPages;
    document.getElementById("prev-page").disabled = previewPage <= 1;
    document.getElementById("next-page").disabled = previewPage >= totalPages;
  } catch (e) {
    box.innerHTML = '<p class="text-center text-red-500 py-10">Error: ' + escapeHtml(e.message) + '</p>';
  }
}

document.getElementById("prev-page").addEventListener("click", () => loadPreview(previewPage - 1));
document.getElementById("next-page").addEventListener("click", () => loadPreview(previewPage + 1));

// ---------- util ----------
function escapeHtml(s) {
  return String(s).replace(/[&<>"']/g, c =>
    ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]));
}

// ---------- init ----------
loadPosts("publish");

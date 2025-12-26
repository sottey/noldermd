const apiBase = "/api/v1";

const app = document.getElementById("app");
const treeContainer = document.getElementById("tree-container");
const refreshBtn = document.getElementById("refresh-btn");
const editor = document.getElementById("editor");
const preview = document.getElementById("preview");
const notePath = document.getElementById("note-path");
const saveBtn = document.getElementById("save-btn");
const viewButtons = Array.from(document.querySelectorAll(".view-btn"));
const contextMenu = document.getElementById("context-menu");
const sidebar = document.getElementById("sidebar");
const sidebarResizer = document.getElementById("sidebar-resizer");
const paneResizer = document.getElementById("pane-resizer");
const editorPane = document.getElementById("editor-pane");
const previewPane = document.getElementById("preview-pane");
const mainContent = document.getElementById("main-content");
const searchInput = document.getElementById("search-input");
const searchBtn = document.getElementById("search-btn");
const searchResults = document.getElementById("search-results");

let currentNotePath = "";
let currentTree = null;
let isDirty = false;

function setView(view) {
  app.dataset.view = view;
  viewButtons.forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.view === view);
  });
}

function escapeHtml(value) {
  if (value === null || value === undefined) {
    return "";
  }
  return String(value)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/\"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

function renderMarkdown(text) {
  if (!window.marked) {
    return `<pre>${escapeHtml(text)}</pre>`;
  }

  const renderer = new marked.Renderer();
  renderer.image = (href, title, text) => {
    const resolved = resolveAssetPath(href);
    const titleAttr = title ? ` title="${escapeHtml(title)}"` : "";
    return `<img src="${resolved}" alt="${escapeHtml(text || "")}"${titleAttr} />`;
  };

  renderer.link = (href, title, text) => {
    const safeHref = escapeHtml(href || "");
    const titleAttr = title ? ` title="${escapeHtml(title)}"` : "";
    const target = safeHref.startsWith("http") ? " target=\"_blank\" rel=\"noopener\"" : "";
    return `<a href="${safeHref}"${titleAttr}${target}>${text}</a>`;
  };

  marked.setOptions({
    gfm: true,
    breaks: true,
    tables: true,
    renderer,
    langPrefix: "language-",
  });

  return marked.parse(text);
}

function applyHighlighting() {
  if (!window.hljs) {
    return;
  }
  preview.querySelectorAll("pre code").forEach((block) => {
    hljs.highlightElement(block);
  });
}

async function apiFetch(path, options = {}) {
  const response = await fetch(`${apiBase}${path}`, {
    headers: {
      "Content-Type": "application/json",
    },
    ...options,
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: "Request failed" }));
    throw new Error(error.error || "Request failed");
  }

  if (response.status === 204) {
    return null;
  }

  return response.json();
}

function buildTreeNode(node, depth = 0) {
  const wrapper = document.createElement("div");
  wrapper.className = `tree-node ${node.type}`;

  const row = document.createElement("div");
  row.className = "node-row";
  row.style.paddingLeft = `${12 + depth * 12}px`;
  row.dataset.path = node.path;
  row.dataset.type = node.type;

  if (node.type === "folder") {
    const caret = document.createElement("span");
    caret.className = "caret";
    row.appendChild(caret);
  } else {
    const spacer = document.createElement("span");
    spacer.style.width = "10px";
    spacer.style.height = "10px";
    row.appendChild(spacer);
  }

  const name = document.createElement("span");
  name.className = "node-name";
  name.textContent = node.name || "(root)";
  row.appendChild(name);

  wrapper.appendChild(row);

  if (node.type === "folder") {
    if (depth > 0) {
      wrapper.classList.add("collapsed");
    }
    const children = document.createElement("div");
    children.className = "node-children";

    (node.children || []).forEach((child) => {
      children.appendChild(buildTreeNode(child, depth + 1));
    });

    wrapper.appendChild(children);

    row.addEventListener("click", () => {
      wrapper.classList.toggle("collapsed");
    });

    row.addEventListener("contextmenu", (event) => {
      event.preventDefault();
      showContextMenu(event.clientX, event.clientY, [
        {
          label: "Edit",
          action: () => renameFolder(node.path),
        },
        {
          label: "New Note",
          action: () => createNote(node.path),
        },
        {
          label: "New Child Folder",
          action: () => createFolder(node.path),
        },
        {
          label: "Delete",
          action: () => deleteFolder(node.path),
        },
      ]);
    });
  } else {
    row.addEventListener("click", (event) => {
      event.stopPropagation();
      openNote(node.path);
    });
  }

  return wrapper;
}

function renderTree(tree) {
  treeContainer.innerHTML = "";
  if (!tree) {
    return;
  }

  const rootNode = buildTreeNode(tree, 0);
  rootNode.classList.remove("collapsed");
  treeContainer.appendChild(rootNode);
  setActiveNode(currentNotePath);
}

function setActiveNode(path) {
  const rows = treeContainer.querySelectorAll(".node-row");
  rows.forEach((row) => {
    row.classList.toggle("active", row.dataset.path === path && row.dataset.type === "file");
  });
}

function expandToPath(path) {
  if (!path) {
    return;
  }
  const parts = path.split("/").filter(Boolean);
  let current = "";
  parts.slice(0, -1).forEach((part) => {
    current = current ? `${current}/${part}` : part;
    const row = treeContainer.querySelector(`.node-row[data-path="${CSS.escape(current)}"]`);
    if (row) {
      const wrapper = row.closest(".tree-node.folder");
      if (wrapper) {
        wrapper.classList.remove("collapsed");
      }
    }
  });
}

async function loadTree(path = "") {
  try {
    const query = path ? `?path=${encodeURIComponent(path)}` : "";
    currentTree = await apiFetch(`/tree${query}`);
    renderTree(currentTree);
  } catch (err) {
    alert(err.message);
  }
}

async function openNote(path) {
  try {
    const data = await apiFetch(`/notes?path=${encodeURIComponent(path)}`);
    currentNotePath = data.path;
    notePath.textContent = data.path;
    editor.value = data.content;
    preview.innerHTML = renderMarkdown(data.content);
    applyHighlighting();
    isDirty = false;
    saveBtn.disabled = false;
    expandToPath(currentNotePath);
    setActiveNode(currentNotePath);
  } catch (err) {
    alert(err.message);
  }
}

async function saveNote() {
  if (!currentNotePath) {
    return;
  }
  try {
    saveBtn.disabled = true;
    saveBtn.textContent = "Saving...";
    await apiFetch("/notes", {
      method: "PATCH",
      body: JSON.stringify({
        path: currentNotePath,
        content: editor.value,
      }),
    });
    isDirty = false;
    saveBtn.textContent = "Save";
    saveBtn.disabled = false;
  } catch (err) {
    saveBtn.textContent = "Save";
    saveBtn.disabled = false;
    alert(err.message);
  }
}

function promptForName(label) {
  const name = window.prompt(label);
  if (!name) {
    return "";
  }
  if (name.includes("/") || name.includes("\\")) {
    alert("Names cannot include slashes.");
    return "";
  }
  return name.trim();
}

async function createNote(parentPath = "") {
  const name = promptForName("New note name");
  if (!name) {
    return;
  }
  const path = parentPath ? `${parentPath}/${name}` : name;
  try {
    const data = await apiFetch("/notes", {
      method: "POST",
      body: JSON.stringify({
        path,
        content: "",
      }),
    });
    await loadTree();
    await openNote(data.path);
  } catch (err) {
    alert(err.message);
  }
}

async function createFolder(parentPath = "") {
  const name = promptForName("New folder name");
  if (!name) {
    return;
  }
  const path = parentPath ? `${parentPath}/${name}` : name;
  try {
    await apiFetch("/folders", {
      method: "POST",
      body: JSON.stringify({ path }),
    });
    await loadTree();
  } catch (err) {
    alert(err.message);
  }
}

async function renameFolder(path) {
  if (!path) {
    alert("Root folder cannot be renamed.");
    return;
  }
  const currentName = path.split("/").pop();
  const name = promptForName(`Rename folder (${currentName})`);
  if (!name) {
    return;
  }
  const base = path.split("/").slice(0, -1).join("/");
  const newPath = base ? `${base}/${name}` : name;
  try {
    await apiFetch("/folders", {
      method: "PATCH",
      body: JSON.stringify({ path, newPath }),
    });
    await loadTree();
  } catch (err) {
    alert(err.message);
  }
}

async function deleteFolder(path) {
  if (!path) {
    alert("Root folder cannot be deleted.");
    return;
  }
  const confirmDelete = window.confirm("Delete this folder and all of its contents?");
  if (!confirmDelete) {
    return;
  }
  try {
    await apiFetch(`/folders?path=${encodeURIComponent(path)}`, {
      method: "DELETE",
    });
    await loadTree();
  } catch (err) {
    alert(err.message);
  }
}

function showContextMenu(x, y, items) {
  contextMenu.innerHTML = "";
  items.forEach((item) => {
    const button = document.createElement("button");
    button.type = "button";
    button.textContent = item.label;
    button.addEventListener("click", () => {
      hideContextMenu();
      item.action();
    });
    contextMenu.appendChild(button);
  });
  contextMenu.style.left = `${x}px`;
  contextMenu.style.top = `${y}px`;
  contextMenu.classList.remove("hidden");
}

function hideContextMenu() {
  contextMenu.classList.add("hidden");
}

function renderSearchResults(matches) {
  searchResults.innerHTML = "";
  if (matches.length === 0) {
    const empty = document.createElement("div");
    empty.className = "search-empty";
    empty.textContent = "No matches";
    searchResults.appendChild(empty);
    return;
  }
  matches.forEach((match) => {
    const button = document.createElement("button");
    button.type = "button";
    button.textContent = match.name || match.path.split("/").pop();
    button.title = match.path;
    button.addEventListener("click", () => {
      hideSearchResults();
      openNote(match.path);
    });
    searchResults.appendChild(button);
  });
}

function showSearchResults() {
  searchResults.classList.remove("hidden");
}

function hideSearchResults() {
  searchResults.classList.add("hidden");
}

async function runSearch() {
  const query = searchInput.value.trim();
  if (!query) {
    hideSearchResults();
    return;
  }
  try {
    const matches = await apiFetch(`/search?query=${encodeURIComponent(query)}`);
    renderSearchResults(matches);
    showSearchResults();
  } catch (err) {
    alert(err.message);
  }
}

function resolveAssetPath(href) {
  if (!href) {
    return "";
  }
  if (/^(https?:|data:|\/)/i.test(href)) {
    return href;
  }
  const base = currentNotePath.split("/").slice(0, -1).join("/");
  const combined = base ? `${base}/${href}` : href;
  return `${apiBase}/files?path=${encodeURIComponent(combined)}`;
}

function setupSplitters() {
  const isStacked = () => window.matchMedia("(max-width: 720px)").matches;

  sidebarResizer.addEventListener("mousedown", (event) => {
    event.preventDefault();
    const startX = event.clientX;
    const startY = event.clientY;
    const startWidth = sidebar.getBoundingClientRect().width;
    const startHeight = sidebar.getBoundingClientRect().height;

    function onMove(moveEvent) {
      if (isStacked()) {
        const delta = moveEvent.clientY - startY;
        const newHeight = Math.max(160, startHeight + delta);
        sidebar.style.height = `${newHeight}px`;
      } else {
        const delta = moveEvent.clientX - startX;
        const newWidth = Math.max(220, startWidth + delta);
        sidebar.style.width = `${newWidth}px`;
        document.documentElement.style.setProperty("--sidebar-width", `${newWidth}px`);
      }
    }

    function onUp() {
      document.removeEventListener("mousemove", onMove);
      document.removeEventListener("mouseup", onUp);
    }

    document.addEventListener("mousemove", onMove);
    document.addEventListener("mouseup", onUp);
  });

  paneResizer.addEventListener("mousedown", (event) => {
    event.preventDefault();
    const startX = event.clientX;
    const startY = event.clientY;
    const startWidth = editorPane.getBoundingClientRect().width;
    const startHeight = editorPane.getBoundingClientRect().height;

    function onMove(moveEvent) {
      if (isStacked()) {
        const delta = moveEvent.clientY - startY;
        const newHeight = Math.max(120, startHeight + delta);
        editorPane.style.height = `${newHeight}px`;
      } else {
        const delta = moveEvent.clientX - startX;
        const containerWidth = mainContent.getBoundingClientRect().width;
        const newWidth = Math.min(containerWidth - 200, Math.max(200, startWidth + delta));
        editorPane.style.flex = "0 0 auto";
        editorPane.style.flexBasis = `${newWidth}px`;
        editorPane.style.width = "";
        previewPane.style.flex = "1 1 0";
        previewPane.style.width = "";
      }
    }

    function onUp() {
      document.removeEventListener("mousemove", onMove);
      document.removeEventListener("mouseup", onUp);
    }

    document.addEventListener("mousemove", onMove);
    document.addEventListener("mouseup", onUp);
  });
}

refreshBtn.addEventListener("click", () => loadTree());

editor.addEventListener("input", () => {
  isDirty = true;
  preview.innerHTML = renderMarkdown(editor.value);
  applyHighlighting();
  saveBtn.disabled = !currentNotePath;
});

saveBtn.addEventListener("click", () => saveNote());

viewButtons.forEach((btn) => {
  btn.addEventListener("click", () => setView(btn.dataset.view));
});

window.addEventListener("keydown", (event) => {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "s") {
    event.preventDefault();
    saveNote();
  }
});

treeContainer.addEventListener("contextmenu", (event) => {
  const row = event.target.closest(".node-row");
  if (row) {
    return;
  }
  event.preventDefault();
  showContextMenu(event.clientX, event.clientY, [
    { label: "New Folder", action: () => createFolder() },
    { label: "New Note", action: () => createNote() },
  ]);
});

document.addEventListener("click", () => hideContextMenu());

searchBtn.addEventListener("click", (event) => {
  event.preventDefault();
  runSearch();
});

searchInput.addEventListener("keydown", (event) => {
  if (event.key === "Enter") {
    event.preventDefault();
    runSearch();
  }
});

document.addEventListener("click", (event) => {
  if (!searchResults.contains(event.target) && event.target !== searchBtn) {
    hideSearchResults();
  }
});

setView("split");
setupSplitters();
loadTree();

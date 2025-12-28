const apiBase = "/api/v1";

const app = document.getElementById("app");
const treeContainer = document.getElementById("tree-container");
const refreshBtn = document.getElementById("refresh-btn");
const editor = document.getElementById("editor");
const preview = document.getElementById("preview");
const notePath = document.getElementById("note-path");
const saveBtn = document.getElementById("save-btn");
const viewButtons = Array.from(document.querySelectorAll(".view-btn"));
const viewSelector = document.querySelector(".view-selector");
const contextMenu = document.getElementById("context-menu");
const sidebar = document.getElementById("sidebar");
const sidebarResizer = document.getElementById("sidebar-resizer");
const paneResizer = document.getElementById("pane-resizer");
const editorPane = document.getElementById("editor-pane");
const previewPane = document.getElementById("preview-pane");
const mainContent = document.getElementById("main-content");
const mainHeader = document.querySelector(".main-header");
const searchInput = document.getElementById("search-input");
const searchBtn = document.getElementById("search-btn");
const searchResults = document.getElementById("search-results");
const tagBar = document.getElementById("tag-bar");
const tagAddBtn = document.getElementById("tag-add-btn");
const tagPills = document.getElementById("tag-pills");
const assetPreview = document.getElementById("asset-preview");
const pdfPreview = document.getElementById("pdf-preview");
const csvPreview = document.getElementById("csv-preview");
const summaryPanel = document.getElementById("summary-panel");
const settingsBtn = document.getElementById("settings-btn");
const settingsPanel = document.getElementById("settings-panel");
const settingsDarkMode = document.getElementById("settings-dark-mode");
const settingsDefaultView = document.getElementById("settings-default-view");
const settingsAutosaveEnabled = document.getElementById("settings-autosave-enabled");
const settingsAutosaveInterval = document.getElementById("settings-autosave-interval");
const settingsDefaultFolder = document.getElementById("settings-default-folder");
const settingsDailyFolder = document.getElementById("settings-daily-folder");
const settingsShowTemplates = document.getElementById("settings-show-templates");
const taskEditor = document.getElementById("task-editor");
const taskTitleInput = document.getElementById("task-title");
const taskProjectInput = document.getElementById("task-project");
const taskTagsInput = document.getElementById("task-tags");
const taskDueDateInput = document.getElementById("task-duedate");
const taskPriorityInput = document.getElementById("task-priority");
const taskCompletedInput = document.getElementById("task-completed");
const taskNotesInput = document.getElementById("task-notes");
const taskCreatedText = document.getElementById("task-created");
const taskUpdatedText = document.getElementById("task-updated");

let currentNotePath = "";
let currentActivePath = "";
let currentTree = null;
let currentTags = [];
let currentTasks = [];
let currentTaskId = "";
let currentTask = null;
let currentMode = "note";
let lastNoteView = "split";
let currentSettings = { darkMode: false };
let settingsLoaded = false;
let autosaveTimer = null;
let autosaveInFlight = false;
let isDirty = false;
let syncingScroll = false;
let activeScrollSource = null;
let clearScrollSourceTimer = null;

const tagPalette = [
  "#fde68a",
  "#fecdd3",
  "#bfdbfe",
  "#bbf7d0",
  "#e9d5ff",
  "#fbcfe8",
  "#bae6fd",
  "#fed7aa",
  "#c7d2fe",
  "#a7f3d0",
  "#f5d0fe",
  "#d9f99d",
  "#fecaca",
  "#cbd5f5",
  "#e0f2fe",
  "#fae8ff",
];

function setView(view, force = false) {
  if (!force && viewSelector.classList.contains("hidden")) {
    return;
  }
  app.dataset.view = view;
  viewButtons.forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.view === view);
  });
  if (currentMode === "note") {
    lastNoteView = view;
  }
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

function displayNodeName(node) {
  if (node.type === "file" && node.name && node.name.toLowerCase().endsWith(".md")) {
    return node.name.slice(0, -3);
  }
  return node.name || "(root)";
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

function extractTags(text) {
  if (!text) {
    return [];
  }
  const pattern = /(^|\s)#([A-Za-z]+)\b/g;
  const seen = new Set();
  const tags = [];
  let match;
  while ((match = pattern.exec(text)) !== null) {
    const tag = match[2];
    const key = tag.toLowerCase();
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    tags.push(tag);
  }
  return tags;
}

function renderTagBar(tags) {
  tagPills.innerHTML = "";
  if (!currentNotePath) {
    tagBar.classList.add("hidden");
    return;
  }
  (tags || []).forEach((tag) => {
    const pill = document.createElement("button");
    pill.type = "button";
    pill.className = "tag-pill";
    pill.textContent = `#${tag}`;
    pill.style.backgroundColor = getTagColor(tag);
    pill.addEventListener("click", () => openTagGroup(tag));
    tagPills.appendChild(pill);
  });
  tagBar.classList.remove("hidden");
}

function formatDateTime(value) {
  if (!value) {
    return "";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  return date.toLocaleString();
}

function parseTagsInput(value) {
  return String(value || "")
    .split(",")
    .map((tag) => tag.trim())
    .filter(Boolean);
}

function showTaskEditor() {
  if (currentMode === "note") {
    lastNoteView = app.dataset.view;
  }
  currentMode = "task";
  currentNotePath = "";
  summaryPanel.classList.add("hidden");
  settingsPanel.classList.add("hidden");
  taskEditor.classList.remove("hidden");
  editor.classList.add("hidden");
  preview.classList.add("hidden");
  assetPreview.classList.add("hidden");
  assetPreview.innerHTML = "";
  pdfPreview.classList.add("hidden");
  pdfPreview.innerHTML = "";
  csvPreview.classList.add("hidden");
  csvPreview.innerHTML = "";
  viewSelector.classList.add("hidden");
  viewButtons.forEach((btn) => {
    btn.disabled = true;
  });
  setView("edit", true);
  tagBar.classList.add("hidden");
}

function showNoteEditor() {
  currentMode = "note";
  currentTaskId = "";
  currentTask = null;
  summaryPanel.classList.add("hidden");
  settingsPanel.classList.add("hidden");
  taskEditor.classList.add("hidden");
  editor.classList.remove("hidden");
  viewSelector.classList.remove("hidden");
  viewButtons.forEach((btn) => {
    btn.disabled = false;
  });
  setView(lastNoteView || "split");
}

function clearTaskEditor() {
  taskTitleInput.value = "";
  taskProjectInput.value = "";
  taskTagsInput.value = "";
  taskDueDateInput.value = "";
  taskPriorityInput.value = "3";
  taskCompletedInput.checked = false;
  taskNotesInput.value = "";
  taskCreatedText.textContent = "";
  taskUpdatedText.textContent = "";
  notePath.textContent = "No task selected";
  currentTaskId = "";
  currentTask = null;
  currentActivePath = "";
  isDirty = false;
  saveBtn.disabled = true;
  showTaskEditor();
  setActiveNode(currentActivePath);
}

function showSummary(title, items, action) {
  currentMode = "summary";
  currentNotePath = "";
  currentTaskId = "";
  currentTask = null;
  notePath.textContent = title;
  saveBtn.disabled = true;
  tagBar.classList.add("hidden");
  viewSelector.classList.add("hidden");
  viewButtons.forEach((btn) => {
    btn.disabled = true;
  });
  taskEditor.classList.add("hidden");
  settingsPanel.classList.add("hidden");
  editor.classList.add("hidden");
  preview.classList.add("hidden");
  assetPreview.classList.add("hidden");
  assetPreview.innerHTML = "";
  pdfPreview.classList.add("hidden");
  pdfPreview.innerHTML = "";
  csvPreview.classList.add("hidden");
  csvPreview.innerHTML = "";
  summaryPanel.innerHTML = "";
  summaryPanel.classList.remove("hidden");
  setView("preview", true);

  const header = document.createElement("div");
  header.className = "summary-header";

  const heading = document.createElement("h2");
  heading.className = "summary-title";
  heading.textContent = title;
  header.appendChild(heading);

  if (action && action.label && action.handler) {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "primary summary-action";
    button.textContent = action.label;
    button.addEventListener("click", action.handler);
    header.appendChild(button);
  }

  summaryPanel.appendChild(header);

  const grid = document.createElement("div");
  grid.className = "summary-grid";
  items.forEach((item) => {
    const card = document.createElement("div");
    card.className = "summary-item";
    const label = document.createElement("div");
    label.className = "summary-label";
    label.textContent = item.label;
    const value = document.createElement("div");
    value.className = "summary-value";
    value.textContent = String(item.value);
    card.appendChild(label);
    card.appendChild(value);
    grid.appendChild(card);
  });
  summaryPanel.appendChild(grid);
}

function getDefaultView(value) {
  if (value === "edit" || value === "preview" || value === "split") {
    return value;
  }
  return "split";
}

function applyAutosave(settings) {
  if (autosaveTimer) {
    clearInterval(autosaveTimer);
    autosaveTimer = null;
  }
  if (!settings.autosaveEnabled) {
    return;
  }
  const intervalMs = Math.max(5000, settings.autosaveIntervalSeconds * 1000);
  autosaveTimer = setInterval(async () => {
    if (autosaveInFlight) {
      return;
    }
    if (currentMode !== "note" || !currentNotePath || !isDirty) {
      return;
    }
    autosaveInFlight = true;
    try {
      await saveNote();
    } finally {
      autosaveInFlight = false;
    }
  }, intervalMs);
}

function applySidebarWidth(width) {
  if (!width || Number.isNaN(width)) {
    return;
  }
  const clamped = Math.min(600, Math.max(220, Math.round(width)));
  sidebar.style.width = `${clamped}px`;
  document.documentElement.style.setProperty("--sidebar-width", `${clamped}px`);
  currentSettings.sidebarWidth = clamped;
}

function findFolderNode(tree, path) {
  if (!tree || !path) {
    return null;
  }
  const queue = [tree];
  while (queue.length > 0) {
    const node = queue.shift();
    if (!node) {
      continue;
    }
    if (node.type === "folder" && node.path === path) {
      return node;
    }
    if (node.children && node.children.length > 0) {
      queue.push(...node.children);
    }
  }
  return null;
}

function applySettings(settings) {
  currentSettings = {
    darkMode: !!settings.darkMode,
    defaultView: getDefaultView(settings.defaultView),
    autosaveEnabled: !!settings.autosaveEnabled,
    autosaveIntervalSeconds: Number(settings.autosaveIntervalSeconds) || 30,
    sidebarWidth: Number(settings.sidebarWidth) || 300,
    defaultFolder: settings.defaultFolder || "",
    dailyFolder: settings.dailyFolder || "",
    showTemplates: settings.showTemplates !== false,
  };
  document.body.classList.toggle("theme-dark", currentSettings.darkMode);
  if (settingsDarkMode) {
    settingsDarkMode.checked = currentSettings.darkMode;
  }
  if (settingsDefaultView) {
    settingsDefaultView.value = currentSettings.defaultView;
  }
  if (settingsAutosaveEnabled) {
    settingsAutosaveEnabled.checked = currentSettings.autosaveEnabled;
  }
  if (settingsAutosaveInterval) {
    settingsAutosaveInterval.value = String(currentSettings.autosaveIntervalSeconds);
  }
  if (settingsDefaultFolder) {
    settingsDefaultFolder.value = currentSettings.defaultFolder;
  }
  if (settingsDailyFolder) {
    settingsDailyFolder.value = currentSettings.dailyFolder;
  }
  if (settingsShowTemplates) {
    settingsShowTemplates.checked = currentSettings.showTemplates;
  }
  applyAutosave(currentSettings);
  applySidebarWidth(currentSettings.sidebarWidth);
}

function showSettings() {
  currentMode = "settings";
  currentNotePath = "";
  currentTaskId = "";
  currentTask = null;
  notePath.textContent = "Settings";
  tagBar.classList.add("hidden");
  viewSelector.classList.add("hidden");
  viewButtons.forEach((btn) => {
    btn.disabled = true;
  });
  summaryPanel.classList.add("hidden");
  taskEditor.classList.add("hidden");
  editor.classList.add("hidden");
  preview.classList.add("hidden");
  assetPreview.classList.add("hidden");
  assetPreview.innerHTML = "";
  pdfPreview.classList.add("hidden");
  pdfPreview.innerHTML = "";
  csvPreview.classList.add("hidden");
  csvPreview.innerHTML = "";
  settingsPanel.classList.remove("hidden");
  setView("edit", true);
  saveBtn.disabled = true;
  isDirty = false;
  if (settingsDarkMode) {
    settingsDarkMode.checked = currentSettings.darkMode;
  }
  if (settingsDefaultView) {
    settingsDefaultView.value = currentSettings.defaultView || "split";
  }
  if (settingsAutosaveEnabled) {
    settingsAutosaveEnabled.checked = !!currentSettings.autosaveEnabled;
  }
  if (settingsAutosaveInterval) {
    settingsAutosaveInterval.value = String(currentSettings.autosaveIntervalSeconds || 30);
  }
  if (settingsDefaultFolder) {
    settingsDefaultFolder.value = currentSettings.defaultFolder || "";
  }
  if (settingsDailyFolder) {
    settingsDailyFolder.value = currentSettings.dailyFolder || "";
  }
  if (settingsShowTemplates) {
    settingsShowTemplates.checked = currentSettings.showTemplates;
  }
}

async function saveSettings() {
  if (
    !settingsDarkMode ||
    !settingsDefaultView ||
    !settingsAutosaveEnabled ||
    !settingsAutosaveInterval ||
    !settingsDefaultFolder ||
    !settingsDailyFolder ||
    !settingsShowTemplates
  ) {
    return;
  }
  try {
    saveBtn.disabled = true;
    saveBtn.textContent = "Saving...";
    const previousShowTemplates = currentSettings.showTemplates;
    const payload = {
      darkMode: settingsDarkMode.checked,
      defaultView: settingsDefaultView.value,
      autosaveEnabled: settingsAutosaveEnabled.checked,
      autosaveIntervalSeconds: Number(settingsAutosaveInterval.value) || 30,
      sidebarWidth: currentSettings.sidebarWidth || 300,
      defaultFolder: settingsDefaultFolder.value.trim(),
      dailyFolder: settingsDailyFolder.value.trim(),
      showTemplates: settingsShowTemplates.checked,
    };
    const updated = await apiFetch("/settings", {
      method: "PATCH",
      body: JSON.stringify(payload),
    });
    applySettings(updated);
    if (previousShowTemplates !== updated.showTemplates) {
      await loadTree();
    }
    isDirty = false;
    saveBtn.textContent = "Save";
    saveBtn.disabled = false;
  } catch (err) {
    saveBtn.textContent = "Save";
    saveBtn.disabled = false;
    alert(err.message);
  }
}

async function saveSidebarWidth(width) {
  const clamped = Math.min(600, Math.max(220, Math.round(width)));
  try {
    await apiFetch("/settings", {
      method: "PATCH",
      body: JSON.stringify({ sidebarWidth: clamped }),
    });
    currentSettings.sidebarWidth = clamped;
  } catch (err) {
    console.warn("Unable to save sidebar width", err);
  }
}

function fillTaskEditor(task) {
  taskTitleInput.value = task.title || "";
  taskProjectInput.value = task.project || "";
  taskTagsInput.value = Array.isArray(task.tags) ? task.tags.join(", ") : "";
  taskDueDateInput.value = task.duedate || "";
  taskPriorityInput.value = String(task.priority || 3);
  taskCompletedInput.checked = !!task.completed;
  taskNotesInput.value = task.notes || "";
  taskCreatedText.textContent = formatDateTime(task.created);
  taskUpdatedText.textContent = formatDateTime(task.updated);
}

function getTagColor(tag) {
  const value = String(tag || "");
  let hash = 0;
  for (let i = 0; i < value.length; i += 1) {
    hash = (hash * 31 + value.charCodeAt(i)) % tagPalette.length;
  }
  return tagPalette[Math.abs(hash) % tagPalette.length];
}

function openTagGroup(tag) {
  if (!tag) {
    return;
  }
  const tagRoot = treeContainer.querySelector(".tree-node.tag-root");
  if (tagRoot) {
    tagRoot.classList.remove("collapsed");
  }
  const tagRow = treeContainer.querySelector(
    `.node-row[data-type="tag"][data-tag="${CSS.escape(tag)}"]`
  );
  if (!tagRow) {
    return;
  }
  const tagWrapper = tagRow.closest(".tree-node.tag-group");
  if (tagWrapper) {
    tagWrapper.classList.remove("collapsed");
  }
  tagRow.scrollIntoView({ block: "center" });
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
  if (node.type === "folder" && depth === 0) {
    wrapper.classList.add("note-root");
  }

  const row = document.createElement("div");
  row.className = "node-row";
  row.style.paddingLeft = `${12 + depth * 12}px`;
  row.dataset.path = node.path;
  row.dataset.type = node.type;

  if (node.type === "folder") {
    const icon = document.createElement("span");
    icon.className = "folder-icon";
    row.appendChild(icon);
  } else if (node.type === "asset") {
    const icon = document.createElement("span");
    icon.className = "asset-icon";
    row.appendChild(icon);
  } else if (node.type === "pdf") {
    const icon = document.createElement("span");
    icon.className = "pdf-icon";
    row.appendChild(icon);
  } else if (node.type === "csv") {
    const icon = document.createElement("span");
    icon.className = "csv-icon";
    row.appendChild(icon);
  } else {
    const icon = document.createElement("span");
    icon.className = "note-icon";
    row.appendChild(icon);
  }

  const name = document.createElement("span");
  name.className = "node-name";
  name.textContent = displayNodeName(node);
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
      hideContextMenu();
      wrapper.classList.toggle("collapsed");
      const counts = countTreeItems(node);
      currentActivePath = node.path || "";
      setActiveNode(currentActivePath);
      const title = depth === 0 ? "Notes" : `Folder: ${node.name}`;
      showSummary(title, [
        { label: "Folders", value: counts.folders },
        { label: "Notes", value: counts.notes },
        { label: "Assets", value: counts.assets },
        { label: "PDFs", value: counts.pdfs },
        { label: "CSVs", value: counts.csvs },
      ], {
        label: "New",
        handler: () => createNote(node.path || ""),
      });
    });

    row.addEventListener("contextmenu", (event) => {
      event.preventDefault();
      const isCollapsed = wrapper.classList.contains("collapsed");
      showContextMenu(event.clientX, event.clientY, [
        {
          label: "New Folder",
          action: () => createFolder(node.path),
        },
        {
          label: "New Note",
          action: () => createNote(node.path),
        },
        {
          label: "Rename",
          action: () => renameFolder(node.path),
        },
        {
          label: "Delete",
          action: () => deleteFolder(node.path),
        },
        {
          label: isCollapsed ? "Expand" : "Collapse",
          action: () => wrapper.classList.toggle("collapsed"),
        },
      ]);
    });
  } else {
    row.addEventListener("click", (event) => {
      event.stopPropagation();
      hideContextMenu();
      if (node.type === "asset") {
        openAsset(node.path);
      } else if (node.type === "pdf") {
        openPdf(node.path);
      } else if (node.type === "csv") {
        openCsv(node.path);
      } else {
        openNote(node.path);
      }
    });

    if (node.type === "file") {
      row.addEventListener("contextmenu", (event) => {
        event.preventDefault();
        const parentPath = node.path.split("/").slice(0, -1).join("/");
        showContextMenu(event.clientX, event.clientY, [
          {
            label: "New Note",
            action: () => createNote(parentPath),
          },
          {
            label: "Rename",
            action: () => renameNote(node.path),
          },
          {
            label: "Delete",
            action: () => deleteNote(node.path),
          },
        ]);
      });
    }
  }

  return wrapper;
}

function getSortableDueDate(value) {
  if (!value) {
    return new Date("9999-12-31T00:00:00Z");
  }
  const date = new Date(`${value}T00:00:00Z`);
  if (Number.isNaN(date.getTime())) {
    return new Date("9999-12-31T00:00:00Z");
  }
  return date;
}

function compareTasks(a, b) {
  const dateA = getSortableDueDate(a.duedate);
  const dateB = getSortableDueDate(b.duedate);
  if (dateA.getTime() !== dateB.getTime()) {
    return dateA - dateB;
  }
  const priorityA = Number(a.priority) || 0;
  const priorityB = Number(b.priority) || 0;
  if (priorityA !== priorityB) {
    return priorityB - priorityA;
  }
  const updatedA = new Date(a.updated || 0);
  const updatedB = new Date(b.updated || 0);
  return updatedA - updatedB;
}

function countTreeItems(node) {
  const counts = {
    folders: 0,
    notes: 0,
    assets: 0,
    pdfs: 0,
    csvs: 0,
  };
  if (!node || !node.children) {
    return counts;
  }
  const stack = [...node.children];
  while (stack.length > 0) {
    const current = stack.pop();
    if (!current) {
      continue;
    }
    switch (current.type) {
      case "folder":
        counts.folders += 1;
        if (current.children && current.children.length > 0) {
          stack.push(...current.children);
        }
        break;
      case "file":
        counts.notes += 1;
        break;
      case "asset":
        counts.assets += 1;
        break;
      case "pdf":
        counts.pdfs += 1;
        break;
      case "csv":
        counts.csvs += 1;
        break;
      default:
        break;
    }
  }
  return counts;
}

function buildTaskNode(task, depth) {
  const wrapper = document.createElement("div");
  wrapper.className = "tree-node task";

  const row = document.createElement("div");
  row.className = "node-row";
  row.style.paddingLeft = `${12 + depth * 12}px`;
  row.dataset.path = `task:${task.id}`;
  row.dataset.type = "task";
  row.dataset.taskId = task.id;
  if (task.completed) {
    row.dataset.completed = "true";
  }

  const icon = document.createElement("span");
  icon.className = "task-icon";
  row.appendChild(icon);

  const name = document.createElement("span");
  name.className = "node-name";
  name.textContent = task.title || "(untitled)";
  row.appendChild(name);

  wrapper.appendChild(row);

  row.addEventListener("click", (event) => {
    event.stopPropagation();
    hideContextMenu();
    openTask(task.id);
  });

  row.addEventListener("contextmenu", (event) => {
    event.preventDefault();
    const isCompleted = !!task.completed;
    showContextMenu(event.clientX, event.clientY, [
      {
        label: "Edit",
        action: () => openTask(task.id),
      },
      {
        label: isCompleted ? "Mark Incomplete" : "Mark Complete",
        action: () => setTaskCompletion(task.id, !isCompleted),
      },
      {
        label: "Delete",
        action: () => deleteTask(task.id),
      },
    ]);
  });

  return wrapper;
}

function buildTaskGroup(name, tasks, depth) {
  const wrapper = document.createElement("div");
  wrapper.className = "tree-node folder task-group collapsed";

  const row = document.createElement("div");
  row.className = "node-row";
  row.style.paddingLeft = `${12 + depth * 12}px`;
  row.dataset.path = `task-group:${name}`;
  row.dataset.type = "task-group";

  const caret = document.createElement("span");
  caret.className = "folder-icon";
  row.appendChild(caret);

  const label = document.createElement("span");
  label.className = "node-name";
  label.textContent = name;
  row.appendChild(label);

  wrapper.appendChild(row);

  const children = document.createElement("div");
  children.className = "node-children";
  const sortedTasks = [...tasks].sort(compareTasks);
  sortedTasks.forEach((task) => {
    children.appendChild(buildTaskNode(task, depth + 1));
  });
  wrapper.appendChild(children);

  row.addEventListener("click", () => {
    hideContextMenu();
    wrapper.classList.toggle("collapsed");
    const total = (tasks || []).length;
    const completed = (tasks || []).filter((task) => task.completed).length;
    const active = total - completed;
    currentActivePath = `task-group:${name}`;
    setActiveNode(currentActivePath);
    let title = `Project: ${name}`;
    if (name === "No Project") {
      title = "No Project";
    } else if (name === "Completed") {
      title = "Completed Tasks";
    }
    const action =
      name === "Completed"
        ? null
        : {
            label: "New",
            handler: () => createTask(name === "No Project" ? "" : name),
          };
    showSummary(
      title,
      [
        { label: "Total Tasks", value: total },
        { label: "Completed", value: completed },
        { label: "Active", value: active },
      ],
      action
    );
  });

  row.addEventListener("contextmenu", (event) => {
    event.preventDefault();
    const isCollapsed = wrapper.classList.contains("collapsed");
    const items = [];
    if (name !== "Completed") {
      items.push({
        label: "New Task",
        action: () => createTask(name === "No Project" ? "" : name),
      });
    }
    items.push({
      label: isCollapsed ? "Expand" : "Collapse",
      action: () => wrapper.classList.toggle("collapsed"),
    });
    showContextMenu(event.clientX, event.clientY, items);
  });

  return wrapper;
}

function buildTasksRoot(tasks) {
  const wrapper = document.createElement("div");
  wrapper.className = "tree-node folder task-root";

  const row = document.createElement("div");
  row.className = "node-row";
  row.style.paddingLeft = "12px";
  row.dataset.path = "__tasks__";
  row.dataset.type = "task-root";

  const caret = document.createElement("span");
  caret.className = "folder-icon";
  row.appendChild(caret);

  const name = document.createElement("span");
  name.className = "node-name";
  name.textContent = "Tasks";
  row.appendChild(name);

  wrapper.appendChild(row);

  const children = document.createElement("div");
  children.className = "node-children";

  const projectMap = new Map();
  const noProject = [];
  const completed = [];

  (tasks || []).forEach((task) => {
    if (task.completed) {
      completed.push(task);
      return;
    }
    const projectName = (task.project || "").trim();
    if (!projectName) {
      noProject.push(task);
      return;
    }
    if (!projectMap.has(projectName)) {
      projectMap.set(projectName, []);
    }
    projectMap.get(projectName).push(task);
  });

  const projectNames = Array.from(projectMap.keys()).sort((a, b) =>
    a.localeCompare(b, undefined, { sensitivity: "base" })
  );

  projectNames.forEach((project) => {
    children.appendChild(buildTaskGroup(project, projectMap.get(project) || [], 1));
  });

  children.appendChild(buildTaskGroup("No Project", noProject, 1));
  children.appendChild(buildTaskGroup("Completed", completed, 1));

  wrapper.appendChild(children);

  row.addEventListener("click", () => {
    hideContextMenu();
    wrapper.classList.toggle("collapsed");
    const completedCount = (tasks || []).filter((task) => task.completed).length;
    const projectSet = new Set();
    let noProjectCount = 0;
    (tasks || []).forEach((task) => {
      const projectName = (task.project || "").trim();
      if (!projectName) {
        noProjectCount += 1;
        return;
      }
      projectSet.add(projectName);
    });
    currentActivePath = "__tasks__";
    setActiveNode(currentActivePath);
    showSummary(
      "Tasks",
      [
        { label: "Total Tasks", value: (tasks || []).length },
        { label: "Completed", value: completedCount },
        { label: "Active", value: (tasks || []).length - completedCount },
        { label: "Projects", value: projectSet.size },
        { label: "No Project", value: noProjectCount },
      ],
      { label: "New", handler: () => createTask("") }
    );
  });

  row.addEventListener("contextmenu", (event) => {
    event.preventDefault();
    const isCollapsed = wrapper.classList.contains("collapsed");
    showContextMenu(event.clientX, event.clientY, [
      {
        label: "New Task",
        action: () => createTask(""),
      },
      {
        label: "Refresh",
        action: () => loadTree(),
      },
      {
        label: isCollapsed ? "Expand" : "Collapse",
        action: () => wrapper.classList.toggle("collapsed"),
      },
    ]);
  });

  return wrapper;
}

function buildTagRoot(tags) {
  const wrapper = document.createElement("div");
  wrapper.className = "tree-node folder tag-root collapsed";

  const row = document.createElement("div");
  row.className = "node-row";
  row.style.paddingLeft = "12px";
  row.dataset.path = "__tags__";
  row.dataset.type = "tag-root";

  const caret = document.createElement("span");
  caret.className = "folder-icon";
  row.appendChild(caret);

  const name = document.createElement("span");
  name.className = "node-name";
  name.textContent = "Tags";
  row.appendChild(name);

  wrapper.appendChild(row);

  const children = document.createElement("div");
  children.className = "node-children";
  tags.forEach((group) => {
    children.appendChild(buildTagGroup(group, 1));
  });
  wrapper.appendChild(children);

  row.addEventListener("click", () => {
    hideContextMenu();
    wrapper.classList.toggle("collapsed");
    const totalTags = (tags || []).length;
    const noteSet = new Set();
    let totalEntries = 0;
    (tags || []).forEach((group) => {
      (group.notes || []).forEach((note) => {
        if (note && note.path) {
          noteSet.add(note.path);
        }
        totalEntries += 1;
      });
    });
    currentActivePath = "__tags__";
    setActiveNode(currentActivePath);
    showSummary("Tags", [
      { label: "Tags", value: totalTags },
      { label: "Tagged Notes", value: noteSet.size },
      { label: "Tag Entries", value: totalEntries },
    ]);
  });

  row.addEventListener("contextmenu", (event) => {
    event.preventDefault();
    const isCollapsed = wrapper.classList.contains("collapsed");
    showContextMenu(event.clientX, event.clientY, [
      {
        label: isCollapsed ? "Expand" : "Collapse",
        action: () => wrapper.classList.toggle("collapsed"),
      },
    ]);
  });

  return wrapper;
}

function buildTagGroup(group, depth) {
  const wrapper = document.createElement("div");
  wrapper.className = "tree-node folder tag-group";
  wrapper.classList.add("collapsed");

  const row = document.createElement("div");
  row.className = "node-row";
  row.style.paddingLeft = `${12 + depth * 12}px`;
  row.dataset.path = `tag:${group.tag}`;
  row.dataset.type = "tag";
  row.dataset.tag = group.tag;

  const caret = document.createElement("span");
  caret.className = "folder-icon";
  row.appendChild(caret);

  const name = document.createElement("span");
  name.className = "node-name tag-label";
  name.textContent = `#${group.tag}`;
  name.style.backgroundColor = getTagColor(group.tag);
  row.appendChild(name);

  wrapper.appendChild(row);

  const children = document.createElement("div");
  children.className = "node-children";
  (group.notes || []).forEach((note) => {
    children.appendChild(buildTagNote(note, depth + 1));
  });
  wrapper.appendChild(children);

  row.addEventListener("click", () => {
    hideContextMenu();
    wrapper.classList.toggle("collapsed");
    const totalTags = (tags || []).length;
    const noteSet = new Set();
    let totalEntries = 0;
    (tags || []).forEach((group) => {
      (group.notes || []).forEach((note) => {
        if (note && note.path) {
          noteSet.add(note.path);
        }
        totalEntries += 1;
      });
    });
    currentActivePath = "__tags__";
    setActiveNode(currentActivePath);
    showSummary("Tags", [
      { label: "Tags", value: totalTags },
      { label: "Tagged Notes", value: noteSet.size },
      { label: "Tag Entries", value: totalEntries },
    ]);
  });

  row.addEventListener("contextmenu", (event) => {
    event.preventDefault();
    const isCollapsed = wrapper.classList.contains("collapsed");
    showContextMenu(event.clientX, event.clientY, [
      {
        label: isCollapsed ? "Expand" : "Collapse",
        action: () => wrapper.classList.toggle("collapsed"),
      },
    ]);
  });

  return wrapper;
}

function buildTagNote(note, depth) {
  const wrapper = document.createElement("div");
  wrapper.className = "tree-node file";

  const row = document.createElement("div");
  row.className = "node-row";
  row.style.paddingLeft = `${12 + depth * 12}px`;
  row.dataset.path = note.path;
  row.dataset.type = "file";

  const spacer = document.createElement("span");
  spacer.className = "note-icon";
  row.appendChild(spacer);

  const name = document.createElement("span");
  name.className = "node-name";
  name.textContent = displayNodeName({ type: "file", name: note.name });
  row.appendChild(name);

  wrapper.appendChild(row);

  row.addEventListener("click", (event) => {
    event.stopPropagation();
    hideContextMenu();
    openNote(note.path);
  });

  return wrapper;
}

function renderTree(tree, tags, tasks) {
  treeContainer.innerHTML = "";
  if (tree) {
    const rootNode = buildTreeNode(tree, 0);
    rootNode.classList.remove("collapsed");
    treeContainer.appendChild(rootNode);
  }
  if (tasks) {
    const tasksRoot = buildTasksRoot(tasks);
    treeContainer.appendChild(tasksRoot);
  }
  if (tags) {
    const tagRoot = buildTagRoot(tags);
    treeContainer.appendChild(tagRoot);
  }
  setActiveNode(currentActivePath);
}

function setActiveNode(path) {
  const rows = treeContainer.querySelectorAll(".node-row");
  rows.forEach((row) => {
    const isSelectable =
      row.dataset.type === "file" ||
      row.dataset.type === "asset" ||
      row.dataset.type === "pdf" ||
      row.dataset.type === "csv" ||
      row.dataset.type === "task" ||
      row.dataset.type === "task-root" ||
      row.dataset.type === "tag-root" ||
      row.dataset.type === "task-group" ||
      row.dataset.type === "folder";
    row.classList.toggle("active", isSelectable && row.dataset.path === path);
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
    const [tree, tags, tasksResponse, settingsResponse] = await Promise.all([
      apiFetch(`/tree${query}`),
      apiFetch("/tags"),
      apiFetch("/tasks"),
      apiFetch("/settings"),
    ]);
    if (tasksResponse.notice) {
      alert(tasksResponse.notice);
    }
    if (settingsResponse.notice && !settingsLoaded) {
      alert(settingsResponse.notice);
    }
    settingsLoaded = true;
    applySettings(settingsResponse.settings || {});
    currentTree = tree;
    currentTags = tags;
    currentTasks = tasksResponse.tasks || [];
    renderTree(currentTree, currentTags, tasksResponse.tasks || []);
    if (currentTree && currentTree.type === "folder" && currentTree.children) {
      const defaultFolder = (currentSettings.defaultFolder || "").trim();
      const targetNode = defaultFolder ? findFolderNode(currentTree, defaultFolder) : currentTree;
      const counts = countTreeItems(targetNode || currentTree);
      currentActivePath = targetNode ? targetNode.path : "";
      setActiveNode(currentActivePath);
      if (targetNode && targetNode.path) {
        expandToPath(targetNode.path);
      }
      const title = targetNode && targetNode.path ? `Folder: ${targetNode.name}` : "Notes";
      const actionPath = targetNode && targetNode.path ? targetNode.path : "";
      showSummary(
        title,
        [
          { label: "Folders", value: counts.folders },
          { label: "Notes", value: counts.notes },
          { label: "Assets", value: counts.assets },
          { label: "PDFs", value: counts.pdfs },
          { label: "CSVs", value: counts.csvs },
        ],
        { label: "New", handler: () => createNote(actionPath) }
      );
    }
  } catch (err) {
    alert(err.message);
  }
}

async function openNote(path) {
  try {
    showNoteEditor();
    const data = await apiFetch(`/notes?path=${encodeURIComponent(path)}`);
    currentNotePath = data.path;
    currentActivePath = data.path;
    notePath.textContent = data.path;
    editor.value = data.content;
    preview.innerHTML = renderMarkdown(data.content);
    preview.classList.remove("hidden");
    assetPreview.classList.add("hidden");
    assetPreview.innerHTML = "";
    pdfPreview.classList.add("hidden");
    pdfPreview.innerHTML = "";
    csvPreview.classList.add("hidden");
    csvPreview.innerHTML = "";
    viewSelector.classList.remove("hidden");
    viewButtons.forEach((btn) => {
      btn.disabled = false;
    });
    setView(getDefaultView(currentSettings.defaultView), true);
    applyHighlighting();
    renderTagBar(extractTags(data.content));
    isDirty = false;
    saveBtn.disabled = false;
    expandToPath(currentNotePath);
    setActiveNode(currentActivePath);
  } catch (err) {
    alert(err.message);
  }
}

function openAsset(path) {
  if (!path) {
    return;
  }
  currentMode = "asset";
  currentTaskId = "";
  currentTask = null;
  summaryPanel.classList.add("hidden");
  settingsPanel.classList.add("hidden");
  taskEditor.classList.add("hidden");
  editor.classList.remove("hidden");
  currentNotePath = "";
  currentActivePath = path;
  notePath.textContent = path;
  editor.value = "";
  preview.innerHTML = "";
  preview.classList.add("hidden");
  assetPreview.classList.remove("hidden");
  assetPreview.innerHTML = "";
  pdfPreview.classList.add("hidden");
  pdfPreview.innerHTML = "";
  csvPreview.classList.add("hidden");
  csvPreview.innerHTML = "";
  const img = document.createElement("img");
  img.src = `${apiBase}/files?path=${encodeURIComponent(path)}`;
  img.alt = path.split("/").pop() || "Image";
  assetPreview.appendChild(img);
  viewSelector.classList.add("hidden");
  viewButtons.forEach((btn) => {
    btn.disabled = true;
  });
  app.dataset.view = "preview";
  saveBtn.disabled = true;
  isDirty = false;
  renderTagBar([]);
  expandToPath(path);
  setActiveNode(currentActivePath);
}

function openPdf(path) {
  if (!path) {
    return;
  }
  currentMode = "asset";
  currentTaskId = "";
  currentTask = null;
  summaryPanel.classList.add("hidden");
  settingsPanel.classList.add("hidden");
  taskEditor.classList.add("hidden");
  editor.classList.remove("hidden");
  currentNotePath = "";
  currentActivePath = path;
  notePath.textContent = path;
  editor.value = "";
  preview.innerHTML = "";
  preview.classList.add("hidden");
  assetPreview.classList.add("hidden");
  assetPreview.innerHTML = "";
  csvPreview.classList.add("hidden");
  csvPreview.innerHTML = "";
  pdfPreview.classList.remove("hidden");
  pdfPreview.innerHTML = "";
  const src = `${apiBase}/files?path=${encodeURIComponent(path)}`;
  const embed = document.createElement("embed");
  embed.type = "application/pdf";
  embed.src = src;
  pdfPreview.appendChild(embed);
  const fallback = document.createElement("div");
  fallback.className = "pdf-fallback";
  const link = document.createElement("a");
  link.href = src;
  link.textContent = "Open PDF";
  link.target = "_blank";
  link.rel = "noopener";
  fallback.appendChild(document.createTextNode("PDF preview unavailable. "));
  fallback.appendChild(link);
  pdfPreview.appendChild(fallback);
  viewSelector.classList.add("hidden");
  viewButtons.forEach((btn) => {
    btn.disabled = true;
  });
  app.dataset.view = "preview";
  saveBtn.disabled = true;
  isDirty = false;
  renderTagBar([]);
  expandToPath(path);
  setActiveNode(currentActivePath);
}

function parseCsv(text) {
  const rows = [];
  let row = [];
  let value = "";
  let inQuotes = false;

  for (let i = 0; i < text.length; i += 1) {
    const char = text[i];
    if (char === "\"") {
      if (inQuotes && text[i + 1] === "\"") {
        value += "\"";
        i += 1;
      } else {
        inQuotes = !inQuotes;
      }
      continue;
    }
    if (char === "," && !inQuotes) {
      row.push(value);
      value = "";
      continue;
    }
    if ((char === "\n" || char === "\r") && !inQuotes) {
      if (char === "\r" && text[i + 1] === "\n") {
        i += 1;
      }
      row.push(value);
      rows.push(row);
      row = [];
      value = "";
      continue;
    }
    value += char;
  }
  row.push(value);
  rows.push(row);
  return rows;
}

function renderCsvTable(rows) {
  csvPreview.innerHTML = "";
  if (!rows || rows.length === 0) {
    return;
  }
  const table = document.createElement("table");
  table.className = "csv-table";
  const thead = document.createElement("thead");
  const headerRow = document.createElement("tr");
  rows[0].forEach((cell) => {
    const th = document.createElement("th");
    th.textContent = cell;
    headerRow.appendChild(th);
  });
  thead.appendChild(headerRow);
  table.appendChild(thead);

  const tbody = document.createElement("tbody");
  rows.slice(1).forEach((row) => {
    const tr = document.createElement("tr");
    row.forEach((cell) => {
      const td = document.createElement("td");
      td.textContent = cell;
      tr.appendChild(td);
    });
    tbody.appendChild(tr);
  });
  table.appendChild(tbody);
  csvPreview.appendChild(table);
}

async function openCsv(path) {
  if (!path) {
    return;
  }
  try {
    const response = await fetch(`${apiBase}/files?path=${encodeURIComponent(path)}`);
    if (!response.ok) {
      throw new Error("Unable to load CSV file");
    }
    const text = await response.text();
    renderCsvTable(parseCsv(text));
    currentMode = "asset";
    currentTaskId = "";
    currentTask = null;
    summaryPanel.classList.add("hidden");
    settingsPanel.classList.add("hidden");
    taskEditor.classList.add("hidden");
    editor.classList.remove("hidden");
    currentNotePath = "";
    currentActivePath = path;
    notePath.textContent = path;
    editor.value = "";
    preview.innerHTML = "";
    preview.classList.add("hidden");
    assetPreview.classList.add("hidden");
    assetPreview.innerHTML = "";
    pdfPreview.classList.add("hidden");
    pdfPreview.innerHTML = "";
    csvPreview.classList.remove("hidden");
    viewSelector.classList.add("hidden");
    viewButtons.forEach((btn) => {
      btn.disabled = true;
    });
    app.dataset.view = "preview";
    saveBtn.disabled = true;
    isDirty = false;
    renderTagBar([]);
    expandToPath(path);
    setActiveNode(currentActivePath);
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

function buildTaskPayloadFromForm() {
  return {
    title: taskTitleInput.value.trim(),
    project: taskProjectInput.value.trim(),
    tags: parseTagsInput(taskTagsInput.value),
    duedate: taskDueDateInput.value,
    priority: Number(taskPriorityInput.value) || 3,
    completed: taskCompletedInput.checked,
    notes: taskNotesInput.value,
  };
}

async function openTask(id) {
  if (!id) {
    return;
  }
  try {
    showTaskEditor();
    const task = await apiFetch(`/tasks/${encodeURIComponent(id)}`);
    currentTaskId = task.id;
    currentTask = task;
    currentActivePath = `task:${task.id}`;
    notePath.textContent = `Task: ${task.title || "(untitled)"}`;
    fillTaskEditor(task);
    isDirty = false;
    saveBtn.disabled = false;
    setActiveNode(currentActivePath);
  } catch (err) {
    alert(err.message);
  }
}

async function saveTask() {
  if (!currentTaskId) {
    return;
  }
  const payload = buildTaskPayloadFromForm();
  if (!payload.title) {
    alert("Title is required.");
    return;
  }
  if (payload.priority < 1 || payload.priority > 5) {
    alert("Priority must be between 1 and 5.");
    return;
  }
  try {
    saveBtn.disabled = true;
    saveBtn.textContent = "Saving...";
    const task = await apiFetch(`/tasks/${encodeURIComponent(currentTaskId)}`, {
      method: "PATCH",
      body: JSON.stringify(payload),
    });
    currentTask = task;
    fillTaskEditor(task);
    notePath.textContent = `Task: ${task.title || "(untitled)"}`;
    isDirty = false;
    saveBtn.textContent = "Save";
    saveBtn.disabled = false;
    await loadTree();
  } catch (err) {
    saveBtn.textContent = "Save";
    saveBtn.disabled = false;
    alert(err.message);
  }
}

function saveCurrent() {
  if (currentMode === "settings") {
    saveSettings();
  } else if (currentMode === "task") {
    saveTask();
  } else {
    saveNote();
  }
}

async function createTask(projectName = "") {
  const title = promptForName("New task title");
  if (!title) {
    return;
  }
  try {
    const task = await apiFetch("/tasks", {
      method: "POST",
      body: JSON.stringify({
        title,
        project: projectName,
        tags: [],
        duedate: "",
        priority: 3,
        completed: false,
        notes: "",
      }),
    });
    await loadTree();
    await openTask(task.id);
  } catch (err) {
    alert(err.message);
  }
}

async function deleteTask(id) {
  if (!id) {
    return;
  }
  const confirmDelete = window.confirm("Delete this task?");
  if (!confirmDelete) {
    return;
  }
  try {
    await apiFetch(`/tasks/${encodeURIComponent(id)}`, {
      method: "DELETE",
    });
    if (currentTaskId === id) {
      clearTaskEditor();
    }
    await loadTree();
  } catch (err) {
    alert(err.message);
  }
}

async function setTaskCompletion(id, completed) {
  if (!id) {
    return;
  }
  try {
    const task = await apiFetch(`/tasks/${encodeURIComponent(id)}`);
    const payload = {
      title: task.title,
      project: task.project,
      tags: Array.isArray(task.tags) ? task.tags : [],
      duedate: task.duedate || "",
      priority: Number(task.priority) || 3,
      completed,
      notes: task.notes || "",
    };
    const updated = await apiFetch(`/tasks/${encodeURIComponent(id)}`, {
      method: "PATCH",
      body: JSON.stringify(payload),
    });
    if (currentTaskId === id) {
      currentTask = updated;
      fillTaskEditor(updated);
    }
    await loadTree();
  } catch (err) {
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

function promptForNameWithDefault(label, defaultValue) {
  const name = window.prompt(label, defaultValue);
  if (!name) {
    return "";
  }
  if (name.includes("/") || name.includes("\\")) {
    alert("Names cannot include slashes.");
    return "";
  }
  return name.trim();
}

function ensureMarkdownName(name) {
  if (name.toLowerCase().endsWith(".md")) {
    return name;
  }
  return `${name}.md`;
}

function ensureTemplateName(name) {
  if (name.toLowerCase().endsWith(".template")) {
    return name;
  }
  return `${name}.template`;
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

async function renameNote(path) {
  if (!path) {
    return;
  }
  const currentName = path.split("/").pop() || "";
  const isTemplate = currentName.toLowerCase().endsWith(".template");
  const displayName = isTemplate
    ? currentName
    : displayNodeName({ type: "file", name: currentName });
  const name = promptForNameWithDefault(`Rename note (${displayName})`, displayName);
  if (!name) {
    return;
  }
  const base = path.split("/").slice(0, -1).join("/");
  const newName = isTemplate ? ensureTemplateName(name) : ensureMarkdownName(name);
  const newPath = base ? `${base}/${newName}` : newName;
  try {
    const data = await apiFetch("/notes/rename", {
      method: "PATCH",
      body: JSON.stringify({ path, newPath }),
    });
    await loadTree();
    if (currentNotePath === path) {
      await openNote(data.newPath || newPath);
    }
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

async function deleteNote(path) {
  if (!path) {
    return;
  }
  const confirmDelete = window.confirm("Delete this note?");
  if (!confirmDelete) {
    return;
  }
  try {
    await apiFetch(`/notes?path=${encodeURIComponent(path)}`, {
      method: "DELETE",
    });
    if (currentNotePath === path) {
      currentNotePath = "";
      currentActivePath = "";
      notePath.textContent = "";
      editor.value = "";
      preview.innerHTML = "";
      preview.classList.remove("hidden");
      assetPreview.classList.add("hidden");
      assetPreview.innerHTML = "";
      pdfPreview.classList.add("hidden");
      pdfPreview.innerHTML = "";
      csvPreview.classList.add("hidden");
      csvPreview.innerHTML = "";
      viewSelector.classList.remove("hidden");
      viewButtons.forEach((btn) => {
        btn.disabled = false;
      });
      saveBtn.disabled = true;
      isDirty = false;
      renderTagBar([]);
    }
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
  const safeMatches = Array.isArray(matches) ? matches : [];
  searchResults.innerHTML = "";
  if (safeMatches.length === 0) {
    const empty = document.createElement("div");
    empty.className = "search-empty";
    empty.textContent = "No matches";
    searchResults.appendChild(empty);
    return;
  }
  safeMatches.forEach((match) => {
    const button = document.createElement("button");
    button.type = "button";
    const isTask = match.type === "task" || match.id;
    if (isTask) {
      button.textContent = `Task: ${match.name || "(untitled)"}`;
      button.title = match.id || "";
    } else {
      const rawName = match.name || match.path.split("/").pop();
      button.textContent = displayNodeName({ type: "file", name: rawName });
      button.title = match.path;
    }
    button.addEventListener("click", () => {
      hideSearchResults();
      if (isTask) {
        openTask(match.id);
      } else {
        openNote(match.path);
      }
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
  let dragOverlay = null;

  function attachDragOverlay(cursor) {
    dragOverlay = document.createElement("div");
    dragOverlay.style.position = "fixed";
    dragOverlay.style.top = "0";
    dragOverlay.style.left = "0";
    dragOverlay.style.width = "100%";
    dragOverlay.style.height = "100%";
    dragOverlay.style.zIndex = "999";
    dragOverlay.style.cursor = cursor;
    dragOverlay.style.background = "transparent";
    document.body.appendChild(dragOverlay);
    document.body.style.userSelect = "none";
  }

  function detachDragOverlay() {
    if (dragOverlay) {
      dragOverlay.remove();
      dragOverlay = null;
    }
    document.body.style.userSelect = "";
  }

  sidebarResizer.addEventListener("mousedown", (event) => {
    event.preventDefault();
    attachDragOverlay("col-resize");
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
      detachDragOverlay();
      if (!isStacked()) {
        const width = sidebar.getBoundingClientRect().width;
        saveSidebarWidth(width);
      }
    }

    document.addEventListener("mousemove", onMove);
    document.addEventListener("mouseup", onUp);
  });

  paneResizer.addEventListener("mousedown", (event) => {
    event.preventDefault();
    attachDragOverlay(isStacked() ? "row-resize" : "col-resize");
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
      detachDragOverlay();
    }

    document.addEventListener("mousemove", onMove);
    document.addEventListener("mouseup", onUp);
  });
}

refreshBtn.addEventListener("click", () => loadTree());
settingsBtn.addEventListener("click", () => {
  hideContextMenu();
  showSettings();
});

editor.addEventListener("input", () => {
  if (currentMode !== "note") {
    return;
  }
  isDirty = true;
  preview.innerHTML = renderMarkdown(editor.value);
  applyHighlighting();
  renderTagBar(extractTags(editor.value));
  saveBtn.disabled = !currentNotePath;
});

function markTaskDirty() {
  if (currentMode !== "task") {
    return;
  }
  isDirty = true;
  saveBtn.disabled = !currentTaskId;
}

[
  taskTitleInput,
  taskProjectInput,
  taskTagsInput,
  taskDueDateInput,
  taskPriorityInput,
  taskNotesInput,
].forEach((input) => {
  if (!input) {
    return;
  }
  input.addEventListener("input", () => markTaskDirty());
});

if (taskCompletedInput) {
  taskCompletedInput.addEventListener("change", () => markTaskDirty());
}

if (settingsDarkMode) {
  settingsDarkMode.addEventListener("change", () => {
    if (currentMode !== "settings") {
      return;
    }
    applySettings({ ...currentSettings, darkMode: settingsDarkMode.checked });
    isDirty = true;
    saveBtn.disabled = false;
  });
}

if (settingsDefaultView) {
  settingsDefaultView.addEventListener("change", () => {
    if (currentMode !== "settings") {
      return;
    }
    currentSettings.defaultView = getDefaultView(settingsDefaultView.value);
    isDirty = true;
    saveBtn.disabled = false;
  });
}

if (settingsAutosaveEnabled) {
  settingsAutosaveEnabled.addEventListener("change", () => {
    if (currentMode !== "settings") {
      return;
    }
    currentSettings.autosaveEnabled = settingsAutosaveEnabled.checked;
    isDirty = true;
    saveBtn.disabled = false;
  });
}

if (settingsAutosaveInterval) {
  settingsAutosaveInterval.addEventListener("change", () => {
    if (currentMode !== "settings") {
      return;
    }
    currentSettings.autosaveIntervalSeconds = Number(settingsAutosaveInterval.value) || 30;
    isDirty = true;
    saveBtn.disabled = false;
  });
}

if (settingsDefaultFolder) {
  settingsDefaultFolder.addEventListener("input", () => {
    if (currentMode !== "settings") {
      return;
    }
    currentSettings.defaultFolder = settingsDefaultFolder.value.trim();
    isDirty = true;
    saveBtn.disabled = false;
  });
}

if (settingsDailyFolder) {
  settingsDailyFolder.addEventListener("input", () => {
    if (currentMode !== "settings") {
      return;
    }
    currentSettings.dailyFolder = settingsDailyFolder.value.trim();
    isDirty = true;
    saveBtn.disabled = false;
  });
}

if (settingsShowTemplates) {
  settingsShowTemplates.addEventListener("change", () => {
    if (currentMode !== "settings") {
      return;
    }
    currentSettings.showTemplates = settingsShowTemplates.checked;
    isDirty = true;
    saveBtn.disabled = false;
  });
}

function normalizeTagInput(value) {
  const trimmed = String(value || "").trim();
  if (!trimmed) {
    return "";
  }
  const cleaned = trimmed.startsWith("#") ? trimmed.slice(1) : trimmed;
  const normalized = cleaned.toLowerCase();
  if (!/^[a-z0-9]+$/.test(normalized)) {
    return "";
  }
  return normalized;
}

function appendTagToNote(tag) {
  if (!currentNotePath) {
    alert("Select a note before adding tags.");
    return;
  }
  const current = editor.value || "";
  const normalizedTag = normalizeTagInput(tag);
  if (!normalizedTag) {
    alert("Tags must contain only letters or numbers.");
    return;
  }
  const lines = current.split("\n");
  const lastIndex = Math.max(0, lines.length - 1);
  const prefix = lines[lastIndex].trim().length === 0 ? "" : " ";
  lines[lastIndex] = `${lines[lastIndex]}${prefix}#${normalizedTag}`;
  editor.value = lines.join("\n");
  isDirty = true;
  preview.innerHTML = renderMarkdown(editor.value);
  applyHighlighting();
  renderTagBar(extractTags(editor.value));
  saveBtn.disabled = !currentNotePath;
}

function getScrollRatio(element) {
  const maxScroll = element.scrollHeight - element.clientHeight;
  if (maxScroll <= 0) {
    return 0;
  }
  return element.scrollTop / maxScroll;
}

function syncScroll(source, target) {
  if (syncingScroll) {
    return;
  }
  if (activeScrollSource && source !== activeScrollSource) {
    return;
  }
  syncingScroll = true;
  const maxSourceScroll = source.scrollHeight - source.clientHeight;
  const maxTargetScroll = target.scrollHeight - target.clientHeight;
  const ratio = maxSourceScroll <= 0 ? 0 : source.scrollTop / maxSourceScroll;
  const topScaled =
    source.scrollHeight <= 0 ? 0 : source.scrollTop * (target.scrollHeight / source.scrollHeight);
  const bottomScaled = Math.max(0, maxTargetScroll) * ratio;
  const blended = topScaled * (1 - ratio) + bottomScaled * ratio;
  target.scrollTop = Math.max(0, Math.min(maxTargetScroll, blended));
  requestAnimationFrame(() => {
    syncingScroll = false;
  });
}

function markActiveScrollSource(source) {
  activeScrollSource = source;
  if (clearScrollSourceTimer) {
    clearTimeout(clearScrollSourceTimer);
  }
  clearScrollSourceTimer = setTimeout(() => {
    activeScrollSource = null;
  }, 140);
}

editor.addEventListener("wheel", () => markActiveScrollSource(editor), { passive: true });
preview.addEventListener("wheel", () => markActiveScrollSource(preview), { passive: true });
editor.addEventListener("touchstart", () => markActiveScrollSource(editor), { passive: true });
preview.addEventListener("touchstart", () => markActiveScrollSource(preview), { passive: true });

editor.addEventListener("scroll", () => {
  if (!activeScrollSource) {
    markActiveScrollSource(editor);
  }
  syncScroll(editor, preview);
});
preview.addEventListener("scroll", () => {
  if (!activeScrollSource) {
    markActiveScrollSource(preview);
  }
  syncScroll(preview, editor);
});

saveBtn.addEventListener("click", () => saveCurrent());

viewButtons.forEach((btn) => {
  btn.addEventListener("click", () => setView(btn.dataset.view));
});

tagAddBtn.addEventListener("click", () => {
  const input = window.prompt("Add tag");
  if (input === null) {
    return;
  }
  appendTagToNote(input);
});

window.addEventListener("keydown", (event) => {
  if (!(event.metaKey || event.ctrlKey)) {
    return;
  }
  const key = event.key.toLowerCase();
  if (key === "s" && event.shiftKey) {
    event.preventDefault();
    saveCurrent();
    return;
  }
  if (key === "e" && event.shiftKey) {
    event.preventDefault();
    setView("edit");
    return;
  }
  if (key === "p" && event.shiftKey) {
    event.preventDefault();
    setView("preview");
    return;
  }
  if (key === "b" && event.shiftKey) {
    event.preventDefault();
    setView("split");
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

treeContainer.addEventListener("mousedown", () => hideContextMenu(), true);
mainHeader.addEventListener("mousedown", () => hideContextMenu(), true);
tagBar.addEventListener("mousedown", () => hideContextMenu(), true);
previewPane.addEventListener("mousedown", () => hideContextMenu(), true);
mainContent.addEventListener("mousedown", () => hideContextMenu(), true);
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

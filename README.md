# NolderMD

NolderMD is a Go-based Markdown notes server with a discrete JSON API and web
UI. Notes are stored as `.md` files under the local `Notes/` directory.

## Run

```bash
go run ./cmd/noldermd serve --notes-dir ./Notes --port 8080
```

Open http://localhost:8080.

## API (base: `/api/v1`)

- `GET /health`
- `GET /tree?path=<folder>`
- `GET /notes?path=<file>`
- `POST /notes` `{ "path": "Folder/Note", "content": "..." }`
- `PATCH /notes` `{ "path": "Folder/Note.md", "content": "..." }`
- `PATCH /notes/rename` `{ "path": "Folder/Note.md", "newPath": "Folder/Renamed" }`
- `DELETE /notes?path=<file>`
- `POST /folders` `{ "path": "Folder/Subfolder" }`
- `PATCH /folders` `{ "path": "Folder", "newPath": "Renamed" }`
- `DELETE /folders?path=<folder>`
- `GET /files?path=<file>` (raw file, used for images)
- `GET /search?query=<text>` (searches filenames + contents)
- `GET /tags` (tags with notes that contain them)
- `GET /tasks` (lists tasks)
- `GET /tasks/<id>` (fetch task)
- `POST /tasks` `{ "title": "...", "project": "...", "tags": [], "duedate": "YYYY-MM-DD", "priority": 3, "completed": false, "notes": "..." }`
- `PATCH /tasks/<id>` `{ "title": "...", "project": "...", "tags": [], "duedate": "YYYY-MM-DD", "priority": 3, "completed": false, "notes": "..." }`
- `DELETE /tasks/<id>`

## Notes rules

- `.md` is appended on note creation when missing.
- Only `.md` files are treated as notes.
- Files starting with `._` are ignored.
- Tree responses return metadata only.
- Tags match `#` followed by letters, preceded by whitespace or start of line.

## Tasks rules

- Tasks live in `Notes/tasks.json` and are created automatically if missing.
- Tasks use UUIDs for stable IDs.
- Priority is 1-5 (1 highest, 5 lowest).
- Due dates are stored as `YYYY-MM-DD`.

## UX behavior

- Left sidebar renders a recursive `Notes/` tree with a draggable width splitter.
- Left sidebar also renders a "Tags" root (collapsed by default), refreshed on
  tree reload.
- Left sidebar renders a "Tasks" root with projects, "No Project", and "Completed".
- Clicking Notes/Tasks/Tags roots or any folder/project group shows a summary panel.
- Folder and tag rows show centered chevrons indicating expanded/collapsed
  state.
- Main pane supports edit, preview, or split view with a draggable splitter.
- View selector (top right) toggles edit/preview/split.
- Preview pane shows a sticky tag bar with clickable tag pills.
- Context menus:
  - Folder: New Folder, New Note, Rename, Delete, Expand/Collapse.
  - Note: New Note, Rename, Delete.
  - Sidebar empty area: New Folder, New Note.
- Refresh button reloads the tree.
- Search input lists matching notes and tasks; selecting one opens it.
- Editor and preview scroll independently from the sidebar.
- Editor and preview panes scroll together (proportional sync).

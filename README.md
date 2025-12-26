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

## Notes rules

- `.md` is appended on note creation when missing.
- Only `.md` files are treated as notes.
- Tree responses return metadata only.
- Tags match `#` followed by letters, preceded by whitespace or start of line.

## UX behavior

- Left sidebar renders a recursive `Notes/` tree with a draggable width splitter.
- Left sidebar also renders a "Tags" root (collapsed by default), refreshed on
  tree reload.
- Main pane supports edit, preview, or split view with a draggable splitter.
- View selector (top right) toggles edit/preview/split.
- Preview pane shows a sticky tag bar with clickable tag pills.
- Context menus:
  - Folder: New Folder, New Note, Rename, Delete, Expand/Collapse.
  - Note: New Note, Rename, Delete.
  - Sidebar empty area: New Folder, New Note.
- Refresh button reloads the tree.
- Search input lists matching notes; selecting one opens it.
- Editor and preview scroll independently from the sidebar.
- Editor and preview panes scroll together (proportional sync).

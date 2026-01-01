# NolderMD

NolderMD is a Go-based Markdown notes server with a discrete JSON API and web
UI. Notes are stored as `.md` files under the local `Notes/` directory.

## Run

```bash
go run ./cmd/noldermd serve --notes-dir ./Notes --port 8080
```

Open http://localhost:8080.

## Docker Compose

1) Use git to clone the repo
2) Docker build -t noldermd:local .
3) Use the compose file below to run the app

```yaml
services:
  noldermd:
    image: sottey/noldermd:local
    ports:
      - 8083:8080
    user: "1000:1000"
    volumes:
      - /path/to/notes/folder:/notes
networks: {}
```

The `user` entry keeps file ownership aligned with your host user so the app
can create notes and templates inside the mounted `/notes` folder.

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
- `GET /settings` (app settings)
- `PATCH /settings` `{ "darkMode": true, "defaultView": "split", "autosaveEnabled": false, "autosaveIntervalSeconds": 30, "sidebarWidth": 300, "defaultFolder": "Folder/Subfolder", "dailyFolder": "Folder/Subfolder", "showTemplates": true }`
- `GET /tasks` (lists tasks parsed from notes)
- `PATCH /tasks/toggle` `{ "path": "Note.md", "lineNumber": 12, "lineHash": "...", "completed": true }`

## Notes rules

- `.md` is appended on note creation when missing.
- Only `.md` files are treated as notes.
- Files starting with `._` are ignored.
- Tree responses return metadata only.
- Tags match `#` followed by letters, preceded by whitespace or start of line.
- If a folder contains `default.template`, new notes created in that folder use
  the template contents.

## Templates

- `default.template` in a folder provides the initial content for new notes
  created in that folder.
- Example: `Notes/00.Daily/default.template` applies to new notes under
  `00.Daily/`.
- Templates can include placeholders that are replaced when the note is created:
  `{{date:YYYY-MM-DD}}`, `{{time:HH:mm}}`, `{{datetime:YYYY-MM-DD HH:mm}}`,
  `{{day:ddd}}` or `{{day:dddd}}`, `{{year:YYYY}}`, `{{month:YYYY-MM}}`,
  `{{title}}`, `{{path}}`, `{{folder}}`. All date/time values use server-local
  time.
- Date/time placeholders must include the token name (for example,
  `{{date:YYYY-MM-DD}}`, not `{{YYYY-MM-DD}}`).

## Tasks rules

- Tasks are parsed on the fly from note contents; no `tasks.json` is used.
- A task line starts with optional whitespace then `- [ ] ` or `- [x] ` (space required after the bracket).
- Completed states accept `[x]`, `[X]`, or `[✓]`.
- Markers in the line: `#tag`, `@mention`, `+project`, `>due`, `^priority` (1-5). Only one project is used (first match wins).
- Due dates are parsed from the `>` marker; unrecognized formats are returned as warnings.

Example:
```
- [ ] Call Mom +Home #family @alice >2025-01-31 ^2
  - [x] File taxes +Finance >2025-02-01 ^1
```

## Settings rules

- Settings live in `Notes/settings.json` and are created automatically if missing.
- `darkMode` toggles the UI theme.
- `defaultView` controls the initial note view (`edit`, `preview`, `split`).
- `autosaveEnabled` and `autosaveIntervalSeconds` control note autosave.
- `sidebarWidth` stores the sidebar width in pixels.
- `defaultFolder` selects a folder dashboard on startup (relative to `Notes/`).
- `dailyFolder` opts into auto-creating a dated note in that folder on startup.
- `showTemplates` toggles visibility of `.template` files in the sidebar.

## UX behavior

- Left sidebar renders a recursive `Notes/` tree with a draggable width splitter.
- Left sidebar also renders a "Tags" root (collapsed by default), refreshed on
  tree reload.
- Left sidebar renders a "Tasks" root with projects, "No Project", and "Completed".
- Clicking Notes/Tags roots or any folder shows a summary panel.
- Clicking Tasks root or any project group shows a task list in the main pane.
- Folder and tag rows show centered chevrons indicating expanded/collapsed
  state.
- Main pane supports edit, preview, or split view with a draggable splitter.
- View selector (top right) toggles edit/preview/split.
- Settings button sits beside refresh in the sidebar header and opens a settings form.
- Settings include dark mode, default view, and autosave options.
- Settings include a Show Templates toggle for `.template` files.
- Settings are grouped into Display, Autosave, and Folders sections.
- Preview pane shows a sticky tag bar with clickable tag pills.
- Context menus:
  - Folder: New Folder, New Note, Edit Template, Rename, Delete, Expand/Collapse.
  - Note: New Note, Rename, Delete.
  - Sidebar empty area: New Folder, New Note.
  - Edit Template creates `default.template` if missing and opens it for editing.
- Refresh button reloads the tree.
- Search input lists matching notes; selecting one opens it.
- Editor and preview scroll independently from the sidebar.
- Editor and preview panes scroll together (proportional sync).

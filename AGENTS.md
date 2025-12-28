# NolderMD (project notes)

NolderMD is a Go-based web application with a separate HTTP API. It presents a
folder tree of Markdown notes from the local `Notes/` directory, with a main
pane that supports edit, preview, or split view.

This file documents the current project shape and interaction model.

## Product behavior (high level)
- Left sidebar shows the `Notes/` directory tree (folders + `.md` files) plus a
  tags root.
- Left sidebar shows a `Tasks` root with project groups, a "No Project" group,
  and a "Completed" group.
- Clicking Notes/Tasks/Tags roots or any folder/project group shows a summary
  panel in the main pane.
- Main pane shows a Markdown editor, a rendered preview, or a split view with a
  draggable splitter.
- Settings button in the sidebar header opens a settings form.
- The sidebar width is adjustable with a draggable splitter.
- A view selector in the top-right provides edit, preview, and split modes.
- Context menus:
  - Right-click on a folder: New Folder, New Note, Rename, Delete, Expand/Collapse.
  - Right-click on a note: New Note, Rename, Delete.
  - Right-click empty area in sidebar: New Folder, New Note.
- A Refresh button reloads the tree view.
- Tags are aggregated into a "Tags" root, collapsed by default, and refreshed
  when the tree reloads.
- The preview pane shows a sticky tag bar with clickable tag pills for the
  current note.
- Editor and preview panes scroll together (proportional sync).
- Folder and tag rows show centered chevrons indicating expanded/collapsed
  state.

## Architecture
- **CLI**: A Cobra-based entrypoint used to run the server and any future admin
  tasks (example: `noldermd serve --notes-dir ./Notes --port 8080`).
- **API**: A JSON HTTP API that handles notes and folder operations.
- **Web app**: A UI that consumes the API and renders the editor + preview.
- **Storage**: Notes live on disk in the `Notes/` tree as Markdown files.

## API responsibilities
- List a recursive folder tree and notes beneath a provided folder.
- Read/write note contents.
- Create/rename/delete folders.
- Create/rename/delete notes.
- Provide a refresh endpoint or tree reload operation.
- List tags extracted from note contents.
- Read/write tasks stored in `Notes/tasks.json`.
- Read/write settings stored in `Notes/settings.json`.

## UI responsibilities
- Render the folder tree and handle context menus.
- Render the tags root and tag groups.
- Render the tasks root and project groups.
- Render the Markdown editor, preview, and split view with draggable splitter.
- Render a task editor form that mirrors the note editor controls.
- Render a settings form (dark mode, default view, autosave, sidebar width, default folder, daily folder).
- Ensure tag labels remain legible in dark mode.
- Render a tag bar in the preview pane.
- Call API endpoints for all mutations and refresh operations.
- Provide filename/content search with a dropdown of matches.

## API shape (proposed)
- **Base path**: `/api/v1` (no auth for now).
- **Identifiers**: use a path string relative to `Notes/` for all note/folder ops.
- **Tree**:
  - `GET /api/v1/tree?path=<folder>` returns a recursive tree under `path`
    (metadata only).
  - If `path` is omitted, the full tree under `Notes/` is returned.
- **Notes**:
  - `GET /api/v1/notes?path=<file>` returns note content and metadata.
  - `POST /api/v1/notes` creates a note at `path` with content.
  - `PATCH /api/v1/notes` updates a note at `path` with content.
  - `PATCH /api/v1/notes/rename` renames a note from `path` to `newPath`.
  - `DELETE /api/v1/notes?path=<file>` removes the note.
- **Folders**:
  - `POST /api/v1/folders` creates a folder at `path`.
  - `PATCH /api/v1/folders` renames a folder from `path` to `newPath`.
  - `DELETE /api/v1/folders?path=<folder>` removes the folder.
- **Files**:
  - `GET /api/v1/files?path=<file>` serves a raw file (used for images).
- **Search**:
  - `GET /api/v1/search?query=<text>` searches note filenames + contents.
- **Tags**:
  - `GET /api/v1/tags` returns tags with the notes that contain them.
- **Tasks**:
  - `GET /api/v1/tasks` returns tasks.
  - `GET /api/v1/tasks/<id>` returns a task.
  - `POST /api/v1/tasks` creates a task.
  - `PATCH /api/v1/tasks/<id>` updates a task.
  - `DELETE /api/v1/tasks/<id>` deletes a task.
- **Settings**:
  - `GET /api/v1/settings` returns settings.
  - `PATCH /api/v1/settings` updates settings.
- **Health**:
  - `GET /api/v1/health` returns status.

## Data rules (confirmed)
- Tree responses include metadata only, never file contents.
- If a note path is missing the `.md` extension, it is appended on create.
- Only `.md` files are considered notes; other files are ignored.
- Files starting with `._` are ignored.
- Tags match `#` followed by letters, preceded by whitespace or start of line.
- Tasks live in `Notes/tasks.json` and use UUIDs for ids.
- Settings live in `Notes/settings.json`.
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

## Open questions to confirm
- None currently.

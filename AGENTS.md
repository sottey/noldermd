# NolderMD (project notes)

NolderMD is a Go-based web application with a separate HTTP API. It presents a
folder tree of Markdown notes from the local `Notes/` directory, with a main
pane that supports edit, preview, or split view.

This file documents the intended project shape and interaction model. Concrete
package layout and commands will be updated as they are implemented.

## Product behavior (high level)
- Left sidebar shows the `Notes/` directory tree (folders + `.md` files).
- Main pane shows a Markdown editor, a rendered preview, or a split view with a
  draggable splitter.
- The sidebar width is adjustable with a draggable splitter.
- A view selector in the top-right provides edit, preview, and split modes.
- Context menus:
  - Right-click on a folder: Edit (rename), New Note, New Child Folder, Delete.
  - Right-click empty area in sidebar: New Folder, New Note.
- A Refresh button reloads the tree view.

## Architecture (intended)
- **CLI**: A Cobra-based entrypoint used to run the server and any future admin
  tasks (example: `noldermd serve --notes-dir ./Notes --port 8080`).
- **API**: A JSON HTTP API that handles notes and folder operations.
- **Web app**: A UI that consumes the API and renders the editor + preview.
- **Storage**: Notes live on disk in the `Notes/` tree as Markdown files.

## API responsibilities (planned)
- List a recursive folder tree and notes beneath a provided folder.
- Read/write note contents.
- Create/rename/delete folders.
- Create/rename/delete notes.
- Provide a refresh endpoint or tree reload operation.

## UI responsibilities (planned)
- Render the folder tree and handle context menus.
- Render the Markdown editor, preview, and split view with draggable splitter.
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
  - `DELETE /api/v1/notes?path=<file>` removes the note.
- **Folders**:
  - `POST /api/v1/folders` creates a folder at `path`.
  - `PATCH /api/v1/folders` renames a folder from `path` to `newPath`.
  - `DELETE /api/v1/folders?path=<folder>` removes the folder.
- **Files**:
  - `GET /api/v1/files?path=<file>` serves a raw file (used for images).
- **Search**:
  - `GET /api/v1/search?query=<text>` searches note filenames + contents.
- **Health**:
  - `GET /api/v1/health` returns status.

## Data rules (confirmed)
- Tree responses include metadata only, never file contents.
- If a note path is missing the `.md` extension, it is appended on create.
- Only `.md` files are considered notes; other files are ignored.

## Open questions to confirm
- None currently.

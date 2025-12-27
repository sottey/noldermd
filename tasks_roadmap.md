# Tasks Feature Roadmap

## Goal
Add a Tasks feature backed by a JSON file in `Notes/`, with a Tasks tree in the sidebar and a single-pane task editor in the main pane.

## Storage
- File: `Notes/tasks.json`
- Create file if missing and notify the user.
- UUID per task (stable identifier, even for identical titles).

## Task model
- `id` (uuid string)
- `title` (string)
- `project` (string; empty/null means "No Project")
- `tags` (string array)
- `created` (ISO datetime)
- `updated` (ISO datetime)
- `duedate` (ISO date)
- `priority` (int, 1-5; 1 is most important)
- `completed` (bool)
- `notes` (string)
- `recurring` (reserved for future use)

## Sidebar tree
- `Tasks` root node (expand/collapse only)
  - One node per project
  - `No Project`
  - `Completed`
- Task leaf nodes appear under their project, or `No Project`, or `Completed`.
- Completed tasks appear only under `Completed` (not duplicated).

## Sorting
Within each project/No Project/Completed node:
- Primary: `duedate` ascending (oldest first)
- Secondary: `priority` descending (5 to 1)
- Tertiary: `updated` ascending

## UI behavior
- Clicking a non-task node (Tasks/project/No Project/Completed) expands/collapses it.
- Selecting a task opens a single-pane task editor in the main pane.
- Task editor should mimic the notes editor (save button, etc.).
- Context menus:
  - Tasks root: New Task, Refresh (and related actions as needed)
  - Project/No Project/Completed: New Task (pre-filled project), Refresh
  - Task node: Edit, Delete, Mark Complete, etc.

## API
- Introduce a new `/api/v1/tasks` API surface dedicated to tasks.
- Endpoints and payloads to be defined during implementation.
- Search should include tasks (title/notes and optionally tags/project).

## Search
- Tasks are included in search results.
- Search indexing should cover `title`, `notes`, and likely `tags`/`project`.

## Open items
- Define the API contract details (request/response shapes, errors).
- Decide where task list views appear if any (current plan: selecting a task opens editor; non-task nodes only expand/collapse).

# NolderMD Introduction

Welcome to NolderMD. This short guide helps you get set up and learn the
basics of notes, tasks, tags, and daily notes.

## Start the app

Run the server and open the UI in your browser:

```bash
go run ./cmd/noldermd serve --notes-dir ./Notes --port 8080
```

Then open http://localhost:8080.

## Notes and folders

- Notes are Markdown files stored under `Notes/`.
- Create folders and notes from the sidebar context menu.
- Only `.md` files are treated as notes.

## View modes

Use the view selector to choose:

- Edit
- Preview
- Split

## Tags

Add tags in your notes with `#tag` syntax. Tags are collected under the Tags
root in the sidebar.

Example:

```md
# Project Plan

- Kickoff meeting #planning
- Draft proposal #writing
```

## Tasks

Tasks live in `Notes/tasks.json` and appear under the Tasks root. Use the task
editor to create and update them.

## Templates

If a folder contains `default.template`, new notes created in that folder use
the template content. Placeholders are replaced at creation time.

Examples:

```md
# Daily Note - {{date:YYYY-MM-DD}}

Time: {{time:HH:mm}}
Title: {{title}}
Path: {{path}}
Folder: {{folder}}
```

This tutorial folder includes `default.template`. Create a new note in
`Tutorials/` and it will start with the template content above.

## Daily notes

If `dailyFolder` is set in settings and the folder exists, NolderMD creates a
note for today. If `default.template` exists in that folder, it is used.

## Settings

Open Settings from the sidebar header to configure:

- Dark mode
- Default view
- Autosave
- Sidebar width
- Default folder
- Daily folder

## Search

Use the search input to find notes by filename or content.

---

If you get stuck, check `README.md` for API details and rules.
#tutorial
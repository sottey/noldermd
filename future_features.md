## TODO

- [ ] Drag-and-drop in the sidebar tree:
  - Move notes/folders by dropping onto another folder.
  - Dragging a task project does nothing.
  - Dropping a task on another project updates its project.
- [ ] tasks in notes (if first char of a line is * followed by data, it is a task. +PROJECT >DUEDATE -PRI #tag #tag #tag)
    Example: * This is a task in the +Home project and is due >2025-12-27 and is priority -3 #home #test

## COMPLETED
- [X] Template use (context menu items?)
- [X] Default startup folder (Daily?)
- [X] Notes root should always be named Notes, not the root folder name
- [X] If there is a "Daily" folder and a note for this day in the form YYYY-MM-DD does not exist in it, create a new note with today's date in the format YYYY-MM-DD and, if in the daily folder there is a daily.template file, use it's contents for the new note

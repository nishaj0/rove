# repository instructions

## project context

- this repository contains a windows desktop app with two parts:
  - backend: go daemon in backend/
  - desktop ui: tauri + react in app/
- backend exposes a localhost api on 127.0.0.1:7331.

## coding style

- prefer small, focused changes over broad rewrites.
- preserve existing project structure and naming unless a change is required.
- keep code readable and avoid unnecessary abstraction for mvp work.

## comments

- add comments only where logic is non-obvious.
- always add comments for new functions and exported types.
- When referencing a function or type in a comment, ensure its name matches the exact case and is wrapped in quotation marks (e.g. "myFunction").
- avoid comments that simply restate what the code does; instead, focus on explaining why the code does something or any non-obvious implications.
- never add the function name as first word in a comment for that function. For example, instead of writing "myFunction does X", write "does X" and ensure the comment is directly above the function definition.
- do not flood files with comments.
- always start comment text with lowercase.
- avoid trivial comments that restate the code.

## frontend style

- keep the ui clean, minimal, and utility-focused.
- maintain readable text contrast and practical spacing.

## verification

- after backend edits, run: go build ./... from backend/ when possible.
- after frontend edits, run: npm run type-check from app/ when possible.
- if a command cannot be run, clearly state that in the response.

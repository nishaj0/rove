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
- do not flood files with comments.
- always start comment text with lowercase.
- avoid trivial comments that restate the code.

## frontend style

- keep the ui clean, minimal, and utility-focused.
- prefer dark + green theme accents unless the user asks for another theme.
- maintain readable text contrast and practical spacing.

## verification

- after backend edits, run: go build ./... from backend/ when possible.
- after frontend edits, run: npm run type-check from app/ when possible.
- if a command cannot be run, clearly state that in the response.

# Graphify for Codex

Use Graphify as the first orientation layer for this workspace when its output is available.

## Always-On Guidance

- Before answering architecture questions or starting broad file discovery, check whether `graphify-out/GRAPH_REPORT.md` exists.
- If `graphify-out/GRAPH_REPORT.md` exists, read it before using raw file search so you can navigate by:
  - god nodes
  - community structure
  - surprising connections
- If the graph output is missing or stale, say so briefly and then fall back to normal code search.

## Precise Graph Queries

Use the Graphify CLI when structural detail matters:

- `/graphify query` for targeted graph lookups
- `/graphify path` for hop-by-hop traversals between nodes
- `/graphify explain` for edge-level detail including relation type, confidence, and source location

Prefer these commands over wide grep/glob passes when the question is about architecture, ownership, or cross-cutting relationships.

## Keeping the Graph Fresh

Choose one of these approaches when Graphify is installed:

- `graphify hook install` to rebuild after commits and branch switches
- `/graphify ./raw --watch` in a background terminal for quick local refreshes during active development

## Codex Note

For parallel extraction support, Codex also needs this global setting in `~/.codex/config.toml`:

```toml
[features]
multi_agent = true
```

That setting is outside this repository, so update it separately if you want full Graphify parallel extraction behavior.

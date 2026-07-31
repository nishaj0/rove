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

## graphify

This project has a knowledge graph at graphify-out/ with god nodes, community structure, and cross-file relationships.

When the user types `/graphify`, invoke the `skill` tool with `skill: "graphify"` before doing anything else.

Rules:
- For codebase questions, first run `graphify query "<question>"` when graphify-out/graph.json exists. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts. These return a scoped subgraph, usually much smaller than GRAPH_REPORT.md or raw grep output.
- Dirty graphify-out/ files are expected after hooks or incremental updates; dirty graph files are not a reason to skip graphify. Only skip graphify if the task is about stale or incorrect graph output, or the user explicitly says not to use it.
- If graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read graphify-out/GRAPH_REPORT.md only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost).

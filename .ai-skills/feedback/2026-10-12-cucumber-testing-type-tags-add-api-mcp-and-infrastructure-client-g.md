---
status: draft
kind: improvement
skill: cucumber-testing
source:
  type: github
  path: https://github.com/InsonusK/ai-skills.git
  tree: master
commit: 00727d0ec3c2fc096f66a50aaea87d9c7a794f9e
skill_path: skills/testing/core/cucumber-testing.skill
created_at: 2026-10-12T09:03:18Z
---

# Type tags: add api/mcp and infrastructure/client/git

The `@type/…` table of "One type tag per feature" has no value for two kinds of adapter a real project needed (Go CLI `go-ai-skill-manager`):

| Feature | What it specifies | Value used | Why no listed value fits |
| --- | --- | --- | --- |
| `internal/mcpserver/features/mcpserver.feature` | an inbound adapter: an MCP server over stdio (tools an agent calls) | `@type/api/mcp` | transports are only `http`, `grpc`, `cli`, `message`; MCP is none of them — `message` means a consumer of a queue or topic |
| `internal/infrastructure/repository/features/git.feature` | an outbound adapter that runs the `git` process against a remote repository (clone, fetch by commit) | `@type/infrastructure/client/git` | protocols are only `http`, `grpc`; `infrastructure/file` does not fit either — the adapter talks to a remote service |

Proposal:
- add `mcp` to the transports of `@type/api/{transport}`;
- add `git` to the protocols of `@type/infrastructure/client/{protocol}` (or say which listed value a client that drives an external process should take);
- make the same change in `tools/testing/tags.json` of `solution-conformance-testing` and in the check list line that enumerates the values.

Related: the rule's Fix says "add a transport, engine, broker or protocol missing from the table to the table and to the project's `tools/testing/tags.json` before using it". A project cannot edit the table (the skill is synced from another repository), so only the second half is doable in a project — it would help to say so explicitly.

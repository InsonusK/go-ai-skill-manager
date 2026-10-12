---
status: sent
kind: bug
skill: solution-conformance-testing
source:
  type: github
  path: https://github.com/InsonusK/ai-skills.git
  tree: master
commit: 00727d0ec3c2fc096f66a50aaea87d9c7a794f9e
skill_path: skills/testing/core/solution-conformance-testing.skill
created_at: 2026-10-12T09:03:18Z
issue_url: https://github.com/InsonusK/ai-skills/issues/215
sent_at: 2026-10-12T09:08:23Z
---

# tags.json: verbatim copy contradicts adding project tag values

Two rules contradict each other about `tools/testing/tags.json`.

- `solution-conformance-testing`, `Implementation/Repository.create.md`: "Copy verbatim, as a folder, to `tools/testing/` … `tags.json`. Identical in every stack", and the check list: "`tools/testing/` (except `kinds/`) and `tools/livingdoc/` are byte-for-byte copies of the assets".
- `cucumber-testing`, "One type tag per feature", Fix: "add a transport, engine, broker or protocol missing from the table to the table and to the project's `tools/testing/tags.json` before using it".

A project that needs a value outside the table (here: `api/mcp` for an MCP stdio server and `infrastructure/client/git` for a git process adapter) has to edit `tags.json`, and then fails the "byte-for-byte copy" check item. Without the edit `kind_scenarios_check` reports the feature as having an unknown type.

Proposal — one of:
- exclude `tags.json` from the verbatim rule: it is the project's own list, seeded from the asset (name it next to `kinds/` in the check list);
- or keep `tags.json` verbatim and read project additions from a second file (for example `tools/testing/tags.local.json`) merged by the normalizers, the tag check and the legend.

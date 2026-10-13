# patch

Change a file that already exists. The model sends an edit in the format its capability profile names (`edit_format`): a **unified diff** or **SEARCH/REPLACE blocks**. The model only ever sees one of the two schemas. Format decision: [ADR-004](../adrs.md#adr-004--edit-formats-and-apply-ladder).

| | |
|--|--|
| **Args** | `udiff`: unified diff. `search_replace`: path plus one or more `{search, replace}` pairs. |
| **Limit** | One file per call |
| **Permission** | edit |

Malformed edits are the top risk for small models, so the harness tries a fixed ladder and stops at the first step that applies cleanly:

1. **Exact.** `git apply` for a diff; exact unique match for SEARCH/REPLACE.
2. **Whitespace-tolerant.** `git apply --recount --ignore-whitespace`; for SEARCH/REPLACE, match ignoring leading/trailing whitespace and line-ending differences.
3. **Fuzzy locate.** Find the hunk's context or search block by line similarity in the current file. Applied only if there is **exactly one** match above the threshold; ambiguous matches are never applied.
4. **Reject.** The file stays as it was. The error names the failing hunk and shows the actual current lines nearest to where it expected to match, so the repair turn has what it needs.

A patch on a file the user changed since the model last read it is rejected before the ladder ("file changed since you read it"). See [tool runtime](../tool-runtime.md#files).

The tool returns the path, a short diff stat, and which ladder step applied. Observability records `apply_strategy` per call, and the eval reports `percent_cases_well_formed` (applied at step 1–2) separately from fuzzy applies, so tolerance does not hide a model that cannot write edits.

## Comparison

| Harness | What it does | Closer to |
|---------|--------------|-----------|
| **thee** | Unified diff or SEARCH/REPLACE, picked per model. One file. Exact, then whitespace-tolerant, then unique fuzzy match, then a targeted reject. |  |
| **OpenCode** | Search-replace and whole-file edits are both common. |  |
| **Aider** | Several edit formats chosen per model, with flexible matching when a hunk does not apply exactly. | closest |
| **Ante** | Edits are whatever tools the profile exposes. |  |
| **Pi** | Patch or write, depending on the tool pack. |  |
| **Goose** | A developer extension edits files. The format is not fixed. |  |

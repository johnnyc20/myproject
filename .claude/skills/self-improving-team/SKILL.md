---
name: self-improving-team
description: Runs a non-trivial code change to this Go API server through a planner → implementer → critic → tester pipeline, with a bounded reflection loop between critic and implementer, instead of writing the change directly. Use when asked to build a feature or fix a bug "properly"/"with review", or explicitly asked to run the self-improving team / dev team / agent pipeline.
---

# Self-Improving Team

Four Claude Code subagents cannot call each other directly — a subagent has
no `Agent` tool, so orchestration has to happen at the top level, in this
session. This skill is that orchestration: it directs you (the top-level
assistant) to call each agent below in sequence, in the current conversation,
rather than making the code change yourself.

Don't use this for a trivial one-line fix — the planning and review overhead
isn't worth it. Use it for anything that touches more than one file or one
layer (`internal/store` + `internal/api` together), or whenever the user
explicitly asks for it.

## Pipeline

1. **Plan** — call the `planner` agent with the request. It reads the repo
   and returns a numbered plan plus any open questions. If it raises open
   questions, resolve them with the user before moving on — don't let
   `implementer` guess.

2. **Implement** — call the `implementer` agent with the plan from step 1.
   It writes the code, runs its own one-pass self-review, runs `make vet`
   and `make test`, and reports back what changed.

3. **Critique** — call the `code-reviewer` agent to review the diff
   `implementer` just produced.
   - If it reports findings: call `implementer` again with those specific
     findings as the instruction (not the original plan — just the fixes).
     Then call `code-reviewer` again.
   - **Bound this loop at 2 rounds of critique.** If round 2 still has
     findings, stop looping — report the remaining findings to the user
     instead of continuing indefinitely. A reflection loop that never
     terminates is worse than no reflection.
   - If it reports nothing: proceed to step 4.

4. **Test** — call the `tester` agent to check edge-case coverage on the
   now-approved diff and run `make test`.
   - If it finds and fills coverage gaps, that's expected and not a
     failure — proceed to step 5.
   - If `make test` fails for a reason unrelated to coverage, treat that as
     a real regression: send it back to `implementer` as a fix, once, then
     re-run `tester`. Don't loop this stage more than once.

5. **Report to the user** — summarize the whole run: the plan, what
   implementer built, how many critique rounds ran and what they found (if
   any), and the final test result. If the critique loop hit its round
   limit with unresolved findings, say so plainly rather than presenting the
   change as fully clean.

## Why this shape

- Rule 1 (reflection) shows up twice: inside `implementer`'s own one-pass
  self-review, and again as the bounded critic↔implementer loop in step 3 —
  self-review catches what the author can see, peer critique catches what
  they can't.
- Rule 2 (tool use) is every agent in the pipeline actually reading files
  and running `make vet`/`make test`, never asserting a change works from
  memory.
- Rule 3 (planning) is step 1 running before any code exists.
- Rule 4 (multi-agent collaboration) is the four distinct roles — planner,
  implementer, critic, tester — each with a narrow job and its own tool
  scope, rather than one agent doing everything.
</content>

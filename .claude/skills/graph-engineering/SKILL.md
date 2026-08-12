---
name: graph-engineering
description: Reference this skill whenever designing, extracting into, querying, or evolving myproject's LESSONS.md-derived knowledge graph — schema decisions (node/edge types, the error_type taxonomy), the extraction pipeline (lessons_to_graph.py), the retrieval pipeline (query_lessons_graph.py), or measuring whether the graph is actually working. Both the graph-architect and graph-evaluator subagents should read this before acting.
---

# Graph Engineering for myproject

## Overview

myproject's knowledge graph exists for one reason: to stop the same mistake from recurring, by surfacing a relevant past lesson at the moment it matters, at a token cost that's worth paying every session. Every schema, extraction, and retrieval decision should be judged against that — not against "is this more complete" or "is this more elegant."

Three roles touch this system and should stay separated:
- **What the graph looks like** (node/edge types, taxonomy) — owned by `graph-architect`.
- **Whether it's working** (recurrence, retrieval relevance, taxonomy drift) — owned by `graph-evaluator`.
- **Whether the data in it is clean** (dedup, staleness, orphaned edges) — owned by `memory-curator`, covered in its own subagent, not here.
- **Whether it costs too much to load** — owned by `context-budget`, covered in its own subagent, not here.

## Core schema

- **Node types**: keep this list short. A new node type is justified only when a recurring pattern genuinely can't be expressed as a property on an existing node type. Prefer adding a property or taxonomy value over adding a type.
- **Edge types**: `caused_by`, `fixed_by`, `occurs_in` (lesson → file/component) are the load-bearing ones — they're what makes retrieval traversal meaningful rather than just a flat lookup. Every edge type must answer "what question does traversing this edge let me answer that a flat list couldn't?" If it doesn't, it's not earning its place.
- **`error_type` taxonomy**: a fixed, controlled vocabulary — Kitchen Sink, Wrong Abstraction, Optimistic Path, Runaway Refactor, Other. This is deliberate: a controlled vocabulary lets the graph consolidate near-duplicate failure patterns into one node instead of fragmenting into free-form variants that never match each other in a query. Extraction must map to one of these values, never invent new ones inline — taxonomy changes go through `graph-architect`, not through the extraction prompt improvising.

## Schema evolution principles

- **Evidence before change.** Schema changes are expensive — they require re-extraction or migration of existing data. Never evolve the schema speculatively; require a specific finding (from `graph-evaluator`, or a concrete recurring case) as the trigger.
- **Prefer the smallest change that fixes the finding.** Order of preference: (1) tune a taxonomy value or property, (2) adjust extraction/retrieval logic within the existing schema, (3) add a new edge type, (4) add a new node type. Each step down this list costs more to migrate and maintain — don't skip straight to (4) when (1) would resolve the actual finding.
- **Version the schema.** Any schema change should be a deliberate, documented step (a comment/changelog in `lessons_to_graph.py` noting what changed and why), not a silent drift between what old nodes look like and what new ones look like. Old nodes should either be migrated or explicitly still valid under the new schema — don't leave the graph half-migrated.
- **"Other" bucket growth is a signal, not a failure to hide.** If Other grows past roughly 10-15% of nodes, that's evidence the taxonomy is missing a real category — worth architect's attention. It is not something to solve by hand-relabeling individual nodes (that's a data-hygiene band-aid, not a schema fix).

## Extraction pipeline (`lessons_to_graph.py`)

- The extraction prompt should be **narrow and structural**: given one LESSONS.md entry, output nodes/edges in the exact schema, nothing else. It is not the place for the model to exercise judgment about whether a lesson is worth keeping (that's `memory-curator`'s job, upstream) or how the schema should be shaped (that's `graph-architect`'s job).
- It should be **idempotent and incremental** — safe to re-run on the same entry without creating duplicate nodes, and cheap to run only on new/changed entries rather than reprocessing the whole file every time.
- If extraction is Haiku-tier (per myproject's model-tier pattern), the prompt needs explicit structure and 1-2 worked examples per node type — Haiku will not reliably infer schema conventions from a description alone. Route prompt-wording changes (as opposed to schema changes) to `prompt-engineer`.

## Retrieval pipeline (`query_lessons_graph.py`)

- Retrieval must be **scoped**, not exhaustive: a query should be triggered by something specific (current file, current error signature, current task type) and return a bounded, ranked set of nodes — not a broad traversal that returns most of the graph back in through the side door. An unscoped "retrieval" step is just injection with extra steps, and `context-budget` will flag it as such.
- Ranking should favor **specificity over recency** by default — a highly specific lesson about the exact file/pattern in question beats a recent-but-generic one. Recency can be a tiebreaker, not the primary signal.
- Retrieval quality is not self-evident from the code being correct — a query can run without error and still return irrelevant nodes. That's what `graph-evaluator`'s relevance sampling is for.

## Measuring effectiveness (what "self-improving" actually means here)

A graph that's schema-correct and data-clean can still be failing at its actual job. The signals that matter:

1. **Recurrence**: did the same `error_type` + file/component combination happen again *after* a lesson covering it was already in the graph? If yes, the graph existing didn't change behavior — the problem is retrieval scope, injection visibility, or the lesson's actionability, not just "add more lessons."
2. **Retrieval relevance**: for a realistic query, are the returned nodes actually useful to the task at hand, or just topically adjacent? This requires human or model judgment on samples — it can't be fully automated by checking the code runs.
3. **Taxonomy fit**: is `error_type` classification staying meaningful, or is everything drifting into one or two buckets (or Other)?
4. **Dead weight**: nodes that no realistic query would ever surface are pure token/maintenance cost with no payoff — candidates for pruning (data hygiene) or evidence that extraction is too granular (schema issue).

These four signals are `graph-evaluator`'s job to measure and report. `graph-architect` acts on schema-shaped findings (3, and granularity issues in 4); `memory-curator` acts on data-shaped findings (dead nodes in 4); `context-budget` acts on injection-shaped findings (1, when the cause is visibility rather than the lesson's existence).

## Anti-patterns to watch for

- **Schema sprawl**: a new node type per lesson category "just in case." This makes the graph harder to query, not more expressive.
- **Taxonomy explosion**: adding a category for every edge case instead of tolerating some Other. A taxonomy with 15 categories has the same fragmentation problem free-form text had.
- **Retrieval as injection**: a "query" that always returns the same broad result regardless of input isn't retrieval — it's SessionStart injection wearing a query-shaped costume, and it should be flagged to `context-budget`.
- **Coupling schema to wording**: if changing the extraction prompt's phrasing requires also changing the schema (or vice versa), the two are too tightly coupled — schema should be stable enough that prompt-engineer can iterate on wording without architect needing to change node/edge definitions.

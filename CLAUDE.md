<!-- psyduck -->

## PsyDuck Spec Lifecycle

This project uses [psyduck](https://github.com/psyduck/psyduck) for spec lifecycle management. Every code package (Go package, JS/TS module, Python package, Rust crate, etc.) may have a `SPEC.md` describing its purpose and public API; design and implementation docs live under `docs/` until archived to `.psy/`.

### `/psy-sync` — Close out a finished change set

**Trigger (MANDATORY, both conditions):**

1. **Automatic** — Immediately after `superpowers:executing-plans` finishes (i.e., after the implementation plan is fully executed, as part of closing out the work). Do not skip this step.
2. **Manual** — Whenever the user runs `/psy-sync` directly.

**What it does:**

- **Generate / update `SPEC.md`** for every code package affected by the current change set. The change set is inferred from context: the plan that just executed, the working tree, recent edits — never hardcoded against a specific base branch.
- **Archive design docs** produced by this work (typically under `docs/superpowers/specs/`) into `.psy/YYYY-MM-DD/<doc-name>.md`. Move with `git mv`, don't copy. Append `-2`, `-3`, ... on name collision.
- **Sync the repo-root `README.md`** to the change set: reconcile code-derived facts (commands, flags, exit codes, directory layout, build/test commands, dependencies, skill reference) and add terse entries mirroring existing structure for newly added surface. Editorial prose is preserved.

See the `/psy-sync` skill for the full procedure.

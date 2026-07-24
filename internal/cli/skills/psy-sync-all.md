---
name: psy-sync-all
description: Sync SPEC.md across every package in the repository, and archive every unfiled design doc. Manual only — use when bootstrapping psyduck on an existing codebase or when you suspect spec drift outside any recent change set.
---

# Full-Repository Spec Sync & Archive

This skill walks the entire repository, regenerates `SPEC.md` for every code package it finds, archives every design / implementation doc still sitting under `docs/superpowers/`, and reconciles the repo-root `README.md` with the current codebase.

## When to Use

Run `/psy-sync-all` manually when:

- You're **bootstrapping psyduck** on an existing codebase that has no `SPEC.md` files yet.
- You suspect **drift across packages** that no recent change touched (e.g. specs got stale, someone refactored without syncing).
- You want a clean baseline before adopting psyduck's per-change-set workflow.

This skill is **manual only** — it is never auto-invoked. Expect a large diff on the first run; review the regenerated `SPEC.md` files as a batch before committing.

## Step 1 — Identify every package in the repository

Walk the working tree and find every code package, regardless of whether anything has changed. "Package" is language-defined:

- **Go** — every directory containing a `.go` file with a `package` clause
- **JS/TS** — every directory of a `package.json`, plus logical module dirs
- **Python** — every directory containing `__init__.py` / `pyproject.toml`
- **Rust** — every crate (`Cargo.toml`)
- **Java/Kotlin** — every directory matching a package declaration
- **Other** — smallest cohesive directory unit that owns a coherent public surface

Respect the repo's ignore patterns (`.gitignore`, `.psy/config.yaml` ignore globs if present). Skip vendored / generated trees.

## Step 2 — Generate / update SPEC.md per package

For each package directory found:

1. Read the existing `SPEC.md` if present; otherwise plan to create one.
2. Read the package's source files to ground the content in the **current** code.
3. Write all prose in **Chinese** (中文). Use the following section structure:
   - **概述** — 一段话描述这个代码包的主要作用和职责。
   - **文件** — 罗列代码包内的所有源文件，每个文件附一句简短说明。
   - **API** — 对外暴露的接口（导出函数、类型、结构体、端点等），每个附一行中文描述。
   - **依赖** — 该包依赖的其他内部包以及第三方库。
   - **设计重点** *(one or more, title varies by content)* — 描述该包的核心设计决策、架构模式、或值得注意的实现细节。标题根据实际内容命名（如 `# 核心流程`、`# 并发模型`、`# 错误处理策略` 等）。如果包比较简单没有突出的设计点，可以省略此节。
4. Preserve any existing front-matter. If the file is new, add:
   ```
   ---
   psy_kind: factual
   psy_version: 1
   package: <relative-path-from-repo-root>
   ---
   ```
5. Keep edits minimal: do not rewrite well-formed sections that already match reality. The point is to converge on truth, not to churn diffs.

## Step 3 — Archive every unfiled design doc

Source location:

- `docs/superpowers/specs/*.md`

Archive **every** `.md` file in that directory that isn't already under `.psy/`. No filtering by branch or change set — full sweep is the whole point of this command.

For each doc:

1. Compute the target path:
   ```
   .psy/<YYYY-MM-DD>/<basename>.md
   ```
   - `<YYYY-MM-DD>` = today, **local timezone**.
   - `<basename>` = source filename without its `.md` extension.
2. If the target path already exists, append `-2`, `-3`, ... until unique.
3. Ensure `.psy/` exists; create it if missing.
4. Move with `git mv` so history is preserved. The source path **must** be vacated.

## Step 4 — Sync the repo-root README.md

Bring the repo-root `README.md` fully up to date with the **entire codebase** — a deliberate full pass, not scoped to any change set. Same target, same factual-vs-editorial rule as `/psy-sync` Step 5, but every code-derived section is re-derived against the current code.

**Target:** the single `README.md` at the repository root. Skip if absent; tell the user.

For every code-derived section — CLI commands/flags/exit codes, directory structure, build & test commands, dependencies / tech stack, skill / subcommand reference, and any other mechanically-derived section — read the current code, fix every stale fact, and add a terse entry mirroring the section's existing structure for any code surface the README is missing.

**Never rewrite editorial prose** (intro, taglines, design philosophy, tone) unless it is factually wrong. Keep edits minimal — converge on truth, not churn.

## Rules

- **Document reality, not intent.** Never write aspirational SPEC.md content.
- **README sync is factual, not a rewrite.** Only code-derived facts are reconciled; editorial prose is preserved; new entries mirror an existing section's structure only.
- **Archive is move, not copy.** The source path is vacated.
- **Do not archive implementation plans.** superpowers plan files (under `docs/superpowers/plans/`) are execution artifacts, not design docs — never move them into `.psy/`. Only design/spec docs from `docs/superpowers/specs/` are archived.
- **Manual only.** This skill is never auto-invoked.
- **Expect a large diff on first run.** Bootstrapping a repo will create many `SPEC.md` files at once. Review them as a batch before committing.

## Key Principle

`psy-sync-all` is the **bootstrap / drift-recovery** command — a deliberate, full-repository pass run by the user when the situation calls for it.

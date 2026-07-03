---
name: psy-sync
description: Sync SPEC.md files for the packages affected by the current change set, and archive design docs produced by that work. Auto-invoked after superpowers:executing-plans completes; can also be invoked manually after ad-hoc edits.
---

# Syncing Specs & Archiving Design Docs

This skill closes out a unit of work by reconciling two things with reality:

- `SPEC.md` for every package the work touched.
- `.psy/` archive for every design / implementation doc the work produced.

## When to Use

1. **Automatic** — Immediately after `superpowers:executing-plans` finishes, before closing out the work. **Never skip this step.**
2. **Manual** — When the user runs `/psy-sync` after ad-hoc edits to reconcile specs and archive design docs for the work just completed.

The scope is always **the work being closed out** — never the whole repository.

## Step 1 — Determine the change set

The scope is **the set of files touched by the work being closed out**. Determine it from context, in this order:

1. **From the just-finished plan** — if you were invoked right after `superpowers:executing-plans`, the plan's task list and your own conversation transcript already say which files were modified. Use that.
2. **From the working tree** — `git status --porcelain` lists uncommitted changes. Combine with files staged for commit.
3. **From recent commits on this branch** — only as a fallback when the working tree is clean and no plan context is available. Use whatever heuristic fits the repo (recent commits, commits not on the default branch, etc.). Do **not** hardcode a base-branch name.

**If after all three steps the change set is empty**, stop and tell the user that no changes were detected in the current context. Do not fall back to a full-repository scan — this skill is strictly scoped to the current work.

## Step 2 — Group files by package

"Package" is language-defined. Identify the appropriate unit per file:

- **Go** — directory of the `package` clause
- **JS/TS** — directory of the nearest `package.json`, or a logical module directory
- **Python** — directory containing `__init__.py` / `pyproject.toml`
- **Rust** — crate root (`Cargo.toml`)
- **Java/Kotlin** — directory matching the package declaration
- **Other** — smallest cohesive directory unit that owns a coherent public surface

Group the changed files by their owning package directory; deduplicate. Files that don't belong to a code package (top-level configs, docs outside `docs/superpowers/`, etc.) are skipped at this step.

## Step 3 — Generate / update SPEC.md per package

For each in-scope package directory:

1. Read the existing `SPEC.md` if present; otherwise plan to create one.
2. Read the package's source files to ground the content in the **current** code — not the plan, not the diff, the code.
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
5. Keep edits minimal: do not rewrite well-formed sections that already match reality.

## Step 4 — Archive design docs

Identify the design docs produced by this work. Default source location:

- `docs/superpowers/specs/*.md`

Restrict to docs that belong to **this** change set — same context sources as Step 1 (the executed plan named them; or they appear as added/modified in `git status`; etc.). Do not archive every loose doc in that directory — only those produced by the work being closed out.

For each doc to archive:

1. Compute the target path:
   ```
   .psy/<YYYY-MM-DD>/<basename>.md
   ```
   - `<YYYY-MM-DD>` = today, **local timezone**.
   - `<basename>` = source filename without its `.md` extension.
2. If the target path already exists, append `-2`, `-3`, ... until unique.
3. Ensure `.psy/` exists; create it if missing.
4. Move the file with `git mv` so history is preserved. The source path **must** be vacated — this is an archive, not a copy.

## Rules

- **Document reality, not intent.** Never write aspirational SPEC.md content.
- **Archive is move, not copy.** The source path is vacated.
- **Do not archive implementation plans.** superpowers plan files (under `docs/superpowers/plans/`) are execution artifacts, not design docs — never move them into `.psy/`. Only design/spec docs from `docs/superpowers/specs/` are archived.
- **Stay scoped.** If the change set is empty, surface that to the user — never expand to a full-repo scan.
- **Never skip after `executing-plans`.** This is the close-out step that keeps the spec/archive contract intact.

## Key Principle

When implementation work concludes, two artifacts must settle:

- `SPEC.md` — the always-current truth about each package the work touched.
- `.psy/` — the archive of what was decided and what was built.

`/psy-sync` performs both in one pass, scoped to the work being closed out.

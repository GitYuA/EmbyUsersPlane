# AGENTS.md

## Scope
- This file applies to the whole repository.
- A deeper `AGENTS.md` in a subdirectory overrides this file only for that subdirectory.

## Instruction Priority
1. System instructions
2. Developer instructions
3. This `AGENTS.md`
4. User request

## Project Facts
- Backend entry: `main.go` (Go).
- HTML templates: `templates/`.
- Frontend assets: `public/` and `public/assets/`.
- Runtime data: `data/` (production data, do not change unless requested).
- Container files: `Dockerfile`, `docker-compose.yml`.

## Default Working Style
- Default response language: Chinese.
- Keep responses concise and outcome-first.
- For coding requests, implement first, then explain.

## Skill Rules
- If the user explicitly names a skill, use it.
- If the task clearly matches a known skill, prefer that skill.
- If a skill is missing or blocked, state it briefly and continue with a practical fallback.

## Change Boundaries
- Do not modify files under `data/` unless explicitly requested.
- Do not change port semantics unless requested (`8086` admin, `8085` query).
- Do not silently rename API action keys used by frontend/backend contracts.
- Never revert unrelated local user changes.

## Encoding and Text Rules
- Save text files as UTF-8 without BOM.
- Avoid introducing garbled text (for example replacement character `U+FFFD`).
- When editing HTML, verify attribute quote closure, especially around `onclick` and `selectOption(...)`.

## Validation Rules
- After Go changes, prefer:
1. `go test ./...`
2. `go build ./...`
- If local Go is unavailable, use container build fallback:
1. `docker build -t <image>:<tag> .`
- After template/frontend changes, at minimum run:
1. `rg -n "\\?\\?" templates public`
2. quick syntax-risk checks for broken quotes in event attributes
- If validation cannot run, explicitly state what was not verified.

## Communication Rules
- Before major edits, send a short 1-2 sentence progress note.
- After completion, include:
1. changed files
2. validation performed
3. remaining risks or next steps

## Failure Fallback Template
1. What failed
2. What was attempted
3. Next actionable step
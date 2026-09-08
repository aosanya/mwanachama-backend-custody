# mwanachama-backend-custody — documentation

## Layout

Four folders, in SDLC order, and everything lives under one of them.

| Folder | What's inside |
|--------|---------------|
| [1. requirements/](1.%20requirements/) | Problem, vision and scope for the audit/compliance cluster extracted out of the gateway. |
| [2. design/](2.%20design/) | The custody/export/contact/consent schema and how it maps onto GORM. |
| [3. implementation/](3.%20implementation/) | The work: `todo.md` (open board), `todo_done.md` (completed rows + board context). |
| [4. qa/](4.%20qa/) | Test coverage and results. |

## Boards and status

| File | What it holds |
| --- | --- |
| [todo.md](3.%20implementation/todo.md) | Open task board |
| [todo_done.md](3.%20implementation/todo_done.md) | Completed rows + board context |

## What this repo is

Extracts `mwanachama-backend-api-gateway`'s `internal/domain/{custody,export,
contact,consent}` packages onto GORM-backed relational storage, imported
directly by [mwanachama-backend-api-gateway](../mwanachama-backend-api-gateway)
— no gRPC, no sub-service shape. Same template as
[mwanachama-backend-actor](../mwanachama-backend-actor) and
[mwanachama-backend-comm](../mwanachama-backend-comm). Decided 2026-09-06 —
see the gateway's `documentation/2. design/design-register.md` and
`architecture-domain-decomposition.md`. Built for the gateway's
`documentation/3. implementation/todo_custody_standup.md`, DEV-1640…1648.

# Project: CTF Submission Portal Security Audit & Assessment

## Architecture Overview
- **Product**: SSH-based CTF Submission Portal with Elm-style Bubble Tea TUI, SQLite WAL persistence, bcrypt authentication, and OpenPGP clearsign verification.
- **Components**:
  - `cmd/server/main.go` & `internal/server/`: Charmbracelet Wish SSH Server listening on port 2222 with middleware stack.
  - `cmd/admin/main.go`: CLI tool for CTF organizers (team management, rounds, scoreboard, hints, database migrations/backups).
  - `internal/auth/`: Password verification via bcrypt, OpenPGP armored key handling, detached signature checking.
  - `internal/scoring/`: Flag submission validation, dynamic scoring decay, first-blood bonuses, hint challenges/redemption, round skipping, team score breakdown.
  - `internal/store/`: SQLite driver with WAL mode, single open connection serialization, schema migrations, and views.
  - `internal/tui/`: Bubble Tea interactive TUI with 8 distinct views.
  - `internal/middleware/`: Connection rate limiting, session concurrency caps, session/idle timeouts, and text sanitization.

## Feature & Scope Inventory
| # | Feature / Area | Description | Milestone | Source |
|---|----------------|-------------|-----------|--------|
| 1 | Architecture & Entry Points | Enumerate SSH daemon, CLI commands, TUI views, and middleware pipeline | M1 | Survey (Explorer 1) |
| 2 | Threat Modeling & Trust Boundaries | Map threat actors, assets, trust boundaries, and attack surface matrix | M1 | Survey (Explorer 1) |
| 3 | Authentication & Session Vulnerabilities | Case sensitivity disconnect, pre-auth bcrypt DoS, session state desync | M2 | Survey (Explorer 2, 3) |
| 4 | Authorization & Registration Gaps | Registration bypass on flag submission/hints/skips (`ErrNotRegistered` unused) | M2 | Survey (Explorer 2) |
| 5 | Scoring & Business Logic Vulnerabilities | Skip-then-solve bypass, zero-cost skip, score ledger truncation (>10k rows) | M2 | Survey (Explorer 3) |
| 6 | Concurrency & Data Integrity Risks | Lack of unique constraints on `hint_usage`, TOCTOU in skips, double point deductions | M2 | Survey (Explorer 2, 3) |
| 7 | Data Exposure & Information Leaks | Plaintext secret flag storage in `submissions` table, database backup leakage | M2 | Survey (Explorer 2) |
| 8 | Input Validation & Injection Risks | Dead sanitization routines, ANSI terminal escape injection in scoreboard/header | M2 | Survey (Explorer 1, 2, 3) |
| 9 | Rate Limiting & DoS Vectors | In-memory token bucket drain during cooldown, inactive round operations | M2 | Survey (Explorer 2, 3) |
| 10 | Code-Level Remediation Specifications | Concrete code patches for all findings (Go code, SQL migrations, config) | M3 | Survey (Explorers 1-3) |
| 11 | Functional Compatibility Verification | Ensure proposed fixes preserve legitimate CTF competition operations | M3 | Survey (Explorers 1-3) |
| 12 | Final Audit Report Compilation | Generate `E:\ieee_ctf\submission-portal\SECURITY_AUDIT_REPORT.md` (R1-R4) | M4 | Synthesis & Worker |
| 13 | Multi-Agent Audit & Gate Verification | Independent Reviewers, Challengers, and Forensic Auditor evaluation | M5 | Reviewers / Auditor |

## Milestones
| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| M1 | Attack Surface & Threat Matrix | Complete inventory of all interfaces, actors, boundaries, and data flows (R1) | Survey | DONE |
| M2 | Vulnerability Identification & Root Causes | Detailed analysis of all High/Med/Low vulnerabilities with code line citations (R2) | M1 | DONE |
| M3 | Hardening & Remediation Specifications | Actionable Go/SQL patch specifications maintaining CTF compatibility (R3) | M2 | DONE |
| M4 | Comprehensive Security Audit Report Delivery | Author `SECURITY_AUDIT_REPORT.md` in `E:\ieee_ctf\submission-portal` (R4) | M1, M2, M3 | DONE |
| M5 | Multi-Agent Verification & Gate Sign-off | 2 Reviewers, 2 Challengers, and Forensic Auditor verification | M4 | DONE |

## Code Layout
- Target Report: `E:\ieee_ctf\submission-portal\SECURITY_AUDIT_REPORT.md`
- Codebase Location: `E:\ieee_ctf\submission-portal/`
- Agent Metadata & Logs: `E:\ieee_ctf\.agents/`

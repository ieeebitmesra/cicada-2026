# IEEE CTF — Submission Portal TUI

Scoring engine served over SSH with a Bubble Tea terminal UI. Teams log in with
`ssh <team> -p 2222`, register (new password + PGP public key), then submit
flags, request hints, skip rounds and watch the live scoreboard.

## Security layers

| Layer | Control | Where |
|-------|---------|-------|
| Network | nftables default-deny + fail2ban | `../host-hardening/` |
| SSH transport | per-IP token bucket, global/per-IP session caps, hard timeout | `internal/middleware/` |
| Application | ANSI/CSI/OSC/DCS stripping, control-char filter, flag format + rate limit | `internal/middleware/sanitize.go`, `internal/scoring/validator.go` |
| Auth | bcrypt password hashing; PGP clearsigned hint requests | `internal/auth/` |
| Resource | single-writer SQLite (WAL), submission cooldowns, idle-timeout quit | `internal/store/db.go`, `internal/tui/root.go` |
| Monitoring | wish logging middleware → journald → fail2ban/SIEM | `internal/server/server.go` |

## Scoring rules

| Action | Cost |
|--------|------|
| Clear a round | +round points |
| Skip a round | −50% of your **current total** |
| Plain hint | −20% of the **round's** points |
| Encoded hint (you decode it) | −10% of the **round's** points |

Hint requests must be PGP clearsigned:

```
HINT-REQ
round=3
type=plain
index=0
ts=<RFC3339>
team=<ssh-user>
```

```bash
gpg --clearsign --local-user YOURKEY   # paste the challenge, output the signed block
```

The server verifies the signature against your registered key, checks the
challenge freshness window, then logs the proof in `hint_usage` before showing
the hint.

## Layout

```
cmd/server        Wish SSH server entry point
cmd/admin         operator CLI (teams, rounds, scoreboard, db)
internal/auth     bcrypt login + PGP clearsign verification
internal/middleware  rate limit / session caps / timeout / input sanitization
internal/scoring  engine (hint & skip costs), flag validator, game service
internal/server   config loader + Wish server assembly
internal/store    SQLite layer + migrations runner (WAL, FK on)
internal/tui      root state machine, styles, components/, views/
migrations        001_initial.sql (teams, rounds, submissions, hint_usage, skips, scoreboard view)
configs           server.yaml (limits) + rounds.yaml (flags/hints)
```

## Run locally

```bash
go run ./cmd/server            # listens on :2222, auto-generates .ssh/id_ed25519
go run ./cmd/admin teams create --name Alpha --ssh-user alpha --password 'changeme123'
ssh alpha@localhost -p 2222    # first login forces registration
```

Admin essentials:

```bash
admin rounds hash-flag --flag 'PANTHEON{...}'      # put digest into rounds.yaml
admin rounds load --file configs/rounds.yaml
admin rounds set-active --round 3 --active false
admin scoreboard export --format csv --output scores.csv
admin hints list --team alpha
admin db backup --output backup.sql
```

## Environment overrides

| Variable | Purpose |
|----------|---------|
| `CTF_CONFIG` | path to server.yaml |
| `CTF_ROUNDS` | path to rounds.yaml |
| `CTF_DB_PATH` | database file override |
| `CTF_HOST_KEY` | host key override |
| `CTF_MIGRATIONS` | migrations dir for admin CLI |

## Docker

See the repo-root `docker-compose.yml`. The container runs read-only with all
caps dropped; only the data volume is writable.

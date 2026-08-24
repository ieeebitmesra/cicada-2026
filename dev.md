# Dev — Run Locally & Test

Everything you need to hack on the platform from a laptop. No root, no KVM
required for Platform 1; Platform 2 needs a Linux box with `/dev/kvm`.

```text
Prereqs (Platform 1): Go 1.25+          git
Optional  (Platform 2): docker, root, /dev/kvm, firecracker binary
Handy:                  gpg, sqlite3, ssh/sftp clients
```

---

## 1. Build & static checks

```bash
cd submission-portal
go build ./...        # compiles everything
go vet ./...          # catches API misuse
go test ./...         # unit tests (scoring, validator, sanitizer, store)
```

Expected: all packages `ok`, no vet findings. The store tests build a real
SQLite DB in a temp dir and exercise migrations, the duplicate-solve guard,
skips, hints and the scoreboard view.

---

## 2. Run the portal locally

The binaries are path-relative by default (`configs/`, `migrations/`), so run
from inside `submission-portal/`:

```bash
cd submission-portal
go run ./cmd/server
# log line: "Submission Portal listening on 0.0.0.0:2222"
```

On first start it will:
- apply `migrations/*.sql` to `ctf.db`
- sync `configs/rounds.yaml` into the rounds table
- generate an SSH host key at `.ssh/id_ed25519`

All paths are overridable via env — useful to keep dev state in a sandbox dir:

| Env var | Default | Used by |
|---------|---------|---------|
| `CTF_CONFIG` | `configs/server.yaml` | server |
| `CTF_ROUNDS` | `configs/rounds.yaml` | server |
| `CTF_DB_PATH` | `ctf.db` | both |
| `CTF_HOST_KEY` | from config | server |
| `CTF_MIGRATIONS` | `migrations` | admin |

Sandboxed dev run:

```bash
mkdir -p ~/ctf-dev && cd ~/ctf-dev
cp -r <repo>/submission-portal/migrations .
cp <repo>/submission-portal/configs/{server,rounds}.yaml .
CTF_CONFIG=server.yaml CTF_ROUNDS=rounds.yaml \
CTF_DB_PATH=dev.db go run <repo>/submission-portal/cmd/server
```

### Tunables you'll flip while developing (`configs/server.yaml`)

| Key | Default | Effect |
|-----|---------|--------|
| `security.timeouts.max_session_minutes` | 30 | hard SSH session kill |
| `security.timeouts.idle_timeout_minutes` | 5 | TUI quits after this idle |
| `security.rate_limit.requests_per_second/burst` | 1.0 / 5 | new-connection token bucket per IP |
| `security.sessions.max_global / max_per_ip` | 100 / 3 | concurrent session caps |
| `security.submissions.per_minute_per_team / cooldown_seconds` | 2 / 30 | flag attempt limiter |
| `security.input.max_flag_length` | 256 | also enforced by format regex |
| `hints.challenge_validity_minutes` | 10 | signed challenge freshness window |

For local testing set idle timeout to 1 minute etc. — restart the server after edits.

---

## 3. End-to-end gameplay walkthrough (the main test)

Terminal A — server as above.

Terminal B — operator:

```bash
cd submission-portal
go build -o /tmp/admin ./cmd/admin

/tmp/admin teams create --name DevTeam --ssh-user dev --password 'devpass123'
/tmp/admin teams list                       # registered=false expected

# wire a known flag into round 1:
HASH=$(/tmp/admin rounds hash-flag --flag 'IEEE{dev_flag_1}')
sed -i "0,/REPLACE_WITH_SHA256_HEX/s//${HASH}/" configs/rounds.yaml   # first round only (GNU sed)
go run ./cmd/server                          # restart if already running
# NOTE: hint TEXT lives in rounds.yaml and is read by the server process —
# always restart the server after editing it (`admin rounds load` alone is not enough).
```

Terminal C — player:

```bash
ssh dev@localhost -p 2222       # password devpass123
```

Walk through the TUI:

1. **Registration** is forced on first login: set a new password (min 8 chars),
   confirm it, paste a PGP public key (§4), `Tab` between fields, `Ctrl+S` saves.
2. **Dashboard** → *Submit Flag* → pick Round 1 → try a wrong flag (error flash,
   rate-limited if spammed) → submit `IEEE{dev_flag_1}` → success flash.
3. **Scoreboard** shows DevTeam with +100 (or whatever points you configured).
4. **My Status** shows the ledger (earned/hints/skips).
5. Reconnect — registration screen must NOT appear again.
6. `Esc` navigates back; `Ctrl+C` quits.

Verify persistence directly:

```bash
sqlite3 ctf.db "SELECT team_id,round_id,is_correct FROM submissions;"
sqlite3 ctf.db "SELECT team_name,total_score,rounds_solved FROM scoreboard;"
```

Operator cross-checks:

```bash
/tmp/admin scoreboard
/tmp/admin db backup --output backup.sql && ls -l backup.sql
```

### Scoring rules to exercise deliberately

| Action | How | Expected |
|--------|-----|----------|
| plain hint cost | §4 flow on any open round | −20% of that round's points |
| encoded hint cost | same, choose "encoded" | −10% |
| skip penalty | Skip Round → Yes | −50% of current total (shown before confirm) |
| unbounded negative | skip repeatedly / many hints | score goes below zero, renders red |
| duplicate solve | resubmit correct flag for solved round | rejected ("closed for you") |
| double skip | second skip attempt on same round | rejected ("already skipped") |

---

## 4. Test the PGP hint flow locally

One-time key generation (throwaway dev key):

```bash
gpg --batch --quick-gen-key "Dev Team" default default never
gpg --armor --export "Dev Team" > devkey.pub && cat devkey.pub
```

In-game:

1. Dashboard → *Request Hint* → select an unsolved round → type (plain = 20%).
2. The TUI prints the challenge block. Save it verbatim:

```
HINT-REQ
round=1
type=plain
index=0
ts=<RFC3339>
team=dev
```

3. Clearsign within the validity window (default ±10 min):

```bash
gpg --clearsign -u "Dev Team" challenge.txt     # writes challenge.txt.asc
cat challenge.txt.asc                           # paste whole thing into the textarea
```

4. Press `Ctrl+S` → hint text appears with the cost flash.
5. Negative tests worth running:
   - wait >10 min then paste → *"challenge timestamp outside validity window"*
   - edit one character of the signed body → signature fails, nothing dispensed
   - request the same hint index twice → next request gets `index=1`;
     exhausting all hints → *"no hints remain"* on challenge issue
6. Proof audit trail:

```bash
/tmp/admin hints list --team dev
sqlite3 ctf.db "SELECT round_id,hint_type,hint_index,cost_points,length(pgp_proof) FROM hint_usage;"
```

---

## 5. Security-control tests (scriptable)

Each control maps to a config knob or middleware file — break them on purpose:

**bcrypt auth**

```bash
ssh dev@localhost -p 2222            # wrong password → Permission denied
# server-side wish logging middleware records auth failures
```

**per-IP connection rate limit** (`ratelimit.go`: 1 rps, burst 5)

```bash
# -tt forces a PTY so each connection reaches the middleware chain, then quits
for i in $(seq 1 8); do ssh -tt dev@localhost -p 2222 </dev/null; sleep 0.2; done
# later attempts print: "Rate limit exceeded. Try again later."
```

**session caps** (`sessions.go`: max_per_ip=3)

```bash
for i in 1 2 3; do ssh -tt dev@localhost -p 2222 & done # keep three TUIs open
ssh dev@localhost -p 2222                              # 4th: "Too many sessions from your IP."
```

**hard timeout / idle quit** — set both to 1 minute in `server.yaml`,
reconnect, wait: session drops with the timeout banner; idling at any screen
quits the TUI after the idle window.

**input sanitizer** (`sanitize_test.go` covers the units; manually: paste ANSI
escape sequences / `` `id` `` / `$(id)` into free-text fields — they're stripped
or rejected).

**flag rate limit** (`validator_test.go`; manually: two wrong submissions back-to-back
→ second says cooldown active).

---

## 6. Platform 2 (challenge microVM) — dev testing

Needs root, docker and `/dev/kvm` (`ls -l /dev/kvm`). Firecracker install steps
are in [deploy.md](deploy.md#21-install-firecracker).

```bash
cd challenge-ssh

# 1. kernel: follow kernel/README.md
# 2. assets + rootfs:
mkdir -p rootfs/assets && echo test > rootfs/assets/challenge.zip
printf 'MThd\x00\x00\x00\x06' > rootfs/assets/track.mid
sudo SEEKER_PASS='devpass123' ASSETS_DIR=rootfs/assets ./rootfs/build-rootfs.sh

# 3. orchestrator:
go build -o /tmp/orchestrator ./orchestrator
sudo /tmp/orchestrator snapshot      # golden boot; check golden_snapshot+golden_mem exist
sudo rm -f /tmp/firecracker-challenge.sock
sudo /tmp/orchestrator restore       # serves host :2223 via DNAT
```

Functional assertions:

```bash
sftp -P 2223 seeker@127.0.0.1           # ls -> challenge.zip track.mid ; get works
ssh  -p 2223 seeker@127.0.0.1           # MUST fail (ForceCommand internal-sftp)
ping -c1 -W1 172.16.0.2                 # ok from host bridge...
sudo iptables -S FORWARD                # inter-VM DROP + egress DROP rules present
# VM has no internet (egress DROP): confirm by absence of outbound conntrack:
sudo conntrack -L 2>/dev/null | grep 172.16.0.2 || echo "no VM egress flows"
```

Snapshot hygiene: after poking around SFTP (e.g., uploading junk), stop the
instance and `restore` again — the junk must be gone (pristine rootfs).

No-KVM laptop? You can still validate everything except booting: `go vet ./orchestrator`,
`shellcheck scripts/*.sh`, review overlay configs, and dry-run network setup on a scratch VM.

---

## 7. Host-hardening validation (staging box)

```bash
# nftables syntax without applying:
nft -c -f host-hardening/nftables.conf
# after applying, probe the intended behavior:
nft list ruleset
nc -w2 -z <host> 9999            # dropped; counter increments under inet filter input
fail2ban-client status sshd      # ban a test IP after N bad passwords
ausearch -k cmd-exec | tail      # execve events flowing
mount | grep ' /tmp '            # tmpfs with noexec,nodev,nosuid
```

## 8. CI-style gate (copy into your pipeline)

```bash
set -euo pipefail
(cd submission-portal && go build ./... && go vet ./... && go test ./...)
(cd challenge-ssh    && go build -o /dev/null ./orchestrator && go vet ./orchestrator)
bash -n challenge-ssh/scripts/*.sh challenge-ssh/rootfs/build-rootfs.sh
```

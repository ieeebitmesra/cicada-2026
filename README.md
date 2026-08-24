# IEEE CTF Event Platform

Two server-side platforms with defense-in-depth (7 security layers), per
`ieee_ctf_platform_plan.md`:

```
┌────────────────────────────┐     ┌─────────────────────────────────┐
│  submission-portal/        │:2222│  challenge-ssh/                 │:2223
│  Scoring TUI over SSH      │     │  Firecracker microVM            │
│  Wish + Bubble Tea + SQLite│     │  SFTP-only chroot, own kernel   │
│  bcrypt auth, PGP hints    │     │  golden snapshot per session    │
└────────────────────────────┘     └─────────────────────────────────┘
              host-hardening/  (nftables · fail2ban · auditd · tmpfs)
```

## Quick start

- **Deploy to production** → [deploy.md](deploy.md)
- **Run locally + test everything** → [dev.md](dev.md)

### 1. Submission Portal

```bash
cd submission-portal
go run ./cmd/server                                   # :2222, auto host-key
go run ./cmd/admin teams create --name Alpha \
    --ssh-user alpha --password 'changeme123'
ssh alpha@localhost -p 2222                           # first login = registration
# put real digests into configs/rounds.yaml via:
go run ./cmd/admin rounds hash-flag --flag 'IEEE{...}'
```

Production: `docker compose up -d submission-portal` from this directory.

### 2. Challenge microVM (bare metal / nested virt with `/dev/kvm`)

```bash
cd challenge-ssh
# fetch kernel (kernel/README.md), then build rootfs:
sudo SEEKER_PASS='<babylonian-pass>' ASSETS_DIR=rootfs/assets ./rootfs/build-rootfs.sh
go build -o orchestrator ./orchestrator
sudo ./orchestrator snapshot                          # one-time golden boot
sudo ./orchestrator restore                           # per team session (~ms)
```

### 3. Host hardening

```bash
sudo nft -f host-hardening/nftables.conf
sudo cp host-hardening/fail2ban/jail.local /etc/fail2ban/
sudo cp host-hardening/fail2ban/filter.d/wish-ssh.conf /etc/fail2ban/filter.d/
sudo cp host-hardening/audit/ctf.rules /etc/audit/rules.d/
cat host-hardening/fstab.append >> /etc/fstab
```

## Verification status

| Component | Check |
|-----------|-------|
| `submission-portal` | `go build`, `go vet`, `go test ./...` — unit tests cover scoring rules, flag validator/rate limiter, ANSI sanitizer, SQLite store incl. duplicate-solve guard & scoreboard view; E2E smoke-tested over real SSH (bcrypt accept/reject, TUI render) |
| `challenge-ssh` orchestrator | `go build`, `go vet`; runtime requires KVM host + Firecracker binary |
| Shell scripts | `shellcheck`-style review; idempotent network setup |

## Layout

- [`submission-portal/`](submission-portal/) — Platform 1 (see its README)
- [`challenge-ssh/`](challenge-ssh/) — Platform 2 (see its README)
- [`host-hardening/`](host-hardening/) — Layer 1 configs for the VPS
- [`docker-compose.yml`](docker-compose.yml) — portal deployment (read-only container, dropped caps)

## Open questions still owned by organizers

1. Team count (tune `max_global`, VM pool sizing)
2. PGP key distribution model (players generate their own today)
3. Credential distribution channel (admin CLI creates them; you deliver them)
4. Final rounds/points — edit `submission-portal/configs/rounds.yaml`
5. VPS must expose `/dev/kvm` for Platform 2

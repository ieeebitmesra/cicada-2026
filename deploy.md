# Deploy — IEEE CTF Platform Production Setup

Target: one Ubuntu 24.04 host (bare metal strongly preferred — Platform 2 needs `/dev/kvm`).

| Resource | Minimum | Recommended |
|----------|---------|-------------|
| CPU      | 2 vCPU  | 4 vCPU      |
| RAM      | 2 GB    | 4 GB        |
| Storage  | 20 GB   | 40 GB       |
| KVM      | `/dev/kvm` present | same |
| Open ports | 22 (admin), 2222 (portal), 2223 (challenge) | same |

Deployment order matters: **hardening last step locks the box** — read §5 before loading the firewall, and never enable the default-drop ruleset before your admin IP is set.

---

## 0. Layout on the host

```text
/opt/ieee-ctf/
├── submission-portal/     # built binaries + configs + data volume target
└── challenge-ssh/         # orchestrator + kernel + rootfs + snapshots
/var/lib/ctf/              # ctf.db lives here (backed up regularly)
```

---

## 1. Submission Portal

### Option A — Docker (simplest)

```bash
cd ieee_ctf                       # repo root (docker-compose.yml lives here)
docker compose up -d --build
docker compose logs -f submission-portal
```

What this gives you (already encoded in `docker-compose.yml`):
- portal on **:2222**, DB persisted in the `portal-data` volume at `/app/data/ctf.db`
- container is `read_only`, all caps dropped, tmpfs `/tmp` noexec
- configs mounted read-only from `./submission-portal/configs`

Create teams right away:

```bash
docker compose exec submission-portal ./admin teams create \
    --name Alpha --ssh-user alpha --password 'InitialPass123'
```

### Option B — Native binary + systemd

```bash
sudo apt-get update && sudo apt-get install -y golang-go git
sudo useradd -r -s /usr/sbin/nologin ctf
sudo mkdir -p /opt/ieee-ctf /var/lib/ctf && sudo chown ctf:ctf /var/lib/ctf

git clone <your-repo> /tmp/src && cd /tmp/src/submission-portal
CGO_ENABLED=0 go build -o /tmp/server ./cmd/server
CGO_ENABLED=0 go build -o /tmp/admin  ./cmd/admin
sudo mkdir /opt/ieee-ctf/submission-portal
sudo cp /tmp/server /tmp/admin /opt/ieee-ctf/submission-portal/
sudo cp -r configs migrations /opt/ieee-ctf/submission-portal/
```

`/etc/systemd/system/ctf-portal.service`:

```ini
[Unit]
Description=IEEE CTF Submission Portal (Wish SSH :2222)
After=network-online.target
Wants=network-online.target

[Service]
User=ctf
Group=ctf
WorkingDirectory=/opt/ieee-ctf/submission-portal
Environment=CTF_DB_PATH=/var/lib/ctf/ctf.db
ExecStart=/opt/ieee-ctf/submission-portal/server
Restart=on-failure
RestartSec=3
NoNewPrivileges=true
ProtectSystem=strict
ReadWritePaths=/var/lib/ctf /opt/ieee-ctf/submission-portal/.ssh
CapabilityBoundingSet=
AmbientCapabilities=
SyslogIdentifier=ctf-portal

[Install]
WantedBy=multi-user.target
```

> The host key is auto-generated on first start under `.ssh/id_ed25519` (relative
> to the working directory). Set `security.server.host_key_path` in
> `configs/server.yaml` if you want it elsewhere.

Enable:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now ctf-portal
sudo systemctl status ctf-portal
ss -tlnp | grep 2222
```

### Configure rounds (both options)

1. Compute flag digests and put them in `submission-portal/configs/rounds.yaml`:

```bash
./admin rounds hash-flag --flag 'PANTHEON{real_flag_here}'
# -> paste hex digest into round's flag_hash field
```

2. Sync to DB:

```bash
./admin rounds load --file configs/rounds.yaml
./admin rounds list          # verify active + hashes
./admin rounds set-active --round N --active false   # hold back unfinished rounds
```

---

## 2. Challenge microVM (Platform 2)

Requires root + working `/dev/kvm`.

### 2.1 Install Firecracker

```bash
FC_VER=1.8.0
curl -L -o /tmp/fc.tgz \
  https://github.com/firecracker-microvm/firecracker/releases/download/v${FC_VER}/firecracker-v${FC_VER}-x86_64.tgz
sudo tar -xzf /tmp/fc.tgz -C /tmp
sudo install -m 0755 /tmp/release-v${FC_VER}-x86_64/firecracker /usr/local/bin/
firecracker --version
ls -l /dev/kvm                 # must exist
```

### 2.2 Kernel + rootfs

```bash
cd /opt/ieee-ctf/challenge-ssh            # copy the repo's challenge-ssh/ here
# fetch vmlinux per kernel/README.md (prebuilt CI artifact or self-built)

# organizer assets first:
mkdir -p rootfs/assets
cp /path/to/challenge.zip rootfs/assets/
cp /path/to/track.mid     rootfs/assets/

sudo apt-get install -y docker.io
sudo SEEKER_PASS='<babylonian-password>' ASSETS_DIR=rootfs/assets \
     ./rootfs/build-rootfs.sh             # -> rootfs/rootfs.ext4 (~200MB)
```

### 2.3 Build orchestrator + golden snapshot

```bash
go build -o orchestrator ./orchestrator

sudo ./orchestrator snapshot     # boots once (~8s settle), writes golden_snapshot+golden_mem
rm -f /tmp/firecracker-challenge.sock                     # golden instance is done
```

### 2.4 Per-event runtime

`/etc/systemd/system/ctf-challenge.service`:

```ini
[Unit]
Description=IEEE CTF challenge microVM (restore from golden snapshot)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=/opt/ieee-ctf/challenge-ssh
ExecStart=/opt/ieee-ctf/challenge-ssh/orchestrator restore
Restart=on-failure
RestartSec=2
# Critical: kill the whole cgroup so the firecracker child dies with the unit
KillMode=control-group

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now ctf-challenge
ss -tlnp | grep 2223
```

Each team session should end with destroy + re-restore so tampering cannot
persist — schedule an hourly pristine reset via cron:

```bash
sudo tee /etc/cron.d/ctf-challenge-reset >/dev/null <<'EOF'
0 * * * * root systemctl restart ctf-challenge.service
EOF
```

Quick functional check from the host:

```bash
sftp -P 2223 seeker@127.0.0.1    # password = SEEKER_PASS; 'ls' shows challenge.zip track.mid
ssh  -p 2223 seeker@127.0.0.1    # must FAIL — no shell, sftp only
```

---

## 3. Host hardening (`host-hardening/`)

Load **in this order**, keeping an admin SSH session open until the end:

```bash
# 3.1 Firewall — EDIT FIRST:
sudoedit host-hardening/nftables.conf        # replace YOUR_ADMIN_IP
sudo nft -c -f host-hardening/nftables.conf  # dry-run syntax check
sudo nft -f host-hardening/nftables.conf
sudo nft list ruleset                        # sanity check
sudo cp host-hardening/nftables.conf /etc/nftables.conf   # persist across reboots

# 3.2 fail2ban
sudo apt-get install -y fail2ban
sudo cp host-hardening/fail2ban/jail.local /etc/fail2ban/
sudo cp host-hardening/fail2ban/filter.d/wish-ssh.conf /etc/fail2ban/filter.d/

# portal auth failures must reach /var/log/wish-server.log (rsyslog):
sudo tee /etc/rsyslog.d/30-ctf.conf >/dev/null <<'EOF'
if $programname == 'ctf-portal' then /var/log/wish-server.log
& stop
EOF
# (the systemd unit below tags its output as ctf-portal via SyslogIdentifier)

sudo systemctl restart rsyslog fail2ban
sudo fail2ban-client status wish-portal

# 3.3 auditd
sudo apt-get install -y auditd
sudo cp host-hardening/audit/ctf.rules /etc/audit/rules.d/
sudo augenrules --load                       # -e 2 makes rules immutable until reboot

# 3.4 tmpfs noexec mounts
cat host-hardening/fstab.append >> /etc/fstab
sudo systemctl daemon-reload && sudo mount -o remount /tmp
```

---

## 4. Event-day operations

```bash
# teams
./admin teams import --csv teams.csv          # header: name,ssh_user,password
./admin teams reset-password --user alpha     # prints a new random password
./admin teams list                            # registered? PGP fingerprint?

# during the event
watch -n5 './admin scoreboard'
./admin hints list --team alpha               # who consumed hints (+ proofs)
./admin rounds set-active --round 4 --active true

# safety net
./admin db backup --output /var/backups/ctf-$(date +%FT%H%M).sql
scp /var/backups/ctf-*.sql backup-host:/offsite/

# closing time
./admin scoreboard export --format csv --output scores.csv
```

Log destinations worth shipping to a SIEM: `journalctl -u ctf-portal`,
`/var/log/wish-server.log`, `ausearch -k cmd-exec`, `nft` drop counters.

---

## 5. Pre-flight checklist

- [ ] `nft list ruleset` loaded, admin IP whitelisted, you still have SSH
- [ ] `ss -tlnp` shows 22, 2222, 2223 and nothing unexpected
- [ ] `systemctl status ctf-portal` healthy; DB reachable; WAL files next to it
- [ ] `admin teams list` matches your roster; all rounds have real flag hashes
- [ ] Round activation state is what you intend (hold locked rounds inactive)
- [ ] One test team completed the full loop: register → solve → hint → skip → scoreboard
- [ ] Golden snapshot exists (`golden_snapshot` + `golden_mem`) and `restore` serves :2223
- [ ] `sftp` works, `ssh` shell refused on 2223; inter-VM ping fails; VM has no internet
- [ ] fail2ban bans after repeated bad passwords (`fail2ban-client status sshd`)
- [ ] Backups cron/scheduled and restored once into a scratch dir to prove they work

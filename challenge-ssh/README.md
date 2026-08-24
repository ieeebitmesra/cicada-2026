# Challenge File SSH Server (Firecracker microVM)

Round 4: participants `ssh -p 2223 seeker@<host>` (password = Babylonian
numbers), receive `challenge.zip` + `track.mid` over SFTP, and get a locked-
down chroot with **no shell**.

Runs inside a Firecracker microVM — its own kernel and rootfs — so "container
escape" is not in the threat model. After each team session the VM is
destroyed and re-restored from a golden snapshot to erase any tampering.

## Architecture

```
Host
 ├─ orchestrator (Go, firecracker-go-sdk) ── lifecycle, snapshots
 ├─ br0 (172.16.0.0/24)
 │   └─ tap-challenge ── VM NIC
 └─ iptables/nftables:
      DNAT host:2223 → 172.16.0.2:22
      DROP inter-VM traffic
      DROP all VM egress (no reverse shells)

microVM (Alpine rootfs.ext4, 200MB)
 ├─ sshd  : SFTP-only, ChrootDirectory /home/seeker
 ├─ files : challenge.zip, track.mid (0444, root-owned)
 ├─ AppArmor profile usr.sbin.sshd.ctf : deny exec/network/shadow
 └─ ulimits : nproc=50 nofile=256 fsize=10M cpu=5min as=256MB
```

## Build

```bash
# 1. Kernel — see kernel/README.md
# 2. Rootfs (requires docker + root):
sudo SEEKER_PASS='<babylonian-password>' ASSETS_DIR=rootfs/assets \
     ./rootfs/build-rootfs.sh
#    drop challenge.zip + track.mid into rootfs/assets/ first

# 3. Orchestrator binary:
cd .. && go build -o orchestrator ./orchestrator
```

## Operate

```bash
sudo ./orchestrator snapshot   # one-time golden boot (~8s settle) → golden_snapshot+golden_mem
sudo ./orchestrator restore    # per team session (~1-5ms restore); destroy + re-restore after
sudo ./orchestrator run        # cold boot instead of snapshots
```

The host publishes port **2223**; the portal advertises it to Round-4 teams.

## Hardening notes

- `sshd_config`: root login off, pubkey off, `ForceCommand internal-sftp`,
  chroot to `/home/seeker`, forwarding/tunneling fully disabled.
- AppArmor denies exec of shells/interpreters, denies network, denies reads of
  `/etc/shadow`, `/proc/*/mem`; profile attaches to `sftp-server`.
- ulimits are enforced via PAM (`pam_limits`) when enabled; the AppArmor
  network/exec denials hold regardless.
- Firecracker applies its own seccomp-BPF jail to the VMM process on the host;
  guest-side syscall filtering is layered via AppArmor + sshd's internal-sftp
  (no user shell exists to issue syscalls from).
- Disk I/O rate limiter (10MB/s / 500IOPS) prevents disk-fill DoS from inside.

## Defense summary (this platform)

| Vector | Control |
|--------|---------|
| Reverse shell | egress DROP + AppArmor deny network + no shell |
| Fork bomb | nproc=50 + Firecracker vCPU/mem caps |
| Privilege esc | root login off, AppArmor deny shadow/mem |
| Tampering between sessions | golden snapshot re-restore per session |
| Disk fill | fsize=10M + Firecracker drive rate limiter |
| Lateral movement | inter-VM DROP, no bridge forwarding |

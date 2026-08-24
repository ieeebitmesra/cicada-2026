#!/bin/bash
# rootfs/build-rootfs.sh — Build hardened Alpine rootfs for Firecracker
#
# Produces rootfs.ext4 (~200MB) containing a locked-down sshd that serves
# /home/seeker via SFTP-only chroot. Challenge artifacts are copied from
# $ASSETS_DIR (default ./assets) if present, otherwise placeholders are made.

set -euo pipefail

ROOTFS_SIZE=200   # MB
ROOTFS_FILE="$(dirname "$0")/rootfs.ext4"
MOUNT_DIR="/tmp/ctf-rootfs"
OVERLAY_DIR="$(dirname "$0")/overlay"
ASSETS_DIR="${ASSETS_DIR:-$(dirname "$0")/assets}"
SEEKER_PASS="${SEEKER_PASS:?Set SEEKER_PASS to the challenge password (Babylonian numbers)}"

[ "$(id -u)" -eq 0 ] || { echo "must run as root"; exit 1; }
command -v docker >/dev/null || { echo "docker required"; exit 1; }

echo "=== Creating ${ROOTFS_SIZE}MB rootfs ==="
dd if=/dev/zero of="$ROOTFS_FILE" bs=1M count="$ROOTFS_SIZE" status=none
mkfs.ext4 -q -F "$ROOTFS_FILE"

mkdir -p "$MOUNT_DIR"
mount "$ROOTFS_FILE" "$MOUNT_DIR"
trap 'umount "$MOUNT_DIR" 2>/dev/null || true; rmdir "$MOUNT_DIR" 2>/dev/null || true' EXIT

echo "=== Populating from Alpine Docker image ==="
docker run --rm -v "$MOUNT_DIR":/rootfs alpine:3.20 sh -c '
    set -e
    apk add --no-cache openssh openrc util-linux apparmor-shadow >/dev/null

    # Copy system directories into the future rootfs
    for d in bin etc lib root sbin usr var; do
        tar c "/$d" | tar x -C /rootfs
    done

    # Create required mountpoint dirs (populated by guest init)
    for dir in dev proc run sys tmp home mnt; do
        mkdir -p /rootfs/${dir}
    done

    # Serial console for Firecracker
    ln -sf agetty /rootfs/etc/init.d/agetty.ttyS0
    echo "ttyS0" > /rootfs/etc/securetty

    # Boot services
    chroot /rootfs rc-update add agetty.ttyS0 default
    chroot /rootfs rc-update add devfs boot
    chroot /rootfs rc-update add procfs boot
    chroot /rootfs rc-update add sysfs boot
    chroot /rootfs rc-update add hostname boot
    chroot /rootfs rc-update add bootmisc boot
    chroot /rootfs rc-update add sshd default

    # Challenge user (locked shell; SFTP comes from sshd internal-sftp)
    chroot /rootfs adduser -D -s /sbin/nologin seeker
'

echo "=== Setting seeker password + generating sshd host keys ==="
chroot "$MOUNT_DIR" sh -c "echo 'seeker:${SEEKER_PASS}' | chpasswd"
chroot "$MOUNT_DIR" ssh-keygen -A

echo "=== Applying overlay ==="
cp -r "$OVERLAY_DIR"/. "$MOUNT_DIR"/

echo "=== Installing challenge artifacts ==="
mkdir -p "$MOUNT_DIR/home/seeker"
if [ -f "$ASSETS_DIR/challenge.zip" ]; then
    cp "$ASSETS_DIR/challenge.zip" "$MOUNT_DIR/home/seeker/challenge.zip"
else
    echo "WARNING: $ASSETS_DIR/challenge.zip not found — creating placeholder" >&2
    echo "placeholder challenge.zip — replace via ASSETS_DIR" > "$MOUNT_DIR/home/seeker/challenge.zip"
fi
if [ -f "$ASSETS_DIR/track.mid" ]; then
    cp "$ASSETS_DIR/track.mid" "$MOUNT_DIR/home/seeker/track.mid"
else
    echo "WARNING: $ASSETS_DIR/track.mid not found — creating placeholder" >&2
    printf 'MThd' > "$MOUNT_DIR/home/seeker/track.mid"
fi

echo "=== Setting permissions ==="
chown root:root "$MOUNT_DIR"/home/seeker
chmod 755 "$MOUNT_DIR"/home/seeker          # chroot dir must be root-owned, non-group/world-writable
chown root:root "$MOUNT_DIR"/home/seeker/challenge.zip "$MOUNT_DIR"/home/seeker/track.mid
chmod 444 "$MOUNT_DIR"/home/seeker/challenge.zip "$MOUNT_DIR"/home/seeker/track.mid
chmod 1777 "$MOUNT_DIR"/tmp

echo "=== Done: $ROOTFS_FILE ==="

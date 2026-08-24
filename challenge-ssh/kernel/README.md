# kernel/

Place a Firecracker-compatible uncompressed kernel here as `vmlinux`.

Options:

1. **Download a prebuilt CI kernel** (fastest):
   Firecracker's CI publishes `vmlinux-*` artifacts; any 5.10/6.1 microvm
   kernel works. Verify with `file vmlinux` (should report "ELF 64-bit ... statically linked").

2. **Build your own** (recommended for the event):

```bash
git clone --depth 1 https://github.com/torvalds/linux -b linux-6.1.y
cd linux
# start from Firecracker's microvm config:
curl -o .config https://raw.githubusercontent.com/firecracker-microvm/firecracker/main/resources/guest_configs/microvm-kernel-ci-x86_64-6.1.config
make olddefconfig
make -j"$(nproc)" vmlinux
cp vmlinux ../kernel/vmlinux
```

The guest only needs: virtio-mmio, ext4, serial console (ttyS0), and
AppArmor if you enable it (`apparmor=1 security=apparmor` in KernelArgs).

// Package main is the Firecracker orchestrator for the challenge-file SSH VM.
//
// Usage:
//
//	orchestrator setup-net                 # bridge + TAP + firewall rules (idempotent)
//	orchestrator run                       # cold-boot the challenge VM
//	orchestrator snapshot                  # boot once, then write golden snapshot
//	orchestrator restore                   # fast-restore VM from golden snapshot (~1-5ms)
package main

import (
	"fmt"
	"os"
)

const usage = `challenge-vm orchestrator

Commands:
  setup-net            Create br0/tap-challenge and install firewall rules
  run                  Cold-boot the challenge microVM
  snapshot             Boot, settle, then create golden_snapshot/golden_mem
  restore              Restore from the golden snapshot onto its TAP device

Environment overrides:
  KERNEL_PATH   (default kernel/vmlinux)
  ROOTFS_PATH   (default rootfs/rootfs.ext4)
  VCPUS         (default 1)
  MEMORY_MIB    (default 512)
  TAP_DEVICE    (default tap-challenge)
  VM_IP         (default 172.16.0.2)
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	cfg := LoadConfig()

	var err error
	switch os.Args[1] {
	case "setup-net":
		err = SetupNetwork(cfg)
	case "run":
		err = RunVM(cfg)
	case "snapshot":
		err = MakeSnapshot(cfg)
	case "restore":
		err = RestoreFromSnapshot(cfg)
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

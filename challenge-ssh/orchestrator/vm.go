package main

import (
	"context"
	"fmt"
	"os"
	"time"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
)

// buildFirecrackerConfig assembles the microVM spec with disk I/O rate limits.
func buildFirecrackerConfig(cfg *VMConfig) firecracker.Config {
	return firecracker.Config{
		SocketPath:      cfg.SocketPath,
		KernelImagePath: cfg.KernelPath,
		KernelArgs:      "console=ttyS0 reboot=k panic=1 pci=off quiet",
		Drives: []models.Drive{{
			DriveID:      firecracker.String("rootfs"),
			PathOnHost:   firecracker.String(cfg.RootfsPath),
			IsRootDevice: firecracker.Bool(true),
			IsReadOnly:   firecracker.Bool(false),
			RateLimiter: &models.RateLimiter{
				// Disk I/O: 10MB/s bandwidth, 500 IOPS
				Bandwidth: &models.TokenBucket{
					Size:       firecracker.Int64(10 * 1024 * 1024),
					RefillTime: firecracker.Int64(1000), // ms
				},
				Ops: &models.TokenBucket{
					Size:       firecracker.Int64(500),
					RefillTime: firecracker.Int64(1000), // ms
				},
			},
		}},
		MachineCfg: models.MachineConfiguration{
			VcpuCount:  firecracker.Int64(cfg.VCPUs),
			MemSizeMib: firecracker.Int64(cfg.MemoryMiB),
		},
		NetworkInterfaces: []firecracker.NetworkInterface{{
			StaticConfiguration: &firecracker.StaticNetworkConfiguration{
				MacAddress:  cfg.GuestMAC,
				HostDevName: cfg.TAPDevice,
			},
			AllowMMDS: false,
		}},
	}
}

// RunVM cold-boots the challenge VM and blocks until it exits.
func RunVM(cfg *VMConfig) error {
	if err := SetupNetwork(cfg); err != nil {
		return err
	}
	ctx := context.Background()
	m, err := startMachine(ctx, cfg)
	if err != nil {
		return err
	}
	defer m.StopVMM()

	fmt.Printf("challenge VM up — ssh -p %s seeker@<host>\n", cfg.HostPort)
	select {} // stay attached; SIGTERM tears the VM down via StopVMM defer
}

func startMachine(ctx context.Context, cfg *VMConfig) (*firecracker.Machine, error) {
	fcCfg := buildFirecrackerConfig(cfg)
	cmd := firecracker.VMCommandBuilder{}.
		WithBin("firecracker").
		WithSocketPath(cfg.SocketPath).
		Build(ctx)

	m, err := firecracker.NewMachine(ctx, fcCfg, firecracker.WithProcessRunner(cmd))
	if err != nil {
		return nil, fmt.Errorf("new machine: %w", err)
	}
	if err := m.Start(ctx); err != nil {
		return nil, fmt.Errorf("start: %w", err)
	}
	fmt.Printf("VM started (socket=%s ip=%s)\n", cfg.SocketPath, cfg.GuestIP)
	return m, nil
}

// MakeSnapshot boots a golden VM once, waits for OpenSSH to settle, pauses and
// snapshots. Later sessions restore from this pair in ~1-5ms.
func MakeSnapshot(cfg *VMConfig) error {
	if err := SetupNetwork(cfg); err != nil {
		return err
	}
	ctx := context.Background()

	m, err := startMachine(ctx, cfg)
	if err != nil {
		return err
	}
	defer m.StopVMM()

	fmt.Printf("letting VM settle for %ds (sshd init, host keys already baked)...\n", cfg.SettleSeconds)
	time.Sleep(time.Duration(cfg.SettleSeconds) * time.Second)

	if err := m.PauseVM(ctx); err != nil {
		return fmt.Errorf("pause: %w", err)
	}
	// NOTE SDK arg order: CreateSnapshot(ctx, memFilePath, snapshotPath).
	if err := m.CreateSnapshot(ctx, cfg.MemFilePath, cfg.SnapshotPath); err != nil {
		return fmt.Errorf("create snapshot: %w", err)
	}

	fmt.Println("golden snapshot saved:", cfg.SnapshotPath, "+", cfg.MemFilePath)
	fmt.Println("Destroy this golden instance now; use `orchestrator restore` per session.")
	return os.ErrProcessDone // signal success; caller exits
}

// RestoreFromSnapshot restores a fresh VM from the golden snapshot. Each team
// session gets a pristine copy; destroy + re-restore afterwards to prevent
// tampering.
func RestoreFromSnapshot(cfg *VMConfig) error {
	if err := SetupNetwork(cfg); err != nil {
		return err
	}
	if _, err := os.Stat(cfg.SnapshotPath); err != nil {
		return fmt.Errorf("snapshot missing (%s): run `orchestrator snapshot` first", cfg.SnapshotPath)
	}

	ctx := context.Background()
	fcCfg := buildFirecrackerConfig(cfg)

	// Restore path: configure the snapshot pair on the machine spec; Start()
	// detects it, loads the microstate/memory and resumes the VM.
	fcCfg.Snapshot = firecracker.SnapshotConfig{
		MemFilePath:  cfg.MemFilePath,
		SnapshotPath: cfg.SnapshotPath,
		ResumeVM:     true,
	}

	cmd := firecracker.VMCommandBuilder{}.
		WithBin("firecracker").
		WithSocketPath(cfg.SocketPath).
		Build(ctx)

	m, err := firecracker.NewMachine(ctx, fcCfg, firecracker.WithProcessRunner(cmd))
	if err != nil {
		return fmt.Errorf("restore: new machine: %w", err)
	}
	if err := m.Start(ctx); err != nil {
		return fmt.Errorf("restore: start (load snapshot): %w", err)
	}
	defer m.StopVMM()

	fmt.Printf("restored from snapshot in ~ms — challenge VM serving on port %s\n", cfg.HostPort)
	select {}
}

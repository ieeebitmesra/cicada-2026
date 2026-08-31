package main

import (
	"os"
	"strconv"
)

// VMConfig carries all tunable VM + network parameters.
type VMConfig struct {
	KernelPath string
	RootfsPath string

	SocketPath  string
	VCPUs       int64
	MemoryMiB   int64
	TAPDevice   string
	BridgeName  string
	GuestMAC    string
	GuestIP     string
	GatewayIP   string
	HostPort    string // host port forwarded to guest sshd (22)
	BridgeSubnet string

	SnapshotPath string
	MemFilePath  string
	SettleSeconds int // how long to let the golden boot settle before pausing
}

// LoadConfig builds the config from environment variables with sane defaults.
func LoadConfig() *VMConfig {
	c := &VMConfig{
		KernelPath: envOr("KERNEL_PATH", "kernel/vmlinux"),
		RootfsPath: envOr("ROOTFS_PATH", "rootfs/rootfs.ext4"),

		SocketPath:   "/tmp/firecracker-challenge.sock",
		VCPUs:        int64(envInt("VCPUS", 1)),
		MemoryMiB:    int64(envInt("MEMORY_MIB", 512)),
		TAPDevice:    envOr("TAP_DEVICE", "tap-challenge"),
		BridgeName:   envOr("BRIDGE", "br0"),
		GuestMAC:     envOr("GUEST_MAC", "AA:FC:00:00:00:02"),
		GuestIP:      envOr("VM_IP", "172.16.0.2"),
		GatewayIP:    envOr("GW_IP", "172.16.0.1"),
		HostPort:     envOr("HOST_PORT", "2223"),
		BridgeSubnet: "172.16.0.0/24",

		SnapshotPath:  envOr("SNAPSHOT_PATH", "golden_snapshot"),
		MemFilePath:   envOr("MEMFILE_PATH", "golden_mem"),
		SettleSeconds: envInt("SETTLE_SECONDS", 8),
	}
	return c
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

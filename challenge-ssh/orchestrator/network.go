package main

import (
	"fmt"
	"net"
	"os/exec"
)

// runCmd executes a system command, returning combined output on failure.
func runCmd(name string, args ...string) error {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %v: %w\n%s", name, args, err, out)
	}
	return nil
}

// SetupNetwork is idempotent: bridge, TAP, firewall rules and the host→VM
// port forward. Egress from the VM and inter-VM traffic are hard-dropped.
func SetupNetwork(cfg *VMConfig) error {
	if err := runCmd("sysctl", "-w", "net.ipv4.ip_forward=1"); err != nil {
		return err
	}

	// Bridge (ignore "already exists")
	if err := runCmd("ip", "link", "add", "name", cfg.BridgeName, "type", "bridge"); err != nil {
		fmt.Printf("note: bridge: %v (continuing)\n", err)
	}
	if err := runCmd("ip", "addr", "add", cfg.GatewayIP+"/24", "dev", cfg.BridgeName); err != nil {
		fmt.Printf("note: addr: %v (continuing)\n", err)
	}
	if err := runCmd("ip", "link", "set", cfg.BridgeName, "up"); err != nil {
		return err
	}

	// TAP device attached to the bridge
	if err := runCmd("ip", "tuntap", "add", "dev", cfg.TAPDevice, "mode", "tap"); err != nil {
		fmt.Printf("note: tap: %v (continuing)\n", err)
	}
	if err := runCmd("ip", "link", "set", "dev", cfg.TAPDevice, "master", cfg.BridgeName); err != nil {
		return err
	}
	if err := runCmd("ip", "link", "set", "dev", cfg.TAPDevice, "up"); err != nil {
		return err
	}

	// Firewall: block inter-VM chatter and ALL egress from the bridge.
	rules := [][]string{
		{"iptables", "-C", "FORWARD", "-i", cfg.BridgeName, "-o", cfg.BridgeName, "-j", "DROP"},
		{"iptables", "-A", "FORWARD", "-i", cfg.BridgeName, "-o", cfg.BridgeName, "-j", "DROP"},

		{"iptables", "-C", "FORWARD", "-i", cfg.BridgeName, "-j", "DROP"},
		{"iptables", "-A", "FORWARD", "-i", cfg.BridgeName, "-j", "DROP"}, // no reverse shells
	}
	for i := 0; i < len(rules); i += 2 {
		check, add := rules[i], rules[i+1]
		if err := runCmd(check[0], check[1:]...); err != nil {
			if err := runCmd(add[0], add[1:]...); err != nil {
				return err
			}
		}
	}

	// Host port forward → guest sshd.
	dnat := []string{
		"iptables", "-t", "nat",
		"-C", "PREROUTING", "-p", "tcp", "--dport", cfg.HostPort,
		"-j", "DNAT", "--to-destination", net.JoinHostPort(cfg.GuestIP, "22"),
	}
	if err := runCmd(dnat[0], dnat[1:]...); err != nil {
		add := []string{"iptables", "-t", "nat", "-A", "PREROUTING", "-p", "tcp",
			"--dport", cfg.HostPort, "-j", "DNAT", "--to-destination",
			net.JoinHostPort(cfg.GuestIP, "22")}
		if err := runCmd(add[0], add[1:]...); err != nil {
			return err
		}
	}
	fwd := []string{"iptables", "-C", "FORWARD", "-p", "tcp", "-d", cfg.GuestIP,
		"--dport", "22", "-j", "ACCEPT"}
	if err := runCmd(fwd[0], fwd[1:]...); err != nil {
		add := []string{"iptables", "-I", "FORWARD", "1", "-p", "tcp", "-d", cfg.GuestIP,
			"--dport", "22", "-j", "ACCEPT"}
		if err := runCmd(add[0], add[1:]...); err != nil {
			return err
		}
	}

	fmt.Printf("network ready — bridge=%s tap=%s vm=%s host-port=%s\n",
		cfg.BridgeName, cfg.TAPDevice, cfg.GuestIP, cfg.HostPort)
	return nil
}

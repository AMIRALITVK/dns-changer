//go:build windows

package dns

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

type windowsManager struct{}

func NewManager() (Manager, error) {
	return &windowsManager{}, nil
}

func execOut(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).Output()
}

func (m *windowsManager) getActiveInterface() (string, error) {
	out, err := execOut("netsh", "interface", "show", "interface")
	if err != nil {
		return "Ethernet", nil
	}

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Connected") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				return parts[len(parts)-1], nil
			}
		}
	}
	return "Ethernet", nil
}

func (m *windowsManager) SetDNS(servers []string) error {
	iface, err := m.getActiveInterface()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	exec.CommandContext(ctx, "netsh", "interface", "ip", "delete", "dns", iface, "all").Run()

	for i, s := range servers {
		if i == 0 {
			out, err := execOut("netsh", "interface", "ip", "set", "dns", iface, "static", s)
			if err != nil {
				return err
			}
			_ = out
		} else {
			exec.Command("netsh", "interface", "ip", "add", "dns", iface, s, "index="+string(rune('0'+i))).Run()
		}
	}
	return nil
}

func (m *windowsManager) RemoveDNS() error {
	iface, err := m.getActiveInterface()
	if err != nil {
		return err
	}

	out, err := execOut("netsh", "interface", "ip", "delete", "dns", iface, "all")
	if err != nil {
		return err
	}
	_ = out

	exec.Command("netsh", "interface", "ip", "set", "dns", iface, "dhcp").Run()
	return nil
}

func (m *windowsManager) GetCurrentDNS() ([]string, error) {
	iface, err := m.getActiveInterface()
	if err != nil {
		return []string{}, nil
	}

	out, err := execOut("netsh", "interface", "ip", "show", "dns", iface)
	if err != nil {
		return []string{}, nil
	}

	servers := []string{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "DNS Server") && strings.Contains(line, ".") {
			parts := strings.Fields(line)
			for _, p := range parts {
				if strings.Count(p, ".") == 3 {
					servers = append(servers, p)
				}
			}
		}
	}
	return servers, nil
}

func (m *windowsManager) GetActiveInterface() (string, error) {
	return m.getActiveInterface()
}

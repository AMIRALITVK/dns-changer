//go:build darwin

package dns

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type darwinManager struct{}

func NewManager() (Manager, error) {
	return &darwinManager{}, nil
}

func execOut(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).Output()
}

func execCombined(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

func (m *darwinManager) getServiceName() (string, error) {
	portsOut, err := execCombined("networksetup", "-listallhardwareports")
	if err != nil {
		return "Wi-Fi", nil
	}

	routeOut, _ := execOut("route", "-n", "get", "default")
	activeIface := ""
	for _, line := range strings.Split(string(routeOut), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "interface:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				activeIface = parts[1]
			}
		}
	}

	currentPort := ""
	for _, line := range strings.Split(string(portsOut), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Hardware Port:") {
			currentPort = strings.TrimPrefix(line, "Hardware Port: ")
		} else if strings.HasPrefix(line, "Device:") && currentPort != "" {
			iface := strings.TrimSpace(strings.TrimPrefix(line, "Device:"))
			if iface == activeIface {
				return currentPort, nil
			}
		}
	}

	for _, name := range []string{"Wi-Fi", "Wi\u2011Fi", "Ethernet", "USB 10/100/1000 LAN"} {
		out, err := execOut("networksetup", "-getinfo", name)
		if err == nil && strings.Contains(string(out), "IP address:") {
			return name, nil
		}
	}

	return "Wi-Fi", nil
}

func (m *darwinManager) networksetupAdmin(args ...string) error {
	shellCmd := "/usr/sbin/networksetup " + strings.Join(args, " ")
	script := "do shell script \"" + shellCmd + "\" with administrator privileges"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed:\n  script: %s\n  error: %v\n  output: %s", shellCmd, err, string(out))
	}
	return nil
}

func (m *darwinManager) SetDNS(servers []string) error {
	service, err := m.getServiceName()
	if err != nil {
		return err
	}

	args := []string{"-setdnsservers", service}
	args = append(args, servers...)

	out, err := execCombined("networksetup", args...)
	if err == nil {
		return nil
	}
	_ = out

	quoted := make([]string, len(args))
	for i, a := range args {
		if strings.Contains(a, " ") {
			quoted[i] = "\"" + a + "\""
		} else {
			quoted[i] = a
		}
	}

	return m.networksetupAdmin(quoted...)
}

func (m *darwinManager) RemoveDNS() error {
	service, err := m.getServiceName()
	if err != nil {
		return err
	}

	out, err := execCombined("networksetup", "-setdnsservers", service, "empty")
	if err == nil {
		return nil
	}
	_ = out

	return m.networksetupAdmin("-setdnsservers", service, "empty")
}

func (m *darwinManager) GetCurrentDNS() ([]string, error) {
	service, err := m.getServiceName()
	if err != nil {
		return []string{}, nil
	}

	out, err := execOut("networksetup", "-getdnsservers", service)
	if err != nil {
		return []string{}, nil
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	servers := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.Contains(line, "not") && !strings.HasPrefix(line, "There") {
			servers = append(servers, line)
		}
	}
	return servers, nil
}

func (m *darwinManager) GetActiveInterface() (string, error) {
	service, err := m.getServiceName()
	if err != nil {
		return "Wi-Fi", nil
	}
	return service, nil
}

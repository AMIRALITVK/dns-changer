//go:build windows

package dns

import (
	"fmt"
	"net"
	"strings"

	"golang.org/x/sys/windows/registry"
)

type windowsManager struct{}

func NewManager() (Manager, error) {
	return &windowsManager{}, nil
}

const interfacesKey = `SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Interfaces`

func getActiveInterfaceGUID() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	activeIPs := make(map[string]struct{})
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if ok && ipnet.IP.To4() != nil && !ipnet.IP.IsLoopback() {
				activeIPs[ipnet.IP.String()] = struct{}{}
			}
		}
	}

	if len(activeIPs) == 0 {
		return "", fmt.Errorf("no active network interface found")
	}

	key, err := registry.OpenKey(registry.LOCAL_MACHINE, interfacesKey, registry.READ)
	if err != nil {
		return "", err
	}
	defer key.Close()

	guids, err := key.ReadSubKeyNames(-1)
	if err != nil {
		return "", err
	}

	for _, guid := range guids {
		subKey, err := registry.OpenKey(registry.LOCAL_MACHINE, interfacesKey+`\`+guid, registry.READ)
		if err != nil {
			continue
		}

		// Match by DHCP IP
		ip, _, err := subKey.GetStringValue("DhcpIPAddress")
		if err == nil && ip != "" && ip != "0.0.0.0" {
			if _, ok := activeIPs[ip]; ok {
				subKey.Close()
				return guid, nil
			}
		}

		// Match by static IP (REG_MULTI_SZ)
		ips, _, err := subKey.GetStringsValue("IPAddress")
		if err == nil {
			for _, ip := range ips {
				if ip != "" && ip != "0.0.0.0" {
					if _, ok := activeIPs[ip]; ok {
						subKey.Close()
						return guid, nil
					}
				}
			}
		}

		subKey.Close()
	}

	return "", fmt.Errorf("no matching registry GUID found for active interface")
}

func openInterfaceKey(guid string, access uint32) (registry.Key, error) {
	return registry.OpenKey(registry.LOCAL_MACHINE, interfacesKey+`\`+guid, access)
}

func (m *windowsManager) SetDNS(servers []string) error {
	guid, err := getActiveInterfaceGUID()
	if err != nil {
		return err
	}

	key, err := openInterfaceKey(guid, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("cannot open registry (run as admin): %w", err)
	}
	defer key.Close()

	return key.SetStringValue("NameServer", strings.Join(servers, " "))
}

func (m *windowsManager) RemoveDNS() error {
	guid, err := getActiveInterfaceGUID()
	if err != nil {
		return err
	}

	key, err := openInterfaceKey(guid, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("cannot open registry (run as admin): %w", err)
	}
	defer key.Close()

	if err := key.DeleteValue("NameServer"); err != nil && err != registry.ErrNotExist {
		return err
	}
	return nil
}

func (m *windowsManager) GetCurrentDNS() ([]string, error) {
	guid, err := getActiveInterfaceGUID()
	if err != nil {
		return []string{}, nil
	}

	key, err := openInterfaceKey(guid, registry.READ)
	if err != nil {
		return []string{}, nil
	}
	defer key.Close()

	dnsStr, _, err := key.GetStringValue("NameServer")
	if err != nil {
		dnsStr, _, err = key.GetStringValue("DhcpNameServer")
		if err != nil {
			return []string{}, nil
		}
	}

	if dnsStr == "" {
		return []string{}, nil
	}

	dnsStr = strings.ReplaceAll(dnsStr, ",", " ")
	return strings.Fields(dnsStr), nil
}

func (m *windowsManager) GetActiveInterface() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				ipnet, ok := addr.(*net.IPNet)
				if ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
					return iface.Name, nil
				}
			}
		}
	}
	return "", fmt.Errorf("no active interface found")
}

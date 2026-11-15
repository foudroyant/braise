// Package netutil finds LAN addresses and opens a browser.
package netutil

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"sort"
)

// LANIPs returns private IPv4 addresses, best candidate first. Public
// addresses are deliberately excluded: Braise is a LAN tool and must not
// advertise itself on the internet.
func LANIPs() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	type candidate struct {
		ip   string
		name string
	}
	var out []candidate
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
			if !ok {
				continue
			}
			ip4 := ipnet.IP.To4()
			if ip4 == nil || ip4.IsLoopback() || !isLAN(ip4) {
				continue
			}
			out = append(out, candidate{ip: ip4.String(), name: iface.Name})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return ifaceRank(out[i].name) < ifaceRank(out[j].name)
	})
	ips := make([]string, 0, len(out))
	seen := map[string]bool{}
	for _, c := range out {
		if seen[c.ip] {
			continue
		}
		seen[c.ip] = true
		ips = append(ips, c.ip)
	}
	return ips
}

// isLAN accepts RFC1918, IPv6 ULA/link-local and the 100.64/10 shared range
// used by some VPN overlays (Tailscale, carrier NAT).
func isLAN(ip net.IP) bool {
	if ip.IsPrivate() || ip.IsLinkLocalUnicast() {
		return true
	}
	if v4 := ip.To4(); v4 != nil && v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 {
		return true
	}
	return false
}

func ifaceRank(name string) int {
	switch {
	case name == "en0" || name == "eth0" || name == "wlan0" || name == "Wi-Fi" || name == "Ethernet":
		return 0
	case len(name) > 2 && (name[:2] == "en" || name[:3] == "eth" || name[:2] == "wl"):
		return 1
	default:
		return 2
	}
}

// URLs builds the browser URLs for a port, LAN first then localhost.
func URLs(port int) []string {
	var urls []string
	for _, ip := range LANIPs() {
		urls = append(urls, fmt.Sprintf("http://%s:%d", ip, port))
	}
	urls = append(urls, fmt.Sprintf("http://localhost:%d", port))
	return urls
}

// OpenBrowser tries to open url with the platform handler.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

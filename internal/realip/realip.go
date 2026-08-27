// Package realip — заголовок X-Real-IP и проверка вхождения адреса в CIDR.
package realip

import (
	"fmt"
	"net"
)

// Header — имя HTTP-заголовка с IP-адресом хоста агента.
const Header = "X-Real-IP"

// ParseCIDR разбирает trusted_subnet. Пустая строка — без ограничений (nil, nil).
func ParseCIDR(cidr string) (*net.IPNet, error) {
	if cidr == "" {
		return nil, nil
	}
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid trusted_subnet %q: %w", cidr, err)
	}
	return network, nil
}

// Allowed сообщает, входит ли raw в network. Пустой network — все адреса разрешены.
func Allowed(network *net.IPNet, raw string) bool {
	if network == nil {
		return true
	}
	ip := net.ParseIP(raw)
	return ip != nil && network.Contains(ip)
}

// Host возвращает IPv4-адрес хоста агента для X-Real-IP.
func Host() string {
	if ip := outboundIPv4(); ip != "" {
		return ip
	}
	if ip := firstNonLoopbackIPv4(); ip != "" {
		return ip
	}
	return "127.0.0.1"
}

func outboundIPv4() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()

	addr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok || addr.IP == nil {
		return ""
	}
	ip := addr.IP.To4()
	if ip == nil {
		return ""
	}
	return ip.String()
}

func firstNonLoopbackIPv4() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() {
			continue
		}
		if ip := ipnet.IP.To4(); ip != nil {
			return ip.String()
		}
	}
	return ""
}

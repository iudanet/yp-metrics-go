package config

import (
	"fmt"
	"net"
	"strings"
)

// IPNets - пользовательский тип для хранения списка net.IPNet
type IPNets []net.IPNet

// String используется для форматирования значения при выводе флага
func (ipns *IPNets) String() string {
	var strAddrs []string
	for _, ipnet := range *ipns {
		strAddrs = append(strAddrs, ipnet.String())
	}
	return strings.Join(strAddrs, ",")
}

// Set вызывается несколько раз для каждого значения, если флаг передан несколько раз,
// либо один раз — если передаётся в виде CSV
func (ipns *IPNets) Set(value string) error {
	for _, ipStr := range strings.Split(value, ",") {
		ipStr = strings.TrimSpace(ipStr)
		_, ipnet, err := net.ParseCIDR(ipStr)
		if err != nil {
			return fmt.Errorf("не удалось распарсить CIDR %q: %w", ipStr, err)
		}
		*ipns = append(*ipns, *ipnet)
	}
	return nil
}

func (ipns *IPNets) Contains(ip net.IP) bool {
	if ip == nil {
		return false
	}
	for _, ipnet := range *ipns {
		if ipnet.Contains(ip) {
			return true
		}
	}
	return false
}

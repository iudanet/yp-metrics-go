package config

import (
	"net"
	"testing"
)

func TestIPNets_String(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		ipnets   IPNets
	}{
		{
			name:     "empty IPNets",
			ipnets:   IPNets{},
			expected: "",
		},
		{
			name: "single IPNet",
			ipnets: IPNets{
				{IP: net.ParseIP("192.168.1.0"), Mask: net.CIDRMask(24, 32)},
			},
			expected: "192.168.1.0/24",
		},
		{
			name: "multiple IPNets",
			ipnets: IPNets{
				{IP: net.ParseIP("192.168.1.0"), Mask: net.CIDRMask(24, 32)},
				{IP: net.ParseIP("10.0.0.0"), Mask: net.CIDRMask(8, 32)},
				{IP: net.ParseIP("172.16.0.0"), Mask: net.CIDRMask(12, 32)},
			},
			expected: "192.168.1.0/24,10.0.0.0/8,172.16.0.0/12",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ipnets.String()
			if result != tt.expected {
				t.Errorf("String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIPNets_Set_SingleCIDR(t *testing.T) {
	var ipnets IPNets
	cidr := "192.168.1.0/24"

	err := ipnets.Set(cidr)
	if err != nil {
		t.Fatalf("Set() failed: %v", err)
	}

	if len(ipnets) != 1 {
		t.Fatalf("Expected 1 IPNet, got %d", len(ipnets))
	}

	expectedIP := net.ParseIP("192.168.1.0")
	if !ipnets[0].IP.Equal(expectedIP) {
		t.Errorf("IP = %v, want %v", ipnets[0].IP, expectedIP)
	}

	expectedMask := net.CIDRMask(24, 32)
	if ipnets[0].Mask.String() != expectedMask.String() {
		t.Errorf("Mask = %v, want %v", ipnets[0].Mask, expectedMask)
	}
}

func TestIPNets_Set_MultipleCIDRs(t *testing.T) {
	var ipnets IPNets
	cidrs := "192.168.1.0/24,10.0.0.0/8,172.16.0.0/12"

	err := ipnets.Set(cidrs)
	if err != nil {
		t.Fatalf("Set() failed: %v", err)
	}

	if len(ipnets) != 3 {
		t.Fatalf("Expected 3 IPNets, got %d", len(ipnets))
	}

	expectedCIDRs := []string{
		"192.168.1.0/24",
		"10.0.0.0/8",
		"172.16.0.0/12",
	}

	for i, expected := range expectedCIDRs {
		_, expectedNet, _ := net.ParseCIDR(expected)
		if !ipnets[i].IP.Equal(expectedNet.IP) || ipnets[i].Mask.String() != expectedNet.Mask.String() {
			t.Errorf("IPNet[%d] = %v, want %v", i, ipnets[i].String(), expected)
		}
	}
}

func TestIPNets_Set_MultipleCalls(t *testing.T) {
	var ipnets IPNets

	// Multiple Set calls
	err := ipnets.Set("192.168.1.0/24")
	if err != nil {
		t.Fatalf("First Set() failed: %v", err)
	}

	err = ipnets.Set("10.0.0.0/8")
	if err != nil {
		t.Fatalf("Second Set() failed: %v", err)
	}

	if len(ipnets) != 2 {
		t.Fatalf("Expected 2 IPNets, got %d", len(ipnets))
	}

	expectedCIDRs := []string{
		"192.168.1.0/24",
		"10.0.0.0/8",
	}

	for i, expected := range expectedCIDRs {
		_, expectedNet, _ := net.ParseCIDR(expected)
		if !ipnets[i].IP.Equal(expectedNet.IP) || ipnets[i].Mask.String() != expectedNet.Mask.String() {
			t.Errorf("IPNet[%d] = %v, want %v", i, ipnets[i].String(), expected)
		}
	}
}

func TestIPNets_Set_WithSpaces(t *testing.T) {
	var ipnets IPNets
	cidrs := "  192.168.1.0/24  ,  10.0.0.0/8  ,  172.16.0.0/12  "

	err := ipnets.Set(cidrs)
	if err != nil {
		t.Fatalf("Set() failed: %v", err)
	}

	if len(ipnets) != 3 {
		t.Fatalf("Expected 3 IPNets, got %d", len(ipnets))
	}

	expectedCIDRs := []string{
		"192.168.1.0/24",
		"10.0.0.0/8",
		"172.16.0.0/12",
	}

	for i, expected := range expectedCIDRs {
		_, expectedNet, _ := net.ParseCIDR(expected)
		if !ipnets[i].IP.Equal(expectedNet.IP) || ipnets[i].Mask.String() != expectedNet.Mask.String() {
			t.Errorf("IPNet[%d] = %v, want %v", i, ipnets[i].String(), expected)
		}
	}
}

func TestIPNets_Set_InvalidCIDR(t *testing.T) {
	var ipnets IPNets
	invalidCIDR := "invalid-cidr"

	err := ipnets.Set(invalidCIDR)
	if err == nil {
		t.Fatal("Expected error for invalid CIDR, but got none")
	}

	if len(ipnets) != 0 {
		t.Errorf("Expected no IPNets to be added for invalid CIDR, got %d", len(ipnets))
	}
}

func TestIPNets_Set_InvalidCIDRInCSV(t *testing.T) {
	var ipnets IPNets
	mixedCIDRs := "192.168.1.0/24,invalid-cidr,10.0.0.0/8"

	err := ipnets.Set(mixedCIDRs)
	if err == nil {
		t.Fatal("Expected error for invalid CIDR in CSV, but got none")
	}

	// No IPNets should be added if any CIDR is invalid
	if len(ipnets) != 1 {
		t.Errorf("Expected no IPNets to be added when any CIDR is invalid, got %d", len(ipnets))
	}
}

func TestIPNets_Set_IPv6CIDR(t *testing.T) {
	var ipnets IPNets
	ipv6CIDR := "2001:db8::/32"

	err := ipnets.Set(ipv6CIDR)
	if err != nil {
		t.Fatalf("Set() failed for IPv6: %v", err)
	}

	if len(ipnets) != 1 {
		t.Fatalf("Expected 1 IPNet, got %d", len(ipnets))
	}

	expectedIP := net.ParseIP("2001:db8::")
	if !ipnets[0].IP.Equal(expectedIP) {
		t.Errorf("IP = %v, want %v", ipnets[0].IP, expectedIP)
	}

	expectedMask := net.CIDRMask(32, 128)
	if ipnets[0].Mask.String() != expectedMask.String() {
		t.Errorf("Mask = %v, want %v", ipnets[0].Mask, expectedMask)
	}
}

func TestIPNets_Set_MixedIPv4IPv6(t *testing.T) {
	var ipnets IPNets
	mixedCIDRs := "192.168.1.0/24,2001:db8::/32"

	err := ipnets.Set(mixedCIDRs)
	if err != nil {
		t.Fatalf("Set() failed for mixed CIDRs: %v", err)
	}

	if len(ipnets) != 2 {
		t.Fatalf("Expected 2 IPNets, got %d", len(ipnets))
	}

	// Check IPv4
	expectedIPv4 := net.ParseIP("192.168.1.0")
	if !ipnets[0].IP.Equal(expectedIPv4) {
		t.Errorf("IPv4 IP = %v, want %v", ipnets[0].IP, expectedIPv4)
	}

	// Check IPv6
	expectedIPv6 := net.ParseIP("2001:db8::")
	if !ipnets[1].IP.Equal(expectedIPv6) {
		t.Errorf("IPv6 IP = %v, want %v", ipnets[1].IP, expectedIPv6)
	}
}

func TestIPNets_Contains(t *testing.T) {
	tests := []struct {
		name     string
		ipnets   IPNets
		ip       net.IP
		expected bool
	}{
		{
			name:     "empty IPNets should return false",
			ipnets:   IPNets{},
			ip:       net.ParseIP("192.168.1.1"),
			expected: false,
		},
		{
			name: "single IPv4 network - IP contained",
			ipnets: IPNets{
				mustParseCIDR("192.168.1.0/24"),
			},
			ip:       net.ParseIP("192.168.1.10"),
			expected: true,
		},
		{
			name: "single IPv4 network - IP not contained",
			ipnets: IPNets{
				mustParseCIDR("192.168.1.0/24"),
			},
			ip:       net.ParseIP("10.0.0.1"),
			expected: false,
		},
		{
			name: "multiple IPv4 networks - IP contained in first",
			ipnets: IPNets{
				mustParseCIDR("192.168.1.0/24"),
				mustParseCIDR("10.0.0.0/8"),
			},
			ip:       net.ParseIP("192.168.1.10"),
			expected: true,
		},
		{
			name: "multiple IPv4 networks - IP contained in second",
			ipnets: IPNets{
				mustParseCIDR("192.168.1.0/24"),
				mustParseCIDR("10.0.0.0/8"),
			},
			ip:       net.ParseIP("10.0.0.1"),
			expected: true,
		},
		{
			name: "multiple IPv4 networks - IP not contained",
			ipnets: IPNets{
				mustParseCIDR("192.168.1.0/24"),
				mustParseCIDR("10.0.0.0/8"),
			},
			ip:       net.ParseIP("172.16.0.1"),
			expected: false,
		},
		{
			name: "IPv6 network - IP contained",
			ipnets: IPNets{
				mustParseCIDR("2001:db8::/32"),
			},
			ip:       net.ParseIP("2001:db8::1"),
			expected: true,
		},
		{
			name: "IPv6 network - IP not contained",
			ipnets: IPNets{
				mustParseCIDR("2001:db8::/32"),
			},
			ip:       net.ParseIP("2001:db9::1"),
			expected: false,
		},
		{
			name: "mixed IPv4 and IPv6 networks - IPv4 contained",
			ipnets: IPNets{
				mustParseCIDR("192.168.1.0/24"),
				mustParseCIDR("2001:db8::/32"),
			},
			ip:       net.ParseIP("192.168.1.10"),
			expected: true,
		},
		{
			name: "mixed IPv4 and IPv6 networks - IPv6 contained",
			ipnets: IPNets{
				mustParseCIDR("192.168.1.0/24"),
				mustParseCIDR("2001:db8::/32"),
			},
			ip:       net.ParseIP("2001:db8::1"),
			expected: true,
		},
		{
			name: "mixed IPv4 and IPv6 networks - IP not contained",
			ipnets: IPNets{
				mustParseCIDR("192.168.1.0/24"),
				mustParseCIDR("2001:db8::/32"),
			},
			ip:       net.ParseIP("10.0.0.1"),
			expected: false,
		},
		{
			name: "nil IP should return false",
			ipnets: IPNets{
				mustParseCIDR("192.168.1.0/24"),
			},
			ip:       nil,
			expected: false,
		},
		{
			name: "IP matching network boundary",
			ipnets: IPNets{
				mustParseCIDR("192.168.1.0/24"),
			},
			ip:       net.ParseIP("192.168.1.0"), // network address
			expected: true,
		},
		{
			name: "IP matching broadcast address",
			ipnets: IPNets{
				mustParseCIDR("192.168.1.0/24"),
			},
			ip:       net.ParseIP("192.168.1.255"), // broadcast address
			expected: true,
		},
		{
			name: "IP just outside network",
			ipnets: IPNets{
				mustParseCIDR("192.168.1.0/24"),
			},
			ip:       net.ParseIP("192.168.2.1"), // outside the /24 network
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ipnets.Contains(tt.ip)
			if result != tt.expected {
				t.Errorf("Contains(%v) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestIPNets_Contains_EdgeCases(t *testing.T) {
	t.Run("empty IP with non-empty networks", func(t *testing.T) {
		ipnets := IPNets{
			mustParseCIDR("192.168.1.0/24"),
		}
		result := ipnets.Contains(net.IP{})
		if result != false {
			t.Errorf("Contains(empty IP) should return false, got %v", result)
		}
	})

	t.Run("IP with wrong length for network", func(t *testing.T) {
		ipnets := IPNets{
			mustParseCIDR("192.168.1.0/24"), // IPv4 network
		}
		// IPv6 address with IPv4 network
		result := ipnets.Contains(net.ParseIP("2001:db8::1"))
		if result != false {
			t.Errorf("Contains(IPv6 with IPv4 network) should return false, got %v", result)
		}
	})

	t.Run("IPv4 address with IPv6 network", func(t *testing.T) {
		ipnets := IPNets{
			mustParseCIDR("2001:db8::/32"), // IPv6 network
		}
		// IPv4 address with IPv6 network
		result := ipnets.Contains(net.ParseIP("192.168.1.1"))
		if result != false {
			t.Errorf("Contains(IPv4 with IPv6 network) should return false, got %v", result)
		}
	})
}

// Вспомогательная функция для парсинга CIDR без ошибок
func mustParseCIDR(cidr string) net.IPNet {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(err)
	}
	return *ipnet
}

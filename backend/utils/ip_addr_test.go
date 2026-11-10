package utils

import "testing"

func TestIsPrivateOrReservedIPRejectsReservedIPv6Prefixes(t *testing.T) {
	tests := []struct {
		name string
		ip   string
	}{
		{name: "nat64 private IPv4", ip: "64:ff9b::a00:1"},
		{name: "local-use nat64 private IPv4", ip: "64:ff9b:1::a00:1"},
		{name: "discard prefix", ip: "100::1"},
		{name: "mapped IPv4", ip: "::ffff:192.0.2.1"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if !IsPrivateOrReservedIP(test.ip) {
				t.Fatalf("IsPrivateOrReservedIP(%q) = false, want true", test.ip)
			}
		})
	}
}

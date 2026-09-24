package service

import "testing"

func TestNormalizeGatewayDomain(t *testing.T) {
	ok := []string{"10.0.0.1:8000", "api.example.com:443", "localhost:8001", "[::1]:8080"}
	for _, in := range ok {
		if _, err := normalizeGatewayDomain(in); err != nil {
			t.Fatalf("%s: %v", in, err)
		}
	}
	bad := []string{"", "10.0.0.1", "example.com", "http://10.0.0.1:80", "10.0.0.1:0", "10.0.0.1:65536", "256.1.1.1:80", "example.com:abc"}
	for _, in := range bad {
		if _, err := normalizeGatewayDomain(in); err == nil {
			t.Fatalf("expected error for %s", in)
		}
	}
}

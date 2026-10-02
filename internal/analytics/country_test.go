package analytics

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFirstPublicIP(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{name: "IPv4", header: "8.8.8.8", want: "8.8.8.8"},
		{name: "first public", header: "127.0.0.1, 192.168.1.2, 8.8.4.4", want: "8.8.4.4"},
		{name: "IPv4 with port", header: "8.8.8.8:443", want: "8.8.8.8"},
		{name: "bracketed IPv6", header: "[2606:4700:4700::1111]:443", want: "2606:4700:4700::1111"},
		{name: "private only", header: "127.0.0.1, 10.0.0.1"},
		{name: "documentation address", header: "192.0.2.1"},
		{name: "malformed", header: "not-an-ip"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			address, ok := FirstPublicIP(test.header)
			if test.want == "" {
				if ok {
					t.Fatalf("FirstPublicIP() = %s, want no address", address)
				}
				return
			}
			if !ok || address.String() != test.want {
				t.Fatalf("FirstPublicIP() = %q, %v; want %q, true", address, ok, test.want)
			}
		})
	}
}

func TestMMDBCountryResolverWithLocalDatabase(t *testing.T) {
	path := filepath.Join("..", "..", "data", "geoip", "dbip-country-lite.mmdb")
	resolver, err := OpenMMDBCountryResolver(path)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("GeoIP database not downloaded")
		}
		t.Fatalf("OpenMMDBCountryResolver() error = %v", err)
	}
	defer resolver.Close()

	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set("X-Forwarded-For", "8.8.8.8")
	code := resolver.CountryCode(request)
	if code == nil || *code != "US" {
		t.Fatalf("CountryCode() = %v, want US", code)
	}
}

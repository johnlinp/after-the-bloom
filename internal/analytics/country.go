package analytics

import (
	"net"
	"net/http"
	"net/netip"
	"strings"

	"github.com/oschwald/maxminddb-golang"
)

type CountryResolver interface {
	CountryCode(request *http.Request) *string
}

type MMDBCountryResolver struct {
	reader *maxminddb.Reader
}

func OpenMMDBCountryResolver(path string) (*MMDBCountryResolver, error) {
	reader, err := maxminddb.Open(path)
	if err != nil {
		return nil, err
	}
	return &MMDBCountryResolver{reader: reader}, nil
}

func (r *MMDBCountryResolver) Close() error {
	return r.reader.Close()
}

func (r *MMDBCountryResolver) CountryCode(request *http.Request) *string {
	address, ok := FirstPublicIP(request.Header.Get("X-Forwarded-For"))
	if !ok {
		return nil
	}

	var record struct {
		Country struct {
			ISOCode string `maxminddb:"iso_code"`
		} `maxminddb:"country"`
		RegisteredCountry struct {
			ISOCode string `maxminddb:"iso_code"`
		} `maxminddb:"registered_country"`
		RepresentedCountry struct {
			ISOCode string `maxminddb:"iso_code"`
		} `maxminddb:"represented_country"`
	}
	if err := r.reader.Lookup(net.ParseIP(address.String()), &record); err != nil {
		return nil
	}

	for _, candidate := range []string{
		record.Country.ISOCode,
		record.RegisteredCountry.ISOCode,
		record.RepresentedCountry.ISOCode,
	} {
		code := strings.ToUpper(strings.TrimSpace(candidate))
		if len(code) == 2 && code[0] >= 'A' && code[0] <= 'Z' && code[1] >= 'A' && code[1] <= 'Z' {
			return &code
		}
	}
	return nil
}

func FirstPublicIP(header string) (netip.Addr, bool) {
	for _, raw := range strings.Split(header, ",") {
		candidate := strings.TrimSpace(raw)
		if candidate == "" {
			continue
		}

		if host, _, err := net.SplitHostPort(candidate); err == nil {
			candidate = host
		} else if strings.HasPrefix(candidate, "[") && strings.HasSuffix(candidate, "]") {
			candidate = strings.TrimSuffix(strings.TrimPrefix(candidate, "["), "]")
		}

		address, err := netip.ParseAddr(candidate)
		if err != nil {
			continue
		}
		address = address.Unmap()
		if isPublicIP(address) {
			return address, true
		}
	}
	return netip.Addr{}, false
}

var nonPublicPrefixes = mustPrefixes(
	"0.0.0.0/8", "10.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8",
	"169.254.0.0/16", "172.16.0.0/12", "192.0.0.0/24", "192.0.2.0/24",
	"192.168.0.0/16", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24",
	"224.0.0.0/4", "240.0.0.0/4",
	"::/128", "::1/128", "100::/64", "2001:db8::/32", "fc00::/7", "fe80::/10", "ff00::/8",
)

func isPublicIP(address netip.Addr) bool {
	if !address.IsValid() || address.IsUnspecified() || address.IsLoopback() ||
		address.IsPrivate() || address.IsLinkLocalUnicast() || address.IsLinkLocalMulticast() ||
		address.IsMulticast() {
		return false
	}
	for _, prefix := range nonPublicPrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}

func mustPrefixes(values ...string) []netip.Prefix {
	prefixes := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		prefixes = append(prefixes, netip.MustParsePrefix(value))
	}
	return prefixes
}

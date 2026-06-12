package service

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

// net.IP.IsPrivate covers RFC-1918 + loopback + ULA (Go 1.17+)

var geoClient = &http.Client{Timeout: 5 * time.Second}

// LookupCountryCode returns the ISO 3166-1 alpha-2 country code for the given IP.
// Returns an empty string if the IP is private/loopback or the lookup fails.
func LookupCountryCode(ip string) string {
	if isPrivateIP(ip) {
		return ""
	}
	resp, err := geoClient.Get(fmt.Sprintf("http://ip-api.com/json/%s?fields=countryCode", ip))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var result struct {
		CountryCode string `json:"countryCode"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}
	return result.CountryCode
}

func isPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	return ip != nil && (ip.IsPrivate() || ip.IsLoopback())
}

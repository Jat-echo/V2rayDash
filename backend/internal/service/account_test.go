package service

import (
	"testing"

	"v2ray-dash/backend/internal/model"
)

func TestServerFlag(t *testing.T) {
	for _, tc := range []struct {
		name, country, want string
	}{
		{"HK node", "JP", "🇯🇵"},
		{"unknown", "us", "🇺🇸"},
		{"HK node", "", "🇭🇰"},
		{"Tokyo", "??", "🌐"},
		{"unknown", "", "🌐"},
	} {
		t.Run(tc.name+"/"+tc.country, func(t *testing.T) {
			if got := serverFlag(&model.Server{Name: tc.name, CountryCode: tc.country}); got != tc.want {
				t.Errorf("serverFlag = %q, want %q", got, tc.want)
			}
		})
	}
}

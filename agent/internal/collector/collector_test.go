package collector

import (
	"slices"
	"testing"
)

func TestLegacyAndManagedStatisticsProtocols(t *testing.T) {
	legacy := New()
	if !slices.Contains(legacy.statsArgs(), "--reset") {
		t.Fatal("original direct nodes must retain delta statistics")
	}
	managed := &Collector{Service: "xray-relay", Cumulative: true}
	if slices.Contains(managed.statsArgs(), "--reset") {
		t.Fatal("managed counters must survive retries without resetting")
	}
}

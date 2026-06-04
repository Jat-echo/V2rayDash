package repository

import "testing"

func TestTimeRangeToBucketSeconds(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"1h", 60},
		{"3h", 120},
		{"6h", 300},
		{"1d", 600},
		{"7d", 7200},
		{"30d", 21600},
		{"unknown", 60},
		{"", 60},
	}
	for _, c := range cases {
		got := timeRangeToBucketSeconds(c.input)
		if got != c.want {
			t.Errorf("timeRangeToBucketSeconds(%q) = %d, want %d", c.input, got, c.want)
		}
	}
}

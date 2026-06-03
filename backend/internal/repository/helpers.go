package repository

// timeRangeToInterval converts a short range key ("1h", "3h", …) to a
// PostgreSQL interval string. defaultInterval is used when the key is unknown.
func timeRangeToInterval(timeRange, defaultInterval string) string {
	m := map[string]string{
		"1h":  "1 hour",
		"3h":  "3 hours",
		"4h":  "4 hours",
		"6h":  "6 hours",
		"12h": "12 hours",
		"1d":  "1 day",
		"3d":  "3 days",
		"7d":  "7 days",
		"30d": "30 days",
	}
	if v, ok := m[timeRange]; ok {
		return v
	}
	return defaultInterval
}

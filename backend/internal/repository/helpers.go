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

// timeRangeToBucketSeconds maps a time range key to bucket size in seconds
// for use in SQL time-bucketing queries.
func timeRangeToBucketSeconds(timeRange string) int {
	m := map[string]int{
		"1h":  60,
		"3h":  120,
		"6h":  300,
		"1d":  600,
		"7d":  7200,
		"30d": 21600,
	}
	if v, ok := m[timeRange]; ok {
		return v
	}
	return 60
}

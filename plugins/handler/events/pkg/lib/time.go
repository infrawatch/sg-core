package lib

import "time"

var (
	isoTimeLayout = "2006-01-02 15:04:05.000000"
	rFC3339       = "2006-01-02T15:04:05.000000"
)

// EpochFromFormat get epoch time from one of select time string formats
func EpochFromFormat(ts string) int64 {
	for _, layout := range []string{rFC3339, time.RFC3339, time.RFC3339Nano, time.ANSIC, isoTimeLayout} {
		stamp, err := time.Parse(layout, ts)
		if err == nil {
			return stamp.Unix()
		}
	}
	return 0
}

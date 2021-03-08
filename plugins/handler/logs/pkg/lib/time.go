package lib

import (
	"time"
	"fmt"
)

var (
	isoTimeLayout = "2006-01-02 15:04:05.000000"
	rFC3339       = "2006-01-02T15:04:05.000000"
)

//TimeFromFormat get time from one of select time string formats
func TimeFromFormat(ts string) (time.Time, error) {
	for _, layout := range []string{rFC3339, time.RFC3339, time.RFC3339Nano, time.ANSIC, isoTimeLayout} {
		stamp, err := time.Parse(layout, ts)
		if err == nil {
			return stamp, nil
		}
	}
	return time.Now(), fmt.Errorf("unable to parse timestamp.")
}

package lib

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEpochFromFormat(t *testing.T) {
	t.Run("parse RFC3339 format", func(t *testing.T) {
		ts := "2021-02-10T03:50:41Z"
		epoch := EpochFromFormat(ts)
		expected := time.Date(2021, 2, 10, 3, 50, 41, 0, time.UTC).Unix()
		assert.Equal(t, expected, epoch)
	})

	t.Run("parse RFC3339 with timezone", func(t *testing.T) {
		ts := "2021-02-10T03:50:41+00:00"
		epoch := EpochFromFormat(ts)
		expected := time.Date(2021, 2, 10, 3, 50, 41, 0, time.UTC).Unix()
		assert.Equal(t, expected, epoch)
	})

	t.Run("parse RFC3339Nano format", func(t *testing.T) {
		ts := "2021-02-10T03:50:41.123456789Z"
		epoch := EpochFromFormat(ts)
		expected := time.Date(2021, 2, 10, 3, 50, 41, 123456789, time.UTC).Unix()
		assert.Equal(t, expected, epoch)
	})

	t.Run("parse custom RFC3339 format without Z", func(t *testing.T) {
		ts := "2021-02-10T03:50:41.123456"
		epoch := EpochFromFormat(ts)
		expected := time.Date(2021, 2, 10, 3, 50, 41, 123456000, time.UTC).Unix()
		assert.Equal(t, expected, epoch)
	})

	t.Run("parse ANSIC format", func(t *testing.T) {
		ts := "Wed Feb 10 03:50:41 2021"
		epoch := EpochFromFormat(ts)
		expected := time.Date(2021, 2, 10, 3, 50, 41, 0, time.UTC).Unix()
		assert.Equal(t, expected, epoch)
	})

	t.Run("parse ISO time format with space", func(t *testing.T) {
		ts := "2021-02-10 03:50:41.123456"
		epoch := EpochFromFormat(ts)
		expected := time.Date(2021, 2, 10, 3, 50, 41, 123456000, time.UTC).Unix()
		assert.Equal(t, expected, epoch)
	})

	t.Run("parse ISO time format without microseconds", func(t *testing.T) {
		ts := "2021-02-10 03:50:41"
		epoch := EpochFromFormat(ts)
		// The isoTimeLayout is flexible and can parse without microseconds
		expected := time.Date(2021, 2, 10, 3, 50, 41, 0, time.UTC).Unix()
		assert.Equal(t, expected, epoch)
	})

	t.Run("invalid format returns zero", func(t *testing.T) {
		ts := "invalid-timestamp"
		epoch := EpochFromFormat(ts)
		assert.Equal(t, int64(0), epoch)
	})

	t.Run("empty string returns zero", func(t *testing.T) {
		ts := ""
		epoch := EpochFromFormat(ts)
		assert.Equal(t, int64(0), epoch)
	})

	t.Run("parse timestamp with different year", func(t *testing.T) {
		ts := "2020-09-14T16:12:49Z"
		epoch := EpochFromFormat(ts)
		expected := time.Date(2020, 9, 14, 16, 12, 49, 0, time.UTC).Unix()
		assert.Equal(t, expected, epoch)
	})

	t.Run("parse timestamp at epoch", func(t *testing.T) {
		ts := "1970-01-01T00:00:00Z"
		epoch := EpochFromFormat(ts)
		assert.Equal(t, int64(0), epoch)
	})

	t.Run("parse timestamp with nanoseconds precision", func(t *testing.T) {
		ts := "2021-02-11T21:43:11.180978123Z"
		epoch := EpochFromFormat(ts)
		expected := time.Date(2021, 2, 11, 21, 43, 11, 180978123, time.UTC).Unix()
		assert.Equal(t, expected, epoch)
	})

	t.Run("parse timestamp with microseconds in custom format", func(t *testing.T) {
		ts := "2021-02-11T21:43:11.180978"
		epoch := EpochFromFormat(ts)
		expected := time.Date(2021, 2, 11, 21, 43, 11, 180978000, time.UTC).Unix()
		assert.Equal(t, expected, epoch)
	})

	t.Run("parse timestamp with microseconds in ISO format", func(t *testing.T) {
		ts := "2021-02-11 21:43:11.180978"
		epoch := EpochFromFormat(ts)
		expected := time.Date(2021, 2, 11, 21, 43, 11, 180978000, time.UTC).Unix()
		assert.Equal(t, expected, epoch)
	})

	t.Run("partial date string returns zero", func(t *testing.T) {
		ts := "2021-02-10"
		epoch := EpochFromFormat(ts)
		assert.Equal(t, int64(0), epoch)
	})

	t.Run("numeric string returns zero", func(t *testing.T) {
		ts := "1612928641"
		epoch := EpochFromFormat(ts)
		assert.Equal(t, int64(0), epoch)
	})

	t.Run("malformed RFC3339 returns zero", func(t *testing.T) {
		ts := "2021-02-10T25:70:99Z"
		epoch := EpochFromFormat(ts)
		assert.Equal(t, int64(0), epoch)
	})
}

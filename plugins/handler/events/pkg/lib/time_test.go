package lib

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type timeTestCase struct {
	TimeString string
	TimeStamp  int64
}

var tCases = []timeTestCase{
	timeTestCase{
		TimeString: "2020-03-06T14:01:07",
		TimeStamp:  1583503267,
	},
	timeTestCase{
		TimeString: "2020-03-06 14:13:30.057411",
		TimeStamp:  1583504010,
	},
	timeTestCase{
		TimeString: "2018-02-16T14:06:54.024856417Z",
		TimeStamp:  1518790014,
	},
}

func TestTime(t *testing.T) {
	t.Run("Test yimestamp calculation.", func(t *testing.T) {
		for _, testCase := range tCases {
			assert.Equal(t, testCase.TimeStamp, EpochFromFormat(testCase.TimeString))
		}
	})

}

package src

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSplitDateTimeRange(t *testing.T) {
	type testcase struct {
		begin      time.Time
		end        time.Time
		splitRange time.Duration
		maxRange   time.Duration
		results    []dateTimeRange
	}
	begin, err := time.Parse("2006/01/02", "2020/11/01")
	if err != nil {
		panic(err)
	}
	var testcases = []testcase{
		{
			begin:      begin,
			end:        begin.AddDate(0, 2, 0),
			splitRange: oneWeekTime,
			maxRange:   fiveWeekTime,
			results: []dateTimeRange{
				{BeginDate: begin.Add(0 * oneWeekTime), EndDate: begin.Add(1 * oneWeekTime)},
				{BeginDate: begin.Add(1 * oneWeekTime), EndDate: begin.Add(2 * oneWeekTime)},
				{BeginDate: begin.Add(2 * oneWeekTime), EndDate: begin.Add(3 * oneWeekTime)},
				{BeginDate: begin.Add(3 * oneWeekTime), EndDate: begin.Add(4 * oneWeekTime)},
				{BeginDate: begin.Add(4 * oneWeekTime), EndDate: begin.Add(5 * oneWeekTime)},
			},
		},
	}
	runAt(begin.AddDate(1, 0, 0), func() {
		for _, tc := range testcases {
			results := splitDateTimeRange(tc.begin, tc.end, tc.splitRange, tc.maxRange)
			if got, need := jsonStr(results), jsonStr(tc.results); got != need {
				t.Errorf("need: %s\ngot: %s\n", need, got)
			}
		}
	})
}

func jsonStr(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func runAt(t time.Time, call func()) {
	Now = func() time.Time {
		return t
	}
	defer func() {
		Now = time.Now
	}()
	call()
}

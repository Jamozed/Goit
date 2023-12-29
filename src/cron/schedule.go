// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package cron

import (
	"fmt"
	"time"

	"github.com/Jamozed/Goit/src/util"
)

type Schedule struct{ Month, Day, Weekday, Hour, Minute, Second int64 }

var (
	Immediate = Schedule{-1, -1, -1, -1, -1, -1}
	Yearly    = Schedule{1, 1, -1, 0, 0, 0}
	Monthly   = Schedule{-1, 1, -1, 0, 0, 0}
	Weekly    = Schedule{-1, -1, 1, 0, 0, 0}
	Daily     = Schedule{-1, -1, -1, 0, 0, 0}
	Hourly    = Schedule{-1, -1, -1, -1, 0, 0}
	Minutely  = Schedule{-1, -1, -1, -1, -1, 0}
)

func (s Schedule) Next(t time.Time) time.Time {
	t = t.Add(1 * time.Second)

	added := false

wrap:
	for s.Month != -1 && int64(t.Month()) != s.Month {
		if !added {
			t = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
			added = true
		}

		t = t.AddDate(0, 1, 0)

		if t.Month() == time.January {
			goto wrap
		}
	}

	for !((s.Day == -1 || int64(t.Day()) == s.Day) && (s.Weekday == -1 || int64(t.Weekday()) == s.Weekday)) {
		if !added {
			t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
			added = true
		}

		t = t.AddDate(0, 0, 1)

		if t.Day() == 1 {
			goto wrap
		}
	}

	for s.Hour != -1 && int64(t.Hour()) != s.Hour {
		if !added {
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
			added = true
		}

		t = t.Add(1 * time.Hour)

		if t.Hour() == 0 {
			goto wrap
		}
	}

	for s.Minute != -1 && int64(t.Minute()) != s.Minute {
		if !added {
			t = t.Truncate(time.Minute)
			added = true
		}

		t = t.Add(1 * time.Minute)

		if t.Minute() == 0 {
			goto wrap
		}
	}

	for s.Second != -1 && int64(t.Second()) != s.Second {
		if !added {
			t = t.Truncate(time.Second)
			added = true
		}

		t = t.Add(1 * time.Second)

		if t.Second() == 0 {
			goto wrap
		}
	}

	return t
}

func (s Schedule) IsImmediate() bool {
	return s == Immediate
}

func (s Schedule) String() string {
	if s.IsImmediate() {
		return "immediate"
	}

	return fmt.Sprintf(
		"%s %s %s %s %s %s",
		util.If(s.Month == -1, "*", fmt.Sprint(s.Month)),
		util.If(s.Day == -1, "*", fmt.Sprint(s.Day)),
		util.If(s.Weekday == -1, "*", fmt.Sprint(s.Weekday)),
		util.If(s.Hour == -1, "*", fmt.Sprint(s.Hour)),
		util.If(s.Minute == -1, "*", fmt.Sprint(s.Minute)),
		util.If(s.Second == -1, "*", fmt.Sprint(s.Second)),
	)
}

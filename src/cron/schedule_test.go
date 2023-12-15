// Copyright (C) 2023, Jakob Wakeling
// All rights reserved.

package cron_test

import (
	"testing"
	"time"

	"github.com/Jamozed/Goit/src/cron"
)

func TestNext(t *testing.T) {
	t.Run("Month", func(t *testing.T) {
		schedule := cron.Schedule{6, -1, -1, -1, -1, -1}
		baseTime := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
		expected := time.Date(1970, 6, 1, 0, 0, 0, 0, time.UTC)

		r := schedule.Next(baseTime)
		if r != expected {
			t.Error("Expected", expected, "got", r)
		}
	})

	t.Run("Month with Wrap", func(t *testing.T) {
		schedule := cron.Schedule{6, -1, -1, -1, -1, -1}
		baseTime := time.Date(1970, 8, 1, 0, 0, 0, 0, time.UTC)
		expected := time.Date(1971, 6, 1, 0, 0, 0, 0, time.UTC)

		r := schedule.Next(baseTime)
		if r != expected {
			t.Error("Expected", expected, "got", r)
		}
	})

	t.Run("Day", func(t *testing.T) {
		schedule := cron.Schedule{-1, 12, -1, -1, -1, -1}
		baseTime := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
		expected := time.Date(1970, 1, 12, 0, 0, 0, 0, time.UTC)

		r := schedule.Next(baseTime)
		if r != expected {
			t.Error("Expected", expected, "got", r)
		}
	})

	t.Run("Day with Wrap", func(t *testing.T) {
		schedule := cron.Schedule{-1, 12, -1, -1, -1, -1}
		baseTime := time.Date(1970, 1, 24, 0, 0, 0, 0, time.UTC)
		expected := time.Date(1970, 2, 12, 0, 0, 0, 0, time.UTC)

		r := schedule.Next(baseTime)
		if r != expected {
			t.Error("Expected", expected, "got", r)
		}
	})

	t.Run("Weekday", func(t *testing.T) {
		schedule := cron.Schedule{-1, -1, 3, -1, -1, -1}
		baseTime := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
		expected := time.Date(1970, 1, 7, 0, 0, 0, 0, time.UTC)

		r := schedule.Next(baseTime)
		if r != expected {
			t.Error("Expected", expected, "got", r)
		}
	})

	t.Run("Day and weekday", func(t *testing.T) {
		schedule := cron.Schedule{-1, 12, 3, -1, -1, -1}
		baseTime := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
		expected := time.Date(1970, 8, 12, 0, 0, 0, 0, time.UTC)

		r := schedule.Next(baseTime)
		if r != expected {
			t.Error("Expected", expected, "got", r)
		}
	})

	t.Run("Hour", func(t *testing.T) {
		schedule := cron.Schedule{-1, -1, -1, 18, -1, -1}
		baseTime := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
		expected := time.Date(1970, 1, 1, 18, 0, 0, 0, time.UTC)

		r := schedule.Next(baseTime)
		if r != expected {
			t.Error("Expected", expected, "got", r)
		}
	})

	t.Run("Minute", func(t *testing.T) {
		schedule := cron.Schedule{-1, -1, -1, -1, 30, -1}
		baseTime := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
		expected := time.Date(1970, 1, 1, 0, 30, 0, 0, time.UTC)

		r := schedule.Next(baseTime)
		if r != expected {
			t.Error("Expected", expected, "got", r)
		}
	})

	t.Run("Second", func(t *testing.T) {
		schedule := cron.Schedule{-1, -1, -1, -1, -1, 30}
		baseTime := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
		expected := time.Date(1970, 1, 1, 0, 0, 30, 0, time.UTC)

		r := schedule.Next(baseTime)
		if r != expected {
			t.Error("Expected", expected, "got", r)
		}
	})

	t.Run("All", func(t *testing.T) {
		schedule := cron.Schedule{3, 6, 2, 6, 45, 15}
		baseTime := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
		expected := time.Date(1973, 3, 6, 6, 45, 15, 0, time.UTC)

		r := schedule.Next(baseTime)
		if r != expected {
			t.Error("Expected", expected, "got", r)
		}
	})

	t.Run("Immediate", func(t *testing.T) {
		schedule := cron.Schedule{-1, -1, -1, -1, -1, -1}
		baseTime := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
		expected := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

		r := schedule.Next(baseTime)
		if r != expected {
			t.Error("Expected", expected, "got", r)
		}
	})
}

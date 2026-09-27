package controllers

import (
	"testing"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/jonboulle/clockwork"
)

func TestHealthCheckRunsDailyAtTenShanghai(t *testing.T) {
	for _, now := range []time.Time{
		time.Date(2026, 1, 3, 1, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 3, 3, 0, 0, 0, time.UTC),
	} {
		t.Run(now.Format(time.RFC3339), func(t *testing.T) {
			scheduler, err := gocron.NewScheduler(gocron.WithLocation(time.UTC), gocron.WithClock(clockwork.NewFakeClockAt(now)))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := scheduler.Shutdown(); err != nil {
					t.Error(err)
				}
			})
			job, err := addHealthCheckJob(scheduler, func() {})
			if err != nil {
				t.Fatal(err)
			}
			scheduler.Start()
			runs, err := job.NextRuns(2)
			if err != nil {
				t.Fatal(err)
			}
			want := time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, time.UTC)
			if !want.After(now) {
				want = want.AddDate(0, 0, 1)
			}
			if len(runs) != 2 || !runs[0].Equal(want) || !runs[1].Equal(want.AddDate(0, 0, 1)) {
				t.Fatalf("next runs = %v, want daily at %v", runs, want)
			}
		})
	}
}

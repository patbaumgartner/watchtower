package cmd

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		input    time.Duration
		expected string
	}{
		{0, "0 seconds"},
		{1 * time.Second, "1 second"},
		{30 * time.Second, "30 seconds"},
		{1 * time.Minute, "1 minute"},
		{1*time.Minute + 1*time.Second, "1 minute, 1 second"},
		{2*time.Minute + 30*time.Second, "2 minutes, 30 seconds"},
		{1 * time.Hour, "1 hour"},
		{2 * time.Hour, "2 hours"},
		{1*time.Hour + 1*time.Minute, "1 hour, 1 minute"},
		{1*time.Hour + 1*time.Second, "1 hour, 1 second"},
		{3*time.Hour + 15*time.Minute + 5*time.Second, "3 hours, 15 minutes, 5 seconds"},
	}
	for _, tc := range cases {
		if actual := formatDuration(tc.input); actual != tc.expected {
			t.Errorf("formatDuration(%v) = %q, expected %q", tc.input, actual, tc.expected)
		}
	}
}

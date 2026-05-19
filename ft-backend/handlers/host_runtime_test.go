package handlers

import (
	"testing"
	"time"

	"github.com/shirou/gopsutil/v4/disk"
)

func TestDiskIOBusyFromCounters(t *testing.T) {
	t.Parallel()
	interval := 320 * time.Millisecond
	c1 := map[string]disk.IOCountersStat{
		"sda": {IoTime: 1000},
	}
	c2 := map[string]disk.IOCountersStat{
		"sda": {IoTime: 1000 + uint64(interval.Milliseconds())/2},
	}
	got := diskIOBusyFromCounters(c1, c2, interval)
	want := 50.0
	if got < want-0.1 || got > want+0.1 {
		t.Fatalf("diskIOBusyFromCounters() = %v, want ~%v", got, want)
	}
}

func TestDiskIOBusyFromCountersReadWriteFallback(t *testing.T) {
	t.Parallel()
	interval := 200 * time.Millisecond
	c1 := map[string]disk.IOCountersStat{
		"disk0": {ReadTime: 10, WriteTime: 20},
	}
	c2 := map[string]disk.IOCountersStat{
		"disk0": {ReadTime: 30, WriteTime: 50},
	}
	got := diskIOBusyFromCounters(c1, c2, interval)
	want := clampPct(float64(50) / float64(interval.Milliseconds()) * 100)
	if got != want {
		t.Fatalf("diskIOBusyFromCounters() = %v, want %v", got, want)
	}
}

func TestDiskNameForIO(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"/dev/sda1":      "sda",
		"/dev/nvme0n1p2": "nvme0n1",
		"/dev/dm-0":      "dm-0",
		"/dev/vda2":      "vda",
	}
	for in, want := range cases {
		if got := diskNameForIO(in); got != want {
			t.Fatalf("diskNameForIO(%q) = %q, want %q", in, got, want)
		}
	}
}

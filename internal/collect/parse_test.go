package collect

import "testing"

func TestParseCPULine(t *testing.T) {
	name, tm, ok := ParseCPULine("cpu0 100 0 50 1000 10 0 5 0 0 0")
	if !ok || name != "cpu0" {
		t.Fatalf("parse failed: %v %q", ok, name)
	}
	if tm.User != 100 || tm.Idle != 1000 || tm.Steal != 0 {
		t.Fatalf("fields wrong: %+v", tm)
	}
	if !ok {
		t.Fatal("expected ok")
	}
}

func TestCPUPctFromDelta(t *testing.T) {
	prev := CPUTimes{User: 100, Idle: 900}
	cur := CPUTimes{User: 150, Idle: 950} // 50 active, 100 total -> 50%
	if p := CPUPctFromDelta(prev, cur); p != 50 {
		t.Fatalf("want 50, got %d", p)
	}
	// NO_HZ negative-delta clamps to 0.
	if p := CPUPctFromDelta(cur, prev); p != 0 {
		t.Fatalf("want 0 on negative, got %d", p)
	}
	// Zero total guard.
	if p := CPUPctFromDelta(prev, prev); p != 0 {
		t.Fatalf("want 0 on zero delta, got %d", p)
	}
}

func TestCPUPctExcludesGuest(t *testing.T) {
	// Last two fields are guest, guest_nice — must NOT be counted.
	_, t1, _ := ParseCPULine("cpu 100 0 50 900 0 0 0 0 500 500")
	if t1.Total() != 1050 {
		t.Fatalf("guest/guest_nice should be excluded, got total %d", t1.Total())
	}
}

func TestParseMeminfo(t *testing.T) {
	raw := "MemTotal:       32000000 kB\nMemAvailable:    8000000 kB\nBuffers:          100000 kB\nCached:          5000000 kB\nSwapTotal:      16000000 kB\nSwapFree:       15000000 kB\n"
	m := ParseMeminfo(raw)
	if m.TotalKiB != 32000000 || m.AvailableKiB != 8000000 || m.SwapTotalKiB != 16000000 {
		t.Fatalf("parse failed: %+v", m)
	}
}

func TestParseDevfreqLoad(t *testing.T) {
	pct, hz := ParseDevfreqLoad("22@300000000Hz")
	if pct != 22 || hz != 300000000 {
		t.Fatalf("want 22,300000000 got %d,%d", pct, hz)
	}
	pct, hz = ParseDevfreqLoad("0@1000000000Hz")
	if pct != 0 || hz != 1000000000 {
		t.Fatalf("idle case wrong: %d,%d", pct, hz)
	}
	// Edge: trailing newline.
	pct, hz = ParseDevfreqLoad("100@1000000000Hz\n")
	if pct != 100 || hz != 1000000000 {
		t.Fatalf("trailing newline case wrong: %d,%d", pct, hz)
	}
}

func TestParseRKNPULoad(t *testing.T) {
	raw := "NPU load:  Core0:  0%, Core1: 12%, Core2: 47%,\n"
	got := ParseRKNPULoad(raw)
	if len(got) != 3 || got[0] != 0 || got[1] != 12 || got[2] != 47 {
		t.Fatalf("want [0 12 47], got %v", got)
	}
}

func TestParseMPPLoad(t *testing.T) {
	raw := `fdbd0000.rkvenc-core      load:  98.26% utilization:  97.25%
fdbe0000.rkvenc-core      load:  10.78% utilization:  10.69%
fdc38100.rkvdec-core      load:  17.41% utilization:  17.09%
fdc48100.rkvdec-core      load:   0.00% utilization:   0.00%
`
	got := ParseMPPLoad(raw)
	if len(got) != 4 {
		t.Fatalf("want 4 entries, got %d", len(got))
	}
	if got[0].LoadPct < 98 || got[0].LoadPct > 99 {
		t.Fatalf("decimal parse wrong: %+v", got[0])
	}
	if got[3].LoadPct != 0 {
		t.Fatalf("zero parse wrong: %+v", got[3])
	}
	if got[0].Device != "rkvenc-core" {
		t.Fatalf("device wrong: %s", got[0].Device)
	}
}

func TestParseRGALoad(t *testing.T) {
	raw := `num of scheduler = 3
================= load ==================
scheduler[0]: rga3
	 load = 42%
-----------------------------------
scheduler[1]: rga3
	 load = 27%
-----------------------------------
scheduler[2]: rga2
	 load = 0%
-----------------------------------
`
	got := ParseRGALoad(raw)
	if len(got) != 3 {
		t.Fatalf("want 3 entries, got %d", len(got))
	}
	if got[0].Name != "rga3" || got[0].LoadPct != 42 {
		t.Fatalf("[0] wrong: %+v", got[0])
	}
	if got[1].Name != "rga3_1" || got[1].LoadPct != 27 {
		t.Fatalf("[1] wrong (want duplicate disambiguation): %+v", got[1])
	}
	if got[2].Name != "rga2" || got[2].LoadPct != 0 {
		t.Fatalf("[2] wrong: %+v", got[2])
	}
}

func TestParseLoadavg(t *testing.T) {
	avg, running, total := ParseLoadavg("1.16 1.33 1.42 8/1395 12345\n")
	if avg[0] != 1.16 || avg[1] != 1.33 || avg[2] != 1.42 {
		t.Fatalf("avg wrong: %+v", avg)
	}
	if running != 8 || total != 1395 {
		t.Fatalf("procs wrong: %d/%d", running, total)
	}
}

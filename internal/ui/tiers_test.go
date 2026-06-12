package ui

import "testing"

func TestParseTiers(t *testing.T) {
	cases := []struct {
		in      string
		want    [3]int8
		wantErr bool
	}{
		{"", [3]int8{0, 0, 0}, false},
		{"all", [3]int8{1, 1, 1}, false},
		{"none", [3]int8{-1, -1, -1}, false},
		{"1", [3]int8{1, -1, -1}, false},
		{"2", [3]int8{-1, 1, -1}, false},
		{"3", [3]int8{-1, -1, 1}, false},
		{"1,3", [3]int8{1, -1, 1}, false},
		{"1,2,3", [3]int8{1, 1, 1}, false},
		{"1,1,2", [3]int8{1, 1, -1}, false},
		{" 1 , 3 ", [3]int8{1, -1, 1}, false},
		{"foo", [3]int8{}, true},
		{"4", [3]int8{}, true},
		{"0", [3]int8{}, true},
		{"-1", [3]int8{}, true},
		{"1,foo", [3]int8{}, true},
		{"i", [3]int8{1, -1, -1}, false},
		{"s", [3]int8{-1, 1, -1}, false},
		{"k", [3]int8{-1, -1, 1}, false},
		{"i,k", [3]int8{1, -1, 1}, false},
		{"I,S,K", [3]int8{1, 1, 1}, false},
		{"io,kernel", [3]int8{1, -1, 1}, false},
		{"sys", [3]int8{-1, 1, -1}, false},
		{"ALL", [3]int8{1, 1, 1}, false},
		{"None", [3]int8{-1, -1, -1}, false},
		{"NONE", [3]int8{-1, -1, -1}, false},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, err := parseTiers(c.in)
			if (err != nil) != c.wantErr {
				t.Fatalf("parseTiers(%q) err=%v wantErr=%v", c.in, err, c.wantErr)
			}
			if !c.wantErr && got != c.want {
				t.Fatalf("parseTiers(%q) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestToggleTier(t *testing.T) {
	cases := []struct {
		state  int8
		height int
		idx    int
		want   int8
	}{
		{0, 60, 0, -1},
		{0, 60, 1, -1},
		{0, 60, 2, -1},
		{0, 30, 0, 1},
		{0, 30, 1, 1},
		{0, 30, 2, 1},
		{1, 0, 0, -1},
		{1, 100, 0, -1},
		{-1, 0, 0, 1},
		{-1, 100, 0, 1},
	}
	for _, c := range cases {
		got := toggleTier(c.state, c.height, c.idx)
		if got != c.want {
			t.Fatalf("toggleTier(state=%d,h=%d,idx=%d) = %d, want %d",
				c.state, c.height, c.idx, got, c.want)
		}
	}
}

func TestTierStateLabel(t *testing.T) {
	cases := []struct {
		in   int8
		want string
	}{
		{0, "auto"},
		{1, "on"},
		{-1, "off"},
	}
	for _, c := range cases {
		got := tierStateLabel(c.in)
		if got != c.want {
			t.Fatalf("tierStateLabel(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFooterStateLabel(t *testing.T) {
	cases := []struct {
		state  int8
		height int
		idx    int
		want   string
	}{
		{0, 60, 0, "on"},
		{0, 60, 2, "on"},
		{0, 30, 0, "off"},
		{0, 30, 2, "off"},
		{1, 0, 0, "on"},
		{1, 100, 2, "on"},
		{-1, 0, 0, "off"},
		{-1, 100, 2, "off"},
	}
	for _, c := range cases {
		got := footerStateLabel(c.state, c.height, c.idx)
		if got != c.want {
			t.Fatalf("footerStateLabel(state=%d,h=%d,idx=%d) = %q, want %q",
				c.state, c.height, c.idx, got, c.want)
		}
	}
}

func TestTierVisible(t *testing.T) {
	cases := []struct {
		state  int8
		height int
		idx    int
		want   bool
	}{
		{1, 0, 0, true},
		{1, 0, 1, true},
		{1, 0, 2, true},
		{-1, 100, 0, false},
		{-1, 100, 2, false},
		{0, 0, 0, false},
		{0, 30, 0, false},
		{0, 44, 0, true},
		{0, 56, 0, true},
		{0, 56, 1, false},
		{0, 57, 1, true},
		{0, 59, 2, false},
		{0, 60, 2, true},
	}
	for _, c := range cases {
		got := tierVisible(c.state, c.height, c.idx)
		if got != c.want {
			t.Fatalf("tierVisible(state=%d,h=%d,idx=%d) = %v, want %v",
				c.state, c.height, c.idx, got, c.want)
		}
	}
}

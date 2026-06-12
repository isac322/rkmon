package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/lucasb-eyer/go-colorful"
)

// Styles holds the lipgloss styles for one frame. NoColor swaps everything
// to the zero (plain) style so the same rendering code works without ANSI.
type Styles struct {
	NoColor bool

	Border  lipgloss.Style
	Title   lipgloss.Style
	Label   lipgloss.Style
	Value   lipgloss.Style
	Dim     lipgloss.Style
	Hint    lipgloss.Style
	Cluster lipgloss.Style

	BarEmpty   lipgloss.Style
	StaleStyle lipgloss.Style

	pctBuckets  [21]lipgloss.Style
	tempBuckets [6]lipgloss.Style

	pctANSI      [21]string
	tempANSI     [6]string
	barEmptyANSI string
	staleANSI    string
	ansiReset    string

	BorderPfx  string
	TitlePfx   string
	LabelPfx   string
	ValuePfx   string
	DimPfx     string
	HintPfx    string
	ClusterPfx string

	BorderBarL string
	BorderBarR string
}

// NewStyles builds a Styles set. When noColor is true every style is a no-op
// so the View can call .Render unconditionally.
func NewStyles(noColor bool) Styles {
	if noColor {
		plain := lipgloss.NewStyle()
		s := Styles{
			NoColor: true,
			Border:  plain, Title: plain, Label: plain, Value: plain,
			Dim: plain, Hint: plain, Cluster: plain, BarEmpty: plain,
			StaleStyle: plain,
			ansiReset:  "",
		}
		for i := range s.pctBuckets {
			s.pctBuckets[i] = plain
		}
		for i := range s.tempBuckets {
			s.tempBuckets[i] = plain
		}
		s.BorderBarL = "│ "
		s.BorderBarR = " │"
		return s
	}
	s := Styles{
		Border:     lipgloss.NewStyle().Foreground(lipgloss.Color("#5a5f6c")),
		Title:      lipgloss.NewStyle().Foreground(lipgloss.Color("#5fafff")).Bold(true),
		Label:      lipgloss.NewStyle().Foreground(lipgloss.Color("#9499a5")),
		Value:      lipgloss.NewStyle().Foreground(lipgloss.Color("#d8dee9")),
		Dim:        lipgloss.NewStyle().Foreground(lipgloss.Color("#5a5f6c")),
		Hint:       lipgloss.NewStyle().Foreground(lipgloss.Color("#7c8290")).Italic(true),
		Cluster:    lipgloss.NewStyle().Foreground(lipgloss.Color("#88c0d0")).Bold(true),
		BarEmpty:   lipgloss.NewStyle().Foreground(lipgloss.Color("#3a3f48")),
		StaleStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#7c8290")).Italic(true),
	}
	for i := 0; i <= 20; i++ {
		s.pctBuckets[i] = lipgloss.NewStyle().Foreground(lipgloss.Color(pctHex(float64(i * 5))))
		s.pctANSI[i] = extractAnsiPrefix(s.pctBuckets[i])
	}
	tempHexes := [6]string{"#88c0d0", "#88c0d0", "#a3be8c", "#ebcb8b", "#d08770", "#bf616a"}
	for i, h := range tempHexes {
		s.tempBuckets[i] = lipgloss.NewStyle().Foreground(lipgloss.Color(h))
		s.tempANSI[i] = extractAnsiPrefix(s.tempBuckets[i])
	}
	s.barEmptyANSI = extractAnsiPrefix(s.BarEmpty)
	s.staleANSI = extractAnsiPrefix(s.StaleStyle)
	s.ansiReset = "\x1b[0m"

	s.BorderPfx = extractAnsiPrefix(s.Border)
	s.TitlePfx = extractAnsiPrefix(s.Title)
	s.LabelPfx = extractAnsiPrefix(s.Label)
	s.ValuePfx = extractAnsiPrefix(s.Value)
	s.DimPfx = extractAnsiPrefix(s.Dim)
	s.HintPfx = extractAnsiPrefix(s.Hint)
	s.ClusterPfx = extractAnsiPrefix(s.Cluster)

	s.BorderBarL = s.BorderPfx + "│ " + s.ansiReset
	s.BorderBarR = s.BorderPfx + " │" + s.ansiReset
	return s
}

func (s Styles) border(t string) string  { return s.BorderPfx + t + s.ansiReset }
func (s Styles) title(t string) string   { return s.TitlePfx + t + s.ansiReset }
func (s Styles) label(t string) string   { return s.LabelPfx + t + s.ansiReset }
func (s Styles) value(t string) string   { return s.ValuePfx + t + s.ansiReset }
func (s Styles) dim(t string) string     { return s.DimPfx + t + s.ansiReset }
func (s Styles) hint(t string) string    { return s.HintPfx + t + s.ansiReset }
func (s Styles) cluster(t string) string { return s.ClusterPfx + t + s.ansiReset }

func extractAnsiPrefix(st lipgloss.Style) string {
	const mark = "\x00MARK\x00"
	out := st.Render(mark)
	idx := strings.Index(out, mark)
	if idx <= 0 {
		return ""
	}
	return out[:idx]
}

// PctStyle returns the precomputed style for a percentage value (5%-bucket
// quantised; smooth gradient is preserved across buckets visually).
func PctStyle(s Styles, pct float64) lipgloss.Style {
	if s.NoColor {
		return s.Value
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return s.pctBuckets[int(pct/5)]
}

func PctStaleStyle(s Styles) lipgloss.Style { return s.StaleStyle }

// gradient endpoints (Nord-ish):
//
//	0%  -> #88c0d0 (frost cyan)   "cool/idle"
//	50% -> #ebcb8b (aurora yellow) "warm/busy"
//	100%-> #bf616a (aurora red)    "hot/saturated"
var (
	gradLow, _  = colorful.Hex("#88c0d0")
	gradMid, _  = colorful.Hex("#ebcb8b")
	gradHigh, _ = colorful.Hex("#bf616a")
)

func pctHex(pct float64) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	t := pct / 100.0
	var c colorful.Color
	if t < 0.5 {
		c = gradLow.BlendHcl(gradMid, t*2).Clamped()
	} else {
		c = gradMid.BlendHcl(gradHigh, (t-0.5)*2).Clamped()
	}
	return c.Hex()
}

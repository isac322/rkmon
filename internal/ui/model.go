package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/isac322/rkmon/internal/collect"
)

type Model struct {
	collector  *collect.Collector
	styles     Styles
	refresh    time.Duration
	snap       *collect.Snapshot
	err        error
	tick       int
	width      int
	height     int
	tiers      [3]int8
	sections   [SecCount]bool
	helpMode   bool
	helpTab    int
	helpScroll int
	mainScroll int
}

func NewModel(c *collect.Collector, refresh time.Duration, color bool) Model {
	return NewModelWithTiers(c, refresh, color, [3]int8{})
}

func NewModelWithTiers(c *collect.Collector, refresh time.Duration, color bool, tiers [3]int8) Model {
	return Model{
		collector: c,
		styles:    NewStyles(!color),
		refresh:   refresh,
		width:     100,
		height:    40,
		tiers:     tiers,
		sections:  DefaultSections(),
	}
}

type snapshotMsg struct {
	snap *collect.Snapshot
	err  error
}

func (m Model) Init() tea.Cmd {
	return collectCmd(m.collector)
}

func collectCmd(c *collect.Collector) tea.Cmd {
	return func() tea.Msg {
		snap, err := c.Snapshot()
		return snapshotMsg{snap: snap, err: err}
	}
}

func tickAndCollectCmd(c *collect.Collector, d time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(d)
		snap, err := c.Snapshot()
		return snapshotMsg{snap: snap, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.mainScroll = m.clampMainScroll(m.mainScroll)
		m.helpScroll = clampHelpScroll(m.helpScroll, m.helpTab, m.height)
		return m, nil

	case tea.KeyMsg:
		if m.helpMode {
			switch msg.String() {
			case "esc", "q", "?":
				m.helpMode = false
				m.helpScroll = 0
				return m, nil
			case "tab", "right", "l", "n":
				m.helpTab = (m.helpTab + 1) % len(helpTabs)
				m.helpScroll = 0
				return m, nil
			case "shift+tab", "left", "h", "p":
				m.helpTab = (m.helpTab + len(helpTabs) - 1) % len(helpTabs)
				m.helpScroll = 0
				return m, nil
			case "up", "k":
				m.helpScroll = clampHelpScroll(m.helpScroll-1, m.helpTab, m.height)
				return m, nil
			case "down", "j":
				m.helpScroll = clampHelpScroll(m.helpScroll+1, m.helpTab, m.height)
				return m, nil
			case "pgup", "b", "ctrl+u":
				m.helpScroll = clampHelpScroll(m.helpScroll-10, m.helpTab, m.height)
				return m, nil
			case "pgdown", "f", " ", "ctrl+d":
				m.helpScroll = clampHelpScroll(m.helpScroll+10, m.helpTab, m.height)
				return m, nil
			case "home", "g":
				m.helpScroll = 0
				return m, nil
			case "end", "G":
				m.helpScroll = helpMaxScroll(m.helpTab, m.height)
				return m, nil
			case "ctrl+c":
				return m, tea.Quit
			}
			return m, nil
		}
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "?":
			m.helpMode = true
			return m, nil
		case "r":
			return m, collectCmd(m.collector)
		case "+", "=":
			if m.refresh > 200*time.Millisecond {
				m.refresh -= 100 * time.Millisecond
			}
			return m, nil
		case "-", "_":
			if m.refresh < 10*time.Second {
				m.refresh += 100 * time.Millisecond
			}
			return m, nil
		case "1", "i", "I":
			m.tiers[0] = toggleTier(m.tiers[0], m.height, 0)
			m.mainScroll = 0
			return m, nil
		case "2", "s", "S":
			m.tiers[1] = toggleTier(m.tiers[1], m.height, 1)
			m.mainScroll = 0
			return m, nil
		case "3", "k", "K":
			m.tiers[2] = toggleTier(m.tiers[2], m.height, 2)
			m.mainScroll = 0
			return m, nil
		case "c", "C":
			m.sections = toggleSection(m.sections, SecCPU)
			m.mainScroll = 0
			return m, nil
		case "m", "M":
			m.sections = toggleSection(m.sections, SecMEM)
			m.mainScroll = 0
			return m, nil
		case "g", "G":
			m.sections = toggleSection(m.sections, SecGPU)
			m.mainScroll = 0
			return m, nil
		case "n", "N":
			m.sections = toggleSection(m.sections, SecNPU)
			m.mainScroll = 0
			return m, nil
		case "v", "V":
			m.sections = toggleSection(m.sections, SecVPU)
			m.mainScroll = 0
			return m, nil
		case "a", "A":
			m.sections = toggleSection(m.sections, SecRGA)
			m.mainScroll = 0
			return m, nil
		case "up":
			m.mainScroll = m.clampMainScroll(m.mainScroll - 1)
			return m, nil
		case "down", "j":
			m.mainScroll = m.clampMainScroll(m.mainScroll + 1)
			return m, nil
		case "pgup":
			m.mainScroll = m.clampMainScroll(m.mainScroll - 10)
			return m, nil
		case "pgdown", " ":
			m.mainScroll = m.clampMainScroll(m.mainScroll + 10)
			return m, nil
		case "home":
			m.mainScroll = 0
			return m, nil
		case "end":
			m.mainScroll = m.clampMainScroll(1 << 30)
			return m, nil
		}

	case snapshotMsg:
		m.snap = msg.snap
		m.err = msg.err
		m.tick++
		// Single source of frames: one snapshot → one render → schedule next.
		return m, tickAndCollectCmd(m.collector, m.refresh)
	}
	return m, nil
}

func (m Model) clampMainScroll(v int) int {
	bodyLen := mainBodyLen(m.styles, m.snap, m.refresh, m.tick, m.width, m.height, m.tiers, m.sections)
	return clampMainScroll(v, bodyLen, m.height)
}

func (m Model) View() string {
	if m.helpMode {
		return renderHelp(m.styles, m.width, m.height, m.helpTab, m.helpScroll)
	}
	return Render(m.styles, m.snap, m.refresh, m.tick, m.width, m.height, m.tiers, m.sections, m.mainScroll)
}

func RenderStatic(snap *collect.Snapshot, color bool, width int, tiers [3]int8) string {
	s := NewStyles(!color)
	return Render(s, snap, time.Second, 0, width, 0, tiers, DefaultSections(), 0)
}

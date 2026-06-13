// Command rkmon is a TUI hardware monitor for Rockchip RK3588 (Radxa Rock 5B+).
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/pprof"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"

	"github.com/isac322/rkmon/internal/collect"
	"github.com/isac322/rkmon/internal/ui"
)

// Version and GitSHA are set via -ldflags at build time.
var (
	Version = "0.3.1"
	GitSHA  = "dev"
)

func main() {
	refresh := flag.Duration("refresh", 1*time.Second, "refresh interval (100ms..60s)")
	noColor := flag.Bool("no-color", false, "disable ANSI colors")
	once := flag.Bool("once", false, "render one frame to stdout and exit")
	width := flag.Int("width", 0, "override terminal width (0 = auto / $COLUMNS / pty size)")
	tiersFlag := flag.String("tiers", "", "per-tier override: '' (auto by height) | 'all' | 'none' | comma list like 'i,k' (1,3 also accepted)")
	cpuProfile := flag.String("cpuprofile", "", "write CPU profile to file (runs --refresh ticks; q quits)")
	memProfile := flag.String("memprofile", "", "write heap profile to file at exit")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "rkmon - RK3588 hardware monitor TUI\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n\nFlags:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nKeybinds (TUI), in screen order:\n  c m g n v a  toggle CPU/MEM/GPU/NPU/VPU/RGA sections\n  i s k        toggle I/O / System / Kernel tiers (1/2/3 also accepted)\n  ↑↓ pgup pgdn home end  scroll main view (j also = down)\n  q/ctrl+c     quit\n  + / -        adjust refresh ±100ms\n  r            force redraw / re-collect\n  ?            open multi-tab help; inside: tab/←→ tabs, ↑↓/pgup/pgdn/g/G scroll, esc close\n")
	}
	flag.Parse()

	if *showVersion {
		fmt.Printf("rkmon %s+%s\n", Version, GitSHA)
		return
	}

	if *refresh < 100*time.Millisecond || *refresh > 60*time.Second {
		fmt.Fprintf(os.Stderr, "invalid --refresh %s: must be 100ms..60s\n", *refresh)
		os.Exit(2)
	}

	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cpuprofile open: %v\n", err)
			os.Exit(1)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			fmt.Fprintf(os.Stderr, "cpuprofile start: %v\n", err)
			os.Exit(1)
		}
		defer func() { _ = f.Close() }()
		defer pprof.StopCPUProfile()
	}
	if *memProfile != "" {
		defer func() {
			f, err := os.Create(*memProfile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "memprofile open: %v\n", err)
				return
			}
			defer func() { _ = f.Close() }()
			if err := pprof.WriteHeapProfile(f); err != nil {
				fmt.Fprintf(os.Stderr, "memprofile write: %v\n", err)
			}
		}()
	}

	c := collect.New()

	tiers, err := ui.ParseTiersFlag(*tiersFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid --tiers %q: %v\n", *tiersFlag, err)
		os.Exit(2)
	}
	if *once {
		if _, err := c.Snapshot(); err != nil {
			fmt.Fprintf(os.Stderr, "snapshot error: %v\n", err)
			os.Exit(1)
		}
		time.Sleep(250 * time.Millisecond)
		snap, err := c.Snapshot()
		if err != nil {
			fmt.Fprintf(os.Stderr, "snapshot error: %v\n", err)
			os.Exit(1)
		}
		w := resolveWidth(*width)
		onceTiers := tiers
		if onceTiers == ([3]int8{ui.TierAuto, ui.TierAuto, ui.TierAuto}) {
			onceTiers = [3]int8{ui.TierOn, ui.TierOn, ui.TierOn}
		}
		fmt.Println(ui.RenderStatic(snap, !*noColor, w, onceTiers))
		return
	}

	model := ui.NewModelWithTiers(c, *refresh, !*noColor, tiers)
	p := tea.NewProgram(model)
	_, err = p.Run()
	c.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func resolveWidth(override int) int {
	if override > 0 {
		return override
	}
	if env := os.Getenv("COLUMNS"); env != "" {
		if v, err := strconv.Atoi(env); err == nil && v > 0 {
			return v
		}
	}
	if w, _, err := term.GetSize(os.Stdout.Fd()); err == nil && w > 0 {
		return w
	}
	return 100
}

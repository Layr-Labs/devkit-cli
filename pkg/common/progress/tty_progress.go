package progress

import (
	"time"

	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"

	"fmt"
	"os"
	"strings"
	"sync"
)

type TTYProgressTracker struct {
	mu         sync.Mutex
	progress   map[string]*iface.ProgressInfo
	order      []string
	maxTracked int
	linesDrawn int
	target     *os.File
}

func NewTTYProgressTracker(max int, target *os.File) *TTYProgressTracker {
	return &TTYProgressTracker{
		progress:   make(map[string]*iface.ProgressInfo),
		order:      make([]string, 0, max),
		maxTracked: max,
		target:     target,
	}
}

// ProgressRows returns all progress entries, in the order they completed.
// It is safe to call from multiple goroutines.
func (s *TTYProgressTracker) ProgressRows() []iface.ProgressRow {
	s.mu.Lock()
	defer s.mu.Unlock()

	rows := make([]iface.ProgressRow, 0, len(s.order))
	for _, id := range s.order {
		info := s.progress[id]
		rows = append(rows, iface.ProgressRow{
			Module: id,
			Pct:    info.Percentage,
			Label:  info.DisplayText,
		})
	}
	return rows
}

func (t *TTYProgressTracker) Set(id string, pct int, label string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	ts := time.Now().Format("2006/01/02 15:04:05")

	if info, exists := t.progress[id]; exists {
		if info.Percentage >= pct {
			return
		}
		info.Percentage = pct
		info.DisplayText = label
		info.Timestamp = ts
	} else {
		if len(t.progress) >= t.maxTracked {
			return
		}
		t.progress[id] = &iface.ProgressInfo{
			Percentage:  pct,
			DisplayText: label,
			Timestamp:   ts,
		}
		t.order = append(t.order, id)
	}
}

func (t *TTYProgressTracker) Render() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.linesDrawn > 0 {
		fmt.Fprintf(t.target, "\033[%dA", t.linesDrawn)
	}
	t.linesDrawn = 0

	for _, id := range t.order {
		info := t.progress[id]
		bar := buildBar(info.Percentage)
		fmt.Fprintf(t.target, "\r\033[K%s %s %3d%% %s\n", info.Timestamp, bar, info.Percentage, info.DisplayText)
		t.linesDrawn++
	}
}

func (t *TTYProgressTracker) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.progress = make(map[string]*iface.ProgressInfo)
	t.order = t.order[:0]
	t.linesDrawn = 0

	// print timestamp line on clear
	ts := time.Now().Format("2006/01/02 15:04:05")
	fmt.Fprintf(t.target, "%s\n", ts)
}

func buildBar(pct int) string {
	total := 20
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := pct * total / 100
	return fmt.Sprintf("[%s%s]", strings.Repeat("=", filled), strings.Repeat(" ", total-filled))
}

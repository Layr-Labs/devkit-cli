package progress

import (
	"fmt"
	"sync"

	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
)

type LogProgressTracker struct {
	mu         sync.Mutex
	logger     iface.Logger
	progress   map[string]*iface.ProgressInfo
	order      []string
	maxTracked int
}

func NewLogProgressTracker(max int, logger iface.Logger) *LogProgressTracker {
	return &LogProgressTracker{
		logger:     logger,
		progress:   make(map[string]*iface.ProgressInfo),
		order:      make([]string, 0, max),
		maxTracked: max,
	}
}

// ProgressRows returns all progress entries, in the order they completed.
// It is safe to call from multiple goroutines.
func (s *LogProgressTracker) ProgressRows() []iface.ProgressRow {
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

func (s *LogProgressTracker) Set(id string, pct int, label string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if info, exists := s.progress[id]; exists {
		// end early if this has already reached 100%
		if info.Percentage >= pct || info.Percentage == 100 {
			return
		}
		info.Percentage = pct
		info.DisplayText = label
	} else {
		if len(s.progress) >= s.maxTracked {
			return
		}
		s.progress[id] = &iface.ProgressInfo{
			Percentage:  pct,
			DisplayText: label,
		}
		s.order = append(s.order, id)
	}
	// print to logger on first 100% progress report
	info := s.progress[id]
	if info.Percentage == 100 {
		s.logger.Info(fmt.Sprintf("Progress: %s - %d%%", info.DisplayText, info.Percentage))
	}
}

func (s *LogProgressTracker) Render() {
	// no-op - only print on 100% at Set
}

func (s *LogProgressTracker) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.progress = make(map[string]*iface.ProgressInfo)
	s.order = s.order[:0]
}

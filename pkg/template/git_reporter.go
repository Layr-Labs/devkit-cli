package template

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Layr-Labs/devkit-cli/pkg/common/logger"
)

// cloneReporter implements Reporter and knows how to render submodules + progress.
type cloneReporter struct {
	logger     logger.ProgressLogger
	repoName   string
	parent     string
	final      string
	discovered []string
	metrics    GitMetrics
}

func NewCloneReporter(repoURL string, lg logger.ProgressLogger, m GitMetrics) Reporter {
	return &cloneReporter{
		repoName: filepath.Base(strings.TrimSuffix(repoURL, ".git")),
		logger:   lg,
		metrics:  m,
	}
}

func (r *cloneReporter) Report(e CloneEvent) {
	switch e.Type {
	case EventSubmoduleDiscovered:
		if r.parent != e.Parent {
			r.discovered = r.discovered[:0]
			r.parent = e.Parent
		}
		r.discovered = append(r.discovered,
			fmt.Sprintf("%s → %s (%s)", e.Name, e.Parent+e.Name, e.URL),
		)

	case EventSubmoduleCloneStart:
		if len(r.discovered) > 0 {
			header := r.repoName
			if e.Parent != "" && r.parent != "." {
				header = strings.TrimSuffix(r.parent, "/")
			}
			r.logger.Info("\nDiscovered submodules for %s", header)
			for _, d := range r.discovered {
				r.logger.Info(" - %s\n", d)
			}
			r.discovered = nil
			r.logger.ClearProgress()
		}

	case EventProgress:
		mod := e.Module
		desc := e.Module
		if mod == "" || mod == "." || mod == r.repoName {
			mod = r.repoName
			desc = fmt.Sprintf("%s (Cloning from ref: %s)", r.repoName, e.Ref)
		}
		r.logger.SetProgress(mod, e.Progress, desc)
		r.logger.PrintProgress()
		r.final = mod

	case EventCloneComplete:
		if r.metrics != nil {
			r.metrics.CloneFinished(r.repoName, nil)
		}
		r.logger.SetProgress(r.final, 100, r.final)
		r.logger.PrintProgress()
		r.logger.ClearProgress()

	case EventCloneFailed:
		r.logger.ClearProgress()
	}
}

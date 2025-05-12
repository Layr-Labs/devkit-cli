package progress

import (
	"os"

	"golang.org/x/term"
)

func IsTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

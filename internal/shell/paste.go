package shell

import (
	"fmt"
	"os"
)

const (
	bracketedPasteStart = "\033[200~"
	bracketedPasteEnd   = "\033[201~"
	enableBracketed     = "\033[?2004h"
	disableBracketed    = "\033[?2004l"
)

func WriteResult(cmd string) error {
	if cmd == "" {
		return nil
	}
	_, err := fmt.Fprintf(os.Stdout, "%s%s%s%s", enableBracketed, bracketedPasteStart, cmd, bracketedPasteEnd)
	return err
}


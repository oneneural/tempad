//go:build darwin

package notify

import (
	"fmt"
	"os/exec"
)

// platformSend sends a macOS notification via osascript.
func platformSend(title, body string) error {
	script := fmt.Sprintf(`display notification %q with title %q`, body, title)
	return exec.Command("osascript", "-e", script).Run()
}

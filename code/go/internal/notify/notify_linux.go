//go:build linux

package notify

import "os/exec"

// platformSend sends a Linux notification via notify-send.
func platformSend(title, body string) error {
	return exec.Command("notify-send", title, body).Run()
}

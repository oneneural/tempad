//go:build !darwin && !linux && !windows

package notify

// platformSend is a no-op on unsupported platforms.
func platformSend(_, _ string) error {
	return nil
}

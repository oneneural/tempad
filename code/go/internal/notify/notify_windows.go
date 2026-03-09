//go:build windows

package notify

import (
	"fmt"
	"os/exec"
)

// platformSend sends a Windows notification via PowerShell toast notification.
func platformSend(title, body string) error {
	// Use BurntToast module if available, fall back to basic .NET toast API.
	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
try {
    New-BurntToastNotification -Text '%s', '%s'
} catch {
    [Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
    $template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)
    $textNodes = $template.GetElementsByTagName('text')
    $textNodes.Item(0).AppendChild($template.CreateTextNode('%s')) | Out-Null
    $textNodes.Item(1).AppendChild($template.CreateTextNode('%s')) | Out-Null
    $toast = [Windows.UI.Notifications.ToastNotification]::new($template)
    [Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('TEMPAD').Show($toast)
}`, title, body, title, body)
	return exec.Command("powershell", "-NoProfile", "-Command", script).Run()
}

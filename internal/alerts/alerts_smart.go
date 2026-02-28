package alerts

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// handleSmartDeviceAlert sends alerts when a SMART device state worsens into WARNING/FAILED.
// This is automatic and does not require user opt-in.
func (am *AlertManager) handleSmartDeviceAlert(e *core.RecordEvent) error {
	oldState := e.Record.Original().GetString("state")
	newState := e.Record.GetString("state")

	if !shouldSendSmartDeviceAlert(oldState, newState) {
		return e.Next()
	}

	systemID := e.Record.GetString("system")
	if systemID == "" {
		return e.Next()
	}

	// Fetch the system record to get the name and users
	systemRecord, err := e.App.FindRecordById("systems", systemID)
	if err != nil {
		e.App.Logger().Error("Failed to find system for SMART alert", "err", err, "systemID", systemID)
		return e.Next()
	}

	systemName := systemRecord.GetString("name")
	systemHost := systemRecord.GetString("host")
	deviceName := e.Record.GetString("name")
	model := e.Record.GetString("model")
	stateLabel := smartStateLabelCN(newState)

	currentTime := time.Now().Format("2006-01-02 15:04:05")

	// Build alert message
	title := fmt.Sprintf("%s 磁盘健康告警", smartStateEmoji(newState))

	message := fmt.Sprintf("节点名称：%s", systemName)
	if systemHost != "" {
		message += fmt.Sprintf("\nIP 地址：%s", systemHost)
	}
	message += fmt.Sprintf("\n磁盘设备：%s", deviceName)
	if model != "" {
		message += fmt.Sprintf("\n磁盘型号：%s", model)
	}
	message += fmt.Sprintf("\nSMART 状态：%s", stateLabel)
	message += fmt.Sprintf("\n时间：%s", currentTime)

	// Get users associated with the system
	userIDs := systemRecord.GetStringSlice("users")
	if len(userIDs) == 0 {
		return e.Next()
	}

	// Send alert to each user
	for _, userID := range userIDs {
		if err := am.SendAlert(AlertMessageData{
			UserID:   userID,
			SystemID: systemID,
			Title:    title,
			Message:  message,
			Link:     "",
			LinkText: "",
		}); err != nil {
			e.App.Logger().Error("Failed to send SMART alert", "err", err, "userID", userID)
		}
	}

	return e.Next()
}

func shouldSendSmartDeviceAlert(oldState, newState string) bool {
	oldSeverity := smartStateSeverity(oldState)
	newSeverity := smartStateSeverity(newState)

	// Ignore unknown states and recoveries; only alert on worsening transitions
	// from known-good/degraded states into WARNING/FAILED.
	return oldSeverity >= 1 && newSeverity > oldSeverity
}

func smartStateSeverity(state string) int {
	switch state {
	case "PASSED":
		return 1
	case "WARNING":
		return 2
	case "FAILED":
		return 3
	default:
		return 0
	}
}

func smartStateEmoji(state string) string {
	switch state {
	case "WARNING":
		return "\U0001F7E0"
	default:
		return "\U0001F534"
	}
}

func smartStateLabelCN(state string) string {
	switch state {
	case "PASSED":
		return "正常"
	case "WARNING":
		return "警告"
	case "FAILED":
		return "故障"
	default:
		return state
	}
}

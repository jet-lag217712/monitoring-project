package control

import (
	"encoding/json"
	"os"
	"time"
)

const agentDebugLogPath = "/Users/jeetlad/Projects/Equate/monitoring-dashboard/.cursor/debug-61ec8d.log"

func agentDebugLog(hypothesisID, location, message string, data map[string]any) {
	// #region agent log
	f, err := os.OpenFile(agentDebugLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	payload := map[string]any{
		"sessionId":    "61ec8d",
		"hypothesisId": hypothesisID,
		"location":     location,
		"message":      message,
		"data":         data,
		"timestamp":    time.Now().UnixMilli(),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_, _ = f.Write(append(b, '\n'))
	// #endregion
}

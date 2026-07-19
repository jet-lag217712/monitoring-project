package setup

import (
	"encoding/json"
	"os"
	"time"
)

const debugLogPath = "/Users/jeetlad/Projects/Equate/monitoring-dashboard/.cursor/debug-36805e.log"

// #region agent log
func agentLog(hypothesisID, location, message, runID string, data map[string]any) {
	payload := map[string]any{
		"sessionId":    "36805e",
		"hypothesisId": hypothesisID,
		"location":     location,
		"message":      message,
		"runId":        runID,
		"data":         data,
		"timestamp":    time.Now().UnixMilli(),
	}
	line, err := json.Marshal(payload)
	if err != nil {
		return
	}
	f, err := os.OpenFile(debugLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(line, '\n'))
}

// #endregion

package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var backendErrorLogMu sync.Mutex

// LogBackendError 将后端接口和外部服务错误写入结构化 JSONL 日志。
func LogBackendError(source, method, path string, status int, detail string) {
	logDir := filepath.Join("logs", "errors")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return
	}

	entry := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339Nano),
		"source":    source,
		"method":    method,
		"path":      path,
		"status":    status,
		"detail":    detail,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	backendErrorLogMu.Lock()
	defer backendErrorLogMu.Unlock()
	file, err := os.OpenFile(filepath.Join(logDir, time.Now().Format("2006-01-02")+".jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.Write(append(data, '\n'))
}

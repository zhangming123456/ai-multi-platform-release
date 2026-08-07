package utils

import (
	"encoding/json"
	"net/http"
)

type ResponseData struct {
	Status   int         `json:"status"`
	Data     interface{} `json:"data,omitempty"`
	Items    interface{} `json:"items,omitempty"`
	Message  string      `json:"message,omitempty"`
	Detail   interface{} `json:"detail,omitempty"`
	Total    int64       `json:"total,omitempty"`
	Page     int         `json:"page,omitempty"`
	PageSize int         `json:"page_size,omitempty"`
}

func WriteJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	if payload == nil {
		return
	}
	data, err := json.Marshal(payload)
	if err != nil {
		w.Write([]byte(`{"detail":"序列化失败"}`))
		return
	}
	w.Write(data)
}

func WriteError(w http.ResponseWriter, statusCode int, detail string) {
	WriteJSON(w, statusCode, map[string]interface{}{
		"detail": detail,
	})
}

func WriteOK(w http.ResponseWriter, data interface{}) {
	WriteJSON(w, http.StatusOK, data)
}

func WriteCreated(w http.ResponseWriter, data interface{}) {
	WriteJSON(w, http.StatusCreated, data)
}

func WriteNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func WritePaginated(w http.ResponseWriter, items interface{}, total int64, page, pageSize int) {
	WriteJSON(w, http.StatusOK, ResponseData{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

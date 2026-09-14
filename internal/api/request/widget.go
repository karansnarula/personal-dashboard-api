package request

import "encoding/json"

type CreateWidget struct {
	Type   string          `json:"type" binding:"required"`
	Config json.RawMessage `json:"config" binding:"required"`
}

type UpdateWidget struct {
	Config json.RawMessage `json:"config" binding:"required"`
}

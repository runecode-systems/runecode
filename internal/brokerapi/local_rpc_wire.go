package brokerapi

import "encoding/json"

type LocalRPCRequest struct {
	Operation string          `json:"operation"`
	Request   json.RawMessage `json:"request"`
}

type LocalRPCResponse struct {
	OK       bool            `json:"ok"`
	Response json.RawMessage `json:"response,omitempty"`
	Error    *ErrorResponse  `json:"error,omitempty"`
}

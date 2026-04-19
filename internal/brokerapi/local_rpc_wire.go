package brokerapi

import "encoding/json"

type LocalRPCRequest struct {
	Operation                  string          `json:"operation"`
	Request                    json.RawMessage `json:"request"`
	SecretIngressPayloadBase64 string          `json:"secret_ingress_payload_base64,omitempty"`
}

type LocalRPCResponse struct {
	OK       bool            `json:"ok"`
	Response json.RawMessage `json:"response,omitempty"`
	Error    *ErrorResponse  `json:"error,omitempty"`
}

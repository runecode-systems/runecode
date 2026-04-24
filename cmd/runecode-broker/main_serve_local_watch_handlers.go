package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/runecode-ai/runecode/internal/brokerapi"
)

func decodeAndHandleRunWatch(service *brokerapi.Service, ctx context.Context, raw json.RawMessage, meta brokerapi.RequestContext) localRPCResponse {
	req := brokerapi.RunWatchRequest{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return localRPCResponse{OK: false, Error: decodeWireError("", err)}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			err = errors.New("request must contain exactly one JSON object")
		}
		return localRPCResponse{OK: false, Error: decodeWireError("", err)}
	}
	ack, errResp := service.HandleRunWatchRequest(ctx, req, meta)
	if errResp != nil {
		return localRPCResponse{OK: false, Error: errResp}
	}
	events, err := service.StreamRunWatchEvents(ack)
	if err != nil {
		return localRPCResponse{OK: false, Error: decodeWireError(ack.RequestID, err)}
	}
	return localRPCOKResponse(events)
}

func decodeAndHandleApprovalWatch(service *brokerapi.Service, ctx context.Context, raw json.RawMessage, meta brokerapi.RequestContext) localRPCResponse {
	req := brokerapi.ApprovalWatchRequest{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return localRPCResponse{OK: false, Error: decodeWireError("", err)}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			err = errors.New("request must contain exactly one JSON object")
		}
		return localRPCResponse{OK: false, Error: decodeWireError("", err)}
	}
	ack, errResp := service.HandleApprovalWatchRequest(ctx, req, meta)
	if errResp != nil {
		return localRPCResponse{OK: false, Error: errResp}
	}
	events, err := service.StreamApprovalWatchEvents(ack)
	if err != nil {
		return localRPCResponse{OK: false, Error: decodeWireError(ack.RequestID, err)}
	}
	return localRPCOKResponse(events)
}

func decodeAndHandleSessionWatch(service *brokerapi.Service, ctx context.Context, raw json.RawMessage, meta brokerapi.RequestContext) localRPCResponse {
	req := brokerapi.SessionWatchRequest{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return localRPCResponse{OK: false, Error: decodeWireError("", err)}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			err = errors.New("request must contain exactly one JSON object")
		}
		return localRPCResponse{OK: false, Error: decodeWireError("", err)}
	}
	ack, errResp := service.HandleSessionWatchRequest(ctx, req, meta)
	if errResp != nil {
		return localRPCResponse{OK: false, Error: errResp}
	}
	events, err := service.StreamSessionWatchEvents(ack)
	if err != nil {
		return localRPCResponse{OK: false, Error: decodeWireError(ack.RequestID, err)}
	}
	return localRPCOKResponse(events)
}

func decodeAndHandleSessionTurnExecutionWatch(service *brokerapi.Service, ctx context.Context, raw json.RawMessage, meta brokerapi.RequestContext) localRPCResponse {
	req := brokerapi.SessionTurnExecutionWatchRequest{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return localRPCResponse{OK: false, Error: decodeWireError("", err)}
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			err = errors.New("request must contain exactly one JSON object")
		}
		return localRPCResponse{OK: false, Error: decodeWireError("", err)}
	}
	ack, errResp := service.HandleSessionTurnExecutionWatchRequest(ctx, req, meta)
	if errResp != nil {
		return localRPCResponse{OK: false, Error: errResp}
	}
	events, err := service.StreamSessionTurnExecutionWatchEvents(ack)
	if err != nil {
		return localRPCResponse{OK: false, Error: decodeWireError(ack.RequestID, err)}
	}
	return localRPCOKResponse(events)
}

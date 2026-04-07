package brokerapi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

type pageCursor struct {
	Offset int `json:"offset"`
}

func decodeCursor(raw string) (pageCursor, error) {
	if strings.TrimSpace(raw) == "" {
		return pageCursor{}, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return pageCursor{}, fmt.Errorf("decode cursor: %w", err)
	}
	var c pageCursor
	if err := json.Unmarshal(b, &c); err != nil {
		return pageCursor{}, fmt.Errorf("decode cursor payload: %w", err)
	}
	if c.Offset < 0 {
		return pageCursor{}, fmt.Errorf("cursor offset must be >= 0")
	}
	return c, nil
}

func encodeCursor(c pageCursor) (string, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func normalizeLimit(limit int, fallback int, max int) int {
	if limit <= 0 {
		limit = fallback
	}
	if limit > max {
		limit = max
	}
	return limit
}

func paginate[T any](items []T, cursor string, limit int) ([]T, string, error) {
	c, err := decodeCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	if c.Offset >= len(items) {
		return []T{}, "", nil
	}
	end := c.Offset + limit
	if end > len(items) {
		end = len(items)
	}
	page := items[c.Offset:end]
	if end == len(items) {
		return page, "", nil
	}
	next, err := encodeCursor(pageCursor{Offset: end})
	if err != nil {
		return nil, "", err
	}
	return page, next, nil
}

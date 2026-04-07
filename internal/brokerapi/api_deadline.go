package brokerapi

import (
	"context"
	"time"
)

func withDefaultDeadline(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := parent.Deadline(); ok {
		return parent, func() {}
	}
	return context.WithTimeout(parent, timeout)
}

func withRequestDeadline(parent context.Context, meta RequestContext, fallback time.Duration) (context.Context, context.CancelFunc) {
	if meta.Deadline != nil {
		return context.WithDeadline(parent, *meta.Deadline)
	}
	return withDefaultDeadline(parent, fallback)
}

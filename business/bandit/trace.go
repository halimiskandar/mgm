package bandit

import "context"

type ctxKey string

const TraceIDKey ctxKey = "trace_id"

func TraceIDFromContext(ctx context.Context) string {
	if v := ctx.Value(TraceIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

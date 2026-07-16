package websocket

import "log/slog"

// TryPush sends ev on ch without blocking. Drops and logs when the buffer is full.
func TryPush[T any](ch chan<- T, ev T, log *slog.Logger, dropMsg string, attrs ...any) {
	if ch == nil {
		return
	}
	select {
	case ch <- ev:
	default:
		if log != nil {
			log.Warn(dropMsg, attrs...)
		}
	}
}

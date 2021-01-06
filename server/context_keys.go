package server

type contextKey string

const (
	// ListenerKey is used to add the Listener configuration
	// that received a HTTP request to the request context.
	ListenerKey = contextKey("http:listener")
)

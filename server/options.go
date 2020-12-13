package server

import "github.com/tierklinik-dobersberg/logger"

// Option can be passed to a server. When called only the
// *gin.Engine of the server is ensured to be set.
type Option func(s *Server) error

// WithLogger configures the server request logger.
func WithLogger(l logger.Logger) Option {
	return func(s *Server) error {
		s.logger = l
		return nil
	}
}

// WithPreHandler configures an additional pre-handler function
// for the server. May be called multiple times.
func WithPreHandler(fn PreHandlerFunc) Option {
	return func(s *Server) error {
		s.preHandler = append(s.preHandler, fn)
		return nil
	}
}

// WithListener configures one or more listeners for t
// the server.
func WithListener(l ...Listener) Option {
	return func(s *Server) error {
		s.listenCfgs = append(s.listenCfgs, l...)
		return nil
	}
}

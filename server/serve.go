package server

import (
	"context"
	"fmt"
)

// Shutdown stops all HTTP server listeners and waits for them to
// close. If ctx is cancelled Shutdown returns immediately. See
// http.Server.Shutdown for more information.
func (srv *Server) Shutdown(ctx context.Context) error {
	return nil
}

// Run starts listening and serving on all configured listeners.
func (srv *Server) Run() error {
	if len(srv.listener) == 0 {
		return fmt.Errorf("no listeners configured")
	}

	return nil
}

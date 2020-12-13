package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/ory/graceful"
	"golang.org/x/sync/errgroup"
)

// Shutdown stops all HTTP server listeners and waits for them to
// close. If ctx is cancelled Shutdown returns immediately. See
// http.Server.Shutdown for more information.
func (srv *Server) Shutdown(ctx context.Context) error {
	errGrp := new(errgroup.Group)

	for idx := range srv.servers {
		s := srv.servers[idx]
		errGrp.Go(func() error {
			return s.Shutdown(ctx)
		})
	}

	ch := make(chan error, 1)
	go func() {
		ch <- errGrp.Wait()
	}()

	select {
	case err := <-ch:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Run starts listening and serving on all configured listeners.
func (srv *Server) Run() error {
	if len(srv.listenCfgs) == 0 {
		return fmt.Errorf("no listeners configured")
	}

	srv.servers = make([]*http.Server, len(srv.listenCfgs))
	for idx, cfg := range srv.listenCfgs {
		s := graceful.WithDefaults(&http.Server{
			Handler: srv,
			Addr:    cfg.Address,
		})

		srv.servers[idx] = s
	}

	errGrp := new(errgroup.Group)

	for idx := range srv.servers {
		s := srv.servers[idx]
		cfg := srv.listenCfgs[idx]

		if cfg.TLSCertFile != "" {
			errGrp.Go(func() error {
				return s.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
			})
		} else {
			errGrp.Go(s.ListenAndServe)
		}
	}

	ch := make(chan error, 1)

	go func() {
		err := errGrp.Wait()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			srv.logger.Errorf("failed to listen: %s", err)
		}

		if err != nil {
			ch <- err
		}

		if err := srv.Shutdown(context.Background()); err != nil {
			srv.logger.Errorf("failed to shutdown server: %s", err)
		}
	}()

	return errGrp.Wait()
}

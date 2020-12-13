package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tierklinik-dobersberg/logger"
	"github.com/tierklinik-dobersberg/service/accesslog"
)

// PreHandlerFunc is called for each http request before the
// actual handler chain executes. It's mainly meant to support
// updating the request context.
type PreHandlerFunc func(*http.Request) *http.Request

// Server wraps a gin.Engine and adds some additional, service-context
// specific methods.
type Server struct {
	*gin.Engine

	preHandler []PreHandlerFunc
	logger     logger.Logger
	listenCfgs []Listener
	servers    []*http.Server
}

// New creates a new server instance.
func New(accessLogPath string, opts ...Option) (*Server, error) {
	srv := new(Server)
	srv.Engine = gin.Default() // TODO

	for _, opt := range opts {
		if err := opt(srv); err != nil {
			return nil, fmt.Errorf("failed to apply options: %w", err)
		}
	}

	if srv.logger == nil {
		srv.logger = logger.DefaultLogger()
	}

	if len(srv.listenCfgs) == 0 {
		return nil, fmt.Errorf("no listeners configured")
	}

	// We always use an access logger, either printing to accessLogPath
	// or to logger.DefaultLogger()
	srv.Engine.Use(accessLogger(accessLogPath))

	return srv, nil
}

func accessLogger(path string) gin.HandlerFunc {
	accessLogger := logger.DefaultLogger()
	if path != "" {
		adapter := logger.MultiAdapter(
			logger.DefaultAdapter(),
			&accesslog.FileWriter{
				Path:         path,
				ErrorAdapter: logger.DefaultAdapter(),
			},
		)
		accessLogger = logger.New(adapter)
	}

	return accesslog.New(accessLogger)
}

// ServeHTTP implements http.Handler and calls through to gin.Engine
// with the addition of setting up service specific things ...
func (srv *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// create a new request context that has a logger attached.
	ctx := req.Context()
	ctx = logger.With(ctx, srv.logger)

	// create a new request with the new context
	// and hand over to gin.Engine
	req = req.WithContext(ctx)

	// run all pre-handlers
	for _, fn := range srv.preHandler {
		req = fn(req)
	}

	srv.Engine.ServeHTTP(w, req)
}

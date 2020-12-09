package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ppacher/system-conf/conf"
	"github.com/tierklinik-dobersberg/logger"
	"github.com/tierklinik-dobersberg/service/accesslog"
)

// Server wraps a gin.Engine and adds some additional, service-context
// specific methods.
type Server struct {
	*gin.Engine
	listener []Listener
}

// New creates a new server instance.
func New(accessLogPath string, listeners []conf.Section) (*Server, error) {
	srv := new(Server)

	srv.Engine = gin.Default() // TODO

	for _, cfg := range listeners {
		l, err := BuildListener(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to build listener: %w", err)
		}
		srv.listener = append(srv.listener, l)
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
	// TODO(ppacher): add request specific logger here
	srv.Engine.ServeHTTP(w, req)
}

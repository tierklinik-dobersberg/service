package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ory/graceful"
	"github.com/ppacher/system-conf/conf"
	"github.com/tierklinik-dobersberg/logger"
	"github.com/tierklinik-dobersberg/service/server"
	"github.com/tierklinik-dobersberg/service/svcenv"
)

type contextKey string

const instanceContextKey = contextKey("service:instance")

type Instance struct {
	Config
	svcenv.ServiceEnv

	cfgFile    *conf.File
	srv        *server.Server
	logAdapter *logAdapter
}

// FromContext returns the service instance associated
// with ctx.
func FromContext(ctx context.Context) *Instance {
	inst, _ := ctx.Value(instanceContextKey).(*Instance)
	return inst
}

// Server returns the built-in HTTP server of the service
// instance. It may be nil if Config.DisableServer is set
// to true. Use Config.ServerOptions to specify additional
// option when creating the server.
func (inst *Instance) Server() *server.Server {
	return inst.srv
}

// ConfigFile returns the parsed conf.File content
// of the service configuration file.
func (inst *Instance) ConfigFile() *conf.File {
	return inst.cfgFile
}

// AddLogger adds adapter to the list of logging adapters
// used by inst. Note that messages with lower severity
// than the threshold set by SetLogLevel will be discarded
// and not passed to adapter.
func (inst *Instance) AddLogger(adapter logger.Adapter) {
	inst.logAdapter.addAdapter(adapter)
}

// SetLogLevel configures the maximum log level for the
// instance logger.
func (inst *Instance) SetLogLevel(s logger.Severity) {
	inst.logAdapter.setMaxSeverity(s)
}

// Serve starts serving the internal, built-in HTTP server.
// It blocks forever but listens for typical server interrupt
// signals like SIGINT and SIGTERM. In case of a signal the
// server is gracefully brought down while in-flight requests
// are allowed to finish.
func (inst *Instance) Serve() error {
	if inst.srv == nil {
		return fmt.Errorf("built-in HTTP server is disabled")
	}

	if err := graceful.Graceful(inst.srv.Run, inst.srv.Shutdown); err != nil {
		return fmt.Errorf("graceful: %w", err)
	}

	return nil
}

func (inst *Instance) serverOption() server.Option {
	return server.WithPreHandler(func(r *http.Request) *http.Request {
		newCtx := context.WithValue(r.Context(), instanceContextKey, inst)

		return r.Clone(newCtx)
	})
}

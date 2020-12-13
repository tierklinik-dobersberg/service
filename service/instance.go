package service

import (
	"fmt"

	"github.com/ory/graceful"
	"github.com/ppacher/system-conf/conf"
	"github.com/tierklinik-dobersberg/service/server"
)

type Instance struct {
	Config

	cfgFile *conf.File
	srv     *server.Server
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

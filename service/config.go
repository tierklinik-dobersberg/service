package service

import (
	"github.com/gin-gonic/gin"
	"github.com/ppacher/system-conf/conf"
	"github.com/tierklinik-dobersberg/logger"
	"github.com/tierklinik-dobersberg/service/server"
)

// Config describes the overall configuration and setup required
// to boot the system service.
type Config struct {
	// AccessLogPath is the path to the access log of the
	// built-in HTTP server. If defined as "AccessLogPath"
	// by ConfigFileSpec, the access log may be overwritten
	// by the configuration file automatically.
	AccessLogPath string

	// ConfigFileName is the name of the configuration file.
	// The extension of the configuration file defaults
	// to .conf.
	ConfigFileName string

	// ConfigFileSpec describes the allowed sections and
	// values of the configuration file. Note that if
	// DisableServer is not set the file spec is extended
	// to include Listener sections for the built-in HTTP(s)
	// server.
	// If no [Listener] section is defined and the built-in
	// HTTP server is enabled a default listener for
	// 127.0.0.1:3000 is created.
	ConfigFileSpec conf.FileSpec

	// ConfigTarget may holds the struct that should be
	// used to unmarshal the configuration file into.
	ConfigTarget interface{}

	// DisableServer disables the built-in HTTP(s) server.
	// If set, calls to Server() will return nil!
	DisableServer bool

	// ServerOptions may hold additional options for the
	// built-in HTTP server. ServerOptions is ignored when
	// DisableServer is set.
	ServerOptions []server.Option

	// RouteSetup configures may be used to configure
	// the available HTTP routes. It's also possible
	// to configure routes after Boot() by adding them
	// to the Instance.Server() directly.
	RouteSetupFunc func(grp gin.IRouter) error

	// LogAdapter configures the standard log adapter to use.
	LogAdapter logger.Adapter
}

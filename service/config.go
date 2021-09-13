package service

import (
	"strings"

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
	// by ConfigSchema, the access log may be overwritten
	// by the configuration file automatically.
	AccessLogPath string

	// ConfigFileName is the name of the configuration file.
	// The extension of the configuration file defaults
	// to .conf.
	ConfigFileName string

	// ConfigDirectory is the name of the directory that may contain
	// separate configuration files.
	ConfigDirectory string

	// UseStdlibLogAdapter can be set to true to immediately add a new
	// logger.StandardAdapter to the service logger.
	UseStdlibLogAdapter bool

	// LogLevel can be set to the default log level. This can later be
	// overwritten by using instance.SetLogLevel().
	LogLevel logger.Severity

	// ConfigSchema describes the allowed sections and
	// values of the configuration file. Note that if
	// DisableServer is not set the schema is extended
	// to include Listener sections for the built-in HTTP(s)
	// server.
	// If no [Listener] section is defined and the built-in
	// HTTP server is enabled a default listener for
	// 127.0.0.1:3000 is created.
	ConfigSchema conf.SectionRegistry

	// ConfigTarget may holds the struct that should be
	// used to unmarshal the configuration file into.
	ConfigTarget interface{}

	// DisableServer disables the built-in HTTP(s) server.
	// If set, calls to Server() will return nil!
	DisableServer bool

	// DisableCORS disables automatic support for CORS
	// configuration using the global configuration file.
	DisableCORS bool

	// ServerOptions may hold additional options for the
	// built-in HTTP server. ServerOptions is ignored when
	// DisableServer is set.
	ServerOptions []server.Option

	// RouteSetup configures may be used to configure
	// the available HTTP routes. It's also possible
	// to configure routes after Boot() by adding them
	// to the Instance.Server() directly.
	RouteSetupFunc func(grp gin.IRouter) error
}

func (cfg *Config) OptionsForSection(secName string) (conf.OptionRegistry, bool) {
	lowerName := strings.ToLower(secName)
	if !cfg.DisableServer {
		if lowerName == "listener" {
			return server.ListenerSpec, true
		}

		if !cfg.DisableCORS && lowerName == "cors" {
			return server.CORSSpec, true
		}
	}
	if cfg.ConfigSchema != nil {
		return cfg.ConfigSchema.OptionsForSection(secName)
	}
	return nil, false
}

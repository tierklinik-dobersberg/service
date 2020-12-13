package service

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/ppacher/system-conf/conf"
	"github.com/tierklinik-dobersberg/logger"
	"github.com/tierklinik-dobersberg/service/server"
	"github.com/tierklinik-dobersberg/service/svcenv"
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

// Boot boots the service and returns the service
// instance.
func Boot(cfg Config) (*Instance, error) {
	// setup logging
	// TODO(ppacher): use a logger.MultiAdapter as
	// a wrapper to easy extending of the logging
	// behavior.
	if cfg.LogAdapter == nil {
		cfg.LogAdapter = new(logger.StdlibAdapter)
	}
	logger.SetDefaultAdapter(cfg.LogAdapter)

	// load the service environment
	env := svcenv.Env()

	// load the configuration file
	cfgFile, err := loadConfigFile(env, &cfg)
	if err != nil {
		return nil, fmt.Errorf("configuration: %w", err)
	}

	// If there's a receiver target for the configuration
	// directly decode it there.
	if cfg.ConfigTarget != nil {
		if err := cfg.ConfigFileSpec.Decode(cfgFile, cfg.ConfigTarget); err != nil {
			return nil, fmt.Errorf("failed to decode config: %w", err)
		}
	}

	// setup the built-in HTTP server
	var srv *server.Server
	if !cfg.DisableServer {
		// parse the [Listener] sections from the configuration
		// file.
		var file struct {
			Listeners []server.Listener `section:"Listener"`
		}
		spec := getFileSpec(&cfg)
		if err := spec.Decode(cfgFile, &file); err != nil {
			return nil, fmt.Errorf("failed to parse listeners: %w", err)
		}

		// If there's no listener section make sure to add the dev-version:
		if len(file.Listeners) == 0 {
			logger.DefaultLogger().Info("no listeners configured, using http://127.0.0.1:3000")
			file.Listeners = []server.Listener{
				{
					Address: "127.0.0.1:3000",
				},
			}
		}

		// If there's an AccessLogPath option in the file spec
		// allow it to overwrite the default access-log:
		if glob, ok := cfg.ConfigFileSpec.FindSection("Global"); ok && glob.HasOption("AccessLogPath") {
			if globSection := cfgFile.Get("Global"); globSection != nil {
				opt, _ := globSection.GetString("AccessLogPath")
				if opt != "" {
					cfg.AccessLogPath = opt
				}
			}
		}

		// prepare the actual HTTP server ...
		srv, err = server.New(cfg.AccessLogPath,
			append(
				[]server.Option{
					server.WithListener(file.Listeners...),
					server.WithLogger(logger.DefaultLogger()),
				},
				cfg.ServerOptions...,
			)...,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare built-in HTTP server: %w", err)
		}

		// any create any routes by using the RouteSetupFunc if it
		// was provided by the user.
		if cfg.RouteSetupFunc != nil {
			if err := cfg.RouteSetupFunc(srv); err != nil {
				return nil, fmt.Errorf("route setup failed: %w", err)
			}
		}
	}

	inst := &Instance{
		Config:  cfg,
		cfgFile: cfgFile,
		srv:     srv,
	}

	return inst, nil
}

func loadConfigFile(env svcenv.ServiceEnv, cfg *Config) (*conf.File, error) {
	// The configuration file is either located in env.ConfigurationDirectory
	// or in the current working-directory of the service.
	// TODO(ppacher): add support to disable the WD fallback.

	dir := env.ConfigurationDirectory
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}
	fpath := filepath.Join(env.ConfigurationDirectory, cfg.ConfigFileName)

	// if cfg.ConfigFileName does not include an extension
	// we default to .conf.
	// TODO(ppacher): should we make this behavior configurable?
	if filepath.Ext(fpath) == "" {
		fpath = fpath + ".conf"
	}

	// open the configuration file
	file, err := os.Open(fpath)
	if err != nil {
		return nil, fmt.Errorf("failed to open: %w", err)
	}
	defer file.Close()

	// finally deserialize it and convert it into a
	// conf.File. Actual decoding of confFile into
	// a struct type happens later using
	// (conf.FileSpec).Decode.
	confFile, err := conf.Deserialize(fpath, file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse: %w", err)
	}

	// get the complete configuration file spec.
	// Depending on (config).DisableServer, this may also
	// include options for the built-in HTTP server.
	spec := getFileSpec(cfg)

	// Validate the configuration file, set defaults and ensure
	// everything is ready to be parsed.
	if err := conf.ValidateFile(confFile, spec); err != nil {
		return nil, fmt.Errorf("invalid config file: %w", err)
	}

	return confFile, nil
}

func getFileSpec(cfg *Config) conf.FileSpec {
	fs := make(conf.FileSpec)

	for k, v := range cfg.ConfigFileSpec {
		fs[k] = v
	}

	if !cfg.DisableServer {
		fs["Listener"] = server.ListenerSpec
	}

	return fs
}

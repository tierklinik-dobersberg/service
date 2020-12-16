package service

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ppacher/system-conf/conf"
	"github.com/tierklinik-dobersberg/logger"
	"github.com/tierklinik-dobersberg/service/server"
	"github.com/tierklinik-dobersberg/service/svcenv"
)

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

	inst := &Instance{
		Config:     cfg,
		ServiceEnv: env,
		cfgFile:    cfgFile,
	}

	// prepare the built-in HTTP server
	srv, err := prepareHTTPServer(&cfg, inst)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare HTTP server: %w", err)
	}
	inst.srv = srv

	return inst, nil
}

func prepareHTTPServer(cfg *Config, inst *Instance) (*server.Server, error) {
	if cfg.DisableServer {
		return nil, nil
	}

	// parse the [Listener] sections from the configuration
	// file.
	var file struct {
		Listeners []server.Listener `section:"Listener"`
		CORS      *server.CORS      `section:"CORS"`
	}

	// Prepare default values for cors
	if !cfg.DisableCORS {
		c := server.CORS{
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"},
			AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type"},
			AllowCredentials: false,
			MaxAge:           "12h",
		}

		file.CORS = (*server.CORS)(&c)
	}

	spec := getFileSpec(cfg)
	if err := spec.Decode(inst.cfgFile, &file); err != nil {
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
		if globSection := inst.cfgFile.Get("Global"); globSection != nil {
			opt, _ := globSection.GetString("AccessLogPath")
			if opt != "" {
				cfg.AccessLogPath = opt
			}
		}
	}

	options := []server.Option{
		server.WithListener(file.Listeners...),
		server.WithLogger(logger.DefaultLogger()),
		inst.serverOption(),
	}
	options = append(options, cfg.ServerOptions...)

	// prepare the actual HTTP server ...
	srv, err := server.New(cfg.AccessLogPath, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare built-in HTTP server: %w", err)
	}

	// Enable the CORS middleware
	if !cfg.DisableCORS {
		srv.Use(server.EnableCORS(*file.CORS))
	}

	// any create any routes by using the RouteSetupFunc if it
	// was provided by the user.
	if cfg.RouteSetupFunc != nil {
		if err := cfg.RouteSetupFunc(srv); err != nil {
			return nil, fmt.Errorf("route setup failed: %w", err)
		}
	}

	return srv, nil
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

		if !cfg.DisableCORS {
			fs["CORS"] = server.CORSSpec
		}
	}

	return fs
}

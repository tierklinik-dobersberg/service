package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ppacher/system-conf/conf"
	"github.com/tierklinik-dobersberg/logger"
	"github.com/tierklinik-dobersberg/service/server"
	"github.com/tierklinik-dobersberg/service/svcenv"
)

// Boot boots the service and returns the service
// instance.
func Boot(cfg Config) (*Instance, error) {
	// setup logging
	log := new(logAdapter)
	log.setMaxSeverity(cfg.LogLevel)
	if cfg.UseStdlibLogAdapter {
		log.addAdapter(new(logger.StdlibAdapter))
	}

	logger.SetDefaultAdapter(log)

	// load the service environment
	env := svcenv.Env()

	// load the configuration file
	cfgFile, err := loadConfig(env, &cfg)
	if err != nil {
		return nil, fmt.Errorf("configuration: %w", err)
	}

	// If there's a receiver target for the configuration
	// directly decode it there.
	if cfg.ConfigTarget != nil {
		// we use ConfigSchema directly instead of cfg so we don't try do
		// decode CORS or Listener sections.
		// TODO(ppacher): consider changing this behavior.
		if err := conf.DecodeFile(cfgFile, cfg.ConfigTarget, cfg.ConfigSchema); err != nil {
			return nil, fmt.Errorf("failed to decode config: %w", err)
		}
	}

	inst := &Instance{
		Config:     cfg,
		ServiceEnv: env,
		cfgFile:    cfgFile,
		logAdapter: log,
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

	if err := conf.DecodeFile(inst.cfgFile, &file, cfg); err != nil {
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
	if glob, ok := cfg.ConfigSchema.OptionsForSection("Global"); ok && glob.HasOption("AccessLogPath") {
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

	// create any routes by using the RouteSetupFunc if it
	// was provided by the user.
	if cfg.RouteSetupFunc != nil {
		if err := cfg.RouteSetupFunc(srv); err != nil {
			return nil, fmt.Errorf("route setup failed: %w", err)
		}
	}

	return srv, nil
}

// newLineReader returns an io.Reader that just emits a new line.
// This is required as not all .conf files in the ConfigurationDirectory
// may end in a new line and we need to avoid merging lines from different
// files.
func newLineReader() io.Reader {
	return strings.NewReader("\n")
}

func loadConfig(env svcenv.ServiceEnv, cfg *Config) (*conf.File, error) {
	// The configuration file is either located in env.ConfigurationDirectory
	// or in the current working-directory of the service.
	// TODO(ppacher): add support to disable the WD fallback.
	log := logger.From(context.TODO())

	dir := env.ConfigurationDirectory
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}
	log.V(5).Logf("configuration directory: %s", dir)

	// a list of io.Readers for each configuration file.
	var configurations []io.Reader
	fpath := filepath.Join(dir, cfg.ConfigFileName)
	if cfg.ConfigFileName != "" {
		// if cfg.ConfigFileName does not include an extension
		// we default to .conf.
		if filepath.Ext(fpath) == "" {
			fpath = fpath + ".conf"
		}
		// TODO(ppacher): should the existance of the main configuration
		// file be optional?
		log.V(5).Logf("trying to load main config file from: %s", fpath)
		mainFile, err := os.Open(fpath)
		if err != nil {
			return nil, fmt.Errorf("failed to open: %w", err)
		}
		defer mainFile.Close()
		configurations = append(
			configurations,
			mainFile,
			newLineReader(),
		)
	}

	// open all .conf files in the configuration-directory.
	if confd := cfg.ConfigDirectory; confd != "" {
		// TODO(ppacher): should we check if that directory actually
		// exists?
		if !filepath.IsAbs(confd) {
			confd = filepath.Join(dir, confd)
		}
		log.V(5).Logf("searching for additional .conf files in: %s", confd)
		matches, err := filepath.Glob(filepath.Join(confd, "*.conf"))
		if err != nil {
			// err can only be filepath.ErrBadPattern which we should never
			// see here. so time to panic
			panic(err)
		}

		sort.Strings(matches)
		for _, file := range matches {
			logger.From(context.TODO()).V(5).Logf("found configuration file: %s", file)
			f, err := os.Open(file)
			if err != nil {
				logger.Errorf(context.TODO(), "failed to open %s: %s, skipping", file, err)
				continue
			}
			defer f.Close()
			configurations = append(
				configurations,
				f,
				newLineReader(),
			)
		}
	} else {
		log.V(5).Logf("no conf.d directory configured, not scanning for additional files ...")
	}

	log.V(5).Logf("loading service configuration from %d sources", int(len(configurations)/2))

	// finally deserialize all configuration files and convert it into a
	// conf.File. Actual decoding of confFile into a struct type happens
	// later.
	confFile, err := conf.Deserialize(fpath, io.MultiReader(configurations...))
	if err != nil {
		return nil, fmt.Errorf("failed to parse: %w", err)
	}

	// Validate the configuration file, set defaults and ensure
	// everything is ready to be parsed.
	if err := conf.ValidateFile(confFile, cfg); err != nil {
		return nil, fmt.Errorf("invalid config file: %w", err)
	}

	return confFile, nil
}

package config

import (
	"log"
	"log/slog"
	"os"

	"github.com/vitistack/gslb-operator/pkg/bslog"
	"github.com/vitistack/gslb-operator/pkg/loaders"
)

var cfg *Config

func init() {
	var err error
	cfg, err = newConfig()
	if err != nil {
		log.Fatalf("unable to load config: %s", err.Error())
	}

	var handler slog.Handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:       slog.LevelDebug,
		ReplaceAttr: bslog.BaseReplaceAttr,
	})

	switch cfg.server.ENV {
	case "dev", "development", "DEV", "DEVELOPMENT":
		handler = bslog.NewHandler(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level:       slog.LevelDebug,
				ReplaceAttr: bslog.BaseReplaceAttr,
			}),
			bslog.InDevMode(),
		)
	case "prod", "production", "PROD", "PRODUCTION":
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level:       slog.LevelInfo,
			ReplaceAttr: bslog.BaseReplaceAttr,
		})
	}

	slog.SetDefault(slog.New(handler))
}

type Config struct {
	server Server
	api    API
	jwt    JWT
}

func GetInstance() *Config {
	if cfg == nil {
		cfg, _ = newConfig()
	}
	return cfg
}

func (c *Config) Server() *Server {
	return &c.server
}

func (c *Config) API() *API {
	return &c.api
}

func (c Config) JWT() *JWT {
	return &c.jwt
}

// Server configuration
type Server struct {
	ENV             string `env:"SRV_ENV" flag:"env"`
	DNSDIST_SERVERS string `env:"SRV_DNSDIST_SERVERS"`
}

func (s *Server) Env() string {
	return s.ENV
}

func (s *Server) DNSDistServers() string {
	return s.DNSDIST_SERVERS
}

// API configuration
type API struct {
	PORT             string `env:"API_PORT" flag:"port"`
	UPSTREAMGSLBHOST string `env:"API_UPSTREAM_GSLB_HOST"`
}

func (a *API) Port() string {
	return a.PORT
}

func (a *API) UpstreamGSLBHost() string {
	return a.UPSTREAMGSLBHOST
}

type JWT struct {
	SECRET string `env:"JWT_SECRET"`
	USER   string `env:"JWT_USER"`
}

func (jwt *JWT) Secret() []byte {
	return []byte(jwt.SECRET)
}

func (jwt *JWT) User() string {
	return jwt.USER
}

func newConfig() (*Config, error) {
	loader := loaders.NewChainLoader(
		loaders.NewEnvloader(),
		loaders.NewFileLoader(".env"),
		loaders.NewFlagLoader(),
	)

	// creating default config variables where possible
	serverCfg := Server{
		ENV: "prod",
	}
	apiCfg := API{
		PORT: ":8080",
	}
	jwtcfg := JWT{}

	configs := []any{
		&serverCfg,
		&apiCfg,
		&jwtcfg,
	}

	for _, cfg := range configs {
		err := loader.Load(cfg)
		if err != nil {
			return nil, err
		}
	}

	return &Config{
		server: serverCfg,
		api:    apiCfg,
		jwt:    jwtcfg,
	}, nil
}

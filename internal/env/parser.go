package env

import (
	"github.com/caarlos0/env/v7"
	"github.com/mainflux/mainflux/internal/clients/grpc"
	"github.com/mainflux/mainflux/internal/server"
)

type Options struct {
	// Environment keys and values that will be accessible for the service
	Environment map[string]string

	// TagName specifies another tagname to use rather than the default env
	TagName string

	// RequiredIfNoDef automatically sets all env as required if they do not declare 'envDefault'
	RequiredIfNoDef bool

	// OnSet allows to run a function when a value is set
	OnSet env.OnSetFn

	// Prefix define a prefix for each key
	Prefix string

	// AltPrefix define a alternate prefix for each key
	AltPrefix string
}

func Parse(v interface{}, opts ...Options) error {
	actOpt := []env.Options{}
	altPrefix := ""

	for _, opt := range opts {
		actOpt = append(actOpt, env.Options{
			Environment:     opt.Environment,
			TagName:         opt.TagName,
			RequiredIfNoDef: opt.RequiredIfNoDef,
			OnSet:           opt.OnSet,
			Prefix:          opt.Prefix,
		})
		if opt.AltPrefix != "" {
			altPrefix = opt.AltPrefix
		}
	}

	if altPrefix == "" {
		return env.Parse(v, actOpt...)
	}

	switch cfg := v.(type) {
	case *grpc.Config:
		return parseGrpcConfig(cfg, altPrefix, actOpt...)
	case *server.Config:
		return parseServerConfig(cfg, altPrefix, actOpt...)
	default:
		return env.Parse(v, actOpt...)
	}
}

func parseGrpcConfig(cfg *grpc.Config, altPrefix string, opts ...env.Options) error {
	if err := env.Parse(cfg, opts...); err != nil {
		return err
	}

	if !cfg.ClientTLS || cfg.CACerts == "" {
		altOpts := []env.Options{}
		for _, opt := range opts {
			if opt.Prefix != "" {
				opt.Prefix = altPrefix
			}
			altOpts = append(altOpts, opt)
		}
		altCfg := grpc.Config{}
		if err := env.Parse(&altCfg, altOpts...); err != nil {
			return err
		}
		if cfg.CACerts == "" && altCfg.CACerts != "" {
			cfg.CACerts = altCfg.CACerts
		}
		if !cfg.ClientTLS && altCfg.ClientTLS {
			cfg.ClientTLS = altCfg.ClientTLS
		}
	}

	return nil
}

func parseServerConfig(cfg *server.Config, altPrefix string, opts ...env.Options) error {
	copyConfig := cfg
	if err := env.Parse(cfg, opts...); err != nil {
		return err
	}

	if cfg.CertFile == "" || cfg.KeyFile == "" || cfg.Port == "" || cfg.Port == copyConfig.Port {
		altOpts := []env.Options{}
		for _, opt := range opts {
			if opt.Prefix != "" {
				opt.Prefix = altPrefix
			}
			altOpts = append(altOpts, opt)
		}
		altCfg := server.Config{}
		if err := env.Parse(&altCfg, altOpts...); err != nil {
			return err
		}
		if cfg.CertFile == "" && altCfg.CertFile != "" {
			cfg.CertFile = altCfg.CertFile
		}
		if cfg.KeyFile == "" && altCfg.KeyFile != "" {
			cfg.KeyFile = altCfg.KeyFile
		}
		if (cfg.Port == "" || cfg.Port == copyConfig.Port) && altCfg.Port != "" {
			cfg.Port = altCfg.Port
		}
	}
	return nil
}

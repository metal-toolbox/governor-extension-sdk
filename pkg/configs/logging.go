package configs

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Logging holds logging configuration
type Logging struct {
	Debug  bool `mapstructure:"debug"`
	Pretty bool `mapstructure:"pretty"`
}

// MustLoggingFlags registers logging related flags and binds them to viper
// Panics on error
func MustLoggingFlags(v *viper.Viper, flags *pflag.FlagSet) {
	flags.Bool("debug", false, "enable debug logging")
	viperBindFlag(v, "logging.debug", flags.Lookup("debug"))

	flags.Bool("pretty", false, "enable pretty (human readable) logging output")
	viperBindFlag(v, "logging.pretty", flags.Lookup("pretty"))
}

// Audit holds audit logging configuration
type Audit struct {
	LogPath string `mapstructure:"log-path"`
}

// MustAuditFlags registers Audit related flags and binds them to viper
// Panics on error
func MustAuditFlags(v *viper.Viper, flags *pflag.FlagSet) {
	flags.String("audit-log-path", "/app-audit/audit.log", "file path to write audit logs to.")
	viperBindFlag(v, "audit.log-path", flags.Lookup("audit-log-path"))
}

// Logger creates a zap.Logger based on the Logging configuration
func (lc Logging) Logger(appName string) *zap.Logger {
	cfg := zap.NewProductionConfig()

	if lc.Pretty {
		cfg = zap.NewDevelopmentConfig()
	}

	if lc.Debug {
		cfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	} else {
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	l, err := cfg.Build()
	if err != nil {
		panic(err)
	}

	logger := l.With(zap.String("app", appName))
	defer logger.Sync() //nolint:errcheck

	return logger
}

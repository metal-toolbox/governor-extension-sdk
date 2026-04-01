// Package configs contains configuration structures and utilities for the application.
package configs

import (
	"time"

	govcfg "github.com/metal-toolbox/governor-api/pkg/configs"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	// DefaultIAMRuntimeTimeoutSeconds is the default timeout for IAM runtime operations in seconds
	DefaultIAMRuntimeTimeoutSeconds = 15
	// DefaultNATSBucketTTL is the default time-to-live duration for NATS Key-Value bucket entries
	DefaultNATSBucketTTL = 5 * time.Minute
)

// AppConfig holds the application configuration
var AppConfig struct {
	govcfg.Configs `mapstructure:",squash"`

	DryRun   bool `mapstructure:"dryrun"`
	Audit    Audit
	Tracing  Tracing
	Logging  Logging
	Governor Governor
	Server   Server
	NATS     NATSConfig
}

// Server holds server configuration
type Server struct {
	Listen string `mapstructure:"listen"`
}

// MustServerFlags registers Server related flags and binds them to viper
// Panics on error
func MustServerFlags(v *viper.Viper, flags *pflag.FlagSet) {
	flags.String("listen", "0.0.0.0:8000", "address and port to listen on")
	viperBindFlag(v, "server.listen", flags.Lookup("listen"))
}

// viperBindFlag provides a wrapper around the viper bindings that handles error checks
func viperBindFlag(v *viper.Viper, name string, flag *pflag.Flag) {
	if err := v.BindPFlag(name, flag); err != nil {
		panic(err)
	}
}

package configs

import (
	govcfg "github.com/metal-toolbox/governor-api/pkg/configs"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// NATSConfig holds NATS configuration
type NATSConfig struct {
	govcfg.NATSConfig `mapstructure:",squash"`
	QueueGroup        string `mapstructure:"queue-group"`
	QueueSize         int    `mapstructure:"queue-size"`
}

// MustNATSFlags registers NATS related flags and binds them to viper
// Panics on error
func MustNATSFlags(v *viper.Viper, flags *pflag.FlagSet) {
	govcfg.AddFlags(flags)
	flags.String("nats-queue-group", "equinixmetal.governor.extensions.gov-ldap-addon", "queue group for load balancing messages across NATS consumers")
	viperBindFlag(v, "nats.queue-group", flags.Lookup("nats-queue-group"))
	flags.Int("nats-queue-size", 3, "queue size for load balancing messages across NATS consumers") //nolint: mnd
	viperBindFlag(v, "nats.queue-size", flags.Lookup("nats-queue-size"))
}

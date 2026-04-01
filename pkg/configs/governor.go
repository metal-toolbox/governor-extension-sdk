package configs

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Governor holds Governor API configuration
type Governor struct {
	URL          string   `mapstructure:"url"`
	ClientID     string   `mapstructure:"client-id"`
	ClientSecret string   `mapstructure:"client-secret"`
	TokenURL     string   `mapstructure:"token-url"`
	Audience     string   `mapstructure:"audience"`
	Scopes       []string `mapstructure:"scopes"`
	ExtensionID  string   `mapstructure:"extension-id"`
	ERDsPath     string   `mapstructure:"erds-path"`
}

// MustGovernorFlags registers Governor related flags and binds them to viper
// Panics on error
func MustGovernorFlags(v *viper.Viper, flags *pflag.FlagSet) {
	flags.String("governor-url", "https://api.iam.equinixmetal.net", "url of the governor api")
	viperBindFlag(v, "governor.url", flags.Lookup("governor-url"))
	flags.String("governor-client-id", "", "oauth client ID for client credentials flow")
	viperBindFlag(v, "governor.client-id", flags.Lookup("governor-client-id"))
	flags.String("governor-client-secret", "", "oauth client secret for client credentials flow")
	viperBindFlag(v, "governor.client-secret", flags.Lookup("governor-client-secret"))
	flags.String("governor-token-url", "http://hydra:4444/oauth2/token", "url used for client credential flow")
	viperBindFlag(v, "governor.token-url", flags.Lookup("governor-token-url"))
	flags.String("governor-audience", "http://api:3001/", "oauth audience for client credential flow")
	viperBindFlag(v, "governor.audience", flags.Lookup("governor-audience"))
	flags.StringSlice("governor-scopes", []string{"read:governor:groups", "read:governor:applications"}, "oauth scopes for the governor client credentials token")
	viperBindFlag(v, "governor.scopes", flags.Lookup("governor-scopes"))
	flags.String("governor-extension-id", "", "extension ID for the governor extension")
	viperBindFlag(v, "governor.extension-id", flags.Lookup("governor-extension-id"))
	flags.String("governor-erds-path", "/app/erds", "path to the ERDs for the governor extension")
	viperBindFlag(v, "governor.erds-path", flags.Lookup("governor-erds-path"))
}

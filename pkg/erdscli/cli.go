package erdscli

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	appname string
	erdpath string

	logger = zap.NewNop()

	erdsCmd = &cobra.Command{
		Use:   "erds",
		Short: "erds commands",
	}

	validateCmd = &cobra.Command{
		Use:   "validate",
		Short: "validate ERDs",
	}

	newERDCmd = &cobra.Command{
		Use:   "new",
		Short: "create a new ERD",
	}
)

// SetLogger sets the logger for ERDsCLI
func SetLogger(l *zap.Logger) {
	logger = l
}

// SetAppName sets the app name for ERDsCLI
func SetAppName(name string) {
	appname = name
}

// SetERDPath sets the path for ERDsCLI
func SetERDPath(path string) {
	erdpath = path
}

// RegisterCobraCommand registers the ERDsCLI to the parent command
func RegisterCobraCommand(root *cobra.Command, setupFunc func()) {
	validateCmd.RunE = func(_ *cobra.Command, _ []string) error {
		setupFunc()
		return validate()
	}

	newERDCmd.RunE = func(_ *cobra.Command, _ []string) error {
		setupFunc()
		return newERD()
	}

	erdsCmd.AddCommand(validateCmd)
	erdsCmd.AddCommand(newERDCmd)
	root.AddCommand(erdsCmd)

	newERDFlags()
}

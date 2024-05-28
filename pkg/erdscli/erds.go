package erdscli

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	governor "github.com/metal-toolbox/governor-api/pkg/api/v1alpha1"
	"github.com/metal-toolbox/governor-extension-sdk/pkg/erdvalidator"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const sampleSchema = `{
  "$id": "{{ .Version }}.{{ .SlugPlural }}.{{ .AppName }}.governor.equinixmetal.com",
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "name": {
      "default": "world",
      "description": "your name here",
      "type": "string"
    },
    "resp": { "description": "hello, name", "type": "string" }
  },
  "title": "{{ .ResourceName }}",
  "type": "object"
}
`

var schemaTemplate = template.Must(template.New("schema").Parse(sampleSchema))

func newERDFlags() {
	newERDCmd.Flags().String("filename", "", "filename of the new ERD, only .json, .yml and .yaml are supported")
	viperBindFlag("filename", newERDCmd.Flags().Lookup("filename"))
	newERDCmd.Flags().String("name", "", "name of the new ERD")
	viperBindFlag("name", newERDCmd.Flags().Lookup("name"))
	newERDCmd.Flags().String("slug-singular", "", "singular slug of the new ERD")
	viperBindFlag("slug-singular", newERDCmd.Flags().Lookup("slug-singular"))
	newERDCmd.Flags().String("slug-plural", "", "plural slug of the new ERD")
	viperBindFlag("slug-plural", newERDCmd.Flags().Lookup("slug-plural"))
	newERDCmd.Flags().String("version", "v1alpha1", "version of the new ERD")
	viperBindFlag("version", newERDCmd.Flags().Lookup("version"))
	newERDCmd.Flags().String("scope", "user", "scope of the new ERD")
	viperBindFlag("scope", newERDCmd.Flags().Lookup("scope"))
	newERDCmd.Flags().String("description", "some-description", "description of the new ERD")
	viperBindFlag("description", newERDCmd.Flags().Lookup("description"))
	newERDCmd.Flags().Bool("enabled", true, "enabled status of the new ERD")
	viperBindFlag("enabled", newERDCmd.Flags().Lookup("enabled"))
}

func newERD() error {
	if erdpath == "" {
		return fmt.Errorf("%w: erds-path", ErrValidatorMissingArgs)
	}

	fn := viper.GetString("filename")
	if fn == "" {
		return fmt.Errorf("%w: filename", ErrValidatorMissingArgs)
	}

	name := viper.GetString("name")
	if name == "" {
		return fmt.Errorf("%w: name", ErrValidatorMissingArgs)
	}

	slugSingular := viper.GetString("slug-singular")
	if slugSingular == "" {
		return fmt.Errorf("%w: slug-singular", ErrValidatorMissingArgs)
	}

	slugPlural := viper.GetString("slug-plural")
	if slugPlural == "" {
		return fmt.Errorf("%w: slug-plural", ErrValidatorMissingArgs)
	}

	version := viper.GetString("version")
	if version == "" {
		return fmt.Errorf("%w: version", ErrValidatorMissingArgs)
	}

	scope := viper.GetString("scope")
	if scope == "" {
		return fmt.Errorf("%w: scope", ErrValidatorMissingArgs)
	}

	description := viper.GetString("description")
	if description == "" {
		return fmt.Errorf("%w: description", ErrValidatorMissingArgs)
	}

	enabled := viper.GetBool("enabled")

	logger.Sugar().Infof("creating new ERD %s", fn)

	var templateOut bytes.Buffer

	if err := schemaTemplate.Execute(&templateOut, map[string]string{
		"Version":      version,
		"ResourceName": name,
		"SlugPlural":   slugPlural,
		"AppName":      appname,
	}); err != nil {
		return fmt.Errorf("%w: %s", ErrFailedCreateFile, err)
	}

	erd := &governor.ExtensionResourceDefinitionReq{
		Name:         name,
		SlugSingular: slugSingular,
		SlugPlural:   slugPlural,
		Version:      version,
		Scope:        governor.ExtensionResourceDefinitionScope(scope),
		Description:  description,
		Schema:       templateOut.Bytes(),
		Enabled:      &enabled,
	}

	fullpath := filepath.Join(erdpath, fn)

	if _, err := os.Stat(fullpath); err == nil {
		return fmt.Errorf("%w: %s already exists", ErrFailedCreateFile, fullpath)
	}

	v, _ := erdvalidator.NewValidator(erdvalidator.WithERD(erd))

	if err := v.Validate(); err != nil {
		logger.Error("failed to validate ERD", zap.Error(err))
		return nil
	}

	var contents erdvalidator.ERDContent

	out := []byte{}
	ext := filepath.Ext(fullpath)

	switch ext {
	case ".json":
		contents = (*erdvalidator.ERDContentJSON)(&out)
	case ".yaml", ".yml":
		contents = (*erdvalidator.ERDContentYAML)(&out)
	default:
		return fmt.Errorf("%w: %s is not a supported file", ErrFailedCreateFile, ext)
	}

	if err := contents.Marshal(erd); err != nil {
		return fmt.Errorf("%w: %s", ErrFailedCreateFile, err)
	}

	fmode := 0o644

	if err := os.WriteFile(fullpath, out, os.FileMode(fmode)); err != nil {
		return fmt.Errorf("%w: %s", ErrFailedCreateFile, err)
	}

	return nil
}

// viperBindFlag provides a wrapper around the viper bindings that handles error checks
func viperBindFlag(name string, flag *pflag.Flag) {
	if err := viper.BindPFlag(name, flag); err != nil {
		panic(err)
	}
}

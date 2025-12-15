package erdvalidator

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	governor "github.com/metal-toolbox/governor-api/pkg/api/v1alpha1"
	"github.com/metal-toolbox/governor-api/pkg/jsonschema"
)

// Validator is an ERD validator.
type Validator struct {
	erd *governor.ExtensionResourceDefinitionReq
}

// Option are options for creating a new Validator.
type Option func(*Validator) error

// NewValidator creates a new Validator.
func NewValidator(opts ...Option) (v *Validator, err error) {
	v = &Validator{
		erd: &governor.ExtensionResourceDefinitionReq{},
	}

	for _, opt := range opts {
		if err = opt(v); err != nil {
			return v, err
		}
	}

	return v, err
}

// WithERD sets the ERD to validate.
func WithERD(erd *governor.ExtensionResourceDefinitionReq) Option {
	return func(v *Validator) error {
		v.erd = erd
		return nil
	}
}

// WithERDContent sets the ERD content to validate.
func WithERDContent(content ERDContent) Option {
	return func(v *Validator) (err error) {
		v.erd, err = content.Unmarshal()
		if err != nil {
			return fmt.Errorf("%w: %s", ErrERDValidationFailed, err.Error())
		}

		return err
	}
}

// Validate validates an ERD, returning an error if it is invalid.
func (v *Validator) Validate() error {
	if v.erd == nil {
		return mkValidationErr("ERD is not set in validator")
	}

	if v.erd.Name == "" {
		return mkValidationErr("ERD name is required")
	}

	if v.erd.SlugSingular == "" || v.erd.SlugPlural == "" {
		return mkValidationErr("ERD slugs are required")
	}

	if !isValidSlug(v.erd.SlugSingular) || !isValidSlug(v.erd.SlugPlural) {
		return mkValidationErr("one or both of ERD slugs are invalid")
	}

	if v.erd.Version == "" {
		return mkValidationErr("ERD version is required")
	}

	if v.erd.Enabled == nil {
		return mkValidationErr("ERD enabled is required")
	}

	if v.erd.Scope == "" {
		return mkValidationErr("ERD scope is required")
	}

	if v.erd.Scope != governor.ExtensionResourceDefinitionScopeUser && v.erd.Scope != governor.ExtensionResourceDefinitionScopeSys {
		return mkValidationErr("invalid ERD scope, must be either system or user")
	}

	if string(v.erd.Schema) == "" {
		return mkValidationErr("ERD schema is required")
	}

	// user may choose to upload the schema as an escaped JSON string, here uses
	// a string unmarshal to "un-escape" the JSON string.
	var schema string
	if err := json.Unmarshal(v.erd.Schema, &schema); err != nil {
		// if the user upload the schema as an object, simply convert the bytes to
		// string should suffice
		schema = string(v.erd.Schema)
	}

	compiler := jsonschema.NewCompiler(
		"extension-validator", v.erd.SlugPlural, v.erd.Version,
		jsonschema.WithUniqueConstraint(
			context.Background(),
			nil, nil, nil,
		),
	)

	if _, err := compiler.Compile(schema); err != nil {
		return fmt.Errorf("%w: %s", ErrERDValidationFailed, err.Error())
	}

	return nil
}

func isValidSlug(s string) bool {
	// This regex ensures the slug is lowercase, uses hyphens to separate words,
	// does not start or end with a hyphen, and contains only alphanumeric characters or hyphens.
	pattern := regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	return pattern.MatchString(s)
}

package erdvalidator

import (
	"encoding/json"

	governor "github.com/metal-toolbox/governor-api/pkg/api/v1alpha1"
	"sigs.k8s.io/yaml"
)

// ERDContent is an interface for ERD content.
type ERDContent interface {
	// Marshal marshals an ERD to a payload, returning an error if it fails.
	Marshal(*governor.ExtensionResourceDefinitionReq) error
	// Unmarshal unmarshals an ERD from a payload, returning an error if it fails.
	Unmarshal() (*governor.ExtensionResourceDefinitionReq, error)
}

// ERDContentJSON is a JSON ERD payload, it provides Marshal and Unmarshal methods.
type ERDContentJSON []byte

// ERDContentJSON implements ERDContent.
var _ ERDContent = (*ERDContentJSON)(nil)

// Unmarshal unmarshals an ERD from a JSON payload, returning an error if it fails.
func (e *ERDContentJSON) Unmarshal() (*governor.ExtensionResourceDefinitionReq, error) {
	erd := &governor.ExtensionResourceDefinitionReq{}

	if err := json.Unmarshal(*e, erd); err != nil {
		return nil, err
	}

	return erd, nil
}

// Marshal marshals an ERD to a JSON payload, returning an error if it fails.
func (e *ERDContentJSON) Marshal(erd *governor.ExtensionResourceDefinitionReq) (err error) {
	*e, err = json.MarshalIndent(erd, "", "  ")
	return err
}

// ERDContentYAML is a YAML ERD payload, it provides Marshal and Unmarshal methods.
type ERDContentYAML []byte

// ERDCOntentYAML implements ERDContent.
var _ ERDContent = (*ERDContentYAML)(nil)

// Unmarshal unmarshals an ERD from a YAML payload, returning an error if it fails.
func (e *ERDContentYAML) Unmarshal() (*governor.ExtensionResourceDefinitionReq, error) {
	erd := &governor.ExtensionResourceDefinitionReq{}

	if err := yaml.Unmarshal(*e, erd); err != nil {
		return nil, err
	}

	return erd, nil
}

// Marshal marshals an ERD to a YAML payload, returning an error if it fails.
func (e *ERDContentYAML) Marshal(erd *governor.ExtensionResourceDefinitionReq) (err error) {
	*e, err = yaml.Marshal(erd)
	return err
}

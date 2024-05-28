package eventprocessor

import (
	"github.com/metal-toolbox/governor-api/pkg/api/v1alpha1"
	"github.com/metal-toolbox/governor-extension-sdk/pkg/eventrouter"
)

// EventProcessor is an interface for an event processor, all extension
// processors must implement this interface
type EventProcessor interface {
	Register(r eventrouter.EventRouter, ext *v1alpha1.Extension)
}

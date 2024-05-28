package eventrouter

import (
	"context"

	govevents "github.com/metal-toolbox/governor-api/pkg/events/v1alpha1"
)

// Handler is an function for processing governor events
type Handler func(context.Context, *govevents.Event) error

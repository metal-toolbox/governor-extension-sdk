// Package eventrouter provides a router for events.
package eventrouter

import (
	"context"

	govevents "github.com/metal-toolbox/governor-api/pkg/events/v1alpha1"
)

// EventRouter is an interface for an event router
type EventRouter interface {
	Create(string, Handler, ...Middleware)
	Update(string, Handler, ...Middleware)
	Delete(string, Handler, ...Middleware)
	Approve(string, Handler, ...Middleware)
	Deny(string, Handler, ...Middleware)
	Revoke(string, Handler, ...Middleware)

	Use(mw Middleware)
	Process(context.Context, string, *govevents.Event) error

	Subjects() []string
}

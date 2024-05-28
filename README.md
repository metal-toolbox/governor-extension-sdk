# Governor Extension SDK

[![Build status](https://badge.buildkite.com/3fbf454cd814f4a07ff34d4109d996c6e0c2c06c23b0d2282a.svg?branch=main)](https://buildkite.com/metal-toolbox/governor-extension-sdk)

Governor extension SDK is the SDK for developing Governor extensions. It provides
a set of tools and utilities to help developers to create, test and deploy
[Governor extensions](https://github.com/equinixmetal/napkins/blob/main/napkins/NAP0012.md).

## CLI

### ERDs Validation

The `validate` command validates ERD files in `./erds`

Usage:

```console
$ governor-extension-sdk erds validate       
2023-11-08T17:44:35.527Z        INFO    cmd/erds.go:93  validating ERDs
2023-11-08T17:44:35.564Z        INFO    cmd/erds.go:140 ERDs are valid
```

Run `governor-extension-sdk erds validate --help` for more information

### Create ERD file

The `new` command creates a new ERD file populated with default place holder
values.

Usage:

```console
$ governor-extension-sdk erds new --filename my-erd.yaml
2023-11-08T17:46:49.134Z        INFO    cmd/erds.go:222 creating new ERD my-erd.yaml
```

The default values can be overriden with additional flags:

```console
Flags:
      --description string     description of the new ERD (default "some-description")
      --enabled                enabled status of the new ERD (default true)
      --filename string        filename of the new ERD, only .json, .yml and .yaml are supported
      --name string            name of the new ERD (default "hello-world")
      --scope string           scope of the new ERD (default "user")
      --slug-plural string     plural slug of the new ERD (default "greetings")
      --slug-singular string   singular slug of the new ERD (default "greeting")
      --version string         version of the new ERD (default "v1alpha1")
```

## Development

### Event Router

The event router listens to
events from the Governor API and dispatches them to the appropriate event
processors.

#### Event router sample usage

1. Create a new event processor that implements `pkg/eventprocessor.Processor`

    ```go
    import (
      "github.com/metal-toolbox/governor-extension-sdk/pkg/eventprocessor"
      "github.com/metal-toolbox/governor-api/pkg/api/v1alpha1"
    )

    type Processor struct {
        // add fields here
    }

    // Processor implements the eventrouter.EventProcessor interface
    var _ eventprocessor.EventProcessor = (*Processor)(nil)

    func (p *Processor) ProcessEvent(ctx context.Context, event *event.Event) error {
      // add event processing logic here
      return nil
    }

    func (p. Processor) Register(router eventrouter.EventRouter, ext *v1alpha1.Extension) error {
      router.Create("groups", p.ProcessEvent)
      router.Update("groups", p.ProcessEvent)
      // ... add more event types here

      return nil
    }

    ```

1. Initialize a new event router

    ```go
    import (
      "github.com/metal-toolbox/governor-extension-sdk/pkg/eventrouter"
    )

    router := eventrouter.NewRouter()
    ```

1. Register the event processor with the event router

    ```go
    processor := &Processor{} // processor instance

    err := processor.Register(roueter, &v1alpha1.Extension{ /* extension metadata */ })
    ```

### Server

The `server` package provides a simple HTTP server that listens to incoming
events from the Governor API.

#### Server sample usage

With the event router initialized and event processor registered, the server
can be created as follows:

```go
import (
  "github.com/metal-toolbox/governor-extension-sdk/pkg/server"
)

natsclient, err := server.NewNATSClient(/* nats client options */)

server := server.NewServer(
  "0.0.0.0:8080",
  "governor-extension-id",
  "paht/to/erds",
  server.WithEventRouter(router),
  server.WithNATSClient(natsclient)
)
```

Developers can also provide only the event processor to the server, and the
server will construct a new event router and register the event processor.

```go
import (
  "github.com/metal-toolbox/governor-extension-sdk/pkg/server"
)

server := server.NewServer(
  "0.0.0.0:8080",
  "governor-extension-id",
  "paht/to/erds",
  server.WithNATSClient(natsclient)
  // multiple processors can be added here
  server.WithEventProcessor(&Processor{}),
  server.WithEventProcessor(/* another processor */),
  server.WithEventProcessor(/* and another */),
)
```

Now the server can be started:

```go
err := server.Run(ctx)
```

### Tracing

Tracing should work out of the box. Top level tracer is defined in `server/server.go`
and it can be passed to any event processors.

Event processors can inherit parent trace context in `event.TraceContext`
populated by the Governor API.

The trace context can be passed to other HTTP services by using the
`WithTraceContext` roundtripper option.

```go
import (
  "net/http"
  "github.com/metal-toolbox/governor-extension-sdk/pkg/eventrouter"
)

client := &http.Client{
  Transport: roundtripper.NewGovExtRoundTripper(
    http.DefaultTransport.RoundTrip,
    roundtripper.WithTraceContext(),
  ),
}
```

### Correlation IDs

To prevent infinite update loop that caused by the extension reacting to its
own update, the event router provides a correlation ID processor that can be
used to track the correlation ID of the incoming events,
see [this](https://github.com/equinixmetal/eis/blob/main/napkins/NAP0018.md#correlation-ids)
for more information.

The correlation ID processor works by extracting the correlation ID from the
incoming event and storing it in the context with a middleware, it also checks
if the correlation ID is already present in the context and if it is, it will
skip the event processing. Finally, it will inject the correlation ID into the
subsequent outgoing API requests to the Governor API.

For this to work, the event router should be initialized with the correlation ID
processor:

```go
import (
  "github.com/metal-toolbox/governor-extension-sdk/pkg/eventrouter"
)

router := eventrouter.NewRouter(eventrouter.WithCorrelationIDProcessor(
  eventrouter.NewCorrelationIDProcessor()
))
```

In addition, the http client used to make API requests to the Governor API should
be initialized with the correlation ID Roundtripper:

```go
import (
  "net/http"
  "github.com/metal-toolbox/governor-api/pkg/client"
  "github.com/metal-toolbox/governor-extension-sdk/pkg/roundtripper"
)

client := governor.NewClient(
  governor.WithHTTPClient(&http.Client{
    Transport: roundtripper.NewGovExtRoundTripper(
      http.DefaultTransport.RoundTrip,
      roundtripper.WithCorrelationID(),
    ),
  }),
  /* other options */
)
```

#### Skip Strategy

The correlation ID processor provides three strategies to skip the event processing:

1. **Skip All**: Skip the any event processing if the correlation ID already exists

    ```go
    import (
      "github.com/metal-toolbox/governor-extension-sdk/pkg/eventrouter"
    )

    cidp := eventrouter.NewCorrelationIDProcessor(
      eventrouter.CorrelationIDProcessorWithSkipStrategySkipAll(),
    )
    ```

1. **Update Only**: Skip the event processing if the correlation ID already exists
  and the event type is `update`

    ```go
    import (
      "github.com/metal-toolbox/governor-extension-sdk/pkg/eventrouter"
    )

    cidp := eventrouter.NewCorrelationIDProcessor(
      eventrouter.CorrelationIDProcessorWithSkipStrategyUpdateOnly(),
    )
    ```

1. **Custom**: Skip the event processing if the correlation ID already exists and
  the routes are provided in the custom skip strategy

    ```go
    import (
      "github.com/metal-toolbox/governor-extension-sdk/pkg/eventrouter"
      "github.com/metal-toolbox/governor-api/pkg/events/v1alpha1"
    )

    cidp := eventrouter.NewCorrelationIDProcessor(
      eventrouter.CorrelationIDProcessorWithSkipStrategyCustom(map[string]map[string]struct{}{
        // skips all updates
        govevents.GovernorEventUpdate:  {"*": {}},
        // skips create groups and create roles
        govevents.GovernorEventCreate:  {
          "groups": {},
          "roles": {},
        },
      }),
    )
    ```

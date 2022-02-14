# Otel watermill AMQP
A watermill AMQP wrapper support open-telemetry tracing.

## Problem
In a scenario like CQRS, the command is sent asynchronously, and there is no way to pass the context in the original Library, and open telemetry is to pass the context to track, so in the AMQP-based CQRS Pub/Sub, I add `Trace`, `Span` to message metada, so that asynchronous commands can continue to trace after AMQP queue unmarshal.

## Install

```
go get -u github.com/hpcslag/otel-watermill-amqp
```

``` go

import (
    watermillotelamqp "github.com/hpcslag/otel-watermill-amqp"
)

// DSN
amqpAddress := fmt.Sprintf("amqp://%s:%s@%s:%s/%s", .... )

// just logger
logger := watermill.NewStdLogger(false, false)

// using protobuf marshaler
cqrsMarshaler := cqrs.ProtobufMarshaler{}

// using this library to wrap the config
commandsAMQPConfig := watermillotelamqp.NewDurableQueueConfig(amqpAddress)

// as usual
commandsPublisher, err := amqp.NewPublisher(commandsAMQPConfig, logger)
if err != nil {
    panic(err)
}
// as usual
commandsSubscriber, err := amqp.NewSubscriber(commandsAMQPConfig, logger)
if err != nil {
    panic(err)
}

router, err := message.NewRouter(message.RouterConfig{}, logger)
if err != nil {
    panic(err)
}
...
// add the middleware to help let trace metadata put in to context
router.AddMiddleware(watermillotelamqp.AMQPTraceTaking)
...
```

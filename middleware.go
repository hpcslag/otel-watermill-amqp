package watermillotelamqp

import (
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/trace"
)

// The original message brings the Open Telemetry Trace Data, this can use to bring out to async command
func AMQPTraceTaking(h message.HandlerFunc) message.HandlerFunc {
	return func(event *message.Message) (events []*message.Message, err error) {

		if event.Metadata.Get("trace_id") != "" && event.Metadata.Get("span_id") != "" {
			traceID, traceErr := trace.TraceIDFromHex(event.Metadata.Get("trace_id"))
			spanID, spanErr := trace.SpanIDFromHex(event.Metadata.Get("span_id"))
			if traceErr == nil && spanErr == nil {
				sc := trace.NewSpanContext(trace.SpanContextConfig{
					TraceID:    traceID,
					SpanID:     spanID,
					TraceFlags: 01,
					Remote:     false,
				})

				ctx := trace.ContextWithSpanContext(event.Context(), sc)
				event.SetContext(ctx)

			}
		}

		return h(event)
	}
}

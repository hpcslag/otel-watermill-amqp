package watermillotelamqp

import (
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel/trace"
)

const DefaultMessageUUIDHeaderKey = "_watermill_message_uuid"

// deprecated, please use DefaultMessageUUIDHeaderKey instead
const MessageUUIDHeaderKey = DefaultMessageUUIDHeaderKey

// Marshaler marshals Watermill's message to amqp.Publishing and unmarshals amqp.Delivery to Watermill's message.
type Marshaler interface {
	Marshal(msg *message.Message) (amqp.Publishing, error)
	Unmarshal(amqpMsg amqp.Delivery) (*message.Message, error)
}

type OtelMarshaler struct {
	// PostprocessPublishing can be used to make some extra processing with amqp.Publishing,
	// for example add CorrelationId and ContentType:
	//
	//  amqp.OtelMarshaler{
	//		PostprocessPublishing: func(publishing stdAmqp.Publishing) stdAmqp.Publishing {
	//			publishing.CorrelationId = "correlation"
	//			publishing.ContentType = "application/json"
	//
	//			return publishing
	//		},
	//	}
	PostprocessPublishing func(amqp.Publishing) amqp.Publishing

	// When true, DeliveryMode will be not set to Persistent.
	//
	// DeliveryMode Transient means higher throughput, but messages will not be
	// restored on broker restart. The delivery mode of publishings is unrelated
	// to the durability of the queues they reside on. Transient messages will
	// not be restored to durable queues, persistent messages will be restored to
	// durable queues and lost on non-durable queues during server restart.
	NotPersistentDeliveryMode bool

	// Header used to store and read message UUID.
	//
	// If value is empty, DefaultMessageUUIDHeaderKey value is used.
	// If header doesn't exist, empty value is passed as message UUID.
	MessageUUIDHeaderKey string
}

func (d OtelMarshaler) Marshal(msg *message.Message) (amqp.Publishing, error) {
	headers := make(amqp.Table, len(msg.Metadata)+1) // metadata + plus uuid

	for key, value := range msg.Metadata {
		headers[key] = value
	}
	headers[d.computeMessageUUIDHeaderKey()] = msg.UUID

	// takeout otel trace id
	span := trace.SpanFromContext(msg.Context())
	if span.SpanContext().IsValid() {
		headers["trace_id"] = span.SpanContext().TraceID().String()
		headers["span_id"] = span.SpanContext().SpanID().String()
	}

	publishing := amqp.Publishing{
		Body:    msg.Payload,
		Headers: headers,
	}
	if !d.NotPersistentDeliveryMode {
		publishing.DeliveryMode = amqp.Persistent
	}

	if d.PostprocessPublishing != nil {
		publishing = d.PostprocessPublishing(publishing)
	}

	return publishing, nil
}

func (d OtelMarshaler) Unmarshal(amqpMsg amqp.Delivery) (*message.Message, error) {
	msgUUIDStr, err := d.unmarshalMessageUUID(amqpMsg)
	if err != nil {
		return nil, err
	}

	msg := message.NewMessage(msgUUIDStr, amqpMsg.Body)
	msg.Metadata = make(message.Metadata, len(amqpMsg.Headers)-1) // headers - minus uuid

	// unmarshal otel trace metadata
	traceID, traceErr := trace.TraceIDFromHex(amqpMsg.Headers["trace_id"].(string))
	spanID, spanErr := trace.SpanIDFromHex(amqpMsg.Headers["span_id"].(string))
	if traceErr != nil || spanErr != nil {
		sc := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: 0,
			Remote:     false,
		})
		ctx := trace.ContextWithSpanContext(msg.Context(), sc)
		msg.SetContext(ctx)
	}

	for key, value := range amqpMsg.Headers {
		if key == d.computeMessageUUIDHeaderKey() {
			continue
		}

		var ok bool
		msg.Metadata[key], ok = value.(string)
		if !ok {
			return nil, errors.Errorf("metadata %s is not a string, but %#v", key, value)
		}
	}

	return msg, nil
}

func (d OtelMarshaler) unmarshalMessageUUID(amqpMsg amqp.Delivery) (string, error) {
	var msgUUIDStr string

	msgUUID, hasMsgUUID := amqpMsg.Headers[d.computeMessageUUIDHeaderKey()]
	if !hasMsgUUID {
		return "", nil
	}

	msgUUIDStr, hasMsgUUID = msgUUID.(string)
	if !hasMsgUUID {
		return "", errors.Errorf("message UUID is not a string, but: %#v", msgUUID)
	}

	return msgUUIDStr, nil
}

func (d OtelMarshaler) computeMessageUUIDHeaderKey() string {
	if d.MessageUUIDHeaderKey != "" {
		return d.MessageUUIDHeaderKey
	}

	return DefaultMessageUUIDHeaderKey
}

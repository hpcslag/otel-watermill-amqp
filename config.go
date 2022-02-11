package watermillotelamqp

import "github.com/ThreeDotsLabs/watermill-amqp/pkg/amqp"

func NewDurableQueueConfig(amqpURI string) amqp.Config {
	return amqp.Config{
		Connection: amqp.ConnectionConfig{
			AmqpURI: amqpURI,
		},

		Marshaler: OtelMarshaler{},

		Exchange: amqp.ExchangeConfig{
			GenerateName: func(topic string) string {
				return ""
			},
		},
		Queue: amqp.QueueConfig{
			GenerateName: amqp.GenerateQueueNameTopicName,
			Durable:      true,
		},
		QueueBind: amqp.QueueBindConfig{
			GenerateRoutingKey: func(topic string) string {
				return ""
			},
		},
		Publish: amqp.PublishConfig{
			GenerateRoutingKey: func(topic string) string {
				return topic
			},
		},
		Consume: amqp.ConsumeConfig{
			Qos: amqp.QosConfig{
				PrefetchCount: 1,
			},
		},
		TopologyBuilder: &amqp.DefaultTopologyBuilder{},
	}
}

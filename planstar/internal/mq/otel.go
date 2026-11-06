package mq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
)

// amqpCarrier 让 W3C TraceContext 可以随 RabbitMQ 消息头跨进程透传。
type amqpCarrier amqp.Table

var _ propagation.TextMapCarrier = amqpCarrier(nil)

func (c amqpCarrier) Get(key string) string {
	if v, ok := amqp.Table(c)[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (c amqpCarrier) Set(key, val string) { amqp.Table(c)[key] = val }

func (c amqpCarrier) Keys() []string {
	keys := make([]string, 0, len(amqp.Table(c)))
	for k := range amqp.Table(c) {
		keys = append(keys, k)
	}
	return keys
}

func injectTrace(ctx context.Context, headers amqp.Table) {
	otel.GetTextMapPropagator().Inject(ctx, amqpCarrier(headers))
}

func extractTrace(ctx context.Context, headers amqp.Table) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, amqpCarrier(headers))
}

func consumeSpan(ctx context.Context, queue, routingKey string) (context.Context, trace.Span) {
	name := "consume " + queue
	if routingKey != "" {
		name = "consume " + routingKey
	}
	return otel.Tracer("rabbitmq").Start(ctx, name,
		trace.WithAttributes(
			semconv.MessagingSystemKey.String("rabbitmq"),
			semconv.MessagingDestinationKey.String(routingKey),
		),
	)
}

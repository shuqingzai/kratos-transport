package broker

import (
	"bytes"
	"encoding/gob"
	"errors"

	"github.com/google/uuid"

	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/sdk/resource"
	traceSdk "go.opentelemetry.io/otel/sdk/trace"
	semConv "go.opentelemetry.io/otel/semconv/v1.12.0"

	"github.com/go-kratos/kratos/v2/encoding"
	_ "github.com/go-kratos/kratos/v2/encoding/json"
	_ "github.com/go-kratos/kratos/v2/encoding/proto"
)

func Marshal(codec encoding.Codec, msg Any) ([]byte, error) {
	if msg == nil {
		return nil, errors.New("message is nil")
	}

	if codec != nil {
		dataBuffer, err := codec.Marshal(msg)
		if err != nil {
			return nil, err
		}
		return dataBuffer, nil
	} else {
		switch t := msg.(type) {
		case []byte:
			return t, nil
		case string:
			return []byte(t), nil
		default:
			var buf bytes.Buffer
			enc := gob.NewEncoder(&buf)
			if err := enc.Encode(msg); err != nil {
				return nil, err
			}
			return buf.Bytes(), nil
		}
	}
}

func Unmarshal(codec encoding.Codec, buf []byte, data interface{}) error {
	if codec != nil {
		if err := codec.Unmarshal(buf, data); err != nil {
			return err
		}
	} else {
		data = buf
	}
	return nil
}

// NewExporter 创建一个导出器，支持：jaeger和zipkin
func NewExporter(exporterName, endpoint string) (traceSdk.SpanExporter, error) {
	switch exporterName {
	case "jaeger":
		return NewJaegerExporter(endpoint)
	case "zipkin":
		return NewZipkinExporter(endpoint)
	default:
		return nil, errors.New("exporter type not support")
	}
}

// NewJaegerExporter 创建一个jaeger导出器
func NewJaegerExporter(endpoint string) (traceSdk.SpanExporter, error) {
	return jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(endpoint)))
}

// NewZipkinExporter 创建一个zipkin导出器
func NewZipkinExporter(endpoint string) (traceSdk.SpanExporter, error) {
	return zipkin.New(endpoint)
}

// NewTracerProvider 创建一个链路追踪器
func NewTracerProvider(exporterName, endpoint, serviceName, instanceId, version string, sampler float64) *traceSdk.TracerProvider {
	if instanceId == "" {
		ud, _ := uuid.NewUUID()
		instanceId = ud.String()
	}
	if version == "" {
		version = "x.x.x"
	}

	opts := []traceSdk.TracerProviderOption{
		traceSdk.WithSampler(traceSdk.ParentBased(traceSdk.TraceIDRatioBased(sampler))),
		traceSdk.WithResource(resource.NewSchemaless(
			semConv.ServiceNameKey.String(serviceName),
			semConv.ServiceInstanceIDKey.String(instanceId),
			semConv.ServiceVersionKey.String(version),
		)),
	}

	if len(endpoint) > 0 {
		exp, err := NewExporter(exporterName, endpoint)
		if err != nil {
			panic(err)
		}

		opts = append(opts, traceSdk.WithBatcher(exp))
	}

	return traceSdk.NewTracerProvider(opts...)
}

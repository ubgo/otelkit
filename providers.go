package otelkit

import (
	"context"

	otellog "go.opentelemetry.io/otel/log"
	lognoop "go.opentelemetry.io/otel/log/noop"
	otelmetric "go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/resource"
	oteltrace "go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"

	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// buildSampler maps the Sampler enum + ratio to an SDK sampler.
func buildSampler(s Sampler, ratio float64) sdktrace.Sampler {
	switch s {
	case SamplerAlwaysOn:
		return sdktrace.AlwaysSample()
	case SamplerAlwaysOff:
		return sdktrace.NeverSample()
	case SamplerTraceIDRatio:
		return sdktrace.TraceIDRatioBased(ratio)
	case SamplerParentBasedTraceIDRatio:
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
	default: // SamplerParentBasedAlwaysOn
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	}
}

// buildTracerProvider returns a real TracerProvider when traces are enabled,
// or a no-op provider otherwise. The returned shutdown closure flushes and
// shuts the provider down (nil for the no-op).
func buildTracerProvider(ctx context.Context, c Config, res *resource.Resource) (oteltrace.TracerProvider, shutdownFunc, error) {
	if !c.Traces.Enabled {
		return tracenoop.NewTracerProvider(), nil, nil
	}
	exp, err := newSpanExporter(ctx, c.Traces, c.TLS)
	if err != nil {
		return nil, nil, err
	}
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(buildSampler(c.Sampler, c.SamplerRatio)),
	}
	if c.Traces.Transport == TransportStdout {
		opts = append(opts, sdktrace.WithSyncer(exp))
	} else {
		bopts := []sdktrace.BatchSpanProcessorOption{}
		if c.Traces.BatchTimeout > 0 {
			bopts = append(bopts, sdktrace.WithBatchTimeout(c.Traces.BatchTimeout))
		}
		opts = append(opts, sdktrace.WithBatcher(exp, bopts...))
	}
	tp := sdktrace.NewTracerProvider(opts...)
	return tp, tp.Shutdown, nil
}

// buildMeterProvider returns a real MeterProvider when metrics are enabled, or
// a no-op provider otherwise.
func buildMeterProvider(ctx context.Context, c Config, res *resource.Resource) (otelmetric.MeterProvider, shutdownFunc, error) {
	if !c.Metrics.Enabled {
		return metricnoop.NewMeterProvider(), nil, nil
	}
	exp, err := newMetricExporter(ctx, c.Metrics, c.MetricTemporality, c.TLS)
	if err != nil {
		return nil, nil, err
	}
	interval := c.MetricInterval
	if interval <= 0 {
		interval = defaultMetricInterval
	}
	reader := sdkmetric.NewPeriodicReader(exp, sdkmetric.WithInterval(interval))
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(reader),
	)
	return mp, mp.Shutdown, nil
}

// buildLoggerProvider returns a real LoggerProvider when logs are enabled, or a
// no-op provider otherwise.
func buildLoggerProvider(ctx context.Context, c Config, res *resource.Resource) (otellog.LoggerProvider, shutdownFunc, error) {
	if !c.Logs.Enabled {
		return lognoop.NewLoggerProvider(), nil, nil
	}
	exp, err := newLogExporter(ctx, c.Logs, c.TLS)
	if err != nil {
		return nil, nil, err
	}
	var proc sdklog.Processor
	if c.Logs.Transport == TransportStdout {
		proc = sdklog.NewSimpleProcessor(exp)
	} else {
		proc = sdklog.NewBatchProcessor(exp)
	}
	lp := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(proc),
	)
	return lp, lp.Shutdown, nil
}

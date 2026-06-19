package otelkit

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"go.opentelemetry.io/otel"
	otellog "go.opentelemetry.io/otel/log"
	logglobal "go.opentelemetry.io/otel/log/global"
	lognoop "go.opentelemetry.io/otel/log/noop"
	otelmetric "go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

// shutdownFunc flushes and shuts down one provider.
type shutdownFunc = func(context.Context) error

// buildResourceFn is the resource builder, indirected so tests can exercise
// Init's resource-error path (resource.New rarely fails in practice).
var buildResourceFn = buildResource

// Telemetry is the single bootstrap handle. It owns the three providers and
// the propagator, exposes accessors, and drives a single ordered Shutdown.
type Telemetry struct {
	tp         oteltrace.TracerProvider
	mp         otelmetric.MeterProvider
	lp         otellog.LoggerProvider
	propagator propagation.TextMapPropagator
	shutdowns  []shutdownFunc
	noop       bool
}

// Init resolves configuration (preset < options < env), builds the enabled
// providers, and returns a handle. It performs no global registration; call
// SetGlobal to opt in. On OTEL_SDK_DISABLED=true it returns a fully no-op
// handle (never nil). When WithSelfTest is set it sends one span synchronously
// and returns the export error if the backend is unreachable.
func Init(ctx context.Context, opts ...Option) (*Telemetry, error) {
	if envDisabled() {
		return newNoop(), nil
	}

	cfg := defaultConfig()
	cfg.apply(opts...)
	if cfg.envOverrides {
		// A declarative config file wins outright (spec): flat env + options
		// are ignored, only ${ENV} substitution inside the YAML applies.
		if path := configFilePath(); path != "" {
			return initFromFile(ctx, path)
		}
		if err := applyEnv(&cfg); err != nil {
			return nil, err
		}
	}
	if cfg.dryRun {
		applyDryRun(&cfg, os.Stderr)
	}

	res, err := buildResourceFn(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("otelkit: build resource: %w", err)
	}

	tp, tpShut, err := buildTracerProvider(ctx, cfg, res)
	if err != nil {
		return nil, fmt.Errorf("otelkit: traces: %w", err)
	}
	mp, mpShut, err := buildMeterProvider(ctx, cfg, res)
	if err != nil {
		return nil, fmt.Errorf("otelkit: metrics: %w", err)
	}
	lp, lpShut, err := buildLoggerProvider(ctx, cfg, res)
	if err != nil {
		return nil, fmt.Errorf("otelkit: logs: %w", err)
	}

	installErrorHandler(cfg)

	t := &Telemetry{tp: tp, mp: mp, lp: lp, propagator: buildPropagator(cfg)}
	// Shutdown order: logs, then metrics, then traces — so a prior phase's
	// errors still reach the log/trace collectors before those close.
	for _, s := range []shutdownFunc{lpShut, mpShut, tpShut} {
		if s != nil {
			t.shutdowns = append(t.shutdowns, s)
		}
	}

	if cfg.selfTest {
		if err := t.SelfTest(ctx); err != nil {
			_ = t.Shutdown(ctx)
			return nil, fmt.Errorf("otelkit: self-test failed: %w", err)
		}
	}
	return t, nil
}

// newNoop returns a handle whose providers are all no-ops.
func newNoop() *Telemetry {
	return &Telemetry{
		tp:         tracenoop.NewTracerProvider(),
		mp:         metricnoop.NewMeterProvider(),
		lp:         lognoop.NewLoggerProvider(),
		propagator: propagation.NewCompositeTextMapPropagator(),
		noop:       true,
	}
}

// TracerProvider returns the trace provider (real or no-op).
func (t *Telemetry) TracerProvider() oteltrace.TracerProvider { return t.tp }

// MeterProvider returns the metric provider (real or no-op).
func (t *Telemetry) MeterProvider() otelmetric.MeterProvider { return t.mp }

// LoggerProvider returns the log provider — the seam github.com/ubgo/logger's
// OTEL sink consumes.
func (t *Telemetry) LoggerProvider() otellog.LoggerProvider { return t.lp }

// Tracer is shorthand for TracerProvider().Tracer(name).
func (t *Telemetry) Tracer(name string) oteltrace.Tracer { return t.tp.Tracer(name) }

// Disabled reports whether this handle is the no-op handle (OTEL_SDK_DISABLED).
func (t *Telemetry) Disabled() bool { return t.noop }

// SetGlobal registers the providers and propagator on the OTEL globals.
func (t *Telemetry) SetGlobal() {
	otel.SetTracerProvider(t.tp)
	otel.SetMeterProvider(t.mp)
	logglobal.SetLoggerProvider(t.lp)
	otel.SetTextMapPropagator(t.propagator)
}

// forceFlusher is implemented by the SDK providers (not the no-ops).
type forceFlusher interface {
	ForceFlush(context.Context) error
}

// ForceFlush flushes any buffered telemetry across all providers.
func (t *Telemetry) ForceFlush(ctx context.Context) error {
	var errs []error
	for _, p := range []any{t.lp, t.mp, t.tp} {
		if f, ok := p.(forceFlusher); ok {
			if err := f.ForceFlush(ctx); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

// SelfTest sends one span synchronously and force-flushes it, surfacing the
// export error the async batch processor would otherwise hide. Used by
// WithSelfTest; safe to call directly.
func (t *Telemetry) SelfTest(ctx context.Context) error {
	_, span := t.Tracer("otelkit.selftest").Start(ctx, "selftest")
	span.End()
	return t.ForceFlush(ctx)
}

// buildPropagator builds the composite propagator. Honors OTEL_PROPAGATORS
// (default tracecontext,baggage) only when env overrides are enabled.
func buildPropagator(c Config) propagation.TextMapPropagator {
	def := func() propagation.TextMapPropagator {
		return propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	}
	if !c.envOverrides {
		return def()
	}
	v := getenv("OTEL_PROPAGATORS")
	if v == "" {
		return def()
	}
	var props []propagation.TextMapPropagator
	for _, tok := range strings.Split(v, ",") {
		switch strings.ToLower(strings.TrimSpace(tok)) {
		case "tracecontext":
			props = append(props, propagation.TraceContext{})
		case "baggage":
			props = append(props, propagation.Baggage{})
		case "none":
			return propagation.NewCompositeTextMapPropagator()
		}
	}
	if len(props) == 0 {
		return def()
	}
	return propagation.NewCompositeTextMapPropagator(props...)
}

// applyDryRun rewrites every enabled signal to the stdout transport and prints
// the resolved effective configuration (auth headers redacted) to w, so the
// wiring can be verified without a backend.
func applyDryRun(c *Config, w *os.File) {
	fmt.Fprintln(w, "otelkit: DRY RUN — effective configuration (no export):")
	fmt.Fprintf(w, "  service=%s version=%s env=%s\n", c.ServiceName, c.ServiceVersion, c.Environment)
	fmt.Fprintf(w, "  sampler=%s ratio=%g temporality=%s tls=%s\n", c.Sampler, c.SamplerRatio, c.MetricTemporality, c.TLS)
	for _, s := range []struct {
		name string
		sc   *SignalConfig
	}{{"traces", &c.Traces}, {"metrics", &c.Metrics}, {"logs", &c.Logs}} {
		if !s.sc.Enabled {
			fmt.Fprintf(w, "  %s: disabled\n", s.name)
			continue
		}
		fmt.Fprintf(w, "  %s: transport=%s endpoint=%s headers=%s\n",
			s.name, s.sc.Transport, s.sc.Endpoint, redactHeaders(s.sc.Headers))
		s.sc.Transport = TransportStdout
	}
}

// redactHeaders returns a deterministic, value-redacted header summary.
func redactHeaders(h map[string]string) string {
	if len(h) == 0 {
		return "{}"
	}
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for i, k := range keys {
		keys[i] = k + "=<redacted>"
	}
	return "{" + strings.Join(keys, ",") + "}"
}

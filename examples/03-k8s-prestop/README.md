# 03 — Kubernetes graceful shutdown

Drain and flush telemetry on `SIGTERM` so no spans/metrics are lost when a pod terminates.

## What it shows

- `tel.RunOnSignal(ctx)` — blocks until `SIGTERM`/`SIGINT`, then flushes all signals on a **fresh** context (a cancelled app context can't abort the flush).
- Running an HTTP server alongside, stopped after telemetry drains.

## When you'd use this

Any long-running service, especially in Kubernetes. When the kubelet terminates a pod it sends `SIGTERM`, waits `terminationGracePeriodSeconds`, then `SIGKILL`s. Unflushed batch processors lose data on a hard kill — `RunOnSignal` flushes them in that window.

## Run

```bash
go run ./03-k8s-prestop
# in another terminal:  kill -TERM <pid>   (or Ctrl+C)
```

## Kubernetes notes

- Set `terminationGracePeriodSeconds` comfortably above your export timeout (the default OTLP timeout is 10s), e.g. `30`.
- otelkit drains telemetry; for a **phased** shutdown of multiple resources (stop accepting traffic → drain in-flight → close DB/queues → flush telemetry last), compose with [`github.com/ubgo/shutdown`](https://github.com/ubgo/shutdown).
- A `preStop` hook that sleeps a few seconds lets load balancers stop routing before the process exits — orthogonal to, and complementary with, this flush.

## Key code

```go
go server.ListenAndServe()
if err := tel.RunOnSignal(ctx); err != nil { // blocks → flush on SIGTERM
	log.Printf("telemetry shutdown errors: %v", err)
}
_ = server.Shutdown(context.Background())
```

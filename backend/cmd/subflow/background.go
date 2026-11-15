package main

import (
	"context"

	"ganigeorgiev.com/pocketbase/pocketbase/core"

	"subflow/internal/upgrade"
)

// backgroundContext is cancelled when PocketBase terminates, which is also
// when it releases its database handles.
func backgroundContext(app core.App) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	app.OnTerminate().BindFunc(func(event *core.TerminateEvent) error {
		cancel()
		return event.Next()
	})
	return ctx
}

// guardBackground wraps work that runs outside a request. A job that is
// already querying when the database is released panics on a nil handle;
// that must not end the process while it is reporting a startup failure.
func guardBackground(ctx context.Context, name string, logf func(string, ...any), job func(context.Context)) func() {
	return func() {
		if ctx.Err() != nil {
			return
		}
		defer func() {
			if recovered := recover(); recovered != nil {
				logf("SubFlow background job %s stopped unexpectedly: %v", name, recovered)
			}
		}()
		job(ctx)
	}
}

// recordServeFailure persists a serve error as an upgrade failure only while
// the upgrade is still in progress. Once the instance is ready the upgrade has
// completed, so a later error must not lock the next start in maintenance mode.
func recordServeFailure(manager *upgrade.Manager, ready bool, err error, logf func(string, ...any)) string {
	if ready {
		return "serve_failed"
	}
	code := problemCode(err)
	if manager.HasLock() {
		if auditErr := manager.Fail(code); auditErr != nil {
			logf("could not persist upgrade failure: %v", auditErr)
		}
	}
	return code
}

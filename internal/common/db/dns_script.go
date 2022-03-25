package db

import (
	"errors"
	"time"

	"go.kuoruan.net/v8go-polyfills/fetch"
	v8 "rogchap.com/v8go"
)

var (
	compileTimeInterval    = 1 * time.Second
	ErrCompileTimeExceeded = errors.New("script compile time exceeded: " + compileTimeInterval.String())
)

// DNSScriptCompile attempts to compile a script
func DNSScriptCompile(script, origin string) error {
	iso := v8.NewIsolate()
	global := v8.NewObjectTemplate(iso)

	if err := fetch.InjectTo(iso, global); err != nil {
		return err
	}

	done := make(chan bool, 1)
	errs := make(chan error, 1)

	go func() {
		ctx := v8.NewContext(iso, global)
		_, err := ctx.RunScript(script, origin)
		if err != nil {
			errs <- err
			return
		}
		done <- true
	}()

	select {
	case <-done:
		return nil
	case err := <-errs:
		iso.Dispose()
		return err
	case <-time.After(compileTimeInterval):
		iso.TerminateExecution()
		iso.Dispose()
		return ErrCompileTimeExceeded
	}
}

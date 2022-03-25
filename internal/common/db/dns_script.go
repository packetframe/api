package db

import (
	"errors"
	"time"

	"go.kuoruan.net/v8go-polyfills/fetch"
	v8 "rogchap.com/v8go"
)

var (
	validationTimeInterval    = 1 * time.Second
	ErrValidationTimeExceeded = errors.New("script validation time exceeded: " + validationTimeInterval.String())
)

// DNSScriptValidate attempts to compile a script
func DNSScriptValidate(script, origin string) error {
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
	case <-time.After(validationTimeInterval):
		iso.TerminateExecution()
		iso.Dispose()
		return ErrValidationTimeExceeded
	}
}

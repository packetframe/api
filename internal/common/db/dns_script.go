package db

import (
	"errors"
	"strings"
	"time"

	"go.kuoruan.net/v8go-polyfills/fetch"
	"gorm.io/gorm"
	v8 "rogchap.com/v8go"
)

var (
	validationTimeInterval    = 1 * time.Second
	ErrValidationTimeExceeded = errors.New("script validation time exceeded: " + validationTimeInterval.String())
)

// ScriptValidate attempts to compile a script
func ScriptValidate(script, origin string) error {
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

// ScriptRecords returns a map of DNS labels to script strings
func ScriptRecords(db *gorm.DB) (map[string]string, error) {
	var records []Record
	if err := db.Order("created_at").Where("type = 'SCRIPT'").Joins("Zone").Find(&records).Error; err != nil {
		return nil, err
	}

	scripts := map[string]string{}
	for _, rec := range records {
		label := rec.Label
		if !strings.HasSuffix(rec.Label, rec.Zone.Zone) {
			label += "." + rec.Zone.Zone
		}
		scripts[label] = rec.Value
	}

	return scripts, nil
}

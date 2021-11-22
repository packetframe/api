package validation

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/miekg/dns"

	"github.com/packetframe/api/internal/common/db"
	"github.com/packetframe/api/internal/common/util"
)

// Validation parameters
var (
	validRRTypes = []string{"A", "AAAA", "CNAME", "TXT", "MX", "SRV", "NS"}
)

// localValidator is the singleton validator used for all validations
var localValidator *validator.Validate

// ErrorResponse stores a validation error to be returned as a HTTP response
type ErrorResponse struct {
	FailedField string
	Tag         string
	Value       string
}

// Register registers custom DNS validation handlers with the validator
func Register() error {
	// Create a new singleton validator
	localValidator = validator.New()

	// Register validators
	for name, function := range map[string]func(fl validator.FieldLevel) bool{
		// Validation functions
		"dns-rrtype": func(fl validator.FieldLevel) bool {
			return util.StrSliceContains(validRRTypes, fl.Field().String())
		},
	} {
		err := localValidator.RegisterValidation(name, function)
		if err != nil {
			return err
		}
	}

	// Register validator for full Record type
	localValidator.RegisterStructValidation(func(sl validator.StructLevel) {
		record := sl.Current().Interface().(db.Record)
		rrString := fmt.Sprintf("%s %d IN %s %s", dns.Fqdn(record.Label), record.TTL, record.Type, record.Value)
		_, err := dns.NewRR(rrString) // This is only used to catch an error so ignore resulting RR
		if err != nil {
			sl.ReportError(record.Value, err.Error(), "", "record", "")
		}
	}, db.Record{})

	return nil // nil error
}

// Validate runs a validation and returns a list of errors
func Validate(s interface{}) []*ErrorResponse {
	if localValidator == nil {
		panic("validator must be registered before validating")
	}

	var errors []*ErrorResponse
	err := localValidator.Struct(s)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			var element ErrorResponse
			element.FailedField = err.StructNamespace()
			element.Tag = err.Tag()
			element.Value = err.Param()
			errors = append(errors, &element)
		}
	}
	return errors
}

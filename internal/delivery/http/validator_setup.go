package httpdelivery

import (
	"errors"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// SetupValidator configures validator to use JSON tags in error fields.
func SetupValidator() error {
	engine, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return errors.New("validator engine type assertion failed")
	}

	engine.RegisterTagNameFunc(func(field reflect.StructField) string {
		tag := field.Tag.Get("json")
		if tag == "" {
			return field.Name
		}

		name := strings.Split(tag, ",")[0]
		if name == "" || name == "-" {
			return field.Name
		}

		return name
	})

	return nil
}

package response

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/go-playground/validator/v10"
)

const unknownFieldMessagePrefix = "json: unknown field "

// ValidationErrorItem is a human-readable validation issue for frontend rendering.
type ValidationErrorItem struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Rule    string `json:"rule,omitempty"`
	Param   string `json:"param,omitempty"`
}

// IsValidationError reports whether an error is related to request payload validation/binding.
func IsValidationError(err error) bool {
	if err == nil {
		return false
	}

	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		return true
	}

	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return true
	}

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return true
	}

	return errors.Is(err, io.EOF) || strings.Contains(err.Error(), unknownFieldMessagePrefix)
}

// FormatValidationError converts technical errors into frontend-friendly error details.
func FormatValidationError(err error) []ValidationErrorItem {
	if err == nil {
		return []ValidationErrorItem{{
			Field:   "payload",
			Message: MessageInvalidRequestBody,
			Rule:    "invalid_payload",
		}}
	}

	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		items := make([]ValidationErrorItem, 0, len(validationErrs))
		for _, item := range validationErrs {
			items = append(items, ValidationErrorItem{
				Field:   item.Field(),
				Message: validationMessage(item),
				Rule:    item.Tag(),
				Param:   item.Param(),
			})
		}
		return items
	}

	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return []ValidationErrorItem{{
			Field:   "payload",
			Message: fmt.Sprintf("invalid JSON format at character %d", syntaxErr.Offset),
			Rule:    "json_syntax",
		}}
	}

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		field := typeErr.Field
		if field == "" {
			field = "payload"
		}

		return []ValidationErrorItem{{
			Field:   field,
			Message: fmt.Sprintf("invalid value type, expected %s", typeErr.Type.String()),
			Rule:    "json_type",
		}}
	}

	if errors.Is(err, io.EOF) {
		return []ValidationErrorItem{{
			Field:   "payload",
			Message: "request body must not be empty",
			Rule:    "required",
		}}
	}

	if strings.Contains(err.Error(), unknownFieldMessagePrefix) {
		field := strings.Trim(strings.TrimPrefix(err.Error(), unknownFieldMessagePrefix), "\"")
		return []ValidationErrorItem{{
			Field:   field,
			Message: "field is not allowed",
			Rule:    "unknown_field",
		}}
	}

	return []ValidationErrorItem{{
		Field:   "payload",
		Message: MessageInvalidRequestBody,
		Rule:    "invalid_payload",
	}}
}

func validationMessage(fieldErr validator.FieldError) string {
	switch fieldErr.Tag() {
	case "required":
		return "field is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return fmt.Sprintf("minimum length is %s", fieldErr.Param())
	case "max":
		return fmt.Sprintf("maximum length is %s", fieldErr.Param())
	case "oneof":
		return fmt.Sprintf("must be one of: %s", fieldErr.Param())
	case "numeric":
		return "must contain only numbers"
	case "len":
		return fmt.Sprintf("length must be %s", fieldErr.Param())
	default:
		return "value is invalid"
	}
}

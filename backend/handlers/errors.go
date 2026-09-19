package handlers

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// friendlyError converts a raw Gin/validator binding error into a plain-English
// message that is safe and useful to show directly to the user.
func friendlyError(err error) string {
	// validator.ValidationErrors is what Gin returns for struct tag failures
	var ve validator.ValidationErrors
	if errors, ok := err.(validator.ValidationErrors); ok {
		ve = errors
	} else {
		// Non-validation error (e.g. malformed JSON) — return a generic message
		msg := err.Error()
		if strings.Contains(msg, "cannot unmarshal") || strings.Contains(msg, "invalid character") {
			return "Invalid request body. Please check your input."
		}
		return "Invalid input."
	}

	// Translate the first validation failure into a sentence.
	// We only surface the first error to keep the message short.
	e := ve[0]
	field := strings.ToLower(e.Field())
	tag := e.Tag()
	param := e.Param()

	switch tag {
	case "required":
		return fmt.Sprintf("%s is required.", humanField(field))
	case "min":
		if e.Kind().String() == "string" {
			return fmt.Sprintf("%s must be at least %s characters.", humanField(field), param)
		}
		return fmt.Sprintf("%s must have at least %s items.", humanField(field), param)
	case "max":
		if e.Kind().String() == "string" {
			return fmt.Sprintf("%s must be at most %s characters.", humanField(field), param)
		}
		return fmt.Sprintf("%s can have at most %s items.", humanField(field), param)
	case "email":
		return "Please enter a valid email address."
	case "dive":
		return "One or more options are invalid."
	default:
		return fmt.Sprintf("%s is invalid.", humanField(field))
	}
}

// humanField converts a struct field name like "title" or "options" into a
// friendly display label like "Question" or "Options".
func humanField(field string) string {
	switch field {
	case "title":
		return "Question"
	case "description":
		return "Description"
	case "options":
		return "Options"
	case "option_id":
		return "Option"
	case "name":
		return "Name"
	case "email":
		return "Email"
	case "password":
		return "Password"
	default:
		return strings.Title(field)
	}
}

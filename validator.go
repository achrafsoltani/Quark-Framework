// Package quark provides struct validation through field tags.
//
// The validation system supports a wide range of validators including:
// - Presence validation (required)
// - Length and size validation (min, max, len)
// - Format validation (email, url, uuid, pattern)
// - Character set validation (alpha, alphanum, numeric)
// - Comparison validation (gt, gte, lt, lte)
// - Enumeration validation (oneof)
//
// Validation is performed recursively on nested structs,
// allowing for complex validation scenarios.
//
// Usage:
//
//	type User struct {
//	    Name  string `validate:"required,min:2,max:50"`
//	    Email string `validate:"required,email"`
//	    Age   int    `validate:"gte:0,lte:150"`
//	}
//
//	user := User{Name: "Jo", Email: "invalid", Age: -1}
//	if errs := quark.Validate(user); errs.HasErrors() {
//	    return c.ErrorWithDetails(400, "Validation failed", errs.ToMap())
//	}
package quark

import (
	"fmt"
	"net/mail"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// ValidationError represents a single validation error for a field.
// It includes the field name, the validation tag that failed, the expected
// value/constraint, and a human-readable error message.
type ValidationError struct {
	Field   string `json:"field"`            // Field name (uses json tag if available)
	Tag     string `json:"tag"`              // Validator tag that failed (e.g., "required", "email")
	Value   string `json:"value,omitempty"`  // Expected value or constraint
	Message string `json:"message"`          // Human-readable error message
}

// Error implements the error interface.
func (e ValidationError) Error() string {
	return e.Message
}

// ValidationErrors is a collection of validation errors.
// It implements the error interface and provides helper methods for
// convenient error handling in HTTP responses.
type ValidationErrors []ValidationError

// Error implements the error interface, joining all error messages with semicolons.
// Returns an empty string if there are no errors.
func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Message)
	}
	return strings.Join(msgs, "; ")
}

// ToMap converts validation errors to a map[field]message for convenient JSON responses.
// Useful for client-side form validation integration.
//
// Example:
//
//	if errs := quark.Validate(input); errs.HasErrors() {
//	    return c.ErrorWithDetails(400, "Validation failed", errs.ToMap())
//	}
func (e ValidationErrors) ToMap() map[string]string {
	result := make(map[string]string)
	for _, err := range e {
		result[err.Field] = err.Message
	}
	return result
}

// HasErrors returns true if there are any validation errors in the collection.
func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

// Validate validates a struct using validate struct field tags.
//
// It performs validation on all exported fields that have a validate tag.
// Fields without a validate tag or with validate:"-" are skipped (but nested
// structs are still validated recursively). Field names in errors use the
// json tag if available, falling back to the struct field name.
//
// Validation is performed recursively on nested structs, with field names
// prefixed by the parent field (e.g., "user.address.street").
//
// Supported validation tags:
//   - required:       field must not be empty/zero
//   - min:n:          minimum length (strings/slices/maps) or value (numbers)
//   - max:n:          maximum length (strings/slices/maps) or value (numbers)
//   - len:n:          exact length (strings/slices/maps)
//   - gt:n:           greater than n (numbers only)
//   - gte:n:          greater than or equal to n (numbers only)
//   - lt:n:           less than n (numbers only)
//   - lte:n:          less than or equal to n (numbers only)
//   - email:          must be a valid email address (RFC 5322)
//   - url:            must be a valid URL (http/https/ftp)
//   - alpha:          must contain only letters (unicode)
//   - alphanum:       must contain only letters and numbers
//   - numeric:        must contain only digits
//   - uuid:           must be a valid UUID (v4 format)
//   - oneof:a b c:    must be one of the space-separated values
//   - pattern:regex:  must match the regex pattern
//
// Tags can be combined with commas, e.g., validate:"required,min:2,max:50"
//
// Example:
//
//	type Address struct {
//	    Street string `json:"street" validate:"required,min:5"`
//	    City   string `json:"city" validate:"required"`
//	}
//
//	type User struct {
//	    Name    string  `json:"name" validate:"required,min:2,max:50"`
//	    Email   string  `json:"email" validate:"required,email"`
//	    Age     int     `json:"age" validate:"gte:0,lte:150"`
//	    Role    string  `json:"role" validate:"oneof:admin user guest"`
//	    Address Address `json:"address"`  // Automatically validated recursively
//	}
//
//	user := User{Name: "Jo", Email: "invalid", Age: -1}
//	if errs := Validate(user); errs.HasErrors() {
//	    // Returns errors for: name (min:2), email (invalid format), age (gte:0)
//	    return c.ErrorWithDetails(400, "Validation failed", errs.ToMap())
//	}
func Validate(v interface{}) ValidationErrors {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return ValidationErrors{{
			Field:   "",
			Tag:     "struct",
			Message: "validation requires a struct",
		}}
	}

	var errors ValidationErrors
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Skip unexported fields
		if !fieldVal.CanInterface() {
			continue
		}

		// Get field name (use json tag if available)
		fieldName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" && parts[0] != "-" {
				fieldName = parts[0]
			}
		}

		// Get validate tag
		tag := field.Tag.Get("validate")

		// Apply validators if tag exists and is not "-"
		if tag != "" && tag != "-" {
			validators := strings.Split(tag, ",")
			for _, validator := range validators {
				validator = strings.TrimSpace(validator)
				if validator == "" {
					continue
				}

				// Parse validator and parameter
				var name, param string
				if idx := strings.Index(validator, ":"); idx != -1 {
					name = validator[:idx]
					param = validator[idx+1:]
				} else {
					name = validator
				}

				// Apply validator
				if err := applyValidator(fieldName, fieldVal, name, param); err != nil {
					errors = append(errors, *err)
				}
			}
		}

		// Recursively validate nested structs (always, regardless of whether
		// the parent field has a validate tag). This ensures complete validation
		// of complex nested structures.
		if fieldVal.Kind() == reflect.Struct {
			nestedErrors := Validate(fieldVal.Interface())
			// Prefix nested field names with parent field name for clarity
			for _, err := range nestedErrors {
				err.Field = fieldName + "." + err.Field
				errors = append(errors, err)
			}
		}
	}

	return errors
}

// applyValidator applies a single named validator to a field value.
// It dispatches to the appropriate validation function based on the validator name.
// Returns nil if validation passes or if the validator is unknown.
// Unknown validators are silently skipped to allow for future extensibility.
func applyValidator(fieldName string, fieldVal reflect.Value, name, param string) *ValidationError {
	switch name {
	case "required":
		return validateRequired(fieldName, fieldVal)
	case "min":
		return validateMin(fieldName, fieldVal, param)
	case "max":
		return validateMax(fieldName, fieldVal, param)
	case "email":
		return validateEmail(fieldName, fieldVal)
	case "url":
		return validateURL(fieldName, fieldVal)
	case "alpha":
		return validateAlpha(fieldName, fieldVal)
	case "alphanum":
		return validateAlphaNum(fieldName, fieldVal)
	case "numeric":
		return validateNumeric(fieldName, fieldVal)
	case "uuid":
		return validateUUID(fieldName, fieldVal)
	case "oneof":
		return validateOneOf(fieldName, fieldVal, param)
	case "pattern":
		return validatePattern(fieldName, fieldVal, param)
	case "len":
		return validateLen(fieldName, fieldVal, param)
	case "gt":
		return validateGt(fieldName, fieldVal, param)
	case "gte":
		return validateGte(fieldName, fieldVal, param)
	case "lt":
		return validateLt(fieldName, fieldVal, param)
	case "lte":
		return validateLte(fieldName, fieldVal, param)
	default:
		return nil // Unknown validator, skip
	}
}

// validateRequired checks if a field has a value.
func validateRequired(fieldName string, val reflect.Value) *ValidationError {
	if isEmpty(val) {
		return &ValidationError{
			Field:   fieldName,
			Tag:     "required",
			Message: fmt.Sprintf("%s is required", fieldName),
		}
	}
	return nil
}

// validateMin checks minimum length/value.
func validateMin(fieldName string, val reflect.Value, param string) *ValidationError {
	min, err := strconv.ParseInt(param, 10, 64)
	if err != nil {
		return nil
	}

	var valid bool
	var actual int64

	switch val.Kind() {
	case reflect.String:
		actual = int64(len(val.String()))
		valid = actual >= min
	case reflect.Slice, reflect.Array, reflect.Map:
		actual = int64(val.Len())
		valid = actual >= min
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		actual = val.Int()
		valid = actual >= min
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		actual = int64(val.Uint())
		valid = actual >= min
	case reflect.Float32, reflect.Float64:
		actual = int64(val.Float())
		valid = val.Float() >= float64(min)
	default:
		return nil
	}

	if !valid {
		return &ValidationError{
			Field:   fieldName,
			Tag:     "min",
			Value:   param,
			Message: fmt.Sprintf("%s must be at least %d", fieldName, min),
		}
	}
	return nil
}

// validateMax checks maximum length/value.
func validateMax(fieldName string, val reflect.Value, param string) *ValidationError {
	max, err := strconv.ParseInt(param, 10, 64)
	if err != nil {
		return nil
	}

	var valid bool
	var actual int64

	switch val.Kind() {
	case reflect.String:
		actual = int64(len(val.String()))
		valid = actual <= max
	case reflect.Slice, reflect.Array, reflect.Map:
		actual = int64(val.Len())
		valid = actual <= max
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		actual = val.Int()
		valid = actual <= max
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		actual = int64(val.Uint())
		valid = actual <= max
	case reflect.Float32, reflect.Float64:
		actual = int64(val.Float())
		valid = val.Float() <= float64(max)
	default:
		return nil
	}

	if !valid {
		return &ValidationError{
			Field:   fieldName,
			Tag:     "max",
			Value:   param,
			Message: fmt.Sprintf("%s must be at most %d", fieldName, max),
		}
	}
	return nil
}

// validateLen checks exact length.
func validateLen(fieldName string, val reflect.Value, param string) *ValidationError {
	length, err := strconv.Atoi(param)
	if err != nil {
		return nil
	}

	var actual int
	switch val.Kind() {
	case reflect.String:
		actual = len(val.String())
	case reflect.Slice, reflect.Array, reflect.Map:
		actual = val.Len()
	default:
		return nil
	}

	if actual != length {
		return &ValidationError{
			Field:   fieldName,
			Tag:     "len",
			Value:   param,
			Message: fmt.Sprintf("%s must have exactly %d characters", fieldName, length),
		}
	}
	return nil
}

// validateEmail checks if the value is a valid email.
func validateEmail(fieldName string, val reflect.Value) *ValidationError {
	if val.Kind() != reflect.String {
		return nil
	}

	email := val.String()
	if email == "" {
		return nil // Empty is handled by required
	}

	_, err := mail.ParseAddress(email)
	if err != nil {
		return &ValidationError{
			Field:   fieldName,
			Tag:     "email",
			Message: fmt.Sprintf("%s must be a valid email address", fieldName),
		}
	}
	return nil
}

// validateURL checks if the value is a valid URL.
func validateURL(fieldName string, val reflect.Value) *ValidationError {
	if val.Kind() != reflect.String {
		return nil
	}

	url := val.String()
	if url == "" {
		return nil
	}

	// Simple URL validation
	urlPattern := `^(https?|ftp)://[^\s/$.?#].[^\s]*$`
	matched, _ := regexp.MatchString(urlPattern, url)
	if !matched {
		return &ValidationError{
			Field:   fieldName,
			Tag:     "url",
			Message: fmt.Sprintf("%s must be a valid URL", fieldName),
		}
	}
	return nil
}

// validateAlpha checks if the value contains only letters.
func validateAlpha(fieldName string, val reflect.Value) *ValidationError {
	if val.Kind() != reflect.String {
		return nil
	}

	s := val.String()
	if s == "" {
		return nil
	}

	for _, r := range s {
		if !unicode.IsLetter(r) {
			return &ValidationError{
				Field:   fieldName,
				Tag:     "alpha",
				Message: fmt.Sprintf("%s must contain only letters", fieldName),
			}
		}
	}
	return nil
}

// validateAlphaNum checks if the value contains only letters and numbers.
func validateAlphaNum(fieldName string, val reflect.Value) *ValidationError {
	if val.Kind() != reflect.String {
		return nil
	}

	s := val.String()
	if s == "" {
		return nil
	}

	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return &ValidationError{
				Field:   fieldName,
				Tag:     "alphanum",
				Message: fmt.Sprintf("%s must contain only letters and numbers", fieldName),
			}
		}
	}
	return nil
}

// validateNumeric checks if the value contains only numbers.
func validateNumeric(fieldName string, val reflect.Value) *ValidationError {
	if val.Kind() != reflect.String {
		return nil
	}

	s := val.String()
	if s == "" {
		return nil
	}

	for _, r := range s {
		if !unicode.IsDigit(r) {
			return &ValidationError{
				Field:   fieldName,
				Tag:     "numeric",
				Message: fmt.Sprintf("%s must contain only numbers", fieldName),
			}
		}
	}
	return nil
}

// validateUUID checks if the value is a valid UUID.
func validateUUID(fieldName string, val reflect.Value) *ValidationError {
	if val.Kind() != reflect.String {
		return nil
	}

	uuid := val.String()
	if uuid == "" {
		return nil
	}

	uuidPattern := `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`
	matched, _ := regexp.MatchString(uuidPattern, uuid)
	if !matched {
		return &ValidationError{
			Field:   fieldName,
			Tag:     "uuid",
			Message: fmt.Sprintf("%s must be a valid UUID", fieldName),
		}
	}
	return nil
}

// validateOneOf checks if the value is one of the allowed values.
func validateOneOf(fieldName string, val reflect.Value, param string) *ValidationError {
	if val.Kind() != reflect.String {
		return nil
	}

	s := val.String()
	if s == "" {
		return nil
	}

	allowed := strings.Split(param, " ")
	for _, a := range allowed {
		if s == a {
			return nil
		}
	}

	return &ValidationError{
		Field:   fieldName,
		Tag:     "oneof",
		Value:   param,
		Message: fmt.Sprintf("%s must be one of: %s", fieldName, strings.Join(allowed, ", ")),
	}
}

// validatePattern checks if the value matches a regex pattern.
func validatePattern(fieldName string, val reflect.Value, param string) *ValidationError {
	if val.Kind() != reflect.String {
		return nil
	}

	s := val.String()
	if s == "" {
		return nil
	}

	matched, err := regexp.MatchString(param, s)
	if err != nil || !matched {
		return &ValidationError{
			Field:   fieldName,
			Tag:     "pattern",
			Value:   param,
			Message: fmt.Sprintf("%s format is invalid", fieldName),
		}
	}
	return nil
}

// validateGt checks if value is greater than param.
func validateGt(fieldName string, val reflect.Value, param string) *ValidationError {
	target, err := strconv.ParseFloat(param, 64)
	if err != nil {
		return nil
	}

	var value float64
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value = float64(val.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value = float64(val.Uint())
	case reflect.Float32, reflect.Float64:
		value = val.Float()
	default:
		return nil
	}

	if value <= target {
		return &ValidationError{
			Field:   fieldName,
			Tag:     "gt",
			Value:   param,
			Message: fmt.Sprintf("%s must be greater than %s", fieldName, param),
		}
	}
	return nil
}

// validateGte checks if value is greater than or equal to param.
func validateGte(fieldName string, val reflect.Value, param string) *ValidationError {
	target, err := strconv.ParseFloat(param, 64)
	if err != nil {
		return nil
	}

	var value float64
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value = float64(val.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value = float64(val.Uint())
	case reflect.Float32, reflect.Float64:
		value = val.Float()
	default:
		return nil
	}

	if value < target {
		return &ValidationError{
			Field:   fieldName,
			Tag:     "gte",
			Value:   param,
			Message: fmt.Sprintf("%s must be at least %s", fieldName, param),
		}
	}
	return nil
}

// validateLt checks if value is less than param.
func validateLt(fieldName string, val reflect.Value, param string) *ValidationError {
	target, err := strconv.ParseFloat(param, 64)
	if err != nil {
		return nil
	}

	var value float64
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value = float64(val.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value = float64(val.Uint())
	case reflect.Float32, reflect.Float64:
		value = val.Float()
	default:
		return nil
	}

	if value >= target {
		return &ValidationError{
			Field:   fieldName,
			Tag:     "lt",
			Value:   param,
			Message: fmt.Sprintf("%s must be less than %s", fieldName, param),
		}
	}
	return nil
}

// validateLte checks if value is less than or equal to param.
func validateLte(fieldName string, val reflect.Value, param string) *ValidationError {
	target, err := strconv.ParseFloat(param, 64)
	if err != nil {
		return nil
	}

	var value float64
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value = float64(val.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value = float64(val.Uint())
	case reflect.Float32, reflect.Float64:
		value = val.Float()
	default:
		return nil
	}

	if value > target {
		return &ValidationError{
			Field:   fieldName,
			Tag:     "lte",
			Value:   param,
			Message: fmt.Sprintf("%s must be at most %s", fieldName, param),
		}
	}
	return nil
}

// isEmpty checks if a reflected value is considered "empty" for validation purposes.
// The definition of empty varies by type:
//   - String: empty string ""
//   - Array/Slice/Map: zero length
//   - Bool: false
//   - Numbers (int/uint/float): zero value
//   - Ptr/Interface: nil
//   - Other types: deep equal to zero value
//
// This is used by the "required" validator to determine if a field has a value.
func isEmpty(val reflect.Value) bool {
	switch val.Kind() {
	case reflect.String:
		return val.String() == ""
	case reflect.Array, reflect.Slice, reflect.Map:
		return val.Len() == 0
	case reflect.Bool:
		return !val.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return val.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return val.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return val.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return val.IsNil()
	default:
		return reflect.DeepEqual(val.Interface(), reflect.Zero(val.Type()).Interface())
	}
}

// ValidateVar validates a single variable against a validation tag string.
// This is useful for validating individual values outside of struct context.
// The field name in errors will be "value".
//
// Example:
//
//	email := "invalid-email"
//	if errs := quark.ValidateVar(email, "required,email"); errs.HasErrors() {
//	    // Returns: value must be a valid email address
//	    return c.BadRequest(errs.Error())
//	}
//
//	age := 200
//	if errs := quark.ValidateVar(age, "gte:0,lte:150"); errs.HasErrors() {
//	    // Returns: value must be at most 150
//	    return c.BadRequest(errs.Error())
//	}
func ValidateVar(value interface{}, tag string) ValidationErrors {
	val := reflect.ValueOf(value)
	validators := strings.Split(tag, ",")

	var errors ValidationErrors
	for _, validator := range validators {
		validator = strings.TrimSpace(validator)
		if validator == "" {
			continue
		}

		var name, param string
		if idx := strings.Index(validator, ":"); idx != -1 {
			name = validator[:idx]
			param = validator[idx+1:]
		} else {
			name = validator
		}

		if err := applyValidator("value", val, name, param); err != nil {
			errors = append(errors, *err)
		}
	}

	return errors
}

package quark

import (
	"testing"
)

func TestValidateRequired(t *testing.T) {
	type Input struct {
		Name  string `validate:"required"`
		Email string `validate:"required"`
	}

	tests := []struct {
		name      string
		input     Input
		expectErr bool
		errField  string
	}{
		{
			name:      "valid",
			input:     Input{Name: "John", Email: "john@example.com"},
			expectErr: false,
		},
		{
			name:      "missing name",
			input:     Input{Name: "", Email: "john@example.com"},
			expectErr: true,
			errField:  "Name",
		},
		{
			name:      "missing email",
			input:     Input{Name: "John", Email: ""},
			expectErr: true,
			errField:  "Email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(tt.input)

			if tt.expectErr {
				if !errs.HasErrors() {
					t.Error("expected validation errors")
				}
				found := false
				for _, e := range errs {
					if e.Field == tt.errField {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error for field %s", tt.errField)
				}
			} else {
				if errs.HasErrors() {
					t.Errorf("unexpected errors: %v", errs)
				}
			}
		})
	}
}

func TestValidateMinMax(t *testing.T) {
	type Input struct {
		Name string `validate:"min:2,max:10"`
		Age  int    `validate:"min:0,max:150"`
	}

	tests := []struct {
		name      string
		input     Input
		expectErr bool
	}{
		{
			name:      "valid",
			input:     Input{Name: "John", Age: 30},
			expectErr: false,
		},
		{
			name:      "name too short",
			input:     Input{Name: "J", Age: 30},
			expectErr: true,
		},
		{
			name:      "name too long",
			input:     Input{Name: "JohnJohnJohn", Age: 30},
			expectErr: true,
		},
		{
			name:      "age too low",
			input:     Input{Name: "John", Age: -5},
			expectErr: true,
		},
		{
			name:      "age too high",
			input:     Input{Name: "John", Age: 200},
			expectErr: true,
		},
		{
			name:      "boundary values valid",
			input:     Input{Name: "Jo", Age: 0},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(tt.input)
			if tt.expectErr && !errs.HasErrors() {
				t.Error("expected validation errors")
			}
			if !tt.expectErr && errs.HasErrors() {
				t.Errorf("unexpected errors: %v", errs)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	type Input struct {
		Email string `validate:"email"`
	}

	tests := []struct {
		name      string
		email     string
		expectErr bool
	}{
		{"valid email", "user@example.com", false},
		{"valid email with subdomain", "user@mail.example.com", false},
		{"invalid email no @", "userexample.com", true},
		{"invalid email no domain", "user@", true},
		{"empty email", "", false}, // empty is OK, use required for mandatory
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(Input{Email: tt.email})
			if tt.expectErr && !errs.HasErrors() {
				t.Error("expected validation errors")
			}
			if !tt.expectErr && errs.HasErrors() {
				t.Errorf("unexpected errors: %v", errs)
			}
		})
	}
}

func TestValidateAlpha(t *testing.T) {
	type Input struct {
		Name string `validate:"alpha"`
	}

	tests := []struct {
		name      string
		value     string
		expectErr bool
	}{
		{"valid alpha", "John", false},
		{"with numbers", "John123", true},
		{"with space", "John Doe", true},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(Input{Name: tt.value})
			if tt.expectErr && !errs.HasErrors() {
				t.Error("expected validation errors")
			}
			if !tt.expectErr && errs.HasErrors() {
				t.Errorf("unexpected errors: %v", errs)
			}
		})
	}
}

func TestValidateAlphaNum(t *testing.T) {
	type Input struct {
		Username string `validate:"alphanum"`
	}

	tests := []struct {
		name      string
		value     string
		expectErr bool
	}{
		{"valid alphanum", "John123", false},
		{"with underscore", "john_doe", true},
		{"with space", "john doe", true},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(Input{Username: tt.value})
			if tt.expectErr && !errs.HasErrors() {
				t.Error("expected validation errors")
			}
			if !tt.expectErr && errs.HasErrors() {
				t.Errorf("unexpected errors: %v", errs)
			}
		})
	}
}

func TestValidateNumeric(t *testing.T) {
	type Input struct {
		Code string `validate:"numeric"`
	}

	tests := []struct {
		name      string
		value     string
		expectErr bool
	}{
		{"valid numeric", "12345", false},
		{"with letters", "123abc", true},
		{"with dot", "123.45", true},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(Input{Code: tt.value})
			if tt.expectErr && !errs.HasErrors() {
				t.Error("expected validation errors")
			}
			if !tt.expectErr && errs.HasErrors() {
				t.Errorf("unexpected errors: %v", errs)
			}
		})
	}
}

func TestValidateUUID(t *testing.T) {
	type Input struct {
		ID string `validate:"uuid"`
	}

	tests := []struct {
		name      string
		value     string
		expectErr bool
	}{
		{"valid uuid", "550e8400-e29b-41d4-a716-446655440000", false},
		{"invalid format", "550e8400-e29b-41d4-a716", true},
		{"not a uuid", "not-a-uuid", true},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(Input{ID: tt.value})
			if tt.expectErr && !errs.HasErrors() {
				t.Error("expected validation errors")
			}
			if !tt.expectErr && errs.HasErrors() {
				t.Errorf("unexpected errors: %v", errs)
			}
		})
	}
}

func TestValidateOneOf(t *testing.T) {
	type Input struct {
		Status string `validate:"oneof:active pending inactive"`
	}

	tests := []struct {
		name      string
		value     string
		expectErr bool
	}{
		{"valid active", "active", false},
		{"valid pending", "pending", false},
		{"valid inactive", "inactive", false},
		{"invalid value", "deleted", true},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(Input{Status: tt.value})
			if tt.expectErr && !errs.HasErrors() {
				t.Error("expected validation errors")
			}
			if !tt.expectErr && errs.HasErrors() {
				t.Errorf("unexpected errors: %v", errs)
			}
		})
	}
}

func TestValidatePattern(t *testing.T) {
	type Input struct {
		Code string `validate:"pattern:^[A-Z]{3}[0-9]{3}$"`
	}

	tests := []struct {
		name      string
		value     string
		expectErr bool
	}{
		{"valid code", "ABC123", false},
		{"lowercase letters", "abc123", true},
		{"too short", "AB12", true},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(Input{Code: tt.value})
			if tt.expectErr && !errs.HasErrors() {
				t.Error("expected validation errors")
			}
			if !tt.expectErr && errs.HasErrors() {
				t.Errorf("unexpected errors: %v", errs)
			}
		})
	}
}

func TestValidateLen(t *testing.T) {
	type Input struct {
		Code string `validate:"len:6"`
	}

	tests := []struct {
		name      string
		value     string
		expectErr bool
	}{
		{"exact length", "123456", false},
		{"too short", "12345", true},
		{"too long", "1234567", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(Input{Code: tt.value})
			if tt.expectErr && !errs.HasErrors() {
				t.Error("expected validation errors")
			}
			if !tt.expectErr && errs.HasErrors() {
				t.Errorf("unexpected errors: %v", errs)
			}
		})
	}
}

func TestValidateComparison(t *testing.T) {
	type Input struct {
		GtValue  int `validate:"gt:5"`
		GteValue int `validate:"gte:5"`
		LtValue  int `validate:"lt:10"`
		LteValue int `validate:"lte:10"`
	}

	tests := []struct {
		name      string
		input     Input
		expectErr bool
	}{
		{
			name:      "all valid",
			input:     Input{GtValue: 6, GteValue: 5, LtValue: 9, LteValue: 10},
			expectErr: false,
		},
		{
			name:      "gt fails",
			input:     Input{GtValue: 5, GteValue: 5, LtValue: 9, LteValue: 10},
			expectErr: true,
		},
		{
			name:      "gte fails",
			input:     Input{GtValue: 6, GteValue: 4, LtValue: 9, LteValue: 10},
			expectErr: true,
		},
		{
			name:      "lt fails",
			input:     Input{GtValue: 6, GteValue: 5, LtValue: 10, LteValue: 10},
			expectErr: true,
		},
		{
			name:      "lte fails",
			input:     Input{GtValue: 6, GteValue: 5, LtValue: 9, LteValue: 11},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(tt.input)
			if tt.expectErr && !errs.HasErrors() {
				t.Error("expected validation errors")
			}
			if !tt.expectErr && errs.HasErrors() {
				t.Errorf("unexpected errors: %v", errs)
			}
		})
	}
}

func TestValidateMultipleTags(t *testing.T) {
	type Input struct {
		Email string `validate:"required,email"`
		Age   int    `validate:"required,min:18,max:100"`
	}

	tests := []struct {
		name       string
		input      Input
		expectErrs int
	}{
		{
			name:       "all valid",
			input:      Input{Email: "user@example.com", Age: 25},
			expectErrs: 0,
		},
		{
			name:       "multiple errors",
			input:      Input{Email: "invalid", Age: 15},
			expectErrs: 2, // invalid email + age below min
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(tt.input)
			if len(errs) != tt.expectErrs {
				t.Errorf("expected %d errors, got %d: %v", tt.expectErrs, len(errs), errs)
			}
		})
	}
}

func TestValidateJSONFieldName(t *testing.T) {
	type Input struct {
		UserName string `json:"user_name" validate:"required"`
	}

	errs := Validate(Input{UserName: ""})

	if len(errs) == 0 {
		t.Error("expected validation error")
	}
	if errs[0].Field != "user_name" {
		t.Errorf("expected field name 'user_name', got %s", errs[0].Field)
	}
}

func TestValidateNestedStruct(t *testing.T) {
	type Address struct {
		City string `validate:"required"`
	}
	type Person struct {
		Name    string  `validate:"required"`
		Address Address // Nested structs are validated automatically
	}

	errs := Validate(Person{Name: "John", Address: Address{City: ""}})

	found := false
	for _, e := range errs {
		if e.Field == "Address.City" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected nested validation error for Address.City, got: %v", errs)
	}
}

func TestValidateSkipTag(t *testing.T) {
	type Input struct {
		Name   string `validate:"required"`
		Secret string `validate:"-"`
	}

	errs := Validate(Input{Name: "John", Secret: ""})

	if errs.HasErrors() {
		t.Error("expected no errors, Secret should be skipped")
	}
}

func TestValidationErrorsToMap(t *testing.T) {
	type Input struct {
		Name  string `json:"name" validate:"required"`
		Email string `json:"email" validate:"required,email"`
	}

	errs := Validate(Input{})

	errMap := errs.ToMap()
	if _, ok := errMap["name"]; !ok {
		t.Error("expected 'name' in error map")
	}
	if _, ok := errMap["email"]; !ok {
		t.Error("expected 'email' in error map")
	}
}

func TestValidateVar(t *testing.T) {
	errs := ValidateVar("test@example.com", "required,email")
	if errs.HasErrors() {
		t.Errorf("unexpected errors: %v", errs)
	}

	errs = ValidateVar("invalid", "email")
	if !errs.HasErrors() {
		t.Error("expected validation error for invalid email")
	}
}

func TestValidateSlice(t *testing.T) {
	type Input struct {
		Tags []string `validate:"min:1,max:5"`
	}

	tests := []struct {
		name      string
		tags      []string
		expectErr bool
	}{
		{"valid", []string{"a", "b"}, false},
		{"empty", []string{}, true},
		{"too many", []string{"a", "b", "c", "d", "e", "f"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(Input{Tags: tt.tags})
			if tt.expectErr && !errs.HasErrors() {
				t.Error("expected validation errors")
			}
			if !tt.expectErr && errs.HasErrors() {
				t.Errorf("unexpected errors: %v", errs)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	type Input struct {
		Website string `validate:"url"`
	}

	tests := []struct {
		name      string
		url       string
		expectErr bool
	}{
		{"valid http", "http://example.com", false},
		{"valid https", "https://example.com/path", false},
		{"invalid no scheme", "example.com", true},
		{"invalid format", "not a url", true},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate(Input{Website: tt.url})
			if tt.expectErr && !errs.HasErrors() {
				t.Error("expected validation errors")
			}
			if !tt.expectErr && errs.HasErrors() {
				t.Errorf("unexpected errors: %v", errs)
			}
		})
	}
}

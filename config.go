package quark

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Config holds the application configuration.
type Config struct {
	Port            string        `env:"PORT" default:"8080"`
	Host            string        `env:"HOST" default:"0.0.0.0"`
	Environment     string        `env:"ENV" default:"development"`
	Debug           bool          `env:"DEBUG" default:"false"`
	ReadTimeout     time.Duration `env:"READ_TIMEOUT" default:"30s"`
	WriteTimeout    time.Duration `env:"WRITE_TIMEOUT" default:"30s"`
	IdleTimeout     time.Duration `env:"IDLE_TIMEOUT" default:"120s"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" default:"30s"`
}

// IsDevelopment returns true if running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development" || c.Environment == "dev"
}

// IsProduction returns true if running in production mode.
func (c *Config) IsProduction() bool {
	return c.Environment == "production" || c.Environment == "prod"
}

// IsTest returns true if running in test mode.
func (c *Config) IsTest() bool {
	return c.Environment == "test" || c.Environment == "testing"
}

// LoadConfig loads configuration from environment variables into the Config struct.
func LoadConfig() (*Config, error) {
	cfg := &Config{}
	if err := LoadFromEnv(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// LoadFromEnv loads configuration from environment variables into any struct.
// It uses the `env` tag to map environment variables and `default` tag for defaults.
//
// Supported types: string, bool, int, int64, uint, uint64, float64, time.Duration
//
// Example:
//
//	type MyConfig struct {
//	    DatabaseURL string        `env:"DATABASE_URL" default:"postgres://localhost/db"`
//	    MaxRetries  int           `env:"MAX_RETRIES" default:"3"`
//	    Timeout     time.Duration `env:"TIMEOUT" default:"10s"`
//	}
func LoadFromEnv(cfg interface{}) error {
	v := reflect.ValueOf(cfg)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("cfg must be a non-nil pointer to a struct")
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("cfg must be a pointer to a struct")
	}

	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		envKey := field.Tag.Get("env")
		if envKey == "" {
			// If no env tag, try to load nested struct
			if fieldValue.Kind() == reflect.Struct {
				if err := LoadFromEnv(fieldValue.Addr().Interface()); err != nil {
					return err
				}
			}
			continue
		}

		defaultValue := field.Tag.Get("default")
		value := os.Getenv(envKey)

		if value == "" {
			value = defaultValue
		}

		if value == "" {
			continue
		}

		if err := setField(fieldValue, value); err != nil {
			return fmt.Errorf("failed to set field %s: %w", field.Name, err)
		}
	}

	return nil
}

// setField sets a reflect.Value from a string.
func setField(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)

	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(b)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Special case for time.Duration
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			d, err := time.ParseDuration(value)
			if err != nil {
				return err
			}
			field.SetInt(int64(d))
		} else {
			i, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return err
			}
			field.SetInt(i)
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(u)

	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(f)

	case reflect.Slice:
		// Handle string slices (comma-separated)
		if field.Type().Elem().Kind() == reflect.String {
			parts := strings.Split(value, ",")
			for i := range parts {
				parts[i] = strings.TrimSpace(parts[i])
			}
			field.Set(reflect.ValueOf(parts))
		}

	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}

	return nil
}

// Env returns an environment variable with a default value.
func Env(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// EnvInt returns an environment variable as an int with a default value.
func EnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return i
}

// EnvInt64 returns an environment variable as an int64 with a default value.
func EnvInt64(key string, defaultValue int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	i, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return defaultValue
	}
	return i
}

// EnvBool returns an environment variable as a bool with a default value.
func EnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	b, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return b
}

// EnvDuration returns an environment variable as a time.Duration with a default value.
func EnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return d
}

// EnvSlice returns an environment variable as a slice of strings (comma-separated).
func EnvSlice(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parts := strings.Split(value, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// MustEnv returns an environment variable or panics if not set.
func MustEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("required environment variable not set: %s", key))
	}
	return value
}

// RequireEnv checks that all required environment variables are set.
func RequireEnv(keys ...string) error {
	var missing []string
	for _, key := range keys {
		if os.Getenv(key) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	return nil
}

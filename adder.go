// Package adder provides a lightweight configuration library for Go, inspired by
// [viper]. It reads YAML config files into Go structs with support for
// environment variable overrides.
//
// Use the package-level functions with the default instance for simple cases:
//
//	adder.SetConfigName("application")
//	adder.SetConfigType("yaml")
//	adder.AddConfigPath(".")
//	adder.ReadInConfig()
//
//	var cfg Config
//	adder.Unmarshal(&cfg)
//
// Or create separate instances with [New] for independent configurations.
//
// [viper]: https://github.com/spf13/viper
package adder

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Adder manages configuration loaded from YAML files with optional environment
// variable overrides. Use [New] to create an instance, or use the package-level
// functions which operate on a default instance.
type Adder struct {
	configName   string
	configType   string
	configPaths  []string
	envReplacer  *strings.Replacer
	autoEnv      bool
	envBindings  map[string]string
	configValues map[string]any
}

// New returns a new Adder instance with empty configuration.
func New() *Adder {
	return &Adder{
		configPaths:  []string{},
		envBindings:  make(map[string]string),
		configValues: make(map[string]any),
	}
}

var defaultAdder = New()

// SetConfigName sets the config filename without extension (e.g. "application").
func SetConfigName(name string) { defaultAdder.SetConfigName(name) }

// SetConfigName sets the config filename without extension (e.g. "application").
func (a *Adder) SetConfigName(name string) {
	a.configName = name
}

// SetConfigType sets the config file format. Supported values: "yaml", "yml".
func SetConfigType(typ string) { defaultAdder.SetConfigType(typ) }

// SetConfigType sets the config file format. Supported values: "yaml", "yml".
func (a *Adder) SetConfigType(typ string) {
	a.configType = typ
}

// AddConfigPath adds a directory to the list of paths to search for the config file.
// Paths are searched in the order they are added.
func AddConfigPath(path string) { defaultAdder.AddConfigPath(path) }

// AddConfigPath adds a directory to the list of paths to search for the config file.
// Paths are searched in the order they are added.
func (a *Adder) AddConfigPath(path string) {
	a.configPaths = append(a.configPaths, path)
}

// SetEnvKeyReplacer sets a [strings.Replacer] for mapping config keys to environment
// variable names. For example, strings.NewReplacer(".", "_") maps "http.port" to "HTTP_PORT".
func SetEnvKeyReplacer(r *strings.Replacer) { defaultAdder.SetEnvKeyReplacer(r) }

// SetEnvKeyReplacer sets a [strings.Replacer] for mapping config keys to environment
// variable names. For example, strings.NewReplacer(".", "_") maps "http.port" to "HTTP_PORT".
func (a *Adder) SetEnvKeyReplacer(r *strings.Replacer) {
	a.envReplacer = r
}

// AutomaticEnv enables automatic environment variable overrides. When enabled,
// [Adder.Unmarshal] checks for an environment variable for each config key before
// using the value from the config file. Use [SetEnvKeyReplacer] to control how
// config keys are mapped to environment variable names.
func AutomaticEnv() { defaultAdder.AutomaticEnv() }

// AutomaticEnv enables automatic environment variable overrides. When enabled,
// [Adder.Unmarshal] checks for an environment variable for each config key before
// using the value from the config file. Use [Adder.SetEnvKeyReplacer] to control how
// config keys are mapped to environment variable names.
func (a *Adder) AutomaticEnv() {
	a.autoEnv = true
}

// BindEnv explicitly binds a config key to a specific environment variable.
// The key uses dot notation for nested fields (e.g. "db.url").
// Explicit bindings take precedence over [AutomaticEnv].
func BindEnv(key string, envVar string) error { return defaultAdder.BindEnv(key, envVar) }

// BindEnv explicitly binds a config key to a specific environment variable.
// The key uses dot notation for nested fields (e.g. "db.url").
// Explicit bindings take precedence over [Adder.AutomaticEnv].
func (a *Adder) BindEnv(key string, envVar string) error {
	a.envBindings[strings.ToLower(key)] = envVar
	return nil
}

// ReadInConfig searches the configured paths for the config file and loads it.
// [SetConfigName], [SetConfigType], and [AddConfigPath] must be called before this.
func ReadInConfig() error { return defaultAdder.ReadInConfig() }

// ReadInConfig searches the configured paths for the config file and loads it.
// [Adder.SetConfigName], [Adder.SetConfigType], and [Adder.AddConfigPath] must be called before this.
func (a *Adder) ReadInConfig() error {
	if a.configName == "" {
		return fmt.Errorf("config name not set")
	}

	var configFile string
	for _, path := range a.configPaths {
		candidate := filepath.Join(path, a.configName+"."+a.configType)
		if _, err := os.Stat(candidate); err == nil {
			configFile = candidate
			break
		}
	}

	if configFile == "" {
		return fmt.Errorf("config file not found: %s.%s", a.configName, a.configType)
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	switch a.configType {
	case "yaml", "yml":
		if err := yaml.Unmarshal(data, &a.configValues); err != nil {
			return fmt.Errorf("failed to parse yaml: %w", err)
		}
	default:
		return fmt.Errorf("unsupported config type: %s", a.configType)
	}

	return nil
}

// Unmarshal decodes the loaded configuration into a struct. The target must be
// a non-nil pointer to a struct. Fields are matched by lowercase name or by
// the "mapstructure" struct tag. Environment variable overrides are applied
// during unmarshalling.
func Unmarshal(v any) error { return defaultAdder.Unmarshal(v) }

// Unmarshal decodes the loaded configuration into a struct. The target must be
// a non-nil pointer to a struct. Fields are matched by lowercase name or by
// the "mapstructure" struct tag. Environment variable overrides are applied
// during unmarshalling.
func (a *Adder) Unmarshal(v any) error {
	return a.unmarshalWithPath(a.configValues, v, "")
}

func (a *Adder) unmarshalWithPath(data map[string]any, v any, prefix string) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("unmarshal target must be a non-nil pointer")
	}

	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("unmarshal target must be a pointer to struct")
	}

	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		fieldValue := rv.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		// Get field name from mapstructure tag or use lowercase field name
		fieldName := strings.ToLower(field.Name)
		if tag := field.Tag.Get("mapstructure"); tag != "" {
			fieldName = tag
		}

		fullKey := fieldName
		if prefix != "" {
			fullKey = prefix + "." + fieldName
		}

		// Check for env override
		if envVal := a.getEnvValue(fullKey); envVal != "" {
			if err := setFieldFromString(fieldValue, envVal); err != nil {
				return err
			}
			continue
		}

		// Get value from config
		configVal, exists := data[fieldName]
		if !exists {
			continue
		}

		if err := a.setFieldValue(fieldValue, configVal, fullKey); err != nil {
			return err
		}
	}

	return nil
}

func (a *Adder) getEnvValue(key string) string {
	lowerKey := strings.ToLower(key)

	// Check explicit bindings first
	if envVar, ok := a.envBindings[lowerKey]; ok {
		return os.Getenv(envVar)
	}

	// Check automatic env
	if a.autoEnv {
		envKey := strings.ToUpper(key)
		if a.envReplacer != nil {
			envKey = a.envReplacer.Replace(envKey)
		}
		return os.Getenv(envKey)
	}

	return ""
}

func (a *Adder) setFieldValue(field reflect.Value, value any, keyPath string) error {
	if value == nil {
		return nil
	}

	switch field.Kind() {
	case reflect.Struct:
		if m, ok := value.(map[string]any); ok {
			return a.unmarshalWithPath(m, field.Addr().Interface(), keyPath)
		}
	case reflect.String:
		if s, ok := value.(string); ok {
			field.SetString(s)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch v := value.(type) {
		case int:
			field.SetInt(int64(v))
		case int64:
			field.SetInt(v)
		case float64:
			field.SetInt(int64(v))
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch v := value.(type) {
		case int:
			if v >= 0 {
				field.SetUint(uint64(v))
			}
		case int64:
			if v >= 0 {
				field.SetUint(uint64(v))
			}
		case uint:
			field.SetUint(uint64(v))
		case uint64:
			field.SetUint(v)
		case float64:
			if v >= 0 {
				field.SetUint(uint64(v))
			}
		}
	case reflect.Bool:
		if b, ok := value.(bool); ok {
			field.SetBool(b)
		}
	case reflect.Slice:
		return setSliceField(field, value)
	}

	return nil
}

func setFieldFromString(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(u)
	case reflect.Bool:
		field.SetBool(value == "true" || value == "1")
	}
	return nil
}

func setSliceField(field reflect.Value, value any) error {
	slice, ok := value.([]any)
	if !ok {
		return nil
	}

	elemType := field.Type().Elem()
	newSlice := reflect.MakeSlice(field.Type(), len(slice), len(slice))

	for i, item := range slice {
		elem := newSlice.Index(i)
		switch elemType.Kind() {
		case reflect.String:
			if s, ok := item.(string); ok {
				elem.SetString(s)
			}
		case reflect.Int, reflect.Int64:
			switch v := item.(type) {
			case int:
				elem.SetInt(int64(v))
			case float64:
				elem.SetInt(int64(v))
			}
		}
	}

	field.Set(newSlice)
	return nil
}

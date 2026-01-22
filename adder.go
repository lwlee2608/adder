package adder

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

type Adder struct {
	configName   string
	configType   string
	configPaths  []string
	envReplacer  *strings.Replacer
	autoEnv      bool
	envBindings  map[string]string
	configValues map[string]any
}

func New() *Adder {
	return &Adder{
		configPaths:  []string{},
		envBindings:  make(map[string]string),
		configValues: make(map[string]any),
	}
}

var defaultAdder = New()

func SetConfigName(name string) { defaultAdder.SetConfigName(name) }
func (a *Adder) SetConfigName(name string) {
	a.configName = name
}

func SetConfigType(typ string) { defaultAdder.SetConfigType(typ) }
func (a *Adder) SetConfigType(typ string) {
	a.configType = typ
}

func AddConfigPath(path string) { defaultAdder.AddConfigPath(path) }
func (a *Adder) AddConfigPath(path string) {
	a.configPaths = append(a.configPaths, path)
}

func SetEnvKeyReplacer(r *strings.Replacer) { defaultAdder.SetEnvKeyReplacer(r) }
func (a *Adder) SetEnvKeyReplacer(r *strings.Replacer) {
	a.envReplacer = r
}

func AutomaticEnv() { defaultAdder.AutomaticEnv() }
func (a *Adder) AutomaticEnv() {
	a.autoEnv = true
}

func BindEnv(key string, envVar string) error { return defaultAdder.BindEnv(key, envVar) }
func (a *Adder) BindEnv(key string, envVar string) error {
	a.envBindings[strings.ToLower(key)] = envVar
	return nil
}

func ReadInConfig() error { return defaultAdder.ReadInConfig() }
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

func Unmarshal(v any) error { return defaultAdder.Unmarshal(v) }
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
		var i int64
		fmt.Sscanf(value, "%d", &i)
		field.SetInt(i)
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

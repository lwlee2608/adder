// Package adder provides a lightweight configuration library for Go. It reads
// YAML config files into Go structs with support for environment variable overrides.
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
package adder

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var durationType = reflect.TypeOf(time.Duration(0))

// Adder manages configuration loaded from YAML files with optional environment
// variable overrides. Use [New] to create an instance, or use the package-level
// functions which operate on a default instance.
type Adder struct {
	configFile   string
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

// SetConfigFile calls [Adder.SetConfigFile] on the default instance.
func SetConfigFile(path string) { defaultAdder.SetConfigFile(path) }

// SetConfigFile sets the exact config file path to use, bypassing the
// name/type/path search. The file is read directly by [Adder.ReadInConfig].
func (a *Adder) SetConfigFile(path string) {
	a.configFile = path
}

// SetConfigName calls [Adder.SetConfigName] on the default instance.
func SetConfigName(name string) { defaultAdder.SetConfigName(name) }

// SetConfigName sets the config filename without extension (e.g. "application").
func (a *Adder) SetConfigName(name string) {
	a.configName = name
}

// SetConfigType calls [Adder.SetConfigType] on the default instance.
func SetConfigType(typ string) { defaultAdder.SetConfigType(typ) }

// SetConfigType sets the config file format. Supported values: "yaml", "yml".
func (a *Adder) SetConfigType(typ string) {
	a.configType = strings.ToLower(typ)
}

// AddConfigPath calls [Adder.AddConfigPath] on the default instance.
func AddConfigPath(path string) { defaultAdder.AddConfigPath(path) }

// AddConfigPath adds a directory to the list of paths to search for the config file.
// Paths are searched in the order they are added.
func (a *Adder) AddConfigPath(path string) {
	a.configPaths = append(a.configPaths, path)
}

// SetEnvKeyReplacer calls [Adder.SetEnvKeyReplacer] on the default instance.
func SetEnvKeyReplacer(r *strings.Replacer) { defaultAdder.SetEnvKeyReplacer(r) }

// SetEnvKeyReplacer sets a [strings.Replacer] for mapping config keys to environment
// variable names. For example, strings.NewReplacer(".", "_") maps "http.port" to "HTTP_PORT".
func (a *Adder) SetEnvKeyReplacer(r *strings.Replacer) {
	a.envReplacer = r
}

// AutomaticEnv calls [Adder.AutomaticEnv] on the default instance.
func AutomaticEnv() { defaultAdder.AutomaticEnv() }

// AutomaticEnv enables automatic environment variable overrides. When enabled,
// [Adder.Unmarshal] checks for an environment variable for each config key before
// using the value from the config file. Use [Adder.SetEnvKeyReplacer] to control how
// config keys are mapped to environment variable names.
//
// Slice elements are addressed by index, so CLIENTS_1_TOKEN overrides
// clients[1].token. An index beyond the elements in the config file appends a new
// element, so a list can be defined entirely by the environment.
func (a *Adder) AutomaticEnv() {
	a.autoEnv = true
}

// BindEnv calls [Adder.BindEnv] on the default instance.
func BindEnv(key string, envVar string) error { return defaultAdder.BindEnv(key, envVar) }

// BindEnv explicitly binds a config key to a specific environment variable.
// The key uses dot notation for nested fields (e.g. "db.url").
// Explicit bindings take precedence over [Adder.AutomaticEnv].
func (a *Adder) BindEnv(key string, envVar string) error {
	a.envBindings[strings.ToLower(key)] = envVar
	return nil
}

// ReadInConfig calls [Adder.ReadInConfig] on the default instance.
func ReadInConfig() error { return defaultAdder.ReadInConfig() }

// ReadInConfig searches the configured paths for the config file and loads it.
// Struct field matching is case-insensitive, so YAML keys like "baseURL", "baseUrl",
// and "baseurl" all match the same struct field. Map keys preserve their original casing.
// Either [Adder.SetConfigFile] or [Adder.SetConfigName]/[Adder.SetConfigType]/[Adder.AddConfigPath] must be called before this.
func (a *Adder) ReadInConfig() error {
	var configFile string

	if a.configFile != "" {
		if _, err := os.Stat(a.configFile); err != nil {
			return fmt.Errorf("config file not found: %s", a.configFile)
		}
		configFile = a.configFile
		if a.configType == "" {
			ext := strings.TrimPrefix(filepath.Ext(configFile), ".")
			if ext != "" {
				a.configType = strings.ToLower(ext)
			} else {
				a.configType = "yaml"
			}
		}
	} else {
		if a.configName == "" {
			return fmt.Errorf("config name not set")
		}
		for _, path := range a.configPaths {
			for _, ext := range configExtensions(a.configType) {
				candidate := filepath.Join(path, a.configName+"."+ext)
				if _, err := os.Stat(candidate); err == nil {
					configFile = candidate
					break
				}
			}
			if configFile != "" {
				break
			}
		}
		if configFile == "" {
			return fmt.Errorf("config file not found: %s.%s", a.configName, a.configType)
		}
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand ${VAR} references in the raw config (bare $VAR is intentionally not expanded)
	data = []byte(expandEnvBraceOnly(string(data)))

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

var envBraceRe = regexp.MustCompile(`\$\{([^}]+)\}`)

func expandEnvBraceOnly(s string) string {
	return envBraceRe.ReplaceAllStringFunc(s, func(match string) string {
		return os.Getenv(match[2 : len(match)-1])
	})
}

// Unmarshal calls [Adder.Unmarshal] on the default instance.
func Unmarshal(v any) error { return defaultAdder.Unmarshal(v) }

// Unmarshal decodes the loaded configuration into a struct. The target must be
// a non-nil pointer to a struct. Fields are matched by lowercase name or by
// the "mapstructure" struct tag. Environment variable overrides are applied
// during unmarshalling.
func (a *Adder) Unmarshal(v any) error {
	return a.unmarshalWithPath(a.configValues, v, configKey{})
}

func (a *Adder) unmarshalWithPath(data map[string]any, v any, prefix configKey) error {
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

		fullKey := prefix.child(fieldName)

		// Check for env override
		if settableFromString(fieldValue.Type()) {
			if envVal := a.getEnvValue(fullKey); envVal != "" {
				if err := setFieldFromString(fieldValue, envVal, fullKey.path); err != nil {
					return err
				}
				continue
			}
		}

		// Get value from config (case-insensitive lookup)
		configVal, exists := caseInsensitiveLookup(data, fieldName)
		if !exists {
			// Still recurse into struct and slice fields to check env bindings
			switch fieldValue.Kind() {
			case reflect.Struct:
				if err := a.unmarshalWithPath(map[string]any{}, fieldValue.Addr().Interface(), fullKey); err != nil {
					return err
				}
			case reflect.Slice:
				if err := a.setSliceField(fieldValue, nil, fullKey); err != nil {
					return err
				}
			}
			continue
		}

		if err := a.setFieldValue(fieldValue, configVal, fullKey); err != nil {
			return err
		}
	}

	return nil
}

func (a *Adder) getEnvValue(key configKey) string {
	if v := a.lookupEnvValue(key.path); v != "" {
		return v
	}

	// A field inside a slice element also honours the unindexed form, so
	// CLIENTS_TOKEN applies to every element of clients.
	if !key.isIndex && key.stripped != key.path {
		return a.lookupEnvValue(key.stripped)
	}

	return ""
}

func (a *Adder) lookupEnvValue(key string) string {
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

// configKey is a config path plus the same path with slice indexes dropped.
type configKey struct {
	path     string
	stripped string
	isIndex  bool
}

func (k configKey) child(name string) configKey {
	return configKey{path: joinKey(k.path, name), stripped: joinKey(k.stripped, name)}
}

func (k configKey) index(i int) configKey {
	return configKey{path: joinKey(k.path, strconv.Itoa(i)), stripped: k.stripped, isIndex: true}
}

func joinKey(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

func (a *Adder) setFieldValue(field reflect.Value, value any, keyPath configKey) error {
	// Slices and structs are handled even when absent from the config file, so
	// that indexed env overrides can still populate them.
	switch field.Kind() {
	case reflect.Slice:
		return a.setSliceField(field, value, keyPath)
	case reflect.Struct:
		m, _ := value.(map[string]any)
		if m == nil {
			m = map[string]any{}
		}
		return a.unmarshalWithPath(m, field.Addr().Interface(), keyPath)
	}

	if value == nil {
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		if s, ok := value.(string); ok {
			field.SetString(s)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Type() == durationType {
			return setDurationField(field, value, keyPath.path)
		}
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
	case reflect.Float32, reflect.Float64:
		switch v := value.(type) {
		case float64:
			field.SetFloat(v)
		case float32:
			field.SetFloat(float64(v))
		case int:
			field.SetFloat(float64(v))
		case int64:
			field.SetFloat(float64(v))
		}
	case reflect.Bool:
		if b, ok := value.(bool); ok {
			field.SetBool(b)
		}
	case reflect.Map:
		m, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		mapType := field.Type()
		if mapType.Key().Kind() != reflect.String || mapType.Elem().Kind() != reflect.String {
			return fmt.Errorf("unsupported map type %s at %s: only map[string]string is supported", mapType, keyPath.path)
		}
		newMap := reflect.MakeMap(mapType)
		for k, v := range m {
			newMap.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(fmt.Sprintf("%v", v)))
		}
		field.Set(newMap)
	}

	return nil
}

func setFieldFromString(field reflect.Value, value string, keyPath string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if field.Type() == durationType {
			d, err := time.ParseDuration(value)
			if err != nil {
				return fmt.Errorf("invalid duration at %s: %w", keyPath, err)
			}
			field.SetInt(int64(d))
			return nil
		}
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
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(f)
	case reflect.Bool:
		field.SetBool(value == "true" || value == "1")
	}
	return nil
}

func setDurationField(field reflect.Value, value any, keyPath string) error {
	switch v := value.(type) {
	case string:
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("invalid duration at %s: %w", keyPath, err)
		}
		field.SetInt(int64(d))
	case int:
		field.SetInt(int64(v))
	case int64:
		field.SetInt(v)
	case float64:
		field.SetInt(int64(v))
	default:
		return fmt.Errorf("cannot convert %T to time.Duration at %s", value, keyPath)
	}
	return nil
}

func caseInsensitiveLookup(m map[string]any, key string) (any, bool) {
	if v, ok := m[key]; ok {
		return v, true
	}
	lower := strings.ToLower(key)
	for k, v := range m {
		if strings.ToLower(k) == lower {
			return v, true
		}
	}
	return nil, false
}

// setSliceField populates a slice field from the config value, applying indexed
// environment variable overrides (e.g. CLIENTS_0_NAME for clients[0].name).
// Elements not present in the config file are appended when env vars define
// them, so a list can be supplied entirely by the environment.
func (a *Adder) setSliceField(field reflect.Value, value any, keyPath configKey) error {
	items, fromConfig := value.([]any)
	elemType := field.Type().Elem()

	length := len(items)
	// Without a config value the existing field holds caller-supplied
	// defaults; env overrides must not truncate them.
	if !fromConfig && field.Len() > length {
		length = field.Len()
	}
	for a.hasEnvForIndex(keyPath, length, elemType) {
		length++
	}

	if length == 0 {
		if fromConfig {
			field.Set(reflect.MakeSlice(field.Type(), 0, 0))
		}
		return nil
	}

	newSlice := reflect.MakeSlice(field.Type(), length, length)
	if !fromConfig {
		reflect.Copy(newSlice, field)
	}
	for i := 0; i < length; i++ {
		var item any
		if i < len(items) {
			item = items[i]
		}

		elem := newSlice.Index(i)
		elemKey := keyPath.index(i)

		switch elem.Kind() {
		case reflect.Struct, reflect.Slice, reflect.Map:
			if err := a.setFieldValue(elem, item, elemKey); err != nil {
				return err
			}
		default:
			if envVal := a.getEnvValue(elemKey); envVal != "" {
				if err := setFieldFromString(elem, envVal, elemKey.path); err != nil {
					return err
				}
				continue
			}
			if err := a.setFieldValue(elem, item, elemKey); err != nil {
				return err
			}
		}
	}

	field.Set(newSlice)
	return nil
}

// hasEnvForIndex reports whether an indexed env var sets a settable field of the
// element at index. Unindexed fallbacks never count, so they cannot append.
func (a *Adder) hasEnvForIndex(keyPath configKey, index int, elemType reflect.Type) bool {
	return a.hasEnvForType(keyPath.index(index), elemType, 0)
}

// maxProbeSliceDepth stops recursive types like Node{Children []Node}.
const maxProbeSliceDepth = 8

func (a *Adder) hasEnvForType(key configKey, t reflect.Type, sliceDepth int) bool {
	switch t.Kind() {
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" {
				continue
			}

			name := strings.ToLower(field.Name)
			if tag := field.Tag.Get("mapstructure"); tag != "" {
				name = tag
			}
			if a.hasEnvForType(key.child(name), field.Type, sliceDepth) {
				return true
			}
		}
		return false
	case reflect.Slice:
		if sliceDepth >= maxProbeSliceDepth {
			return false
		}
		return a.hasEnvForType(key.index(0), t.Elem(), sliceDepth+1)
	default:
		return settableFromString(t) && a.lookupEnvValue(key.path) != ""
	}
}

// settableFromString must match the kinds setFieldFromString handles.
func settableFromString(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

func configExtensions(configType string) []string {
	switch configType {
	case "yaml", "yml":
		return []string{"yaml", "yml"}
	default:
		return []string{configType}
	}
}

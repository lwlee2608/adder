package adder

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
)

const maskChar = "*"

type maskRule struct {
	valid bool
	first int
	last  int
}

// PrettyJSON returns an indented JSON string with sensitive fields masked.
//
// String struct fields tagged with `mask:"..."` are masked using the provided
// options. Supported tag formats are:
//   - `mask:"true"` for full masking
//   - `mask:"first=N"` to keep the first N runes
//   - `mask:"last=N"` to keep the last N runes
//   - `mask:"first=N,last=M"` to keep both ends
//
// Masking is length-preserving and rune-aware. The input value is not modified.
func PrettyJSON(v any) (string, error) {
	masked := maskSensitiveCopy(v)
	b, err := json.MarshalIndent(masked, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func maskSensitiveCopy(v any) any {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return nil
	}

	masked := cloneAndMaskValue(rv)
	if !masked.IsValid() {
		return nil
	}

	return masked.Interface()
}

func cloneAndMaskValue(v reflect.Value) reflect.Value {
	switch v.Kind() {
	case reflect.Struct:
		cp := reflect.New(v.Type()).Elem()
		cp.Set(v)
		maskStruct(cp)
		return cp
	case reflect.Ptr:
		if v.IsNil() {
			return v
		}
		elem := v.Elem()
		if elem.Kind() != reflect.Struct {
			return v
		}
		cp := reflect.New(elem.Type())
		cp.Elem().Set(elem)
		maskStruct(cp.Elem())
		return cp
	default:
		return v
	}
}

func maskStruct(v reflect.Value) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		fieldType := t.Field(i)
		if !fieldType.IsExported() {
			continue
		}

		field := v.Field(i)

		switch field.Kind() {
		case reflect.Struct:
			maskStruct(field)
		case reflect.Ptr:
			if !field.IsNil() && field.Elem().Kind() == reflect.Struct {
				cp := reflect.New(field.Elem().Type())
				cp.Elem().Set(field.Elem())
				maskStruct(cp.Elem())
				field.Set(cp)
			}
		}

		if field.Kind() != reflect.String {
			continue
		}

		rule, shouldMask := parseMaskTag(fieldType.Tag.Get("mask"))
		if !shouldMask {
			continue
		}

		field.SetString(maskString(field.String(), rule))
	}
}

func parseMaskTag(tag string) (maskRule, bool) {
	tag = strings.TrimSpace(tag)
	if tag == "" || strings.EqualFold(tag, "false") {
		return maskRule{}, false
	}
	if strings.EqualFold(tag, "true") {
		return maskRule{valid: true}, true
	}

	rule := maskRule{valid: true}
	for _, part := range strings.Split(tag, ",") {
		part = strings.TrimSpace(part)
		if part == "" || strings.EqualFold(part, "true") {
			continue
		}
		if strings.EqualFold(part, "false") {
			return maskRule{}, false
		}

		key, val, ok := strings.Cut(part, "=")
		if !ok {
			rule.valid = false
			continue
		}

		key = strings.ToLower(strings.TrimSpace(key))
		val = strings.TrimSpace(val)

		n, err := strconv.Atoi(val)
		if err != nil || n < 0 {
			rule.valid = false
			continue
		}

		switch key {
		case "first":
			rule.first = n
		case "last":
			rule.last = n
		default:
			rule.valid = false
		}
	}

	return rule, true
}

func maskString(s string, rule maskRule) string {
	runes := []rune(s)
	n := len(runes)
	if n == 0 {
		return ""
	}

	if !rule.valid {
		return strings.Repeat(maskChar, n)
	}

	keepFirst := min(rule.first, n)
	keepLast := min(rule.last, n)

	if keepFirst+keepLast >= n {
		total := n - 1
		if total <= 0 {
			keepFirst, keepLast = 0, 0
		} else {
			keepFirst = min(keepFirst, total)
			keepLast = min(keepLast, total-keepFirst)
		}
	}

	maskedCount := n - keepFirst - keepLast
	if maskedCount <= 0 {
		return strings.Repeat(maskChar, n)
	}

	var b strings.Builder
	b.Grow(n)
	if keepFirst > 0 {
		b.WriteString(string(runes[:keepFirst]))
	}
	b.WriteString(strings.Repeat(maskChar, maskedCount))
	if keepLast > 0 {
		b.WriteString(string(runes[n-keepLast:]))
	}

	return b.String()
}

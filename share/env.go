/*

Copyright 2018-2025 David Walter.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

*/

package share

import (
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
)

// applyEnvironmentVariables applies environment variables to config struct fields
// It converts field names to UPPER_SNAKE_CASE (e.g., RestrictedCIDR -> RESTRICTED_CIDR)
// Only applies env vars if the field still has its default value (from struct tag)
func applyEnvironmentVariables(config interface{}) {
	val := reflect.ValueOf(config)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !fieldType.IsExported() || !field.CanSet() {
			continue
		}

		// Handle nested structs
		if field.Kind() == reflect.Struct {
			applyEnvironmentVariables(field.Addr().Interface())
			continue
		}

		// Convert field name to UPPER_SNAKE_CASE
		envName := camelToUpperSnake(fieldType.Name)

		// Check if environment variable exists
		if envValue, exists := os.LookupEnv(envName); exists {
			// Check if field has a default tag
			defaultValue := fieldType.Tag.Get("default")

			// Only apply env var if:
			// 1. There's no default tag, OR
			// 2. The current value matches the default (meaning user didn't override via flag)
			shouldApply := false
			if defaultValue == "" {
				shouldApply = true
			} else {
				currentValue := getFieldValueAsString(field)
				if currentValue == defaultValue {
					shouldApply = true
				}
			}

			if shouldApply {
				if err := setFieldValue(field, envValue); err != nil {
					log.Printf("Warning: Failed to set %s from environment variable %s: %v", fieldType.Name, envName, err)
				} else {
					log.Printf("Applied environment variable %s=%s", envName, envValue)
				}
			}
		}
	}
}

// getFieldValueAsString returns the field value as a string
func getFieldValueAsString(field reflect.Value) string {
	switch field.Kind() {
	case reflect.String:
		return field.String()
	case reflect.Bool:
		return strconv.FormatBool(field.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(field.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(field.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(field.Float(), 'f', -1, 64)
	}
	return ""
}

// setFieldValue sets a field value from a string
func setFieldValue(field reflect.Value, value string) error {
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
	}
	return nil
}

// camelToUpperSnake converts CamelCase to UPPER_SNAKE_CASE
func camelToUpperSnake(s string) string {
	if s == "" {
		return ""
	}

	var result strings.Builder
	runes := []rune(s)

	for i, r := range runes {
		// Add underscore before uppercase letter if:
		// 1. Not at start
		// 2. Previous character was lowercase OR next character is lowercase (for acronyms)
		if i > 0 && r >= 'A' && r <= 'Z' {
			prevLower := i > 0 && runes[i-1] >= 'a' && runes[i-1] <= 'z'
			nextLower := i < len(runes)-1 && runes[i+1] >= 'a' && runes[i+1] <= 'z'

			if prevLower || nextLower {
				result.WriteRune('_')
			}
		}

		result.WriteRune(r)
	}

	return strings.ToUpper(result.String())
}

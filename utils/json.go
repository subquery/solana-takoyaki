package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

func parseTag(jsonTag string) (string, bool) {
	parts := strings.Split(jsonTag, ",")

	omitEmpty := false
	if len(parts) > 1 && parts[1] == "omitempty" {
		omitEmpty = true
	}

	return parts[0], omitEmpty
}

// MarshalWithEmptySlices ensures nil slices are omitted while empty slices are serialized as [].
func MarshalWithEmptySlices(v interface{}) ([]byte, error) {
	val := reflect.ValueOf(v)

	// Ensure we're working with a struct pointer
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %T", v)
	}

	result := make(map[string]interface{})

	// Iterate through the struct fields
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := val.Type().Field(i)
		jsonTag := fieldType.Tag.Get("json")

		jsonName, jsonOmit := parseTag(jsonTag)

		// Ignore fields with `-` JSON tags
		if jsonName == "-" {
			continue
		}

		// Use the JSON tag name or fallback to field name
		fieldName := fieldType.Name
		if jsonName != "" {
			fieldName = jsonName
		}

		// Handle slice fields separately
		if field.Kind() == reflect.Slice {
			if field.IsNil() && jsonOmit {
				continue // Omit nil slices
			}
			if field.Len() == 0 {
				result[fieldName] = []interface{}{} // Ensure empty slices are `[]`
				continue
			}
		}

		// Recursively process nested structs
		if field.Kind() == reflect.Struct {
			processedStruct, err := MarshalWithEmptySlices(field.Interface())
			if err != nil {
				return nil, err
			}
			var nestedMap map[string]interface{}
			if err := json.Unmarshal(processedStruct, &nestedMap); err != nil {
				return nil, err
			}
			result[fieldName] = nestedMap
			continue
		}

		// Process pointers to structs
		if field.Kind() == reflect.Ptr && field.Elem().Kind() == reflect.Struct {
			if field.IsNil() {
				if jsonOmit {
					continue // Omit nil struct pointers
				}
				result[fieldName] = nil
			} else {
				processedStruct, err := MarshalWithEmptySlices(field.Elem().Interface())
				if err != nil {
					return nil, err
				}
				var nestedMap map[string]interface{}
				if err := json.Unmarshal(processedStruct, &nestedMap); err != nil {
					return nil, err
				}
				result[fieldName] = nestedMap
			}
			continue
		}

		// Process maps
		if field.Kind() == reflect.Map && field.IsValid() && jsonOmit && field.Len() == 0 {
			continue
		}

		// Process nil pointers
		if field.Kind() == reflect.Ptr && field.IsNil() && jsonOmit {
			continue
		}

		// Add non-slice fields as they are
		result[fieldName] = field.Interface()
	}

	return json.Marshal(result)
}

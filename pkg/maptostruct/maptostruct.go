package maptostruct

import (
	"fmt"
	"reflect"
	"strconv"
)

func MapToStruct(data map[string]string, result interface{}) error {
	val := reflect.ValueOf(result).Elem()
	typ := val.Type()

	// Iterate over struct fields and assign values from the Redis hash map
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		// Get the field name from the "redis" tag or struct field name
		fieldName := field.Tag.Get("redis")
		if fieldName == "" {
			fieldName = field.Name
		}

		// Fetch the corresponding value from Redis hash
		value, ok := data[fieldName]
		if !ok {
			continue // Skip fields not in Redis
		}

		// Assign the value to the struct field based on the field type
		switch fieldValue.Kind() {
		case reflect.String:
			fieldValue.SetString(value)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if num, err := strconv.ParseUint(value, 10, 64); err == nil {
				fieldValue.SetUint(num)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if num, err := strconv.ParseInt(value, 10, 64); err == nil {
				fieldValue.SetInt(num)
			}
		case reflect.Float32, reflect.Float64:
			if num, err := strconv.ParseFloat(value, 64); err == nil {
				fieldValue.SetFloat(num)
			}
		default:
			return fmt.Errorf("unsupported field type: %v", fieldValue.Kind())
		}
	}

	return nil
}

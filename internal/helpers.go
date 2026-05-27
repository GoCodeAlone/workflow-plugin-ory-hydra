package internal

import (
	"encoding/json"
	"fmt"
)

func stringValue(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	switch value := values[key].(type) {
	case string:
		return value
	default:
		return ""
	}
}

func intValue(values map[string]any, key string, fallback int) int {
	if values == nil {
		return fallback
	}
	switch value := values[key].(type) {
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return fallback
	}
}

func boolValue(values map[string]any, keys ...string) (bool, bool) {
	if values == nil {
		return false, false
	}
	for _, key := range keys {
		switch value := values[key].(type) {
		case bool:
			return value, true
		case string:
			switch value {
			case "true", "1", "yes", "on":
				return true, true
			case "false", "0", "no", "off":
				return false, true
			}
		}
	}
	return false, false
}

func mapValue(values map[string]any, key string) map[string]any {
	if values == nil {
		return nil
	}
	value, ok := values[key].(map[string]any)
	if !ok {
		return nil
	}
	return value
}

func stringSliceValue(values map[string]any, key string) []string {
	if values == nil {
		return nil
	}
	switch value := values[key].(type) {
	case []string:
		return append([]string(nil), value...)
	case []any:
		out := make([]string, 0, len(value))
		for _, item := range value {
			if text, ok := item.(string); ok && text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}

func mergeMaps(sources ...map[string]any) map[string]any {
	merged := map[string]any{}
	for _, source := range sources {
		for key, value := range source {
			merged[key] = value
		}
	}
	return merged
}

func decodeMap[T any](values map[string]any, target *T) error {
	data, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("marshal input: %w", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode input: %w", err)
	}
	return nil
}

func encodeValue(value any) (map[string]any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal output: %w", err)
	}
	out := map[string]any{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode output: %w", err)
	}
	return out, nil
}

func encodeAny(value any) (any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal output: %w", err)
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode output: %w", err)
	}
	return out, nil
}

func encodeList(key string, value any) (map[string]any, error) {
	encoded, err := encodeValue(value)
	if err != nil {
		return nil, err
	}
	return map[string]any{key: encoded}, nil
}

func errResult(err error) map[string]any {
	return map[string]any{"error": err.Error()}
}

func stringPtrValue(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func boolPtrValue(value *bool) any {
	if value == nil {
		return nil
	}
	return *value
}

func compactMap(values map[string]any) map[string]any {
	for key, value := range values {
		if value == nil {
			delete(values, key)
		}
	}
	return values
}

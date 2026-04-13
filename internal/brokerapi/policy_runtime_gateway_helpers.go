package brokerapi

import "strings"

func firstString(values []string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func toInterfaceMap(input map[string]any) map[string]interface{} {
	out := map[string]interface{}{}
	for key, value := range input {
		out[key] = value
	}
	return out
}

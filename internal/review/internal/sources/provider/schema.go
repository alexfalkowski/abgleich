package provider

// Schema builds the shared strict argument schema used by read-only source tools.
// Line bounds are nullable; paths is an array; other fields are strings.
func Schema(names ...string) map[string]any {
	properties := make(map[string]any, len(names))
	for _, name := range names {
		switch name {
		case "paths":
			properties[name] = map[string]any{"type": "array", "items": map[string]any{"type": "string"}}
		case "start_line", "end_line":
			properties[name] = map[string]any{"type": []string{"integer", "null"}}
		default:
			properties[name] = map[string]any{"type": "string"}
		}
	}

	return map[string]any{"type": "object", "properties": properties, "required": names, "additionalProperties": false}
}

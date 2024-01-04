package editorjs

import (
	`encoding/json`
	`html/template`
)

func FromString(input string) (template.JS, error) {
	block := map[string]interface{}{
		"type": "paragraph",
		"data": map[string]string{
			"text": input,
		},
	}
	jsonData := map[string]interface{}{
		"blocks": []interface{}{block},
	}
	rawJson, json_err := json.Marshal(jsonData)
	return template.JS(rawJson), json_err
}

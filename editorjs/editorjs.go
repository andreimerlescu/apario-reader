package editorjs

import (
	`encoding/json`
	"log"
)

func ParseEditorJSON(editorJS string) EditorJS {
	var result EditorJS

	err := json.Unmarshal([]byte(editorJS), &result)
	if err != nil {
		log.Println(err)
	}

	return result
}

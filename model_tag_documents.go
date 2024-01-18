package main

import (
	`path/filepath`
)

// TagDocuments will be loaded from reading <tags.db>/tag/kind/documents.json
type TagDocuments struct {
	TSTag
	Identifiers []string `json:"i"`
}

func (td *TagDocuments) Save() error {
	err := td.database_document.Lock()
	if err != nil {
		return err
	}
	defer td.database_document.Unlock()
	return write_to_file(filepath.Join(*flag_s_tag_database_path, td.TSTag.Tag, td.TSTag.Kind, "documents.json"), td)
}

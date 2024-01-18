package main

import (
	`path/filepath`
)

// TagPages will be loaded from reading <tags.db>/tag/kind/pages.json
type TagPages struct {
	TSTag
	Identifiers []string `json:"i"`
}

func (tp *TagPages) Save() error {
	err := tp.database_document.Lock()
	if err != nil {
		return err
	}
	defer tp.database_document.Unlock()
	return write_to_file(filepath.Join(*flag_s_tag_database_path, tp.TSTag.Tag, tp.TSTag.Kind, "pages.json"), tp)
}

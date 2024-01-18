package main

import (
	`path/filepath`
)

// TSTag will store its struct to disk as <tags.db>/tag/kind/tag.json
type TSTag struct {
	Tag  string `json:"t"`
	Kind string `json:"k"`
	database_document
}

func (tst *TSTag) Save() error {
	err := tst.database_document.Lock()
	if err != nil {
		return err
	}
	defer tst.database_document.Unlock()
	return write_to_file(filepath.Join(*flag_s_tag_database_path, tst.Tag, tst.Kind, "tag.json"), tst)
}

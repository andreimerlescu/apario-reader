package main

import (
	`path/filepath`
)

// TagAliases will be loaded from reading <tags.db>/tag/kind/aliases.json
type TagAliases struct {
	TSTag
	Aliases []TSTag `json:"a"`
}

func (ta *TagAliases) Save() error {
	err := ta.database_document.Lock()
	if err != nil {
		return err
	}
	defer ta.database_document.Unlock()
	return write_to_file(filepath.Join(*flag_s_tag_database_path, ta.TSTag.Tag, ta.TSTag.Kind, "aliases.json"), ta)
}

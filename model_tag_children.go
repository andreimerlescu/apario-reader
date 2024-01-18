package main

import (
	`path/filepath`
)

// TagChildren will be loaded from reading <tags.db>/tag/kind/children.json
type TagChildren struct {
	TSTag
	Children []TSTag `json:"children"`
}

func (tt *TagChildren) Save() error { // this saves children
	err := tt.database_document.Lock()
	if err != nil {
		return err
	}
	defer tt.database_document.Unlock()
	return write_to_file(filepath.Join(*flag_s_tag_database_path, tt.TSTag.Tag, tt.TSTag.Kind, "children.json"), tt)
}

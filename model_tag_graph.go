package main

// TagGraph is not exported to any database file but is used to map the memory relationship of all of the TSTag(s) that exist.
type TagGraph struct {
	Tag      *TSTag   `json:"t"`
	Parent   *TSTag   `json:"p"` // root node when &SmartTag.Parent == &SmartTag.Tag
	Aliases  []*TSTag `json:"a"`
	Children []*TSTag `json:"c"`
}

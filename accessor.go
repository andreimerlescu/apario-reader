package main

import (
	`encoding/json`
	`errors`
	`fmt`
	`os`
	`path/filepath`
)

// store tags into *flag_s_tags_database_path
// <tags.db>/tag/kind/tag.json = json.Marshal(TSTag)

type ts_all_page_versions struct {
	IdentifierVersions map[string]string   `json:"identifier_versions"` // map[identifier]version
	VersionIdentifiers map[string][]string `json:"version_identifiers"` // map[version][]identifiers
}

// version_exists_in_database_path assumes that you're going to specify the bulk of <path>/versions/<version>.json where
// the path is the database_path and the version is formatted as v0.0.0 to use with ParseDDVersion(). When it returns
// nil it means there is no version requested.
func version_exists_in_database_path(version string, database_path string) (*DDVersion, error) {
	v, v_err := ParseDDVersion(version)
	if v_err != nil {
		return nil, v_err
	}
	path := filepath.Join(database_path, fmt.Sprintf("%s.json", v.String()))
	info, info_err := os.Stat(path)
	if info_err != nil {
		return nil, info_err
	}
	if info.Size() > 0 {
		return v, nil
	}
	return nil, errors.New("entry is 0 bytes")
}

func read_payload_from_database(path string) (any, error) {
	info, info_err := os.Stat(path)
	if info_err != nil {
		return nil, info_err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("expect identifier_path to be a directory")
	}
	bytes, bytes_err := os.ReadFile(path)
	if bytes_err != nil {
		return nil, bytes_err
	}
	var payload any
	err := json.Unmarshal(bytes, &payload)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

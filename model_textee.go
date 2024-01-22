package main

import (
	`encoding/json`
	`errors`
	`os`
	`path/filepath`
	`strconv`

	go_gematria `github.com/andreimerlescu/go-gematria`
)

type TexteeSubstring struct {
	Substring string `json:"s"`
	// kinds are "document", "page", "snippet", "tag"
	Identifiers map[string]string `json:"i"` // map[Kind]Identifier eg Identifiers["document"] = "202300ABCDEF"
}

// TSTextee will save to <textee.db>/<gematria.English>/<gematria.Jewish>/<gematria.Simple>/textee.json
type TSTextee struct {
	database_document
	Gematria   go_gematria.Gematria       `json:"g"` // required to use .Load()
	Substrings map[string]TexteeSubstring `json:"s"`
}

func (t *TSTextee) Load() error {
	err := t.database_document.RLock()
	if err != nil {
		return err
	}
	defer t.database_document.RUnlock()

	if t.Gematria.English == 0 && t.Gematria.Jewish == 0 && t.Gematria.Simple == 0 && len(t.Substrings) == 0 {
		return errors.New(".Load() requires the .Gematria value to be defined")
	}
	gem_path := filepath.Join(strconv.Itoa(int(t.Gematria.English)), strconv.Itoa(int(t.Gematria.Jewish)), strconv.Itoa(int(t.Gematria.Simple)))
	path := filepath.Join(*flag_s_textee_database_path, gem_path, "textee.json")
	info, info_err := os.Stat(path)
	if info_err != nil {
		return info_err
	}

	if info.IsDir() || info.Size() == 0 {
		return errors.New("invalid path for textee save due to size or directory status")
	}

	bytes, bytes_err := os.ReadFile(path)
	if bytes_err != nil {
		return bytes_err
	}

	// temporary destination
	maybe_textee := &TSTextee{}

	// load file into memory
	json_err := json.Unmarshal(bytes, maybe_textee)
	if json_err != nil {
		return json_err
	}

	bytes = nil // flush bytes from read file

	// authenticity check
	if maybe_textee.Gematria.English != t.Gematria.English {
		return errors.New("english gematria mismatch")
	}

	if maybe_textee.Gematria.Jewish != t.Gematria.Jewish {
		return errors.New("jewish gematria mismatch")
	}

	if maybe_textee.Gematria.Simple != t.Gematria.Simple {
		return errors.New("simple gematria mismatch")
	}

	// load identifiers into struct
	t.Substrings = maybe_textee.Substrings
	maybe_textee.Substrings = nil // flush the bulk data from the maybe load
	return nil
}

func (t *TSTextee) Save() error {
	err := t.database_document.Lock()
	if err != nil {
		return err
	}
	defer t.database_document.Unlock()

	return write_to_file(filepath.Join(*flag_s_textee_database_path, strconv.Itoa(int(t.Gematria.English)), strconv.Itoa(int(t.Gematria.Jewish)), strconv.Itoa(int(t.Gematria.Simple))), t)
}

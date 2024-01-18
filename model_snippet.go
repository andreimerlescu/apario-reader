package main

import (
	`encoding/json`
	`fmt`
	`log`
	`path/filepath`
	`time`

	go_textee `github.com/andreimerlescu/go-textee`
)

// TSSnippet will be loaded from reading <snippets.db>/<identifier-path>/snippet.json
type TSSnippet struct {
	database_document
	Identifier         string            `json:"i"`
	Version            DDVersion         `json:"v"`
	DocumentIdentifier string            `json:"di"`
	PageIdentifier     string            `json:"pi"`
	X                  int               `json:"x"`
	Y                  int               `json:"y"`
	H                  int               `json:"h"`
	W                  int               `json:"w"`
	Title              string            `json:"t"`
	Description        string            `json:"d"`
	OCRText            string            `json:"o"`
	Metadata           map[string]string `json:"m"`
	Textee             *go_textee.Textee
}

func (tss *TSSnippet) Save() error {
	err := tss.database_document.Lock()
	if err != nil {
		return err
	}
	defer tss.database_document.Unlock()

	// snippet version control
	// for snippet_version file will be = arg2=[<snippets.db>/<identifier-path>]/versions/arg1=[<version>].json
	snippet_db_path := filepath.Join(*flag_s_snippets_database, identifier_to_path(tss.Identifier))
	version, version_err := version_exists_in_database_path(tss.Version.String(), snippet_db_path)
	if version_err != nil {
		log.Printf("%v", version_err)
	}

	// no version exists on disk, lets save this document to the disk
	if version == nil {
		// no version exists on disk
		// perform a version bump and backup
		dd := database_document{}
		dd.is_safe() // ensure that .Save() can run
		bytes, bytes_err := json.Marshal(tss)
		if bytes_err != nil {
			return bytes_err
		}
		checksum := Sha256(string(bytes))
		sv := &SnippetVersion{
			database_document: dd,               // this is unique to this SnippetVersion struct
			Identifier:        tss.Identifier,   // this is the snippet identifier
			DateCreated:       time.Now().UTC(), // ensure that .UTC() is used always
			Checksum:          checksum,         // checksum can be used to provide basic sanity checking but not secure checking
			Version:           tss.Version,      // dont touch the version here
			Snippet:           *tss,             // dont sent the pointer of the tss, send the actual tss data
		}
		sv_err := sv.Save() // persist struct as json to disk
		if sv_err != nil {
			log.Printf("failed to save the SnippetVersion due to err %v", sv_err)
			return sv_err
		}
	}
	return write_to_file(filepath.Join(snippet_db_path, "snippet.json"), tss)
}

// SnippetVersion is <snippets.db>/<identifier-path>/versions/<version>.json
type SnippetVersion struct {
	database_document
	Identifier  string    `json:"identifier"`
	DateCreated time.Time `json:"date_created"`
	Checksum    string    `json:"checksum"`
	Version     DDVersion `json:"version"`
	Snippet     TSSnippet `json:"snippet"`
}

func (sv *SnippetVersion) Save() error {
	err := sv.database_document.Lock()
	if err != nil {
		return err
	}
	defer sv.database_document.Unlock()
	return write_to_file(filepath.Join(*flag_s_snippets_database, identifier_to_path(sv.Identifier), "versions", fmt.Sprintf("%s.json", sv.Version.String())), sv)
}

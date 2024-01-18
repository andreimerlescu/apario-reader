package main

import (
	`encoding/json`
	`fmt`
	`log`
	`path/filepath`
	`time`
)

// TSDocument is <database>/<document-identifier-path>/document.json
type TSDocument struct {
	database_document
	Collection      string            `json:"c"`
	Identifier      string            `json:"i"`
	Version         DDVersion         `json:"v"`
	PageIdentifiers map[int]string    `json:"pis"` // map[PageNumber]PageIdentifier
	Metadata        map[string]string `json:"m"`   // map[Key]Value
	TotalPages      int               `json:"tp"`
	URL             string            `json:"u"`
	Tags            []TSTag           `json:"t"`
}

// Save reads the value inside the disk and replaces the runtime with the stored value bv
func (tsd *TSDocument) Save() error {
	err := tsd.database_document.Lock()
	if err != nil {
		return err
	}
	defer tsd.database_document.Unlock()

	// document version control
	// for document_version file will be = arg2=[<database>/<document-identifier-path>]/versions/arg1=[<version>].json
	db_path := filepath.Join(*flag_s_database, identifier_to_path(tsd.Identifier))
	version, version_err := version_exists_in_database_path(tsd.Version.String(), db_path)
	if version_err != nil {
		log.Printf("%v", version_err)
	}

	// no version exists on disk, lets save this document to the disk
	if version == nil {
		// no version exists on disk
		// perform a version bump and backup
		dd := database_document{}
		dd.is_safe() // ensure that .Save() can run
		bytes, bytes_err := json.Marshal(tsd)
		if bytes_err != nil {
			return bytes_err
		}
		checksum := Sha256(string(bytes))
		dv := &DocumentVersion{
			database_document: dd,               // this is unique to this DocumentVersion struct
			Identifier:        tsd.Identifier,   // this is the document identifier
			DateCreated:       time.Now().UTC(), // ensure that .UTC() is used always
			Checksum:          checksum,         // checksum can be used to provide basic sanity checking but not secure checking
			Version:           tsd.Version,      // dont touch the version here
			Document:          *tsd,             // dont sent the pointer of the tsd, send the actual tsd data
		}
		dv_err := dv.Save() // persist struct as json to disk
		if dv_err != nil {
			log.Printf("failed to save the DocumentVersion due to err %v", dv_err)
			return dv_err
		}
	}
	return write_to_file(filepath.Join(*flag_s_database, identifier_to_path(tsd.Identifier), "document.json"), tsd)
}

func (tsd *TSDocument) CoverPageIdentifier() string {
	if tsd.PageIdentifiers == nil {
		tsd.PageIdentifiers = make(map[int]string)
		return ""
	}
	return tsd.PageIdentifiers[0]
}

// DocumentVersion is <documents.db>/<identifier-path>/versions/<version>.json
type DocumentVersion struct {
	database_document
	Identifier  string     `json:"identifier"` // document identifier
	DateCreated time.Time  `json:"date_created"`
	Checksum    string     `json:"checksum"`
	Version     DDVersion  `json:"version"`
	Document    TSDocument `json:"document"`
}

func (dv *DocumentVersion) Save() error {
	err := dv.database_document.Lock()
	if err != nil {
		return err
	}
	defer dv.database_document.Unlock()
	return write_to_file(filepath.Join(*flag_s_database, identifier_to_path(dv.Identifier), "versions", fmt.Sprintf("%s.json", dv.Version.String())), dv)
}

package main

import (
	`encoding/json`
	`fmt`
	`log`
	`path/filepath`
	`time`

	go_textee `github.com/andreimerlescu/go-textee`
)

// TSPage is <database>/<document-identifier-path>/pages/<page-identifier-path>/page.json
type TSPage struct {
	database_document
	Identifier             string            `json:"i"`
	Version                DDVersion         `json:"v"`
	DocumentIdentifier     string            `json:"di"`
	Tags                   []TSTag           `json:"t"`
	ExtractedText          string            `json:"ft"`
	OCRText                string            `json:"ot"`
	PageNumber             int               `json:"pn"`
	PreviousPageIdentifier string            `json:"ppi"` // empty string == no previous page
	NextPageIdentifier     string            `json:"npi"` // empty string == no next page
	Metadata               map[string]string `json:"m"`
	ProposalIdentifiers    []string          `json:"pi"`
	OCRTextee              *go_textee.Textee
	ExtractedTextee        *go_textee.Textee
}

func (tsp *TSPage) Save() error {
	err := tsp.database_document.Lock()
	if err != nil {
		return err
	}
	defer tsp.database_document.Unlock()

	// page version control
	// for document_version file will be = arg2=[<database>/<document-identifier-path>/pages/<page-identifier-path>]/versions/arg1=[<version>].json
	db_path := filepath.Join(*flag_s_database, identifier_to_path(tsp.DocumentIdentifier), "pages", identifier_to_path(tsp.Identifier))
	version, version_err := version_exists_in_database_path(tsp.Version.String(), db_path)
	if version_err != nil {
		log.Printf("%v", version_err)
	}

	// no version exists on disk, lets save this document to the disk
	if version == nil {
		// no version exists on disk
		// perform a version bump and backup
		dd := database_document{}
		dd.is_safe() // ensure that .Save() can run
		bytes, bytes_err := json.Marshal(tsp)
		if bytes_err != nil {
			return bytes_err
		}
		checksum := Sha256(string(bytes))
		pv := &PageVersion{
			database_document:  dd,
			PageIdentifier:     tsp.Identifier,
			DocumentIdentifier: tsp.DocumentIdentifier,
			DateCreated:        time.Now().UTC(),
			Checksum:           checksum,
			Version:            tsp.Version,
			Page:               *tsp,
		}
		pv_err := pv.Save() // persist struct as json to disk
		if pv_err != nil {
			log.Printf("failed to save the DocumentVersion due to err %v", pv_err)
			return pv_err
		}
	}
	return write_to_file(filepath.Join(*flag_s_database, identifier_to_path(tsp.DocumentIdentifier), "pages", identifier_to_path(tsp.Identifier), "page.json"), tsp)
}

// PageVersion is <documents.db>/<document-identifier-path>/pages/<page-identifier-path>/versions/<version>.json
type PageVersion struct {
	database_document
	PageIdentifier     string    `json:"page_identifier"` // page identifier
	DocumentIdentifier string    `json:"document_identifier"`
	DateCreated        time.Time `json:"date_created"`
	Checksum           string    `json:"checksum"`
	Version            DDVersion `json:"version"`
	Page               TSPage    `json:"page"`
}

func (pv *PageVersion) Save() error {
	err := pv.database_document.Lock()
	if err != nil {
		return err
	}
	defer pv.database_document.Unlock()
	return write_to_file(filepath.Join(*flag_s_database, identifier_to_path(pv.DocumentIdentifier), "pages", identifier_to_path(pv.PageIdentifier), "versions", fmt.Sprintf("%s.json", pv.Version.String())), pv)
}

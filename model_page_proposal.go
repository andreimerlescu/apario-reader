package main

import (
	`fmt`
	`path/filepath`

	ai `github.com/andreimerlescu/go-apario-identifier`
)

// TSPageProposal will be loaded from reading <documents.db>/<document-identifier-path>/pages/<page-identifier-path>/proposals/<username>.json
type TSPageProposal struct {
	database_document
	// unique identifiers
	Identifier         string    `json:"i"`
	Version            DDVersion `json:"v"`
	DocumentIdentifier string    `json:"id"`
	PageIdentifier     string    `json:"ip"`
	// versions
	DocumentVersion string `json:"dv"`
	PageVersion     string `json:"pv"`
	// author
	Author string `json:"au"` // username of author and value of the .json filename
	Action string `json:"ac"`
	// major release changes
	IsRotate      bool `json:"ir"`
	RotateDegrees int  `json:"rd"`
	// minor release changes
	IsTranslation       bool   `json:"itr"`
	TranslationLanguage string `json:"trl"`
	TranslationText     string `json:"trt"`
	IsTranscription     bool   `json:"itra"`
	TranscriptionKind   string `json:"trak"` // extracted or ocr
	TranscriptionText   string `json:"trat"`
	// patch release changes
	Tags []TSTag `json:"tags"`
}

func (tspp *TSPageProposal) Save() error {
	err := tspp.database_document.Lock()
	if err != nil {
		return err
	}
	defer tspp.database_document.Unlock()
	pid, pidErr := ai.ParseIdentifier(tspp.PageIdentifier)
	did, didErr := ai.ParseIdentifier(tspp.DocumentIdentifier)
	if pidErr != nil {
		return pidErr
	}
	if didErr != nil {
		return didErr
	}
	return write_to_file(filepath.Join(*flag_s_database, did.Path(), "pages", pid.Path(), fmt.Sprintf("%s.json", tspp.Author)), tspp)
}

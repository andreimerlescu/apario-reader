package main

import (
	`log`
	`log/slog`
	`strings`
	`sync`

	go_gematria `github.com/andreimerlescu/go-gematria`
	go_textee `github.com/andreimerlescu/go-textee`
)

type ts_idx_textee_substring struct {
	Gematria            go_gematria.Gematria `json:"g"`
	PageIdentifiers     map[string]int32     `json:"p"` // map[PageIdentifier]Counter
	DocumentIdentifiers map[string]int32     `json:"d"` // map[DocumentIdentifier]PageNumber
}

var m_idx_textee_substring = make(map[string]ts_idx_textee_substring)
var mu_idx_textee_substring = sync.RWMutex{}

func ocr_to_textee_idx(pageIdentifier string, documentIdentifier string, pageNumber uint, ocr string) {
	textee := go_textee.NewTextee(ocr)
	for substring, counter := range textee.Substrings {
		if len(substring) == 0 || counter == nil {
			continue
		}
		gem, gem_err := go_gematria.NewGematria(substring)
		if gem_err != nil {
			log.Printf("received gematria error %v for substring %v", gem_err, substring)
		}

		mu_idx_textee_substring.Lock()
		substring_entry, entry_exists := m_idx_textee_substring[substring]
		if !entry_exists {
			m_idx_textee_substring[substring] = ts_idx_textee_substring{
				Gematria:            gem,
				PageIdentifiers:     map[string]int32{pageIdentifier: counter.Load()},
				DocumentIdentifiers: map[string]int32{documentIdentifier: int32(pageNumber)},
			}
			mu_idx_textee_substring.Unlock()
			continue
		}

		// delete
		delete(m_idx_textee_substring, substring)

		if substring_entry.DocumentIdentifiers == nil {
			substring_entry.DocumentIdentifiers = map[string]int32{documentIdentifier: int32(pageNumber)}
		}
		_, document_exists := substring_entry.DocumentIdentifiers[documentIdentifier]
		if !document_exists {
			substring_entry.DocumentIdentifiers[documentIdentifier] = int32(pageNumber)
		}

		if substring_entry.PageIdentifiers == nil {
			substring_entry.PageIdentifiers = map[string]int32{pageIdentifier: counter.Load()}
		}
		_, page_exists := substring_entry.PageIdentifiers[pageIdentifier]
		if !page_exists {
			substring_entry.PageIdentifiers[pageIdentifier] = counter.Load()
		}

		m_idx_textee_substring[substring] = substring_entry

		mu_idx_textee_substring.Unlock()

		save_textee_gematria(substring, pageIdentifier)
	}
}

func save_textee_gematria(word string, pageIdentifier string) {
	word = strings.ToLower(word)

	word_score, gem_err := go_gematria.NewGematria(word)
	if gem_err != nil {
		return // gem_err
	}

	mu_word_pages.Lock()
	_, word_pages_defined := m_word_pages[word]
	if !word_pages_defined {
		m_word_pages[word] = make(map[string]struct{})
	}
	m_word_pages[word][pageIdentifier] = struct{}{}
	mu_word_pages.Unlock()

	index_textee_word_gematria_against_page_identifier(word, word_score, pageIdentifier)
}

func save_dictionary_textee_index(word string, pageIdentifier string) {
	word = strings.ToLower(word)

	wg := sync.WaitGroup{}
	for _, language := range s_dictionary_languages {
		wg.Add(1)
		go func(wg *sync.WaitGroup, language string, word string, pageIdentifier string) {
			defer wg.Done()
			err := only_permit_dictionary_substring_indexing(language, word, pageIdentifier)
			if err != nil {
				slog.Error("received an error loading %v: %v", language, err)
			}
		}(&wg, language, word, pageIdentifier)
	}
	wg.Wait()
}

func only_permit_dictionary_substring_indexing(language string, word string, pageIdentifier string) error {
	word_score, gem_err := go_gematria.NewGematria(word)
	if gem_err != nil {
		return gem_err
	}
	mu_words.RLock()
	_, language_found := m_words[language]
	if !language_found {
		mu_words.RUnlock()
		return nil // no result, just return and do nothing
	}

	_, word_defined := m_words[language][word]
	if !word_defined {
		mu_words.RUnlock()
		return nil // no result, just return and do nothing
	}
	mu_words.RUnlock()

	// the word/substring is a dictionary word, so lets index it

	mu_word_pages.Lock()
	_, word_pages_defined := m_word_pages[word]
	if !word_pages_defined {
		m_word_pages[word] = make(map[string]struct{})
	}
	m_word_pages[word][pageIdentifier] = struct{}{}
	mu_word_pages.Unlock()

	index_textee_word_gematria_against_page_identifier(word, word_score, pageIdentifier)

	return nil
}

func index_textee_word_gematria_against_page_identifier(word string, word_score go_gematria.Gematria, pageIdentifier string) {
	// english
	mu_page_gematria_english.Lock()
	_, word_english_gematria_defined := m_page_gematria_english[word_score.English]
	if !word_english_gematria_defined {
		m_page_gematria_english[word_score.English] = make(map[string]map[string]struct{})
	}
	_, page_exists_in_english_gematria := m_page_gematria_english[word_score.English][pageIdentifier]
	if !page_exists_in_english_gematria {
		m_page_gematria_english[word_score.English][pageIdentifier] = make(map[string]struct{})
	}
	m_page_gematria_english[word_score.English][pageIdentifier][word] = struct{}{}
	mu_page_gematria_english.Unlock()

	// jewish
	mu_page_gematria_jewish.Lock()
	_, word_jewish_gematria_defined := m_page_gematria_jewish[word_score.Jewish]
	if !word_jewish_gematria_defined {
		m_page_gematria_jewish[word_score.Jewish] = make(map[string]map[string]struct{})
	}
	_, page_exists_in_jewish_gematria := m_page_gematria_jewish[word_score.Jewish][pageIdentifier]
	if !page_exists_in_jewish_gematria {
		m_page_gematria_jewish[word_score.Jewish][pageIdentifier] = make(map[string]struct{})
	}
	m_page_gematria_jewish[word_score.Jewish][pageIdentifier][word] = struct{}{}
	mu_page_gematria_jewish.Unlock()

	// simple
	mu_page_gematria_simple.Lock()
	_, word_simple_gematria_defined := m_page_gematria_simple[word_score.Simple]
	if !word_simple_gematria_defined {
		m_page_gematria_simple[word_score.Simple] = make(map[string]map[string]struct{})
	}
	_, page_exists_in_simple_gematria := m_page_gematria_simple[word_score.Simple][pageIdentifier]
	if !page_exists_in_simple_gematria {
		m_page_gematria_simple[word_score.Simple][pageIdentifier] = make(map[string]struct{})
	}
	m_page_gematria_simple[word_score.Simple][pageIdentifier][word] = struct{}{}
	mu_page_gematria_simple.Unlock()
}

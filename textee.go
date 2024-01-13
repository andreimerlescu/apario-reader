package main

import (
	`log`
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
		mu_idx_textee_substring.RLock()
		substring_entry, entry_exists := m_idx_textee_substring[substring]
		mu_idx_textee_substring.RUnlock()
		if !entry_exists {
			gem, gem_err := go_gematria.NewGematria(substring)
			if gem_err != nil {
				log.Printf("received gematria error %v for substring %v", gem_err, substring)
			}

			mu_idx_textee_substring.Lock()
			m_idx_textee_substring[substring] = ts_idx_textee_substring{
				Gematria:            gem,
				PageIdentifiers:     map[string]int32{pageIdentifier: counter.Load()},
				DocumentIdentifiers: map[string]int32{documentIdentifier: int32(pageNumber)},
			}
			mu_idx_textee_substring.Unlock()
			continue
		}

		mu_idx_textee_substring.Lock()

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

	mu_word_pages.RLock()
	_, word_pages_defined := m_word_pages[word]
	mu_word_pages.RUnlock()
	if !word_pages_defined {
		mu_word_pages.Lock()
		m_word_pages[word] = make(map[string]struct{})
		mu_word_pages.Unlock()
	}

	mu_word_pages.Lock()
	m_word_pages[word][pageIdentifier] = struct{}{}
	mu_word_pages.Unlock()

	word_score := NewGemScore(word)

	// english
	mu_page_gematria_english.RLock()
	_, word_english_gematria_defined := m_page_gematria_english[word_score.English]
	mu_page_gematria_english.RUnlock()
	if !word_english_gematria_defined {
		mu_page_gematria_english.Lock()
		m_page_gematria_english[word_score.English] = make(map[string]map[string]struct{})
		mu_page_gematria_english.Unlock()
	}
	mu_page_gematria_english.RLock()
	_, page_exists_in_english_gematria := m_page_gematria_english[word_score.English][pageIdentifier]
	mu_page_gematria_english.RUnlock()
	if !page_exists_in_english_gematria {
		mu_page_gematria_english.Lock()
		m_page_gematria_english[word_score.English][pageIdentifier] = make(map[string]struct{})
		mu_page_gematria_english.Unlock()
	}
	mu_page_gematria_english.Lock()
	m_page_gematria_english[word_score.English][pageIdentifier][word] = struct{}{}
	mu_page_gematria_english.Unlock()

	// jewish
	mu_page_gematria_jewish.RLock()
	_, word_jewish_gematria_defined := m_page_gematria_jewish[word_score.Jewish]
	mu_page_gematria_jewish.RUnlock()
	if !word_jewish_gematria_defined {
		mu_page_gematria_jewish.Lock()
		m_page_gematria_jewish[word_score.Jewish] = make(map[string]map[string]struct{})
		mu_page_gematria_jewish.Unlock()
	}
	mu_page_gematria_jewish.RLock()
	_, page_exists_in_jewish_gematria := m_page_gematria_jewish[word_score.Jewish][pageIdentifier]
	mu_page_gematria_jewish.RUnlock()
	if !page_exists_in_jewish_gematria {
		mu_page_gematria_jewish.Lock()
		m_page_gematria_jewish[word_score.Jewish][pageIdentifier] = make(map[string]struct{})
		mu_page_gematria_jewish.Unlock()
	}
	mu_page_gematria_jewish.Lock()
	m_page_gematria_jewish[word_score.Jewish][pageIdentifier][word] = struct{}{}
	mu_page_gematria_jewish.Unlock()

	// simple
	mu_page_gematria_simple.RLock()
	_, word_simple_gematria_defined := m_page_gematria_simple[word_score.Simple]
	mu_page_gematria_simple.RUnlock()
	if !word_simple_gematria_defined {
		mu_page_gematria_simple.Lock()
		m_page_gematria_simple[word_score.Simple] = make(map[string]map[string]struct{})
		mu_page_gematria_simple.Unlock()
	}
	mu_page_gematria_simple.RLock()
	_, page_exists_in_simple_gematria := m_page_gematria_simple[word_score.Simple][pageIdentifier]
	mu_page_gematria_simple.RUnlock()
	if !page_exists_in_simple_gematria {
		mu_page_gematria_simple.Lock()
		m_page_gematria_simple[word_score.Simple][pageIdentifier] = make(map[string]struct{})
		mu_page_gematria_simple.Unlock()
	}
	mu_page_gematria_simple.Lock()
	m_page_gematria_simple[word_score.Simple][pageIdentifier][word] = struct{}{}
	mu_page_gematria_simple.Unlock()
}

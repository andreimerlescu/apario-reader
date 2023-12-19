package main

import (
	`encoding/json`
	`fmt`
	`io/fs`
	`log`
	`log/slog`
	`os`
	`path/filepath`
	`strings`
)

func database_load() error {
	directory := *flag_s_database
	if len(directory) == 0 {
		return fmt.Errorf("failed to load database %v", directory)
	}

	wg_active_tasks.Add(1)
	defer wg_active_tasks.Done()

	resolvedPath, symlink_err := resolve_symlink(directory)
	if symlink_err != nil {
		return symlink_err
	}
	directory = resolvedPath

	log.Printf("using directory %v", directory)

	err := filepath.WalkDir(filepath.Join(".", directory, "."), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if strings.HasSuffix(path, "/pages") {
				return nil
			}
			for {
				if ch_db_directories.CanWrite() {
					err := ch_db_directories.Write(path)
					if err != nil {
						return err
					}
					break
				}
			}
		}

		return nil
	})
	if err != nil {
		log.Printf("failed to walk directory %v due to error %v", directory, err)
		return err
	}
	return nil
}

func resolve_symlink(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		log.Printf("error obtaining file information for %v: %v", path, err)
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		resolvedPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			log.Printf("error resolving symlink %v: %v", path, err)
			return "", err
		}
		return resolvedPath, nil
	}
	return path, nil
}

func process_directories(ch <-chan interface{}) {
	for {
		select {
		case i_data, ok := <-ch:
			if ok {
				path, valid := i_data.(string)
				if valid {
					sem_db_directories.Acquire()
					wg_active_tasks.Add(1)
					go analyze_document_directory(path)
				} else {
					log.Printf("failed to typecast i_data %T into string", i_data)
				}
			} else {
				slog.Warn("channel is closed")
			}
		}
	}
}

func analyze_document_directory(path string) {
	defer sem_db_directories.Release()
	defer wg_active_tasks.Done()

	bytes_json_record, record_json_err := os.ReadFile(filepath.Join(path, "record.json"))
	if record_json_err != nil {
		log.Printf("failed to read the file record.json due to %v", record_json_err)
		return
	}

	var record ResultData
	json_err := json.Unmarshal(bytes_json_record, &record)
	if json_err != nil {
		log.Printf("failed to parse the json for %v/record.json due to %v", path, json_err)
		return
	}

	var total_pages uint
	if record.TotalPages > 0 {
		total_pages = uint(record.TotalPages)
	}

	var record_number string
	_, record_number_defined := record.Metadata["record_number"]
	if record_number_defined {
		record_number = record.Metadata["record_number"]
	}

	collection_name, collection_defined := record.Metadata["collection"]
	if collection_defined {
		mu_collections.RLock()
		_, found_collection := m_collections[collection_name]
		mu_collections.RUnlock()
		if !found_collection {
			mu_collections.Lock()
			m_collections[collection_name] = Collection{
				Name:      collection_name,
				Documents: make(map[string]Document),
			}
			mu_collections.Unlock()
		}
	}

	title, title_defined := record.Metadata["title"]
	if !title_defined && len(title) == 0 {
		title = record.Identifier
	}

	mu_collection_documents.RLock()
	_, documents_defined := m_collection_documents[collection_name]
	mu_collection_documents.RUnlock()
	if !documents_defined {
		mu_collection_documents.Lock()
		m_collection_documents[collection_name] = make(map[string]Document)
		mu_collection_documents.Unlock()
	}

	mu_collection_documents.RLock()
	_, document_defined := m_collection_documents[collection_name][record.Identifier]
	mu_collection_documents.RUnlock()
	if !document_defined {
		mu_collection_documents.Lock()
		m_collection_documents[collection_name][record.Identifier] = Document{
			Identifier:   record.Identifier,
			RecordNumber: record_number,
			Pages:        make(map[uint]Page),
			Metadata:     record.Metadata,
			TotalPages:   total_pages,
			Hyperlink:    record.URL,
		}
		mu_collection_documents.Unlock()
	}

	for i := uint(1); i <= total_pages; i++ {
		wg_active_tasks.Add(1)
		sem_analyze_pages.Acquire()
		go analyze_page(record.Identifier, path, i)
	}
}

func analyze_page(record_identifier string, path string, i uint) {
	defer wg_active_tasks.Done()
	defer sem_analyze_pages.Release()

	ocr_path := filepath.Join(path, "pages", fmt.Sprintf("ocr.%06d.txt", i))
	ocr_bytes, ocr_err := os.ReadFile(ocr_path)
	if ocr_err != nil {
		log.Printf("failed to read the ocr_path %v due to error %v", ocr_path, ocr_err)
		return
	}

	page_data_path := filepath.Join(path, "pages", fmt.Sprintf("page.%06d.json", i))
	page_data_bytes, page_data_err := os.ReadFile(page_data_path)
	if page_data_err != nil {
		log.Printf("failed to read the pages JSON data due to error %v", page_data_err)
		return
	}

	var page_data PendingPage
	page_err := json.Unmarshal(page_data_bytes, &page_data)
	if page_err != nil {
		log.Printf("failed to unmarshal the page JSON due to error %v", page_err)
	}

	ocr := string(ocr_bytes)
	gematria := NewGemScore(ocr)

	mu_page_words.RLock()
	_, page_words_defined := m_page_words[page_data.Identifier]
	mu_page_words.RUnlock()
	if !page_words_defined {
		mu_page_words.Lock()
		m_page_words[page_data.Identifier] = make(map[string]struct{})
		mu_page_words.Unlock()
	}

	words := strings.Fields(ocr)
	for _, word := range words {
		word = strings.ToLower(word)

		mu_page_words.Lock()
		m_page_words[page_data.Identifier][word] = struct{}{}
		mu_page_words.Unlock()

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
		_, page_exists_in_english_gematria := m_page_gematria_english[word_score.English][page_data.Identifier]
		mu_page_gematria_english.RUnlock()
		if !page_exists_in_english_gematria {
			mu_page_gematria_english.Lock()
			m_page_gematria_english[word_score.English][page_data.Identifier] = make(map[string]struct{})
			mu_page_gematria_english.Unlock()
		}
		mu_page_gematria_english.Lock()
		m_page_gematria_english[word_score.English][page_data.Identifier][word] = struct{}{}
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
		_, page_exists_in_jewish_gematria := m_page_gematria_jewish[word_score.Jewish][page_data.Identifier]
		mu_page_gematria_jewish.RUnlock()
		if !page_exists_in_jewish_gematria {
			mu_page_gematria_jewish.Lock()
			m_page_gematria_jewish[word_score.Jewish][page_data.Identifier] = make(map[string]struct{})
			mu_page_gematria_jewish.Unlock()
		}
		mu_page_gematria_jewish.Lock()
		m_page_gematria_jewish[word_score.Jewish][page_data.Identifier][word] = struct{}{}
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
		_, page_exists_in_simple_gematria := m_page_gematria_simple[word_score.Simple][page_data.Identifier]
		mu_page_gematria_simple.RUnlock()
		if !page_exists_in_simple_gematria {
			mu_page_gematria_simple.Lock()
			m_page_gematria_simple[word_score.Simple][page_data.Identifier] = make(map[string]struct{})
			mu_page_gematria_simple.Unlock()
		}
		mu_page_gematria_simple.Lock()
		m_page_gematria_simple[word_score.Simple][page_data.Identifier][word] = struct{}{}
		mu_page_gematria_simple.Unlock()
	}

	mu_document_pages.RLock()
	_, pages_defined := m_document_pages[record_identifier]
	mu_document_pages.RUnlock()
	if !pages_defined {
		mu_document_pages.Lock()
		m_document_pages[record_identifier] = make(map[uint]Page)
		mu_document_pages.Unlock()
	}

	mu_document_pages.RLock()
	page, page_defined := m_document_pages[record_identifier][i]
	mu_document_pages.RUnlock()
	if len(page.Identifier) == 0 || !page_defined {
		mu_document_pages.Lock()
		m_document_pages[record_identifier][i] = Page{
			FullText:   ocr,
			PageNumber: i,
			Identifier: page_data.Identifier,
			Gematria:   gematria,
		}
		mu_document_pages.Unlock()
	}
}

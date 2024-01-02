package main

import (
	`bufio`
	`encoding/json`
	`errors`
	`fmt`
	`io/fs`
	`log`
	`log/slog`
	`os`
	`path/filepath`
	`strings`
	`sync`
	`time`
)

func database_load() error {
	directory := *flag_s_database
	if len(directory) == 0 {
		return fmt.Errorf("failed to load database %v", directory)
	}

	wg_active_tasks.Add(1)
	defer wg_active_tasks.Done()

	log.Printf("database_load => directory = %v", directory)

	if f_b_path_is_symlink(directory) {
		resolvedPath, symlink_err := resolve_symlink(directory)
		if symlink_err != nil {
			return symlink_err
		}
		log.Printf("database_load => determined that directory is a symlink to %v", resolvedPath)
		directory = resolvedPath
		log.Printf("database_load => assigned directory = %v", directory)
	}

	log.Printf("database_load => using directory = %v\n", directory)

	err := filepath.WalkDir(filepath.Join(directory, "."), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if strings.HasSuffix(path, string(filepath.Separator)+"pages") {
				return nil
			}
			log.Printf("database_load => filepath.Walk() => sending path %v into the ch_db_directories channel", path)
			timeout := time.NewTicker(30 * time.Second)

			if f_b_path_is_symlink(path) {
				log.Printf("path %v is a symlink ", path)
				var err error
				var new_path string
				new_path, err = resolve_symlink(path)
				if err != nil {
					log.Printf("failed to resolve symblink %v with err %v", path, err)
					return nil
				}
				path = new_path
			}

			document_record_filename := filepath.Join(path, "record.json")
			document_record_info, info_err := os.Stat(document_record_filename)
			if info_err != nil {
				log.Printf("failed to find record.json inside the path %v therefore we are skipping", path)
				return nil
			}

			if document_record_info.Size() == 0 {
				log.Printf("the document_record_info.Size() == 0 for path %v", path)
				return nil
			}

			for {
				select {
				case <-time.Tick(time.Second):
					if ch_db_directories.CanWrite() {
						err := ch_db_directories.Write(path)
						if err != nil {
							return err
						}
						return nil
					}
				case <-timeout.C:
					return nil
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

	document_index := a_i_total_documents.Add(1)

	log.Printf("working on analyze_document_directory(path = %v)", path)

	path = strings.ReplaceAll(path, `mnt/volume_nyc3_01/stargate-tmp/mnt/volume_nyc3_01/stargate-tmp`, `mnt/volume_nyc3_01/stargate-tmp`)

	log.Printf("checking path = %v", path)

	bytes_json_record, record_json_err := os.ReadFile(filepath.Join(path, "record.json"))
	if record_json_err != nil {
		log.Printf("failed to read the file record.json due to %v from path = %v/record.json", record_json_err, path)
		return
	}

	var record ResultData
	json_err := json.Unmarshal(bytes_json_record, &record)
	if json_err != nil {
		log.Printf("failed to parse the json for %v/record.json due to %v", path, json_err)
		return
	}

	if len(record.Identifier) == 0 {
		log.Printf("skipping over path %v due to record.Identifier being 0 bytes", path)
		log.Printf("-> skipped record = %v", record)
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

	mu_document_identifier_directory.RLock()
	_, document_identifier_directory_defined := m_document_identifier_directory[record.Identifier]
	mu_document_identifier_directory.RUnlock()
	if !document_identifier_directory_defined {
		mu_document_identifier_directory.Lock()
		m_document_identifier_directory[record.Identifier] = path
		mu_document_identifier_directory.Unlock()
	}

	mu_document_total_pages.RLock()
	_, document_total_pages_defined := m_document_total_pages[record.Identifier]
	mu_document_total_pages.RUnlock()
	if !document_total_pages_defined {
		mu_document_total_pages.Lock()
		m_document_total_pages[record.Identifier] = total_pages
		mu_document_total_pages.Unlock()
	}

	mu_document_source_url.RLock()
	_, document_source_url_defined := m_document_source_url[record.Identifier]
	mu_document_source_url.RUnlock()
	if !document_source_url_defined {
		mu_document_source_url.Lock()
		m_document_source_url[record.Identifier] = record.URL
		mu_document_source_url.Unlock()
	}

	mu_document_metadata.RLock()
	_, document_metadata_defined := m_document_metadata[record.Identifier]
	mu_document_metadata.RUnlock()
	if !document_metadata_defined {
		mu_document_metadata.Lock()
		m_document_metadata[record.Identifier] = record.Metadata
		mu_document_metadata.Unlock()
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

	mu_document_page_identifiers_pgno.RLock()
	_, document_page_identifiers_pgno_defined := m_document_page_identifiers_pgno[record.Identifier]
	mu_document_page_identifiers_pgno.RUnlock()
	if !document_page_identifiers_pgno_defined {
		mu_document_page_identifiers_pgno.Lock()
		m_document_page_identifiers_pgno[record.Identifier] = make(map[string]uint)
		mu_document_page_identifiers_pgno.Unlock()
	}

	mu_document_pgno_page_identifier.RLock()
	_, document_pgno_page_identifier_defined := m_document_pgno_page_identifier[record.Identifier]
	mu_document_pgno_page_identifier.RUnlock()
	if !document_pgno_page_identifier_defined {
		mu_document_pgno_page_identifier.Lock()
		m_document_pgno_page_identifier[record.Identifier] = make(map[uint]string)
		mu_document_pgno_page_identifier.Unlock()
	}

	mu_index_document_identifier.RLock()
	_, index_document_identifier_defined := m_index_document_identifier[document_index]
	mu_index_document_identifier.RUnlock()
	if !index_document_identifier_defined {
		mu_index_document_identifier.Lock()
		m_index_document_identifier[document_index] = record.Identifier
		mu_index_document_identifier.Unlock()
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

	page_index := a_i_total_pages.Add(1)

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

	if page_data.PageNumber == 1 {
		mu_document_identifier_cover_page_identifier.RLock()
		_, document_identifier_cover_page_identifier_defined := m_document_identifier_cover_page_identifier[record_identifier]
		mu_document_identifier_cover_page_identifier.RUnlock()
		if !document_identifier_cover_page_identifier_defined {
			mu_document_identifier_cover_page_identifier.Lock()
			m_document_identifier_cover_page_identifier[record_identifier] = page_data.Identifier
			mu_document_identifier_cover_page_identifier.Unlock()
		}
	}

	mu_document_page_identifiers_pgno.RLock()
	_, document_page_identifiers_pgno_defined := m_document_page_identifiers_pgno[record_identifier][page_data.Identifier]
	mu_document_page_identifiers_pgno.RUnlock()
	if !document_page_identifiers_pgno_defined {
		mu_document_page_identifiers_pgno.Lock()
		m_document_page_identifiers_pgno[record_identifier][page_data.Identifier] = uint(page_data.PageNumber)
		mu_document_page_identifiers_pgno.Unlock()
	}

	mu_document_pgno_page_identifier.RLock()
	_, document_pgno_page_identifier_defined := m_document_pgno_page_identifier[record_identifier][uint(page_data.PageNumber)]
	mu_document_pgno_page_identifier.RUnlock()
	if !document_pgno_page_identifier_defined {
		mu_document_pgno_page_identifier.Lock()
		m_document_pgno_page_identifier[record_identifier][uint(page_data.PageNumber)] = page_data.Identifier
		mu_document_pgno_page_identifier.Unlock()
	}

	mu_page_identifier_document.RLock()
	_, page_identifier_document_defined := m_page_identifier_document[page_data.Identifier]
	mu_page_identifier_document.RUnlock()
	if !page_identifier_document_defined {
		mu_page_identifier_document.Lock()
		m_page_identifier_document[page_data.Identifier] = record_identifier
		mu_page_identifier_document.Unlock()
	}

	mu_page_identifier_page_number.RLock()
	_, page_identifier_page_number_defined := m_page_identifier_page_number[page_data.Identifier]
	mu_page_identifier_page_number.RUnlock()
	if !page_identifier_page_number_defined {
		mu_page_identifier_page_number.Lock()
		m_page_identifier_page_number[page_data.Identifier] = uint(page_data.PageNumber)
		mu_page_identifier_page_number.Unlock()
	}

	mu_index_page_identifier.RLock()
	existing_entry, page_index_defined := m_index_page_identifier[page_index]
	mu_index_page_identifier.RUnlock()
	if !page_index_defined {
		mu_index_page_identifier.Lock()
		m_index_page_identifier[page_index] = page_data.Identifier
		mu_index_page_identifier.Unlock()
	} else {
		log.Printf("[skipping] found a duplicate m_index_page_identifier[page_index] %d = %v", page_index, existing_entry)
	}

	words := strings.Fields(ocr)
	for _, word := range words {
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
		m_word_pages[word][page_data.Identifier] = struct{}{}
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

	mu_document_page_number_page.RLock()
	_, pages_defined := m_document_page_number_page[record_identifier]
	mu_document_page_number_page.RUnlock()
	if !pages_defined {
		mu_document_page_number_page.Lock()
		m_document_page_number_page[record_identifier] = make(map[uint]Page)
		mu_document_page_number_page.Unlock()
	}

	mu_document_page_number_page.RLock()
	page, page_defined := m_document_page_number_page[record_identifier][i]
	mu_document_page_number_page.RUnlock()
	if len(page.Identifier) == 0 || !page_defined {
		mu_document_page_number_page.Lock()
		m_document_page_number_page[record_identifier][i] = Page{
			Identifier:         page_data.Identifier,
			DocumentIdentifier: record_identifier,
			FullText:           ocr,
			PageNumber:         i,
			Gematria:           gematria,
		}
		mu_document_page_number_page.Unlock()
	}
}

func dump_database_to_disk() {
	if !*flag_b_persist_runtime_database {
		log.Printf("skipping dump_database_to_disk because config.yaml persist-runtime-database is set to false [the default]")
		return
	}
	err := os.MkdirAll(*flag_s_persistent_database_file, 0755)
	if err != nil {
		return
	}

	wg := sync.WaitGroup{}
	go write_payload_to_file("m_cryptonyms.json", &wg, &mu_cryptonyms, &m_cryptonyms)
	go write_payload_to_file("m_collections.json", &wg, &mu_collections, &m_collections)
	go write_payload_to_file("m_collection_documents.json", &wg, &mu_collection_documents, &m_collection_documents)
	go write_payload_to_file("m_word_pages.json", &wg, &mu_word_pages, &m_word_pages)
	go write_payload_to_file("m_document_page_number_page.json", &wg, &mu_document_page_number_page, &m_document_page_number_page)
	go write_payload_to_file("m_document_page_identifiers_pgno.json", &wg, &mu_document_page_identifiers_pgno, &m_document_page_identifiers_pgno)
	go write_payload_to_file("m_document_pgno_page_identifier.json", &wg, &mu_document_pgno_page_identifier, &m_document_pgno_page_identifier)
	go write_payload_to_file("m_page_identifier_document.json", &wg, &mu_page_identifier_document, &m_page_identifier_document)
	go write_payload_to_file("m_page_identifier_page_number.json", &wg, &mu_page_identifier_page_number, &m_page_identifier_page_number)
	go write_payload_to_file("m_document_total_pages.json", &wg, &mu_document_total_pages, &m_document_total_pages)
	go write_payload_to_file("m_document_source_url.json", &wg, &mu_document_source_url, &m_document_source_url)
	go write_payload_to_file("m_document_metadata.json", &wg, &mu_document_metadata, &m_document_metadata)
	go write_payload_to_file("m_index_page_identifier.json", &wg, &mu_index_page_identifier, &m_index_page_identifier)
	go write_payload_to_file("m_index_document_identifier.json", &wg, &mu_index_document_identifier, &m_index_document_identifier)
	go write_payload_to_file("m_document_identifier_directory.json", &wg, &mu_document_identifier_directory, &m_document_identifier_directory)
	go write_payload_to_file("m_document_identifier_cover_page_identifier.json", &wg, &mu_document_identifier_cover_page_identifier, &m_document_identifier_cover_page_identifier)
	go write_payload_to_file("m_page_gematria_english.json", &wg, &mu_page_gematria_english, &m_page_gematria_english)
	go write_payload_to_file("m_page_gematria_jewish.json", &wg, &mu_page_gematria_jewish, &m_page_gematria_jewish)
	go write_payload_to_file("m_page_gematria_simple.json", &wg, &mu_page_gematria_simple, &m_page_gematria_simple)
	//go write_payload_to_file("m_location_cities.json", &wg, &mu_location_cities, &m_location_cities)
	//go write_payload_to_file("m_location_countries.json", &wg, &mu_location_countries, &m_location_countries)
	//go write_payload_to_file("m_location_states.json", &wg, &mu_location_states, &m_location_states)
	wg.Wait()
	log.Println("finished writing database to disk")

}

func restore_database_from_disk() {
	if !can_restore_database_from_disk() {
		log.Printf("cannot restore_database_from_disk due to failed santity check")
		return
	}
	wg := sync.WaitGroup{}
	go load_file_into_payload("m_cryptonyms.json", &wg, &mu_cryptonyms, &m_cryptonyms)
	go load_file_into_payload("m_collections.json", &wg, &mu_collections, &m_collections)
	go load_file_into_payload("m_collection_documents.json", &wg, &mu_collection_documents, &m_collection_documents)
	go load_file_into_payload("m_word_pages.json", &wg, &mu_word_pages, &m_word_pages)
	go load_file_into_payload("m_document_page_number_page.json", &wg, &mu_document_page_number_page, &m_document_page_number_page)
	go load_file_into_payload("m_document_page_identifiers_pgno.json", &wg, &mu_document_page_identifiers_pgno, &m_document_page_identifiers_pgno)
	go load_file_into_payload("m_document_pgno_page_identifier.json", &wg, &mu_document_pgno_page_identifier, &m_document_pgno_page_identifier)
	go load_file_into_payload("m_page_identifier_document.json", &wg, &mu_page_identifier_document, &m_page_identifier_document)
	go load_file_into_payload("m_page_identifier_page_number.json", &wg, &mu_page_identifier_page_number, &m_page_identifier_page_number)
	go load_file_into_payload("m_document_total_pages.json", &wg, &mu_document_total_pages, &m_document_total_pages)
	go load_file_into_payload("m_document_source_url.json", &wg, &mu_document_source_url, &m_document_source_url)
	go load_file_into_payload("m_document_metadata.json", &wg, &mu_document_metadata, &m_document_metadata)
	go load_file_into_payload("m_index_page_identifier.json", &wg, &mu_index_page_identifier, &m_index_page_identifier)
	go load_file_into_payload("m_index_document_identifier.json", &wg, &mu_index_document_identifier, &m_index_document_identifier)
	go load_file_into_payload("m_document_identifier_directory.json", &wg, &mu_document_identifier_directory, &m_document_identifier_directory)
	go load_file_into_payload("m_document_identifier_cover_page_identifier.json", &wg, &mu_document_identifier_cover_page_identifier, &m_document_identifier_cover_page_identifier)
	go load_file_into_payload("m_page_gematria_english.json", &wg, &mu_page_gematria_english, &m_page_gematria_english)
	go load_file_into_payload("m_page_gematria_jewish.json", &wg, &mu_page_gematria_jewish, &m_page_gematria_jewish)
	go load_file_into_payload("m_page_gematria_simple.json", &wg, &mu_page_gematria_simple, &m_page_gematria_simple)
	//go load_file_into_payload("m_location_cities.json", &wg, &mu_location_cities, &m_location_cities)
	//go load_file_into_payload("m_location_countries.json", &wg, &mu_location_countries, &m_location_countries)
	//go load_file_into_payload("m_location_states.json", &wg, &mu_location_states, &m_location_states)
	wg.Wait()
	a_b_locations_loaded.Store(true)
	a_i_total_documents.Store(int64(len(m_document_total_pages)))
	a_i_total_pages.Store(int64(len(m_page_identifier_page_number)))
	log.Println("finished reading database into memory")
	return
}

func f_clear_db_restore_file() {
	file := filepath.Join(".", *flag_s_flush_db_cache_watch_file)
	flush_cache_file_info, flush_cache_file_err := os.Stat(file)
	if flush_cache_file_err != nil {
		log.Printf("cannot remove the %v because it does not exist [%v]", file, flush_cache_file_err)
		return // file not present
	}
	mode := flush_cache_file_info.Mode()
	if mode.IsDir() {
		log.Printf("PROBLEM: the %v file is a directory when the program expected it to be an empty text file [%d bytes]\n", file, flush_cache_file_info.Size())
		return
	} else if (mode & os.ModeSymlink) != 0 {
		log.Printf("PROBLEM: the %v file is a symlink when the program expected it to be an empty text file [%d bytes]\n", file, flush_cache_file_info.Size())
		return
	} else if (mode & os.ModeType) == 0 {
		log.Printf("found a regular file %v and will flush the database cache because this file is present then we'll delete this file after", file)
		rm_rf_err := os.RemoveAll(filepath.Join(*flag_s_persistent_database_file, "*.json"))
		if rm_rf_err != nil {
			log.Printf("error removing the *.json files from %v/* due to err %v", *flag_s_persistent_database_file, rm_rf_err)
			return
		}
		err := os.Remove(file)
		if err != nil {
			log.Printf("failed to remove the %v due to err %v", file, err)
			return
		}
	}
}

func f_b_db_flush_file_set() bool {
	file := filepath.Join(".", *flag_s_flush_db_cache_watch_file)
	flush_cache_file_info, flush_cache_file_err := os.Stat(file)
	if flush_cache_file_err != nil {
		return false // file not present
	}
	mode := flush_cache_file_info.Mode()
	if mode.IsDir() {
		log.Printf("PROBLEM: the %v file is a directory when the program expected it to be an empty text file [%d bytes]\n", file, flush_cache_file_info.Size())
		return false
	} else if (mode & os.ModeSymlink) != 0 {
		log.Printf("PROBLEM: the %v file is a symlink when the program expected it to be an empty text file [%d bytes]\n", file, flush_cache_file_info.Size())
		return false
	} else if (mode & os.ModeType) == 0 {
		log.Printf("found a regular file %v and will flush the database cache because this file is present then we'll delete this file after", file)
		return true
	}

	return false
}

func f_b_path_is_symlink(file string) bool {
	flush_cache_file_info, flush_cache_file_err := os.Stat(file)
	if flush_cache_file_err != nil {
		return false // file not present
	}
	mode := flush_cache_file_info.Mode()

	if (mode & os.ModeSymlink) != 0 {
		return true
	}

	return false
}

func can_restore_database_from_disk() bool {
	return *flag_b_load_persistent_runtime_database &&
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_cryptonyms.json")) &&
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_collections.json")) &&
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_collection_documents.json")) &&
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_word_pages.json")) &&
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_document_page_number_page.json")) &&
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_document_page_identifiers_pgno.json")) &&
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_document_pgno_page_identifier.json")) &&
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_page_identifier_document.json")) &&
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_page_identifier_page_number.json")) &&
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_document_total_pages.json")) &&
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_document_source_url.json")) &&
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_document_metadata.json")) &&
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_index_page_identifier.json")) &&
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_index_document_identifier.json")) &&
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_document_identifier_directory.json")) &&
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_document_identifier_cover_page_identifier.json")) &&
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_page_gematria_english.json")) &&
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_page_gematria_jewish.json")) &&
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_page_gematria_simple.json")) //&&
	//f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_location_cities.json")) &&
	//f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_location_countries.json")) &&
	//f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_location_states.json"))
}

func f_b_path_exists(path string) bool {
	info, info_err := os.Stat(path)
	if errors.Is(info_err, os.ErrNotExist) || errors.Is(info_err, os.ErrPermission) {
		log.Printf("skipping %v due to err %v", path, info_err)
		return false
	}

	if info.Size() == 0 {
		log.Printf("skipping %v due to size = 0 bytes", path)
		return false
	}
	return true
}

func load_file_into_payload(filename string, wg *sync.WaitGroup, mu *sync.RWMutex, payload any) {
	wg.Add(1)
	defer wg.Done()

	path := filepath.Join(*flag_s_persistent_database_file, filename)
	if !f_b_path_exists(path) {
		log.Printf("skipping %v due to size = 0 bytes", path)
		return
	}

	bytes, bytes_err := os.ReadFile(path)
	if bytes_err != nil {
		log.Printf("failed to read file %v due to err %v", path, bytes_err)
		return
	}
	mu.Lock()
	err := json.Unmarshal(bytes, &payload)
	mu.Unlock()
	if err != nil {
		log.Printf("failed to unmarshal bytes for %v due to err %v", filename, err)
		return
	}

	log.Printf("completed loading file %v into the payload\n", filename)
}

func write_payload_to_file(filename string, wg *sync.WaitGroup, mu *sync.RWMutex, payload any) {
	wg.Add(1)
	defer wg.Done()
	file, err := os.Create(filepath.Join(*flag_s_persistent_database_file, filename))
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	bufferedWriter := bufio.NewWriter(file)
	mu.RLock()
	marshal, err := json.Marshal(payload)
	mu.RUnlock()
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}

	_, err = bufferedWriter.Write(marshal)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}

	if err := bufferedWriter.Flush(); err != nil {
		fmt.Println("Error flushing writer:", err)
		return
	}
}

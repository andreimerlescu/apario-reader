package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	go_smartchan "github.com/andreimerlescu/go-smartchan"
)

var (
	db_counter_completed_documents = atomic.Int64{}
	db_counter_completed_pages     = atomic.Int64{}
	db_counter_pending_documents   = atomic.Int64{}
	db_counter_pending_pages       = atomic.Int64{}
)

func database_load() error {
	directory := *flag_s_database
	if len(directory) == 0 {
		return log_boot.TraceReturnf("failed to load database %v", directory)
	}

	wg_active_tasks.Add(1)
	defer wg_active_tasks.Done()

	if f_b_path_is_symlink(directory) {
		resolvedPath, symlink_err := resolve_symlink(directory)
		if symlink_err != nil {
			return symlink_err
		}
		directory = resolvedPath
		log_boot.Printf("database_load => assigned directory = %v", directory)
	}

	walkPath := filepath.Join(directory, ".")
	if !strings.HasPrefix(walkPath, string(os.PathSeparator)) {
		walkPath = filepath.Join(".", walkPath)
	}

	log_boot.Printf("walkPath = %+v", walkPath)

	err := filepath.WalkDir(walkPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if strings.HasSuffix(path, string(filepath.Separator)+"pages") || strings.EqualFold(path, directory) {
				return nil
			}
			timeout := time.NewTicker(30 * time.Second)

			if f_b_path_is_symlink(path) {
				log_boot.Printf("path %v is a symlink ", path)
				var err error
				var new_path string
				new_path, err = resolve_symlink(path)
				if err != nil {
					log_boot.Tracef("failed to resolve symblink %v with err %v", path, err)
					return nil
				}
				path = new_path
			}

			document_record_filename := filepath.Join(path, "record.json")
			document_record_info, info_err := os.Stat(document_record_filename)
			if info_err != nil {
				log_boot.Tracef("failed to find record.json inside the path %v therefore we are skipping", path)
				return nil
			}

			if document_record_info.Size() == 0 {
				log_boot.Tracef("the document_record_info.Size() == 0 for path %v", path)
				return nil
			}

			for {
				select {
				case <-time.Tick(time.Millisecond * 3):
					if ch_db_directories.CanWrite() {
						err := ch_db_directories.Write(path)
						if err != nil {
							return log_boot.TraceReturn(err)
						}
						return nil
					} else {
						return log_boot.TraceReturn(errors.New("cant write to closed ch_db_directories channel"))
					}
				case <-timeout.C:
					return nil
				}

			}
		}

		return nil
	})
	if err != nil {
		log_boot.Tracef("failed to walk directory %v due to error %v", directory, err)
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

func process_directories(ctx context.Context, sch *go_smartchan.SmartChan) {
	for {
		select {
		case <-ctx.Done():
			return
		case i_data, ok := <-sch.Chan():
			if ok {
				path, valid := i_data.(string)
				if valid {
					requestedAt := time.Now().UTC()
					sem_db_directories.Acquire()
					if since := time.Since(requestedAt).Seconds(); since >= 1.0 {
						log_boot.Printf("took %.0f seconds to acquire sem_db_directories queue position", since)
					}
					wg_active_tasks.Add(1)
					go analyze_document_directory(path)
				} else {
					log_debug.Tracef("failed to typecast i_data %T into string", i_data)
				}
			} else {
				log_error.Trace("channel is closed")
			}
		}
	}
}

func analyze_document_directory(path string) {
	defer func() {
		sem_db_directories.Release()
		wg_active_tasks.Done()
		db_counter_completed_documents.Add(1)
	}()
	document_index := a_i_total_documents.Add(1)

	path = strings.ReplaceAll(path, fmt.Sprintf("%v%v", *flag_s_database, *flag_s_database), *flag_s_database)

	db_counter_pending_documents.Add(1)

	bytes_json_record, record_json_err := os.ReadFile(filepath.Join(path, "record.json"))
	if record_json_err != nil {
		log_boot.Tracef("failed to read the file record.json due to %v from path = %v/record.json", record_json_err, path)
		return
	}

	var record ResultData
	json_err := json.Unmarshal(bytes_json_record, &record)
	if json_err != nil {
		log_boot.Tracef("failed to parse the json for %v/record.json due to %v", path, json_err)
		return
	}

	if len(record.Identifier) == 0 {
		log_error.Printf("skipping over path %v due to record.Identifier being 0 bytes", path)
		log_error.Printf("-> skipped record = %v", record)
		return
	}

	var total_pages uint
	if record.TotalPages > 0 {
		total_pages = uint(record.TotalPages)
	}

	if record.Metadata == nil {
		record.Metadata = make(map[string]string)
	}

	var record_number string
	_, record_number_defined := record.Metadata["record_number"]
	if record_number_defined {
		record_number = record.Metadata["record_number"]
	}

	collection_name, collection_defined := record.Metadata["collection"]
	if collection_defined {
		mu_collections.Lock()
		_, found_collection := m_collections[collection_name]
		if !found_collection {
			m_collections[collection_name] = Collection{
				Name:      collection_name,
				Documents: make(map[string]Document),
			}
		}
		mu_collections.Unlock()
	}

	title, title_defined := record.Metadata["title"]
	if !title_defined && len(title) == 0 {
		title = record.Identifier
	}

	mu_document_identifier_directory.Lock()
	_, document_identifier_directory_defined := m_document_identifier_directory[record.Identifier]
	if !document_identifier_directory_defined {
		m_document_identifier_directory[record.Identifier] = path
	}
	mu_document_identifier_directory.Unlock()

	mu_document_total_pages.Lock()
	_, document_total_pages_defined := m_document_total_pages[record.Identifier]
	if !document_total_pages_defined {
		m_document_total_pages[record.Identifier] = total_pages
	}
	mu_document_total_pages.Unlock()

	mu_document_source_url.Lock()
	_, document_source_url_defined := m_document_source_url[record.Identifier]
	if !document_source_url_defined {
		m_document_source_url[record.Identifier] = record.URL
	}
	mu_document_source_url.Unlock()

	mu_document_metadata.Lock()
	_, document_metadata_defined := m_document_metadata[record.Identifier]
	if !document_metadata_defined {
		m_document_metadata[record.Identifier] = record.Metadata
	}
	mu_document_metadata.Unlock()

	mu_collection_documents.Lock()
	_, documents_defined := m_collection_documents[collection_name]
	if !documents_defined {
		m_collection_documents[collection_name] = make(map[string]Document)
	}
	mu_collection_documents.Unlock()

	mu_collection_documents.Lock()
	_, document_defined := m_collection_documents[collection_name][record.Identifier]
	if !document_defined {
		m_collection_documents[collection_name][record.Identifier] = Document{
			Identifier:   record.Identifier,
			RecordNumber: record_number,
			Pages:        make(map[uint]Page),
			Metadata:     record.Metadata,
			TotalPages:   total_pages,
			Hyperlink:    record.URL,
		}
	}
	mu_collection_documents.Unlock()

	mu_document_page_identifiers_pgno.Lock()
	_, document_page_identifiers_pgno_defined := m_document_page_identifiers_pgno[record.Identifier]
	if !document_page_identifiers_pgno_defined {
		m_document_page_identifiers_pgno[record.Identifier] = make(map[string]uint)
	}
	mu_document_page_identifiers_pgno.Unlock()

	mu_document_pgno_page_identifier.Lock()
	_, document_pgno_page_identifier_defined := m_document_pgno_page_identifier[record.Identifier]
	if !document_pgno_page_identifier_defined {
		m_document_pgno_page_identifier[record.Identifier] = make(map[uint]string)
	}
	mu_document_pgno_page_identifier.Unlock()

	mu_index_document_identifier.Lock()
	_, index_document_identifier_defined := m_index_document_identifier[document_index]
	if !index_document_identifier_defined {
		m_index_document_identifier[document_index] = record.Identifier
	}
	mu_index_document_identifier.Unlock()

	for i := uint(1); i <= total_pages; i++ {
		requestedAt := time.Now().UTC()
		sem_analyze_pages.Acquire()
		if since := time.Since(requestedAt).Seconds(); since >= 1.0 {
			log.Printf("took %.0f seconds to acquire sem_analyze_pages queue position", since)
		}
		wg_active_tasks.Add(1)
		go analyze_page(record.Identifier, path, i)
	}
}

func analyze_page(record_identifier string, path string, i uint) {
	defer func() {
		sem_analyze_pages.Release()
		wg_active_tasks.Done()
		db_counter_completed_pages.Add(1)
	}()

	page_index := a_i_total_pages.Add(1)

	db_counter_pending_pages.Add(1)

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

	ocr_to_textee_idx(page_data.Identifier, page_data.RecordIdentifier, uint(page_data.PageNumber), string(ocr_bytes))

	if page_data.PageNumber == 1 {
		mu_document_identifier_cover_page_identifier.Lock()
		_, document_identifier_cover_page_identifier_defined := m_document_identifier_cover_page_identifier[record_identifier]
		if !document_identifier_cover_page_identifier_defined {
			m_document_identifier_cover_page_identifier[record_identifier] = page_data.Identifier
		}
		mu_document_identifier_cover_page_identifier.Unlock()
	}

	mu_document_page_identifiers_pgno.Lock()
	_, document_page_identifiers_pgno_defined := m_document_page_identifiers_pgno[record_identifier][page_data.Identifier]
	if !document_page_identifiers_pgno_defined {
		m_document_page_identifiers_pgno[record_identifier][page_data.Identifier] = uint(page_data.PageNumber)
	}
	mu_document_page_identifiers_pgno.Unlock()

	mu_document_pgno_page_identifier.Lock()
	_, document_pgno_page_identifier_defined := m_document_pgno_page_identifier[record_identifier][uint(page_data.PageNumber)]
	if !document_pgno_page_identifier_defined {
		m_document_pgno_page_identifier[record_identifier][uint(page_data.PageNumber)] = page_data.Identifier
	}
	mu_document_pgno_page_identifier.Unlock()

	mu_page_identifier_document.Lock()
	_, page_identifier_document_defined := m_page_identifier_document[page_data.Identifier]
	if !page_identifier_document_defined {
		m_page_identifier_document[page_data.Identifier] = record_identifier
	}
	mu_page_identifier_document.Unlock()

	mu_page_identifier_page_number.Lock()
	_, page_identifier_page_number_defined := m_page_identifier_page_number[page_data.Identifier]
	if !page_identifier_page_number_defined {
		m_page_identifier_page_number[page_data.Identifier] = uint(page_data.PageNumber)
	}
	mu_page_identifier_page_number.Unlock()

	mu_index_page_identifier.Lock()
	existing_entry, page_index_defined := m_index_page_identifier[page_index]
	if !page_index_defined {
		m_index_page_identifier[page_index] = page_data.Identifier
	} else {
		log.Printf("[skipping] found a duplicate m_index_page_identifier[page_index] %d = %v", page_index, existing_entry)
	}
	mu_index_page_identifier.Unlock()

	mu_document_page_number_page.Lock()
	_, pages_defined := m_document_page_number_page[record_identifier]
	if !pages_defined {
		m_document_page_number_page[record_identifier] = make(map[uint]Page)
	}
	mu_document_page_number_page.Unlock()

	mu_document_page_number_page.Lock()
	page, page_defined := m_document_page_number_page[record_identifier][i]
	if len(page.Identifier) == 0 || !page_defined {
		m_document_page_number_page[record_identifier][i] = Page{
			Identifier:         page_data.Identifier,
			DocumentIdentifier: record_identifier,
			PageNumber:         i,
		}
	}
	mu_document_page_number_page.Unlock()
}

func dump_database_to_disk() {
	if !*flag_b_persist_runtime_database {
		log_boot.Printf("skipping dump_database_to_disk because config.yaml persist-runtime-database is set to false [the default]")
		return
	}
	err := os.MkdirAll(*flag_s_persistent_database_file, 0755)
	if err != nil {
		log_boot.Trace(err)
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
	go write_payload_to_file("m_idx_textee_substring.json", &wg, &mu_idx_textee_substring, &m_idx_textee_substring)
	go write_payload_to_file("m_words.json", &wg, &mu_words, &m_words)
	go write_payload_to_file("m_words_english_gematria_english.json", &wg, &mu_words_english_gematria_english, &m_words_english_gematria_english)
	go write_payload_to_file("m_words_english_gematria_jewish.json", &wg, &mu_words_english_gematria_jewish, &m_words_english_gematria_jewish)
	go write_payload_to_file("m_words_english_gematria_simple.json", &wg, &mu_words_english_gematria_simple, &m_words_english_gematria_simple)
	go write_payload_to_file("m_gematria_english.json", &wg, &mu_gematria_english, &m_gematria_english)
	go write_payload_to_file("m_gematria_jewish.json", &wg, &mu_gematria_jewish, &m_gematria_jewish)
	go write_payload_to_file("m_gematria_simple.json", &wg, &mu_gematria_simple, &m_gematria_simple)
	//go write_payload_to_file("m_location_cities.json", &wg, &mu_location_cities, &m_location_cities)
	//go write_payload_to_file("m_location_countries.json", &wg, &mu_location_countries, &m_location_countries)
	//go write_payload_to_file("m_location_states.json", &wg, &mu_location_states, &m_location_states)
	wg.Wait()
	log_boot.Println("finished writing database to disk")

}

func restore_database_from_disk() {
	if !can_restore_database_from_disk() {
		log_boot.Trace("cannot restore_database_from_disk due to failed santity check")
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
	go load_file_into_payload("m_idx_textee_substring.json", &wg, &mu_idx_textee_substring, &m_idx_textee_substring)
	go load_file_into_payload("m_words.json", &wg, &mu_words, &m_words)
	go load_file_into_payload("m_words_english_gematria_english.json", &wg, &mu_words_english_gematria_english, &m_words_english_gematria_english)
	go load_file_into_payload("m_words_english_gematria_jewish.json", &wg, &mu_words_english_gematria_jewish, &m_words_english_gematria_jewish)
	go load_file_into_payload("m_words_english_gematria_simple.json", &wg, &mu_words_english_gematria_simple, &m_words_english_gematria_simple)
	go load_file_into_payload("m_gematria_english.json", &wg, &mu_gematria_english, &m_gematria_english)
	go load_file_into_payload("m_gematria_jewish.json", &wg, &mu_gematria_jewish, &m_gematria_jewish)
	go load_file_into_payload("m_gematria_simple.json", &wg, &mu_gematria_simple, &m_gematria_simple)
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
		log_boot.Tracef("cannot remove the %v because it does not exist [%v]", file, flush_cache_file_err)
		return // file not present
	}
	mode := flush_cache_file_info.Mode()
	if mode.IsDir() {
		log_boot.Tracef("PROBLEM: the %v file is a directory when the program expected it to be an empty text file [%d bytes]\n", file, flush_cache_file_info.Size())
		return
	} else if (mode & os.ModeSymlink) != 0 {
		log_boot.Tracef("PROBLEM: the %v file is a symlink when the program expected it to be an empty text file [%d bytes]\n", file, flush_cache_file_info.Size())
		return
	} else if (mode & os.ModeType) == 0 {
		log_boot.Printf("found a regular file %v and will flush the database cache because this file is present then we'll delete this file after", file)
		rm_rf_err := os.RemoveAll(filepath.Join(*flag_s_persistent_database_file, "*.json"))
		if rm_rf_err != nil {
			log_boot.Tracef("error removing the *.json files from %v/* due to err %v", *flag_s_persistent_database_file, rm_rf_err)
			return
		}
		err := os.Remove(file)
		if err != nil {
			log_boot.Tracef("failed to remove the %v due to err %v", file, err)
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
		log_boot.Tracef("PROBLEM: the %v file is a directory when the program expected it to be an empty text file [%d bytes]\n", file, flush_cache_file_info.Size())
		return false
	} else if (mode & os.ModeSymlink) != 0 {
		log_boot.Tracef("PROBLEM: the %v file is a symlink when the program expected it to be an empty text file [%d bytes]\n", file, flush_cache_file_info.Size())
		return false
	} else if (mode & os.ModeType) == 0 {
		log_boot.Tracef("found a regular file %v and will flush the database cache because this file is present then we'll delete this file after", file)
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
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_page_gematria_simple.json")) &&
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_idx_textee_substring.json")) &&
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_words.json")) &&
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_words_english_gematria_english.json")) &&
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_words_english_gematria_jewish.json")) &&
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_words_english_gematria_simple.json")) &&
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_gematria_english.json")) &&
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_gematria_jewish.json")) &&
		f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_gematria_simple.json")) //&&
	//f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_location_cities.json")) &&
	//f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_location_countries.json")) &&
	//f_b_path_exists(filepath.Join(*flag_s_persistent_database_file, "m_location_states.json"))
}

func f_b_path_exists(path string) bool {
	info, info_err := os.Stat(path)
	if errors.Is(info_err, os.ErrNotExist) || errors.Is(info_err, os.ErrPermission) {
		log_boot.Tracef("skipping %v due to err %v", path, info_err)
		return false
	}

	if info.Size() == 0 {
		log_boot.Tracef("skipping %v due to size = 0 bytes", path)
		return false
	}
	return true
}

func load_file_into_payload(filename string, wg *sync.WaitGroup, mu *sync.RWMutex, payload any) {
	wg.Add(1)
	defer wg.Done()

	path := filepath.Join(*flag_s_persistent_database_file, filename)
	if !f_b_path_exists(path) {
		log_boot.Tracef("skipping %v due to size = 0 bytes", path)
		return
	}

	bytes, bytes_err := os.ReadFile(path)
	if bytes_err != nil {
		log_boot.Tracef("failed to read file %v due to err %v", path, bytes_err)
		return
	}
	mu.Lock()
	err := json.Unmarshal(bytes, &payload)
	mu.Unlock()
	if err != nil {
		log_boot.Tracef("failed to unmarshal bytes for %v due to err %v", filename, err)
		return
	}

	log_boot.Printf("completed loading file %v into the payload\n", filename)
}

func write_to_file(filename string, payload any) error {

	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error opening file %v: %v", filename, err)
		return err
	}
	defer file.Close()

	bufferedWriter := bufio.NewWriter(file)
	marshal, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return err
	}

	_, err = bufferedWriter.Write(marshal)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return err
	}

	if err := bufferedWriter.Flush(); err != nil {
		fmt.Println("Error flushing writer:", err)
		return err
	}
	return nil
}

func write_any_to_file(database string, filename string, payload any) error {
	file, err := os.Create(filepath.Join(database, filename))
	if err != nil {
		fmt.Printf("Error opening file %v: %v", filename, err)
		return err
	}
	defer file.Close()

	bufferedWriter := bufio.NewWriter(file)
	marshal, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return err
	}

	_, err = bufferedWriter.Write(marshal)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return err
	}

	if err := bufferedWriter.Flush(); err != nil {
		fmt.Println("Error flushing writer:", err)
		return err
	}
	return nil
}

func write_payload_to_file(filename string, wg *sync.WaitGroup, mu *sync.RWMutex, payload any) {
	wg.Add(1)
	defer wg.Done()
	mu.Lock()
	defer mu.Unlock()
	err := write_any_to_file(*flag_s_persistent_database_file, filename, payload)
	if err != nil {
		log_boot.Tracef("write_payload_to_file err: %+v", err)
		return
	}
}

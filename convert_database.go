package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

func convert_database() {
	wg := sync.WaitGroup{}
	wg.Add(7)
	go convert_m_document_total_pages(&wg)
	go convert_m_document_source_url(&wg)
	go convert_m_document_metadata(&wg)
	go convert_m_document_identifier_directory(&wg)
	go convert_m_page_gematria_english(&wg)
	go convert_m_page_gematria_jewish(&wg)
	go convert_m_page_gematria_simple(&wg)
	//convert_m_word_pages()
	//convert_m_document_page_number_page()
	//convert_m_document_page_identifiers_pgno()
	//convert_m_document_pgno_page_identifier()
	//convert_m_page_identifier_document()
	//convert_m_page_identifier_page_number()
	//convert_m_index_page_identifier()
	//convert_m_index_document_identifier()
	//convert_m_document_identifier_cover_page_identifier()
	//convert_m_location_cities()
	//convert_m_location_countries()
	//convert_m_location_states()
	wg.Wait()
}

type Group struct {
	Batch      int
	FirstChars [5]rune
	Tag        string
	File       *os.File
}

func (g *Group) CloseAll() error {
	err := g.File.Close()
	if err != nil {
		return err
	}
	return nil
}

type Grouping map[string]Group

func convert_m_document_total_pages(wg *sync.WaitGroup) {
	// concurrency support for goroutine execution of this function
	defer wg.Done()

	// safely access the data
	mu_document_total_pages.RLock()
	defer mu_document_total_pages.RUnlock()

	// create a grouping for files to be batched inside
	var files Grouping = make(Grouping)
	defer func(files Grouping) {
		for _, group := range files {
			if err := group.CloseAll(); err != nil {
				log_boot.Trace(err)
			}
		}
	}(files)

	// define the batching parameters
	batch := 0
	var batchSize int32 = 10000
	counter := atomic.Int32{}

	// iterate over data
	for documentIdentifier, totalPages := range m_document_total_pages {

		// update counters
		if counter.Add(1) == batchSize {
			batch++
		}

		var firstChars [5]rune
		var tag string
		// get first char and tag
		if len(documentIdentifier) < 5 {
			firstChars = [5]rune{rune(documentIdentifier[0])}
			tag = fmt.Sprintf("%d_%c", batch, firstChars[0])
		} else {
			firstChars = [5]rune{rune(documentIdentifier[0]), rune(documentIdentifier[1]), rune(documentIdentifier[2]), rune(documentIdentifier[3]), rune(documentIdentifier[4])}
			tag = fmt.Sprintf("%d_%c%c%c%c%c", batch, firstChars[0], firstChars[1], firstChars[2], firstChars[3], firstChars[4])
		}

		// create an empty group
		var group Group

		// look for existing tag in group
		if found, exists := files[tag]; !exists {
			group = Group{
				Batch:      batch,
				FirstChars: firstChars,
				Tag:        tag,
				File:       nil,
			}
		} else {
			group = found
		}

		// if no file is defined
		if group.File == nil {

			// build the path of the database file
			fileName := fmt.Sprintf("m_document_total_pages.db/%s.bin", tag)
			filePath := filepath.Join(*flag_s_persistent_database_file, fileName)
			dirPath := filepath.Dir(filePath)

			// ensure that the data's directory exists for the batch bin of data
			if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
				log_boot.Tracef("error creating directory %v: %v", dirPath, err)
				return
			}

			// open the file for writing with append and create mode enabled
			f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log_boot.Tracef("fileName: %+v\nerror opening file %v: %v", fileName, filePath, err)
				return
			}

			// assign the file to the group
			group.File = f
		}

		// generate the content being written to the file
		content := fmt.Sprintf("%s###%d\n", documentIdentifier, totalPages)

		// write content to file
		if _, err := group.File.WriteString(content); err != nil {
			log_boot.Tracef("Error writing to file: %v", err)
		}

		// assign group to files[tag]
		files[tag] = group
	}
}

func convert_m_document_source_url(wg *sync.WaitGroup) {
	// concurrency support for goroutine execution of this function
	defer wg.Done()

	// create a grouping for files to be batched inside
	var files Grouping = make(Grouping)
	defer func(files Grouping) {
		for _, group := range files {
			if err := group.CloseAll(); err != nil {
				log_boot.Trace(err)
			}
		}
	}(files)

	// define the batching parameters
	batch := 0
	var batchSize int32 = 10000
	counter := atomic.Int32{}

	// iterate over data
	for documentIdentifier, url := range m_document_source_url {

		// update counters
		if counter.Add(1) == batchSize {
			batch++
		}

		var firstChars [5]rune
		var tag string
		// get first char and tag
		if len(documentIdentifier) < 5 {
			firstChars = [5]rune{rune(documentIdentifier[0])}
			tag = fmt.Sprintf("%d_%c", batch, firstChars[0])
		} else {
			firstChars = [5]rune{rune(documentIdentifier[0]), rune(documentIdentifier[1]), rune(documentIdentifier[2]), rune(documentIdentifier[3]), rune(documentIdentifier[4])}
			tag = fmt.Sprintf("%d_%c%c%c%c%c", batch, firstChars[0], firstChars[1], firstChars[2], firstChars[3], firstChars[4])
		}

		// create an empty group
		var group Group

		// look for existing tag in group
		if found, exists := files[tag]; !exists {
			group = Group{
				Batch:      batch,
				FirstChars: firstChars,
				Tag:        tag,
				File:       nil,
			}
		} else {
			group = found
		}

		// if no file is defined
		if group.File == nil {

			// build the path of the database file
			fileName := fmt.Sprintf("m_document_source_url.db/%s.bin", tag)
			filePath := filepath.Join(*flag_s_persistent_database_file, fileName)
			dirPath := filepath.Dir(filePath)

			// ensure that the data's directory exists for the batch bin of data
			if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
				log_boot.Tracef("error creating directory %v: %v", dirPath, err)
				return
			}

			// open the file for writing with append and create mode enabled
			f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log_boot.Tracef("fileName: %+v\nerror opening file %v: %v", fileName, filePath, err)
				return
			}

			// assign the file to the group
			group.File = f
		}

		// generate the content being written to the file
		content := fmt.Sprintf("%s###%s\n", documentIdentifier, url)

		// write content to file
		if _, err := group.File.WriteString(content); err != nil {
			log_boot.Tracef("Error writing to file: %v", err)
		}

		// assign group to files[tag]
		files[tag] = group
	}
}

func convert_m_document_metadata(wg *sync.WaitGroup) {
	// concurrency support for goroutine execution of this function
	defer wg.Done()

	mu_document_metadata.RLock()
	defer mu_document_metadata.RUnlock()

	// create a grouping for files to be batched inside
	var files Grouping = make(Grouping)
	defer func(files Grouping) {
		for _, group := range files {
			if err := group.CloseAll(); err != nil {
				log_boot.Trace(err)
			}
		}
	}(files)

	// define the batching parameters
	batch := 0
	var batchSize int32 = 10000
	counter := atomic.Int32{}

	// iterate over data
	for documentIdentifier, metadata := range m_document_metadata {

		// update counters
		if counter.Add(1) == batchSize {
			batch++
		}

		var firstChars [5]rune
		var tag string
		// get first char and tag
		if len(documentIdentifier) < 5 {
			firstChars = [5]rune{rune(documentIdentifier[0])}
			tag = fmt.Sprintf("%d_%c", batch, firstChars[0])
		} else {
			firstChars = [5]rune{rune(documentIdentifier[0]), rune(documentIdentifier[1]), rune(documentIdentifier[2]), rune(documentIdentifier[3]), rune(documentIdentifier[4])}
			tag = fmt.Sprintf("%d_%c%c%c%c%c", batch, firstChars[0], firstChars[1], firstChars[2], firstChars[3], firstChars[4])
		}

		// create an empty group
		var group Group

		// look for existing tag in group
		if found, exists := files[tag]; !exists {
			group = Group{
				Batch:      batch,
				FirstChars: firstChars,
				Tag:        tag,
				File:       nil,
			}
		} else {
			group = found
		}

		// if no file is defined
		if group.File == nil {

			// build the path of the database file
			fileName := fmt.Sprintf("m_document_metadata.db/%s.bin", tag)
			filePath := filepath.Join(*flag_s_persistent_database_file, fileName)
			dirPath := filepath.Dir(filePath)

			// ensure that the data's directory exists for the batch bin of data
			if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
				log_boot.Tracef("error creating directory %v: %v", dirPath, err)
				return
			}

			// open the file for writing with append and create mode enabled
			f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log_boot.Tracef("fileName: %+v\nerror opening file %v: %v", fileName, filePath, err)
				return
			}

			// assign the file to the group
			group.File = f
		}

		// generate the content being written to the file
		var metadataList []string
		for key, value := range metadata {
			metadataList = append(metadataList, fmt.Sprintf("%s=%s", key, value))
		}
		content := fmt.Sprintf("%s###%s\n", documentIdentifier, strings.Join(metadataList, "|"))

		// write content to file
		if _, err := group.File.WriteString(content); err != nil {
			log_boot.Tracef("Error writing to file: %v", err)
		}

		// assign group to files[tag]
		files[tag] = group
	}
}

func convert_m_index_page_identifier() {
	mu_index_page_identifier.RLock()
	defer mu_index_page_identifier.RUnlock()
	filePath := filepath.Join(*flag_s_persistent_database_file, "m_index_page_identifier.db")
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log_boot.Tracef("error opening file %v: %v", filePath, err)
		return
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log_boot.Trace(err)
			return
		}
	}()

	if file == nil {
		log_boot.Fatal("file is nil")
	}

	for index, pageIdentifier := range m_index_page_identifier {
		content := fmt.Sprintf("%d###%s\n", index, pageIdentifier)
		if _, err := file.WriteString(content); err != nil {
			log_boot.Tracef("Error writing to file: %v", err)
		}
	}
}

func convert_m_index_document_identifier() {
	mu_index_document_identifier.RLock()
	defer mu_index_document_identifier.RUnlock()

	filePath := filepath.Join(*flag_s_persistent_database_file, "m_index_document_identifier.db")
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log_boot.Tracef("error opening file %v: %v", filePath, err)
		return
	}
	defer func() {
		err := file.Close()
		if err != nil {
			log_boot.Trace(err)
			return
		}
	}()

	if file == nil {
		log_boot.Fatal("file is nil")
	}

	for index, documentIdentifier := range m_index_document_identifier {
		content := fmt.Sprintf("%d###%s\n", index, documentIdentifier)
		if _, err := file.WriteString(content); err != nil {
			log_boot.Tracef("Error writing to file: %v", err)
		}
	}
}

func convert_m_document_identifier_directory(wg *sync.WaitGroup) {
	// concurrency support for goroutine execution of this function
	defer wg.Done()

	mu_document_identifier_directory.RLock()
	defer mu_document_identifier_directory.RUnlock()

	// create a grouping for files to be batched inside
	var files Grouping = make(Grouping)
	defer func(files Grouping) {
		for _, group := range files {
			if err := group.CloseAll(); err != nil {
				log_boot.Trace(err)
			}
		}
	}(files)

	// define the batching parameters
	batch := 0
	var batchSize int32 = 10000
	counter := atomic.Int32{}

	// iterate over data
	for documentIdentifier, directory := range m_document_identifier_directory {

		// update counters
		if counter.Add(1) == batchSize {
			batch++
		}

		var firstChars [5]rune
		var tag string
		// get first char and tag
		if len(documentIdentifier) < 5 {
			firstChars = [5]rune{rune(documentIdentifier[0])}
			tag = fmt.Sprintf("%d_%c", batch, firstChars[0])
		} else {
			firstChars = [5]rune{rune(documentIdentifier[0]), rune(documentIdentifier[1]), rune(documentIdentifier[2]), rune(documentIdentifier[3]), rune(documentIdentifier[4])}
			tag = fmt.Sprintf("%d_%c%c%c%c%c", batch, firstChars[0], firstChars[1], firstChars[2], firstChars[3], firstChars[4])
		}

		// create an empty group
		var group Group

		// look for existing tag in group
		if found, exists := files[tag]; !exists {
			group = Group{
				Batch:      batch,
				FirstChars: firstChars,
				File:       nil,
			}
		} else {
			group = found
		}

		// if no file is defined
		if group.File == nil {

			// build the path of the database file
			fileName := fmt.Sprintf("m_document_identifier_directory.db/%s.bin", tag)
			filePath := filepath.Join(*flag_s_persistent_database_file, fileName)
			dirPath := filepath.Dir(filePath)

			// ensure that the data's directory exists for the batch bin of data
			if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
				log_boot.Tracef("error creating directory %v: %v", dirPath, err)
				return
			}

			// open the file for writing with append and create mode enabled
			f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log_boot.Tracef("fileName: %+v\nerror opening file %v: %v", fileName, filePath, err)
				return
			}

			// assign the file to the group
			group.File = f
		}

		// generate the content being written to the file
		content := fmt.Sprintf("%s###%s\n", documentIdentifier, directory)

		// write content to file
		if _, err := group.File.WriteString(content); err != nil {
			log_boot.Tracef("Error writing to file: %v", err)
		}

		// assign group to files[tag]
		files[tag] = group
	}
}

func convert_m_page_gematria_english(wg *sync.WaitGroup) {
	// concurrency support for goroutine execution of this function
	defer wg.Done()

	mu_page_gematria_english.RLock()
	defer mu_page_gematria_english.RUnlock()

	// create a grouping for files to be batched inside
	var files Grouping = make(Grouping)
	defer func(files Grouping) {
		for _, group := range files {
			if err := group.CloseAll(); err != nil {
				log_boot.Trace(err)
			}
		}
	}(files)

	// define the batching parameters
	batch := 0
	var batchSize int32 = 10000
	counter := atomic.Int32{}

	// iterate over data
	// gemScoreEnglish uint
	// pageMap = map[word]map[PageIdentifier]struct{}
	for gemScoreEnglish, pageMap := range m_page_gematria_english {

		// update counters
		if counter.Add(1) == batchSize {
			batch++
		}

		tag := fmt.Sprintf("%d_%d", batch, gemScoreEnglish)

		// create an empty group
		var group Group

		// look for existing tag in group
		if found, exists := files[tag]; !exists {
			group = Group{
				Batch: batch,
				Tag:   tag,
				File:  nil,
			}
		} else {
			group = found
		}

		// if no file is defined
		if group.File == nil {

			// build the path of the database file
			fileName := fmt.Sprintf("m_page_gematria_english.db/%s.bin", tag)
			filePath := filepath.Join(*flag_s_persistent_database_file, fileName)
			dirPath := filepath.Dir(filePath)

			// ensure that the data's directory exists for the batch bin of data
			if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
				log_boot.Tracef("error creating directory %v: %v", dirPath, err)
				return
			}

			// open the file for writing with append and create mode enabled
			f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log_boot.Tracef("fileName: %+v\nerror opening file %v: %v", fileName, filePath, err)
				return
			}

			// assign the file to the group
			group.File = f
		}

		// generate the content being written to the file
		results := make(map[string]string)
		for pageIdentifier, words := range pageMap {
			var allWords []string
			for word, _ := range words {
				allWords = append(allWords, word)
			}
			results[pageIdentifier] = strings.Join(allWords, "|")
		}
		var sb *strings.Builder = &strings.Builder{}
		for identifier, words := range results {
			sb.WriteString(fmt.Sprintf("###%s==%s", identifier, words))
		}
		content := fmt.Sprintf("%d%s\n", gemScoreEnglish, sb.String())

		// write content to file
		if _, err := group.File.WriteString(content); err != nil {
			log_boot.Tracef("Error writing to file: %v", err)
		}

		// assign group to files[tag]
		files[tag] = group
	}
}

func convert_m_page_gematria_jewish(wg *sync.WaitGroup) {
	// concurrency support for goroutine execution of this function
	defer wg.Done()

	mu_page_gematria_jewish.RLock()
	defer mu_page_gematria_jewish.RUnlock()

	// create a grouping for files to be batched inside
	var files Grouping = make(Grouping)
	defer func(files Grouping) {
		for _, group := range files {
			if err := group.CloseAll(); err != nil {
				log_boot.Trace(err)
			}
		}
	}(files)

	// define the batching parameters
	batch := 0
	var batchSize int32 = 10000
	counter := atomic.Int32{}

	// iterate over data
	// gemScoreEnglish uint
	// pageMap = map[word]map[PageIdentifier]struct{}
	for gemScoreJewish, pageMap := range m_page_gematria_jewish {

		// update counters
		if counter.Add(1) == batchSize {
			batch++
		}

		tag := fmt.Sprintf("%d_%d", batch, gemScoreJewish)

		// create an empty group
		var group Group

		// look for existing tag in group
		if found, exists := files[tag]; !exists {
			group = Group{
				Batch: batch,
				Tag:   tag,
				File:  nil,
			}
		} else {
			group = found
		}

		// if no file is defined
		if group.File == nil {

			// build the path of the database file
			fileName := fmt.Sprintf("m_page_gematria_jewish.db/%s.bin", tag)
			filePath := filepath.Join(*flag_s_persistent_database_file, fileName)
			dirPath := filepath.Dir(filePath)

			// ensure that the data's directory exists for the batch bin of data
			if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
				log_boot.Tracef("error creating directory %v: %v", dirPath, err)
				return
			}

			// open the file for writing with append and create mode enabled
			f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log_boot.Tracef("fileName: %+v\nerror opening file %v: %v", fileName, filePath, err)
				return
			}

			// assign the file to the group
			group.File = f
		}

		// generate the content being written to the file
		results := make(map[string]string)
		for pageIdentifier, words := range pageMap {
			var allWords []string
			for word, _ := range words {
				allWords = append(allWords, word)
			}
			results[pageIdentifier] = strings.Join(allWords, "|")
		}
		var sb *strings.Builder = &strings.Builder{}
		for identifier, words := range results {
			sb.WriteString(fmt.Sprintf("###%s==%s", identifier, words))
		}
		content := fmt.Sprintf("%d%s\n", gemScoreJewish, sb.String())

		// write content to file
		if _, err := group.File.WriteString(content); err != nil {
			log_boot.Tracef("Error writing to file: %v", err)
		}

		// assign group to files[tag]
		files[tag] = group
	}
}

func convert_m_page_gematria_simple(wg *sync.WaitGroup) {
	// concurrency support for goroutine execution of this function
	defer wg.Done()

	mu_page_gematria_simple.RLock()
	defer mu_page_gematria_simple.RUnlock()

	// create a grouping for files to be batched inside
	var files Grouping = make(Grouping)
	defer func(files Grouping) {
		for _, group := range files {
			if err := group.CloseAll(); err != nil {
				log_boot.Trace(err)
			}
		}
	}(files)

	// define the batching parameters
	batch := 0
	var batchSize int32 = 10000
	counter := atomic.Int32{}

	// iterate over data
	// gemScoreEnglish uint
	// pageMap = map[word]map[PageIdentifier]struct{}
	for gemScoreSimple, pageMap := range m_page_gematria_simple {

		// update counters
		if counter.Add(1) == batchSize {
			batch++
		}

		tag := fmt.Sprintf("%d_%d", batch, gemScoreSimple)

		// create an empty group
		var group Group

		// look for existing tag in group
		if found, exists := files[tag]; !exists {
			group = Group{
				Batch: batch,
				Tag:   tag,
				File:  nil,
			}
		} else {
			group = found
		}

		// if no file is defined
		if group.File == nil {

			// build the path of the database file
			fileName := fmt.Sprintf("m_page_gematria_simple.db/%s.bin", tag)
			filePath := filepath.Join(*flag_s_persistent_database_file, fileName)
			dirPath := filepath.Dir(filePath)

			// ensure that the data's directory exists for the batch bin of data
			if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
				log_boot.Tracef("error creating directory %v: %v", dirPath, err)
				return
			}

			// open the file for writing with append and create mode enabled
			f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log_boot.Tracef("fileName: %+v\nerror opening file %v: %v", fileName, filePath, err)
				return
			}

			// assign the file to the group
			group.File = f
		}

		// generate the content being written to the file
		results := make(map[string]string)
		for pageIdentifier, words := range pageMap {
			var allWords []string
			for word, _ := range words {
				allWords = append(allWords, word)
			}
			results[pageIdentifier] = strings.Join(allWords, "|")
		}
		var sb *strings.Builder = &strings.Builder{}
		for identifier, words := range results {
			sb.WriteString(fmt.Sprintf("###%s==%s", identifier, words))
		}
		content := fmt.Sprintf("%d%s\n", gemScoreSimple, sb.String())

		// write content to file
		if _, err := group.File.WriteString(content); err != nil {
			log_boot.Tracef("Error writing to file: %v", err)
		}

		// assign group to files[tag]
		files[tag] = group
	}
}

func convert_m_location_cities() {
	mu_location_cities.RLock()
	defer mu_location_cities.RUnlock()

	files := make(map[rune]*os.File)
	defer func() {
		for _, file := range files {
			err := file.Close()
			if err != nil {
				log_boot.Trace(err)
				return
			}
		}
	}()

	for _, location := range m_location_cities {
		firstChar := rune(location.City[0])
		if _, exists := files[firstChar]; !exists {
			fileName := fmt.Sprintf("m_location_cities_%c.db", firstChar)
			filePath := filepath.Join(*flag_s_persistent_database_file, fileName)
			file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log_boot.Tracef("fileName: %+v\nerror opening file %v: %v", fileName, filePath, err)
				return
			}
			files[firstChar] = file
		}

		content := fmt.Sprintf("%s###%s###%s###%s###%f###%f\n",
			location.City,
			location.State,
			location.Country,
			location.CountryCode,
			location.Latitude,
			location.Longitude)
		file := files[firstChar]
		if file == nil {
			panic("file is nil")
		}
		if _, err := file.WriteString(content); err != nil {
			log_boot.Tracef("Error writing to file: %v", err)
		}
	}
}

func convert_m_location_states() {
	mu_location_states.RLock()
	defer mu_location_states.RUnlock()

	files := make(map[rune]*os.File)
	defer func() {
		for _, file := range files {
			err := file.Close()
			if err != nil {
				log_boot.Trace(err)
				return
			}
		}
	}()

	for _, location := range m_location_states {
		firstChar := rune(location.State[0])
		if _, exists := files[firstChar]; !exists {
			fileName := fmt.Sprintf("m_location_states_%c.db", firstChar)
			filePath := filepath.Join(*flag_s_persistent_database_file, fileName)
			file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log_boot.Tracef("fileName: %+v\nerror opening file %v: %v", fileName, filePath, err)
				return
			}
			files[firstChar] = file
		}

		content := fmt.Sprintf("%s###%s###%s###%f###%f\n",
			location.State,
			location.Country,
			location.CountryCode,
			location.Latitude,
			location.Longitude)
		file := files[firstChar]
		if file == nil {
			panic("file is nil")
		}
		if _, err := file.WriteString(content); err != nil {
			log_boot.Tracef("Error writing to file: %v", err)
		}
	}
}

func convert_m_location_countries() {
	mu_location_countries.RLock()
	defer mu_location_countries.RUnlock()

	files := make(map[rune]*os.File)
	defer func() {
		for _, file := range files {
			err := file.Close()
			if err != nil {
				log_boot.Trace(err)
				return
			}
		}
	}()

	for _, location := range m_location_countries {
		firstChar := rune(location.Country[0])
		if _, exists := files[firstChar]; !exists {
			fileName := fmt.Sprintf("m_location_countries_%c.db", firstChar)
			filePath := filepath.Join(*flag_s_persistent_database_file, fileName)
			file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log_boot.Tracef("fileName: %+v\nerror opening file %v: %v", fileName, filePath, err)
				return
			}
			files[firstChar] = file
		}

		content := fmt.Sprintf("%s###%s###%f###%f\n",
			location.Country,
			location.CountryCode,
			location.Latitude,
			location.Longitude)
		file := files[firstChar]
		if file == nil {
			panic("file is nil")
		}
		if _, err := file.WriteString(content); err != nil {
			log_boot.Tracef("Error writing to file: %v", err)
		}
	}
}

func convert_m_document_page_identifiers_pgno() {
	mu_document_page_identifiers_pgno.RLock()
	defer mu_document_page_identifiers_pgno.RUnlock()

	files := make(map[rune]*os.File)
	defer func() {
		for _, file := range files {
			err := file.Close()
			if err != nil {
				log.Println(err)
				return
			}
		}
	}()

	for documentIdentifier, pagesData := range m_document_page_identifiers_pgno {
		firstChar := rune(documentIdentifier[0])
		if _, exists := files[firstChar]; !exists {
			fileName := fmt.Sprintf("m_document_page_identifiers_pgno_%c.db", firstChar)
			filePath := filepath.Join(*flag_s_persistent_database_file, fileName)
			file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Println(fileName)
				log.Printf("error opening file %v: %v", filePath, err)
				continue
			}
			files[firstChar] = file
		}

		content := documentIdentifier + "###"
		var parts []string
		for pageIdentifier, pageNumber := range pagesData {
			parts = append(parts, fmt.Sprintf("%s=%d", pageIdentifier, pageNumber))
		}
		content = content + strings.Join(parts, "|")

		file := files[firstChar]
		if file == nil {
			panic("file is nil")
		}
		if _, err := file.WriteString(content); err != nil {
			log.Printf("Error writing to file %s: %v", files[firstChar].Name(), err)
		}

	}
}

func convert_m_document_page_number_page() {
	mu_document_page_number_page.RLock()
	defer mu_document_page_number_page.RUnlock()

	files := make(map[rune]*os.File)
	defer func() {
		for _, file := range files {
			err := file.Close()
			if err != nil {
				log.Println(err)
				return
			}
		}
	}()

	for documentIdentifier, pagesData := range m_document_page_number_page {
		firstChar := rune(documentIdentifier[0])
		if _, exists := files[firstChar]; !exists {
			fileName := fmt.Sprintf("m_document_page_number_page_%c.db", firstChar)
			filePath := filepath.Join(*flag_s_persistent_database_file, fileName)
			file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Println(firstChar)
				log.Println(fileName)
				log.Printf("error opening file %v: %v", filePath, err)
				continue
			}
			files[firstChar] = file
		}

		content := documentIdentifier + "###"
		var parts []string
		for pageNumber, page := range pagesData {
			parts = append(parts, fmt.Sprintf("%d=%s", pageNumber, page.Identifier))
		}
		content = content + strings.Join(parts, "|")

		file := files[firstChar]
		if file == nil {
			panic("file is nil")
		}
		if _, err := file.WriteString(content); err != nil {
			log.Printf("Error writing to file %s: %v", file.Name(), err)
		}
	}
}

func convert_m_document_identifier_cover_page_identifier() {
	mu_document_identifier_cover_page_identifier.RLock()
	defer mu_document_identifier_cover_page_identifier.RUnlock()

	files := make(map[rune]*os.File)
	defer func() {
		for _, file := range files {
			err := file.Close()
			if err != nil {
				log.Println(err)
				return
			}
		}
	}()

	for documentIdentifier, coverPageIdentifier := range m_document_identifier_cover_page_identifier {
		firstChar := rune(documentIdentifier[0])
		if _, exists := files[firstChar]; !exists {
			fileName := fmt.Sprintf("m_document_identifier_cover_page_identifier_%c.db", firstChar)
			filePath := filepath.Join(*flag_s_persistent_database_file, fileName)
			file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Println(fileName)
				log.Printf("error opening file %v: %v", filePath, err)
				continue
			}
			files[firstChar] = file
		}

		content := documentIdentifier + "###" + coverPageIdentifier

		file := files[firstChar]
		if file == nil {
			panic("file is nil")
		}
		if _, err := file.WriteString(content); err != nil {
			log.Printf("Error writing to file %s: %v", files[firstChar].Name(), err)
		}
	}
}

func convert_m_word_pages() {
	mu_word_pages.RLock()
	defer mu_word_pages.RUnlock()

	files := make(map[rune]*os.File)
	defer func() {
		for _, file := range files {
			err := file.Close()
			if err != nil {
				log.Println(err)
				return
			}
		}
	}()

	for word, identifiers := range m_word_pages {
		firstChar := rune(word[0])
		if _, exists := files[firstChar]; !exists {
			fileName := fmt.Sprintf("m_word_pages_%c.db", firstChar)
			filePath := filepath.Join(*flag_s_persistent_database_file, fileName)
			file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Println(fileName)
				log.Printf("error opening file %v: %v", filePath, err)
				continue
			}
			files[firstChar] = file
		}

		var identifiersList []string
		for id := range identifiers {
			identifiersList = append(identifiersList, id)
		}
		content := word + "###" + strings.Join(identifiersList, "|") + "\n"

		file := files[firstChar]
		if file == nil {
			panic("file is nil")
		}
		if _, err := file.WriteString(content); err != nil {
			log.Printf("Error writing to file %s: %v", files[firstChar].Name(), err)
		}
	}
}

func convert_m_document_pgno_page_identifier() {
	mu_document_pgno_page_identifier.RLock()
	defer mu_document_pgno_page_identifier.RUnlock()

	files := make(map[rune]*os.File)
	defer func() {
		for _, file := range files {
			err := file.Close()
			if err != nil {
				log.Println(err)
				return
			}
		}
	}()

	for documentIdentifier, pages := range m_document_pgno_page_identifier {
		firstChar := rune(documentIdentifier[0])
		if _, exists := files[firstChar]; !exists {
			fileName := fmt.Sprintf("m_document_pgno_page_identifier_%c.db", firstChar)
			filePath := filepath.Join(*flag_s_persistent_database_file, fileName)
			file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Println(fileName)
				log.Printf("error opening file %v: %v", filePath, err)
				continue
			}
			files[firstChar] = file
		}

		var content strings.Builder
		content.WriteString(documentIdentifier + "###")
		for pageNumber, pageIdentifier := range pages {
			content.WriteString(fmt.Sprintf("%d=%s|", pageNumber, pageIdentifier))
		}
		content.WriteString("\n")

		file := files[firstChar]
		if file == nil {
			panic("file is nil")
		}
		if _, err := files[firstChar].WriteString(content.String()); err != nil {
			log.Printf("Error writing to file %s: %v", files[firstChar].Name(), err)
		}
	}
}

func convert_m_page_identifier_document() {
	mu_page_identifier_document.RLock()
	defer mu_page_identifier_document.RUnlock()

	files := make(map[rune]*os.File)
	defer func() {
		for _, file := range files {
			err := file.Close()
			if err != nil {
				log.Println(err)
				return
			}
		}
	}()

	for pageIdentifier, documentIdentifier := range m_page_identifier_document {
		firstChar := rune(pageIdentifier[0])
		if _, exists := files[firstChar]; !exists {
			fileName := fmt.Sprintf("m_page_identifier_document_%c.db", firstChar)
			filePath := filepath.Join(*flag_s_persistent_database_file, fileName)
			file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Println(fileName)
				log.Printf("error opening file %v: %v", filePath, err)
				continue
			}
			files[firstChar] = file
		}

		content := fmt.Sprintf("%s###%s\n", pageIdentifier, documentIdentifier)

		file := files[firstChar]
		if file == nil {
			panic("file is nil")
		}
		if _, err := file.WriteString(content); err != nil {
			log.Printf("Error writing to file %s: %v", files[firstChar].Name(), err)
		}
	}
}

func convert_m_page_identifier_page_number() {
	mu_page_identifier_page_number.RLock()
	defer mu_page_identifier_page_number.RUnlock()

	files := make(map[rune]*os.File)
	defer func() {
		for _, file := range files {
			err := file.Close()
			if err != nil {
				log.Println(err)
				return
			}
		}
	}()

	for pageIdentifier, pageNumber := range m_page_identifier_page_number {
		firstChar := rune(pageIdentifier[0])
		if _, exists := files[firstChar]; !exists {
			fileName := fmt.Sprintf("m_page_identifier_page_number_%c.db", firstChar)
			filePath := filepath.Join(*flag_s_persistent_database_file, fileName)
			file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Println(fileName)
				log.Printf("error opening file %v: %v", filePath, err)
				continue
			}
			files[firstChar] = file
		}

		content := fmt.Sprintf("%s###%d\n", pageIdentifier, pageNumber)

		file := files[firstChar]
		if file == nil {
			panic("file is nil")
		}
		if _, err := file.WriteString(content); err != nil {
			log.Printf("Error writing to file %s: %v", files[firstChar].Name(), err)
		}
	}
}

func lookupPageIdentifiersForWord(word string) []string {
	firstChar := word[0]
	fileName := fmt.Sprintf("m_word_pages_%c.db", firstChar)

	file, err := os.Open(filepath.Join(*flag_s_persistent_database_file, fileName))
	if err != nil {
		log.Printf("Error reading file %s: %v", fileName, err)
		return nil
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Printf("Error closing file %s: %v", fileName, err)
		}
	}(file)

	var identifiers []string
	chunkSize := 3 * 1024 * 1024 // 3MB
	buffer := make([]byte, chunkSize)
	var overflow []byte

	for {
		n, err := file.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading file %s: %v", fileName, err)
			}
			break
		}

		// Combine overflow from previous read with the new buffer
		data := append(overflow, buffer[:n]...)
		lines := bytes.Split(data, []byte("\n"))

		// The last line might be incomplete, so keep it for the next read
		overflow = lines[len(lines)-1]
		lines = lines[:len(lines)-1]

		for _, line := range lines {
			text := string(line)
			if strings.HasPrefix(text, word+"###") {
				parts := strings.SplitN(text, "###", 2)
				if len(parts) == 2 {
					ids := strings.Split(parts[1], "|")
					identifiers = append(identifiers, ids...)
				}
			}
		}

		// Stop if we read less than chunkSize, indicating end of file
		if n < chunkSize {
			break
		}
	}

	// Process any remaining overflow line
	if len(overflow) > 0 {
		text := string(overflow)
		if strings.HasPrefix(text, word+"###") {
			parts := strings.SplitN(text, "###", 2)
			if len(parts) == 2 {
				ids := strings.Split(parts[1], "|")
				identifiers = append(identifiers, ids...)
			}
		}
	}

	return identifiers
}

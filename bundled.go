package main

import (
	`bufio`
	`context`
	`embed`
	`encoding/csv`
	`encoding/json`
	`fmt`
	`io/fs`
	`log`
	`log/slog`
	`os`
	`os/exec`
	`strconv`
	`strings`
	`sync`
	`sync/atomic`
)

//go:embed bundled/*
var bundled_files embed.FS

func bundled_load_all_words() {
	languages := []string{"english", "french", "german", "romanian", "russian", "spanish"}
	for _, language := range languages {
		err := bundled_load_language(language)
		if err != nil {
			slog.Error("received an error loading %v: %v", language, err)
		}
	}
}

func bundled_load_cryptonyms() {
	wg_active_tasks.Add(1)
	defer wg_active_tasks.Done()
	a_b_cryptonyms_loaded.Store(false)
	cryptonymFile, cryptonymFileErr := bundled_files.ReadFile("bundled/intelligence/cryptonyms.json")
	if cryptonymFileErr != nil {
		log.Printf("failed to parse cryptonyms.json file from the data directory due to error: %v", cryptonymFileErr)
	} else {
		cryptonymMarshalErr := json.Unmarshal(cryptonymFile, &m_cryptonyms)
		if cryptonymMarshalErr != nil {
			log.Printf("failed to load the m_cryptonyms due to error %v", cryptonymMarshalErr)
		}
		out := ""
		var cryptonyms []string
		for cryptonym, _ := range m_cryptonyms {
			cryptonyms = append(cryptonyms, cryptonym)
		}
		out = strings.Join(cryptonyms, ",")
		log.Printf("Cryptonyms to search for: %v", out)
	}
	a_b_cryptonyms_loaded.Store(true)
}

func bundled_load_locations(ctx context.Context, callback CallbackFunc) error {
	filename := "bundled/geography/locations.csv"
	file, openErr := bundled_files.Open(filename)
	if openErr != nil {
		log.Printf("cant open the file because of err: %v", openErr)
		return openErr
	}
	defer func(file fs.File) {
		closeErr := file.Close()
		if closeErr != nil {
			log.Fatalf("failed to close the file %v caused error %v", filename, closeErr)
		}
	}(file)
	bufferedReader := bufio.NewReaderSize(file, reader_buffer_bytes)
	reader := csv.NewReader(bufferedReader)
	reader.FieldsPerRecord = -1
	headerFields, bufferReadErr := reader.Read()
	if bufferReadErr != nil {
		log.Printf("cant read the csv buffer because of err: %v", bufferReadErr)
		return bufferReadErr
	}
	log.Printf("headerFields = %v", strings.Join(headerFields, ","))
	row := make(chan []Column, channel_buffer_size)
	totalRows, rowWg := atomic.Uint32{}, sync.WaitGroup{}
	done := make(chan struct{})
	go ReceiveRows(ctx, row, filename, callback, done)
	for {
		rowFields, readerErr := reader.Read()
		if readerErr != nil {
			log.Printf("skipping row due to error %v with data %v", readerErr, rowFields)
			break
		}
		totalRows.Add(1)
		rowWg.Add(1)
		go ProcessRow(headerFields, rowFields, &rowWg, row)
	}

	rowWg.Wait()
	close(row)
	<-done
	log.Printf("totalRows = %d", totalRows.Load())
	return nil
}

func bundled_load_language(language string) error {
	filename := fmt.Sprintf("bundled/dictionaries/words-%s.txt", language)
	wordsFile, fileErr := bundled_files.Open(filename)
	if fileErr != nil {
		return fmt.Errorf("failed to load dictionary %v due to error %v", language, fileErr)
	}
	defer func(wordsFile fs.File) {
		err := wordsFile.Close()
		if err != nil {
			slog.Error("failed to close the wordsFile handler with %v", err)
		}
	}(wordsFile)

	scanner := bufio.NewScanner(wordsFile)
	for scanner.Scan() {
		word := scanner.Text()
		_, language_found := m_words[language]
		if !language_found {
			m_words[language] = make(map[string]struct{})
		}
		m_words[language][word] = struct{}{}
		if language != "english" {
			continue
		}
		gematria := NewGemScore(word)
		// english
		_, englished_declared := m_gematria_english[gematria.English]
		if !englished_declared {
			m_gematria_english[gematria.English] = make(map[string]struct{})
		}
		m_gematria_english[gematria.English][word] = struct{}{}
		m_words_english_gematria_english[word] = gematria.English
		// jewish
		_, jewish_declared := m_gematria_jewish[gematria.Jewish]
		if !jewish_declared {
			m_gematria_jewish[gematria.Jewish] = make(map[string]struct{})
		}
		m_gematria_jewish[gematria.Jewish][word] = struct{}{}
		m_words_english_gematria_jewish[word] = gematria.Jewish
		// simple
		_, simple_declared := m_gematria_simple[gematria.Simple]
		if !simple_declared {
			m_gematria_simple[gematria.Simple] = make(map[string]struct{})
		}
		m_gematria_simple[gematria.Simple][word] = struct{}{}
		m_words_english_gematria_simple[word] = gematria.Simple
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file %v: %w", filename, err)
	}

	return nil
}

func parsePIDs(output string) []int {
	var pids []int

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			pid, err := strconv.Atoi(fields[1])
			if err == nil {
				pids = append(pids, pid)
			}
		}
	}

	return pids
}

func terminatePID(pid int) {
	cmd := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/F")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println("Error terminating PID", pid, ":", err)
	}
}

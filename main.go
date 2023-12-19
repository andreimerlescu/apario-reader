package main

import (
	`context`
	`flag`
	`fmt`
	`log`
	`log/slog`
	`os`
	`os/exec`
	`os/signal`
	`path/filepath`
	`runtime`
	`syscall`
	`time`
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, arg := range os.Args {
		if arg == "help" {
			fmt.Println(config.Usage())
			os.Exit(0)
		}
		if arg == "show" {
			for _, innerArg := range os.Args {
				if innerArg == "w" || innerArg == "c" {
					license, err := os.ReadFile(filepath.Join(".", "LICENSE"))
					if err != nil {
						fmt.Printf("Cannot find the license file to load to comply with the GNU-3 license terms. This program was modified outside of its intended runtime use.")
						os.Exit(1)
					} else {
						fmt.Printf("%v\n", string(license))
						os.Exit(1)
					}
				}
			}
		}
	}

	configErr := config.Parse(filepath.Join(".", "config.yaml"))
	if configErr != nil {
		log.Fatalf("failed to parse config.yaml due to err: %v", configErr)
	}

	if *flag_s_database == "" {
		flag.Usage()
		os.Exit(1)
	}

	if *flag_i_sem_limiter > 0 {
		channel_buffer_size = *flag_i_sem_limiter
	}

	if *flag_i_buffer > 0 {
		reader_buffer_bytes = *flag_i_buffer
	}

	logFile, logFileErr := os.OpenFile(*flag_g_log_file, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
	if logFileErr != nil {
		log.Fatal("Failed to open log file: ", logFileErr)
	}
	log.SetOutput(logFile)

	watchdog := make(chan os.Signal, 1)
	signal.Notify(watchdog, os.Kill, syscall.SIGTERM, os.Interrupt)
	go func() {
		<-watchdog
		err := logFile.Close()
		if err != nil {
			log.Printf("failed to close the logFile due to error: %v", err)
		}
		cancel()

		wg_active_tasks.PreventAdd()

		if !sem_analyze_pages.IsEmpty() {
			log.Printf("sem_analyze_pages has %d items left inside it", sem_analyze_pages.Len())
		}

		if !sem_db_directories.IsEmpty() {
			log.Printf("sem_db_directories has %d items left inside it", sem_db_directories.Len())
		}

		ch_db_directories.Close()

		fmt.Printf("Completed running in %d", time.Since(startedAt))

		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			cmd = exec.Command("tasklist", "/FI", "IMAGENAME eq apario-contribution.exe")
		default:
			cmd = exec.Command("pgrep", "apario-contribution")
		}

		output, err := cmd.Output()
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		pids := parsePIDs(string(output))

		for _, pid := range pids {
			terminatePID(pid)
		}

		os.Exit(0)
	}()

	bundled_load_all_words()

	log.Printf("m_words_english_gematria_simple = %T len() = %d", m_words_english_gematria_simple, len(m_words_english_gematria_simple))
	log.Printf("m_words_english_gematria_jewish = %T len() = %d", m_words_english_gematria_jewish, len(m_words_english_gematria_jewish))
	log.Printf("m_words_english_gematria_english = %T len() = %d", m_words_english_gematria_english, len(m_words_english_gematria_english))
	log.Printf("m_gematria_simple = %T len() = %d", m_gematria_simple, len(m_gematria_simple))
	log.Printf("m_gematria_english = %T len() = %d", m_gematria_english, len(m_gematria_english))
	log.Printf("m_gematria_jewish = %T len() = %d", m_gematria_jewish, len(m_gematria_jewish))
	slog.Info("Break here")

	go process_directories(ch_db_directories.Chan())
	defer ch_db_directories.Close()

	go bundled_load_cryptonyms()

	go func() {
		wg_active_tasks.Add(1)
		defer wg_active_tasks.Done()

		locationsCsvErr := bundled_load_locations(ctx, processLocation)
		if locationsCsvErr != nil {
			log.Printf("received an error while loading the locations: %v", locationsCsvErr) // a problem habbened
			return
		}

		a_b_locations_loaded.Store(true)
	}()

	err := database_load()
	if err != nil {
		slog.Error("failed to load the database with error %v", err)
		return
	}
	wg_active_tasks.Wait()
	slog.Info("done loading the application's database into memory")

}

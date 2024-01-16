package main

import (
	`context`
	`flag`
	`fmt`
	`io/fs`
	`log`
	`log/slog`
	`os`
	`os/signal`
	`path/filepath`
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

	// Attempt to read from the `--config` as a file, default: config.yaml
	configErr := config.Parse(*flag_s_config_file)
	if configErr != nil {
		log.Fatalf("failed to parse config file: %v", configErr)
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

	logFile, logFileErr := os.OpenFile(*flag_s_log_file, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
	if logFileErr != nil {
		log.Fatal("Failed to open log file: ", logFileErr)
	}
	log.SetOutput(logFile)

	watchdog := make(chan os.Signal, 1)
	signal.Notify(watchdog, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)
	go watch_for_signal(watchdog, logFile, cancel)

	bundled_load_all_words()

	if f_b_db_flush_file_set() {
		f_clear_db_restore_file()
	}

	if !*flag_b_load_persistent_runtime_database && can_restore_database_from_disk() {
		log.Printf("restore cache db from disk")
		restore_database_from_disk()
	}

	germinatedAt := time.Now().UTC()

	// shard the database checksum directory
	go func() {
		err := filepath.Walk(*flag_s_database, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() && len(info.Name()) == 1 && info.Name() != "." {
				// beginning of a sharded directory
				return nil
			} else if info.IsDir() && len(info.Name()) == 64 {
				// not yet sharded
				err := shard_checksum_path(path, info)
				if err != nil {
					return err
				}
			} else if info.IsDir() && len(info.Name()) == 19 {
				// last octet, meaning the complete checksum is the last
				ok, err := check_path_checksum(path, *flag_s_database)
				if !ok || err != nil {
					return err
				} else {
					return nil
				}
			} else {
				// other length directory, means keep walking until
				return nil
			}
			return nil
		})
		if err != nil {
			return
		}

	}()

	go process_directories(ch_db_directories)

	go bundled_load_cryptonyms()

	//go func() {
	//	wg_active_tasks.Add(1)
	//	defer wg_active_tasks.Done()
	//
	//	locationsCsvErr := bundled_load_locations(ctx, processLocation)
	//	if locationsCsvErr != nil {
	//		log.Printf("received an error while loading the locations: %v", locationsCsvErr) // a problem habbened
	//		return
	//	}
	//
	//	a_b_locations_loaded.Store(true)
	//}()

	err := database_load()
	if err != nil {
		slog.Error("failed to load the database with error %v", err)
		return
	}

	wg_active_tasks.Wait()
	log.Printf("completed loading the database in %.0f seconds", time.Since(germinatedAt).Seconds())
	a_b_database_loaded.Store(true)

	// memory stuff now
	commencedAt := time.Now().UTC()
	m_collections = nil
	m_gematria_jewish = nil
	m_gematria_english = nil
	m_gematria_simple = nil
	m_words = nil
	m_words_english_gematria_english = nil
	m_words_english_gematria_jewish = nil
	m_words_english_gematria_simple = nil
	log.Printf("flushed unnecessary maps out of memory. took %d ms", time.Since(commencedAt).Milliseconds())

	log.Printf("wg_active_tasks has completed!")

	if *flag_b_persist_runtime_database {
		dump_database_to_disk()
	}

	slog.Info("done loading the application's database into memory")

	go NewWebServer(ctx)

	for {
		select {
		case <-ctx.Done():
			fatalf_stout("Main context canceled, exiting application now. Reason: %v", ctx.Err())
		case <-ch_webserver_done.Chan():
			cancel()
		}
	}
}

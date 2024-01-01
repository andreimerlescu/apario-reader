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
		ch_cert_reloader_cancel.Close()
		ch_webserver_done.Close()

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

	//bundled_load_all_words()
	slog.Info("Break here")

	if f_b_db_flush_file_set() {
		f_clear_db_restore_file()
	}

	if can_restore_database_from_disk() {
		restore_database_from_disk()
	} else {
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

		go dump_database_to_disk()
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

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	arg_config_yaml := ""

	for i, arg := range os.Args {
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
		if strings.HasPrefix(arg, `-`) && strings.HasSuffix(arg, "config") {
			if len(os.Args) >= i+1 {
				arg_config_yaml = strings.Clone(os.Args[i+1])
			}
		}
	}

	// Attempt to read from the `--config` as a file, default: config.yaml
	var configErr error
	var configStatErr error
	var configFile string
	if len(arg_config_yaml) > 0 {
		_, configStatErr = os.Stat(arg_config_yaml)
		if configStatErr == nil || !errors.Is(configStatErr, os.ErrNotExist) {
			configFile = strings.Clone(arg_config_yaml)
		}
	} else {
		if len(*flag_s_config_file) == 0 {
			_, configStatErr = os.Stat(filepath.Join(".", "config.yaml"))
			if configStatErr == nil || !errors.Is(configStatErr, os.ErrNotExist) {
				configFile = filepath.Join(".", "config.yaml")
			}
		} else {
			_, configStatErr = os.Stat(*flag_s_config_file)
			if configStatErr == nil || !errors.Is(configStatErr, os.ErrNotExist) {
				configFile = strings.Clone(*flag_s_config_file)
			}
		}
	}

	configErr = config.Parse(configFile)
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

	log_path := *flag_s_log_file
	log_dir := filepath.Dir(log_path)
	_, ldiErr := os.Stat(log_dir)
	if ldiErr != nil && errors.Is(ldiErr, os.ErrNotExist) {
		mkErr := os.MkdirAll(log_dir, 0755)
		if mkErr != nil {
			log.Panicf("failed to open file %v due to %+v", *flag_s_log_file, ldiErr)
		}
	}
	log_files = make(map[string]*os.File)

	// Initialize log files with truncation
	debugFile, err := os.OpenFile(filepath.Join(log_dir, "reader.debug.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("failed to open debug log: %v", err)
	}
	log_files[cLogDebug] = debugFile

	infoFile, err := os.OpenFile(filepath.Join(log_dir, "reader.info.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		closeLogFiles()
		log.Fatalf("failed to open info log: %v", err)
	}
	log_files[cLogInfo] = infoFile

	errorFile, err := os.OpenFile(filepath.Join(log_dir, "reader.error.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		closeLogFiles()
		log.Fatalf("failed to open error log: %v", err)
	}
	log_files[cLogError] = errorFile

	bootFile, err := os.OpenFile(filepath.Join(log_dir, "reader.boot.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		closeLogFiles()
		log.Fatalf("failed to open error log: %v", err)
	}
	log_files[cLogError] = errorFile

	// Initialize loggers
	log_debug = NewCustomLogger(debugFile, "DEBUG: ", log.Ldate|log.Ltime|log.Llongfile, 10)
	log_info = NewCustomLogger(infoFile, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile, 10)
	log_error = NewCustomLogger(errorFile, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile, 10)
	log_boot = NewCustomLogger(bootFile, "BOOT: ", log.Ldate|log.Ltime|log.Lshortfile, 10)

	watchdog := make(chan os.Signal, 1)
	signal.Notify(watchdog, syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)
	go watch_for_signal(watchdog, cancel)

	bundled_load_all_words()

	if f_b_db_flush_file_set() {
		f_clear_db_restore_file()
	}

	if !*flag_b_load_persistent_runtime_database && can_restore_database_from_disk() {
		log_boot.Println("restore cache db from disk")
		restore_database_from_disk()
	}

	germinatedAt := time.Now().UTC()

	// receiver waiting for data to write to ch_db_directories
	go process_directories(ctx, ch_db_directories)

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

	// scheduler that writes to ch_db_directories
	err = database_load()
	if err != nil {
		log_boot.Fatalf("failed to load the database with error %v", err)
	}

	wg_active_tasks.Wait()
	log_boot.Printf("completed loading the database in %.0f seconds", time.Since(germinatedAt).Seconds())
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
	log_boot.Printf("flushed unnecessary maps out of memory. took %d ms", time.Since(commencedAt).Milliseconds())

	log_boot.Printf("wg_active_tasks has completed!")

	if *flag_b_persist_runtime_database {
		dump_database_to_disk()
	}

	log_boot.Println("done loading the application's database into memory")

	// go sync_user_directory(ctx)

	convert_database()

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

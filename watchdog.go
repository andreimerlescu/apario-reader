package main

import (
	`context`
	`fmt`
	`log`
	`os`
	`os/exec`
	`runtime`
	`strconv`
	`strings`
	`time`
)

func watch_for_signal(watchdog chan os.Signal, logFile *os.File, cancel context.CancelFunc) {
	<-watchdog
	log.Printf("watchdog signal received")
	wg_active_tasks.PreventAdd()
	err := ch_webserver_done.Write(struct{}{})
	if err != nil {
		log.Printf("cant close ch_webserver_done")
	}
	err = logFile.Close()
	if err != nil {
		log.Printf("failed to close the logFile due to error: %v", err)
	}
	cancel()

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

	pids := find_process_identifiers(string(output))

	for _, pid := range pids {
		terminate_process_identifier(pid)
	}

	os.Exit(0)
}

func find_process_identifiers(output string) []int {
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

func terminate_process_identifier(pid int) {
	cmd := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/F")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println("Error terminating PID", pid, ":", err)
	}
}

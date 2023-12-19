package main

import (
	`log`
	`strings`
)

func fatalf_log(f string, args ...interface{}) {
	log.Printf(f, args...)
	ch_webserver_done <- struct{}{}
}

func fatalf_stderr(f string, args ...interface{}) {
	log.Printf(f, args...)
	fatalf_log(f, args...)
}

func fatalf_stout(f string, args ...interface{}) {
	log.Printf(f, args...)
	fatalf_log(f, args...)
}

func slice_contains(in []string, what string) bool {
	for _, i := range in {
		if strings.EqualFold(i, what) {
			return true
		}
	}
	return false
}

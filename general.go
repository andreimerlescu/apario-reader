package main

import (
	`bytes`
	`log`
	`os`
	`strconv`
	`strings`
)

func fatalf_log(f string, args ...interface{}) {
	log.Printf(f, args...)
	if ch_webserver_done.CanWrite() {
		err := ch_webserver_done.Write(struct{}{})
		if err != nil {
			log.Printf("failed to close ch_webserver_done due to error %v", err)
			return
		}
	}

}

func shard_checksum_path(path string, info os.FileInfo) error {
	old_path := strings.ReplaceAll(path, *flag_s_database, ``)
	checksum, checksum_err := checksum_to_path(info.Name())
	if checksum_err != nil {
		return checksum_err
	}
	new_path := strings.ReplaceAll(checksum, *flag_s_database, ``)
	err := os.Rename(old_path, new_path)
	if err != nil {
		return err
	}
	return nil
}

func fibonacci(n int) int {
	if n <= 1 {
		return n
	}
	return fibonacci(n-1) + fibonacci(n-2)
}

func check_path_checksum(path string, basename string) (bool, error) {
	path_checksum := strings.ReplaceAll(path, basename, ``)
	checksum := strings.ReplaceAll(path_checksum, string(os.PathSeparator), ``)
	checksum_check, check_err := checksum_to_path(checksum)
	if check_err != nil {
		return false, check_err
	}
	if len(checksum) == 64 && checksum_check == path_checksum {
		log.Printf("successfully verified path %v", path)
		return true, nil
	}
	return false, nil
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

func human_int(i int64, opts ...string) string {
	var comma string
	if len(*flag_s_number_decimal_place) == 0 {
		comma = ","
	} else {
		comma = *flag_s_number_decimal_place
	}

	if len(opts) > 0 && opts[0] != "" {
		comma = opts[0]
	}

	str := strconv.FormatInt(i, 10)

	var result bytes.Buffer

	for i, c := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result.WriteString(comma)
		}
		result.WriteByte(byte(c))
	}

	return result.String()
}

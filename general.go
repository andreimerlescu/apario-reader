package main

import (
	`bytes`
	`log`
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

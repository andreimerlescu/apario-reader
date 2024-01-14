package main

import (
	crypto_rand `crypto/rand`
	`fmt`
	`math`
	`math/big`
	`net/http`
	`strings`

	`golang.org/x/text/cases`
	`golang.org/x/text/language`
)

func f_i_random_int(limit int) int {
	if limit <= 0 {
		return 0
	}

	newInt, err := crypto_rand.Int(crypto_rand.Reader, big.NewInt(int64(limit)))
	if err != nil {
		fmt.Println("Error:", err)
		return 0
	}
	return int(newInt.Int64())
}

// f_i_random_int_range return a random int that is between start and limit, will recursively run until match found
func f_i_random_int_range(start int, limit int) int {
	i := f_i_random_int(limit)
	if i >= start && i <= limit {
		return i
	} else {
		return f_i_random_int_range(start, limit) // retry
	}
}

func f_i_plus(a, b int) int {
	return a + b
}

func f_i_minus(a, b int) int {
	return a - b
}

func f_i_sequence(start, end int) []int {
	var seq []int
	for i := start; i <= end; i++ {
		seq = append(seq, i)
	}
	return seq
}

func f_i_max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func f_i_min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func f_s_client_ip(r *http.Request) string {
	headers := []string{"X-Real-IP", "X-Forwarded-For"}

	for _, header := range headers {
		clientIP := r.Header.Get(header)
		if clientIP != "" {
			return clientIP
		}
	}

	return r.RemoteAddr
}

func f_s_titleize(input string) string {
	input = strings.ReplaceAll(input, `_`, ` `)
	caser := cases.Title(language.English)
	return caser.String(input)
}

func f_i_online_users() int64 {
	return a_i_cached_online_counter.Load()
}

func f_m_online_entry() map[string]online_entry {
	mu_online_list.RLock()
	entries := m_online_list
	mu_online_list.Unlock()
	return entries
}

func f_i_online_cache_delay() int {
	return *flag_i_online_refresh_delay_minutes
}

func f_s_human_bytes(bytes int64, decimals int) string {
	format := fmt.Sprintf("%%.%df %%s", decimals)

	var result float64
	var suffix string

	switch {
	case bytes < kilobyte:
		return fmt.Sprintf(format, float64(bytes), "B")
	case bytes < megabyte:
		result = float64(bytes) / kilobyte
		suffix = "KB"
	case bytes < gigabyte:
		result = float64(bytes) / megabyte
		suffix = "MB"
	case bytes < terabyte:
		result = float64(bytes) / gigabyte
		suffix = "GB"
	case bytes < petabyte:
		result = float64(bytes) / terabyte
		suffix = "TB"
	default:
		result = float64(bytes) / petabyte
		suffix = "PB"
	}

	result = math.Round(result*math.Pow(10, float64(decimals))) / math.Pow(10, float64(decimals))

	return fmt.Sprintf(format, result, suffix)
}

func f_s_get_page_identifier_from_document_identifier_default_identifier_and_current_page_number(document_identifier string, current_page_identifier string, page_number int) string {
	var page_identifier string
	var page_data map[uint]string
	var is_found bool
	mu_document_pgno_page_identifier.RLock()
	page_data, is_found = m_document_pgno_page_identifier[document_identifier]
	mu_document_pgno_page_identifier.RUnlock()
	if !is_found {
		return current_page_identifier
	}
	var has_page bool
	page_identifier, has_page = page_data[uint(page_number)]
	if !has_page {
		return current_page_identifier
	}
	return page_identifier
}

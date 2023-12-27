package main

import (
	`context`
	`fmt`
	`html/template`
	`log`
	`net/http`
	`strings`
	`sync`
	`time`

	go_smartchan `github.com/andreimerlescu/go-smartchan`
	`github.com/gin-gonic/gin`

	`badbitchreads/sema`
)

func r_get_search(c *gin.Context) {
	a_i_waiting_room.Add(1)
	if sem_concurrent_searches.Len() > *flag_i_concurrent_searches {
		c.Redirect(http.StatusTemporaryRedirect, "/waiting-room")
		return
	}
	sem_concurrent_searches.Acquire()
	defer sem_concurrent_searches.Release()
	a_i_waiting_room.Add(-1)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(*flag_i_search_timeout_seconds))
	defer cancel()
	query := c.DefaultQuery("query", "")
	log.Printf("r_get_search using algorithm = %v ; query = %v", *flag_s_search_algorithm, query)

	if len(query) == 0 {
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
	}

	query_analysis := AnalyzeQuery(query)
	log.Printf("query_analysis = %v", query_analysis)

	var mu_inclusive sync.RWMutex
	var inclusive_page_identifiers map[string]struct{}
	inclusive_page_identifiers = make(map[string]struct{})

	var mu_exclusive sync.RWMutex
	var exclusive_page_identifiers map[string]struct{}
	exclusive_page_identifiers = make(map[string]struct{})

	var wg sync.WaitGroup
	sch_found_inclusive_identifiers := go_smartchan.NewSmartChan(*flag_i_search_concurrency_buffer)
	sch_found_exclusive_identifiers := go_smartchan.NewSmartChan(*flag_i_search_concurrency_buffer)
	ch_done_searching := make(chan struct{})

	sem_query_limiter := sema.New(*flag_i_search_concurrency_limiter)

	// ands
	for _, word := range query_analysis.Ands {
		wg.Add(1)
		sem_query_limiter.Acquire()
		go func(ctx context.Context, word string, sch *go_smartchan.SmartChan, sem sema.Semaphore) {
			defer wg.Done()
			defer sem.Release()
			err := find_pages_for_word(ctx, sch, word)
			if err != nil {
				log.Printf("failed to [AND] find_pages_for_word(%v) due to err: %v", word, err)
				return
			}
		}(ctx, word, sch_found_inclusive_identifiers, sem_query_limiter)
	}

	// nots
	for _, word := range query_analysis.Nots {
		wg.Add(1)
		sem_query_limiter.Acquire()
		go func(ctx context.Context, word string, sch *go_smartchan.SmartChan, sem sema.Semaphore) {
			defer wg.Done()
			defer sem.Release()
			err := find_pages_for_word(ctx, sch, word)
			if err != nil {
				log.Printf("failed to [NOT] find_pages_for_word(%v) due to err: %v", word, err)
				return
			}
		}(ctx, word, sch_found_exclusive_identifiers, sem_query_limiter)
	}

	go func() {
		wg.Wait()
		sch_found_inclusive_identifiers.Close()
		sch_found_exclusive_identifiers.Close()
		close(ch_done_searching)
	}()

	for {
		select {
		case <-ctx.Done():
			deliver_search_results(c, query, query_analysis, inclusive_page_identifiers, exclusive_page_identifiers)
			return
		case <-ch_done_searching:
			deliver_search_results(c, query, query_analysis, inclusive_page_identifiers, exclusive_page_identifiers)
			return
		case data, channel_open := <-sch_found_inclusive_identifiers.Chan():
			if channel_open {
				page_identifier, ok := data.(string)
				if ok {
					mu_inclusive.RLock()
					_, identifier_already_defined := inclusive_page_identifiers[page_identifier]
					mu_inclusive.RUnlock()
					if !identifier_already_defined {
						mu_inclusive.Lock()
						inclusive_page_identifiers[page_identifier] = struct{}{}
						mu_inclusive.Unlock()
					}
				} else {
					log.Printf("failed to cast data, channel_open := <-sch_found_inclusive_identifiers.Chan() as a string")
				}
			}
		case data, channel_open := <-sch_found_exclusive_identifiers.Chan():
			if channel_open {
				page_identifier, ok := data.(string)
				if ok {
					mu_exclusive.RLock()
					_, identifier_already_defined := exclusive_page_identifiers[page_identifier]
					mu_exclusive.RUnlock()
					if !identifier_already_defined {
						mu_exclusive.Lock()
						exclusive_page_identifiers[page_identifier] = struct{}{}
						mu_exclusive.Unlock()
					}
				}
			}
		}
	}

}

func r_get_search_results(c *gin.Context) {
	data, err := bundled_files.ReadFile("bundled/assets/html/search-results.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load search-results.html")
		return
	}

	tmpl := template.Must(template.New("search-results").Funcs(gin_func_map).Parse(string(data)))

	var htmlBuilder strings.Builder
	if err := tmpl.Execute(&htmlBuilder, gin.H{
		"title":             fmt.Sprintf("%v - Search Results", *flag_s_site_title),
		"company":           *flag_s_site_company,
		"domain":            *flag_s_primary_domain,
		"active_searches":   human_int(int64(sem_concurrent_searches.Len())),
		"i_active_searches": int64(sem_concurrent_searches.Len()),
		"max_searches":      human_int(int64(*flag_i_concurrent_searches)),
		"i_max_searches":    int64(*flag_i_concurrent_searches),
		"in_waiting_room":   human_int(a_i_waiting_room.Load()),
		"i_in_waiting_room": a_i_waiting_room.Load(),
		"dark_mode":         gin_is_dark_mode(c),
	}); err != nil {
		c.String(http.StatusInternalServerError, "error executing template", err)
		log.Println(err)
		return
	}
	c.Header("Content-Type", "text/html; charset=UTF-8")
	c.String(http.StatusOK, htmlBuilder.String())
}

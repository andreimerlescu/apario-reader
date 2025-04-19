package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	go_gematria "github.com/andreimerlescu/go-gematria"
	sema "github.com/andreimerlescu/go-sema"
	go_smartchan "github.com/andreimerlescu/go-smartchan"
	"github.com/gin-gonic/gin"
)

func r_get_search(c *gin.Context) {
	a_i_waiting_room.Add(1)
	if sem_concurrent_searches.Len() > *flag_i_concurrent_searches {
		c.Redirect(http.StatusTemporaryRedirect, "/waiting-room")
		return
	}
	requestedAt := time.Now().UTC()
	sem_concurrent_searches.Acquire()
	if since := time.Since(requestedAt).Seconds(); since > 1.7 {
		log.Printf("took %.0f seconds to acquire sem_concurrent_searches queue position", since)
	}
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
		requestedAt := time.Now().UTC()
		sem_query_limiter.Acquire()
		if since := time.Since(requestedAt).Seconds(); since > 1.7 {
			log.Printf("took %.0f seconds to acquire sem_query_limiter queue position", since)
		}
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
		requestedAt := time.Now().UTC()
		sem_query_limiter.Acquire()
		if since := time.Since(requestedAt).Seconds(); since > 1.7 {
			log.Printf("took %.0f seconds to acquire sem_query_limiter queue position", since)
		}
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

func deliver_search_results(c *gin.Context, query string, analysis SearchAnalysis, inclusive map[string]struct{}, exclusive map[string]struct{}) {
	data, err := bundled_files.ReadFile("bundled/assets/html/search-results.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load search-results.html")
		return
	}

	mu_gin_func_map.RLock()
	tmpl := template.Must(template.New("search-results").Funcs(gin_func_map).Parse(string(data)))
	mu_gin_func_map.RUnlock()

	mode := gin_is_dark_mode(c)

	template_vars := gin.H{
		"title":     fmt.Sprintf("%v - Search Results on %v", query, *flag_s_site_title),
		"company":   *flag_s_site_company,
		"domain":    *flag_s_primary_domain,
		"dark_mode": mode,
	}
	template_vars["inverse_dark_mode"] = "dark" // default is light mode, therefore inverse default is dark mode
	if mode == "dark" || mode == "1" {
		template_vars["inverse_dark_mode"] = "light"
	}

	s_page := c.DefaultQuery("page", "1")
	s_limit := c.DefaultQuery("limit", "12")

	var page int
	var limit int

	i_page, i_page_err := strconv.Atoi(s_page)
	if i_page_err != nil {
		page = 1
	} else {
		page = i_page
	}

	i_limit, i_limit_err := strconv.Atoi(s_limit)
	if i_limit_err != nil {
		limit = 12
	} else {
		limit = i_limit
	}

	template_vars["i_page"] = i_page
	template_vars["s_page"] = s_page
	template_vars["i_limit"] = i_limit
	template_vars["s_limit"] = s_limit
	template_vars["query"] = query

	var i_page_counter atomic.Int64
	var matching_page_identifiers map[uint]string = make(map[uint]string)
	for identifier, _ := range inclusive {
		_, excluded := exclusive[identifier]
		if !excluded {
			idx := i_page_counter.Add(1)
			matching_page_identifiers[uint(idx)] = identifier
		}
	}
	result := SearchResult{
		Query:    query,
		Analysis: analysis,
	}

	marshal, err := json.Marshal(result)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	log.Printf("json_output = %v\n", marshal)

	start_index := (page - 1) * limit
	end_index := start_index + limit

	total_document_pages := len(matching_page_identifiers)
	total_result_pages := (total_document_pages + limit - 1) / limit // Calculate total pages, rounding up
	var increased_index_by atomic.Int64
	var result_page_identifiers []string
	for i := start_index; i < end_index; i++ {
		identifier, found := matching_page_identifiers[uint(i)]
		if found {
			result_page_identifiers = append(result_page_identifiers, identifier)
		} else {
			total_increases := increased_index_by.Add(1)
			if total_increases < 3 {
				end_index++
			}
		}
	}

	result.Results = result_page_identifiers
	result.Total = len(matching_page_identifiers)

	query_gem_score, _ := go_gematria.NewGematria(result.Query)

	template_vars["total_matching_page_identifiers"] = len(matching_page_identifiers)
	template_vars["s_total_matching_page_identifiers"] = human_int(int64(len(matching_page_identifiers)))
	template_vars["total_result_pages"] = total_result_pages
	template_vars["s_total_result_pages"] = human_int(int64(total_result_pages))
	template_vars["result_page_identifiers"] = result_page_identifiers
	template_vars["total_results"] = len(result_page_identifiers)
	template_vars["s_total_results"] = human_int(int64(len(result_page_identifiers)))
	template_vars["GemScore"] = query_gem_score
	template_vars["from"] = query

	type GematriaPages struct {
		EnglishPages int
		JewishPages  int
		SimplePages  int
	}

	var gematria_pages GematriaPages

	mu_page_gematria_english.RLock()
	gematria_pages.EnglishPages = len(m_page_gematria_english[query_gem_score.English])
	mu_page_gematria_english.RUnlock()

	mu_page_gematria_jewish.RLock()
	gematria_pages.JewishPages = len(m_page_gematria_jewish[query_gem_score.Jewish])
	mu_page_gematria_jewish.RUnlock()

	mu_page_gematria_simple.RLock()
	gematria_pages.SimplePages = len(m_page_gematria_simple[query_gem_score.Simple])
	mu_page_gematria_simple.RUnlock()

	template_vars["gematria_pages"] = gematria_pages

	var htmlBuilder strings.Builder
	if err := tmpl.Execute(&htmlBuilder, template_vars); err != nil {
		c.String(http.StatusInternalServerError, "error executing template", err)
		log.Println(err)
		return
	}
	c.Header("Content-Type", "text/html; charset=UTF-8")
	c.String(http.StatusOK, htmlBuilder.String()) // serve html
	// http.ServeContent(c.Writer, c.Request, "", time.Now(), bytes.NewReader(marshal)) // serve json
	return
}

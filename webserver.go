package main

import (
	"bytes"
	"context"
	"crypto/tls"
	`encoding/json`
	`errors`
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	`sync`
	"time"

	go_smartchan `github.com/andreimerlescu/go-smartchan`
	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/didip/tollbooth_gin"
	"github.com/gin-gonic/gin"
	`github.com/xrash/smetrics`

	`badbitchreads/sema`
)

func NewWebServer(ctx context.Context) {
	once_server_start.Do(func() {
		defer func() {
			if ch_webserver_done.CanWrite() {
				err := ch_webserver_done.Write(struct{}{})
				if err != nil {
					log.Printf("failed to close ch_webserver_done due to err %v", err)
					return
				}
			}
		}()

		// Rate Limiting
		defaultRateLimiter := tollbooth.NewLimiter(*flag_f_rate_limit, &limiter.ExpirableOptions{
			DefaultExpirationTTL: time.Duration(*flag_i_rate_limit_entry_ttl) * time.Second,
			ExpireJobInterval:    time.Duration(*flag_i_rate_limit_cleanup_delay) * time.Second,
		})

		assetRateLimiter := tollbooth.NewLimiter(*flag_f_asset_rate_limit, &limiter.ExpirableOptions{
			DefaultExpirationTTL: time.Duration(*flag_i_asset_rate_limit_entry_ttl) * time.Second,
			ExpireJobInterval:    time.Duration(*flag_i_asset_rate_limit_cleanup_delay) * time.Second,
		})

		// Web Server Configuration
		r := gin.Default()

		// Rate Limit
		r.Use(tollbooth_gin.LimitHandler(defaultRateLimiter))

		// Content Security Policy
		if *flag_b_enable_csp {
			r.Use(middleware_content_security_policy)
		}

		// Cross Origin Resource Sharing (CORS)
		r.Use(func(c *gin.Context) {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(http.StatusOK)
				return
			}

			c.Next()
		})

		// Respond to a basic ping
		r.Any("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "pong",
			})
		})

		// Content Security Policy
		if *flag_b_enable_csp {
			r.POST(*flag_s_csp_report_uri, func(c *gin.Context) {
				var report map[string]interface{}
				if err := c.ShouldBindJSON(&report); err != nil {
					c.String(http.StatusBadRequest, "Invalid report data")
					return
				}

				log.Println("Received CSP report:", report)

				c.Status(http.StatusOK)
			})
		}

		// Serve all static assets using this entry point
		r.GET("/assets/:directory/:filename", tollbooth_gin.LimitHandler(assetRateLimiter), getAsset)

		// Go Web Server Index Path
		r.GET("/", func(c *gin.Context) {
			data, err := bundled_files.ReadFile("bundled/assets/html/index.html")
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to load index.html")
				return
			}

			tmpl := template.Must(template.New("index").Parse(string(data)))

			var htmlBuilder strings.Builder
			if err := tmpl.Execute(&htmlBuilder, gin.H{
				"title":           *flag_s_site_title,
				"company":         *flag_s_site_company,
				"domain":          *flag_s_primary_domain,
				"total_documents": human_int(a_i_total_documents.Load()),
				"total_pages":     human_int(a_i_total_pages.Load() - 1),
				"dark_mode":       getIfDarkMode(c),
			}); err != nil {
				c.String(http.StatusInternalServerError, "error executing template", err)
				log.Println(err)
				return
			}
			c.Header("Content-Type", "text/html; charset=UTF-8")
			c.String(http.StatusOK, htmlBuilder.String())
		})

		// Web App Routes
		r.GET("/search", getSearch)

		r.GET("/dark", func(c *gin.Context) {
			c.SetCookie(*flag_s_dark_mode_cookie, strconv.Itoa(1), 31881600, "/", *flag_s_primary_domain, false, true)
			c.SetCookie(*flag_s_dark_mode_cookie, strconv.Itoa(1), 31881600, "/", *flag_s_primary_domain, true, true)
			c.Redirect(http.StatusTemporaryRedirect, "/")
		})

		r.GET("/light", func(c *gin.Context) {
			c.SetCookie(*flag_s_dark_mode_cookie, strconv.Itoa(0), 31881600, "/", *flag_s_primary_domain, false, true)
			c.SetCookie(*flag_s_dark_mode_cookie, strconv.Itoa(0), 31881600, "/", *flag_s_primary_domain, true, true)
			c.Redirect(http.StatusTemporaryRedirect, "/")
		})

		// Start HTTP Server
		go func(r *gin.Engine) {
			server := &http.Server{
				Addr:    ":" + strconv.Itoa(*flag_i_webserver_default_port),
				Handler: r,
			}

			go func() {
				if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					fatalf_log("listen: %s\n", err)
				}
			}()

			<-ctx.Done()

			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := server.Shutdown(shutdownCtx); err != nil {
				fatalf_stderr("Server Shutdown Failed:%+v", err)
			}

			log.Println("Server exiting properly")
		}(r)

		// Start HTTPS Server
		go func(r *gin.Engine) {
			cert = loadSSLCertificate()
			startCertReloader()
			server := &http.Server{
				Addr:    ":" + strconv.Itoa(*flag_i_webserver_secure_port),
				Handler: r,
				TLSConfig: &tls.Config{
					GetCertificate: getCertificate,
				},
			}

			go func() {
				if err := server.ListenAndServeTLS("", ""); !errors.Is(err, http.ErrServerClosed) {
					fatalf_stderr("ListenAndServeTLS(): %s", err)
				}
			}()

			<-ctx.Done()
			ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := server.Shutdown(ctxShutDown); err != nil {
				fatalf_stderr("Server forced to shutdown: %s", err)
			}

			log.Println("Server exiting properly")
		}(r)

		// Wait for the main context to be canceled
		for {
			select {
			case <-ctx.Done():
				return
			}
		}
	})
}

func getIfDarkMode(c *gin.Context) string {
	// 0 = light mode ; 1 = dark mode
	dark_mode, dark_mode_err := c.Cookie(*flag_s_dark_mode_cookie)
	if dark_mode_err != nil {
		return "0"
	} else {
		if dark_mode == "0" || dark_mode == "1" {
			return dark_mode
		} else {
			return "0"
		}
	}
}

func getIcon(c *gin.Context) {
	name := c.Param("name")

	filePath := fmt.Sprintf("bundled/assets/icons/%v", name)

	fileData, err := bundled_files.ReadFile(filePath)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	switch {
	case strings.HasSuffix(name, ".js"):
		c.Header("Content-Type", "text/javascript")
	case strings.HasSuffix(name, ".css"):
		c.Header("Content-Type", "text/css")
	case strings.HasSuffix(name, ".woff"):
		c.Header("Content-Type", "font/woff")
	case strings.HasSuffix(name, ".woff2"):
		c.Header("Content-Type", "font/woff2")
	case strings.HasSuffix(name, ".ico"):
		c.Header("Content-Type", "image/x-icon")
	case strings.HasSuffix(name, ".jpg"):
		c.Header("Content-Type", "image/jpeg")
	case strings.HasSuffix(name, ".png"):
		c.Header("Content-Type", "image/png")
	case strings.HasSuffix(name, ".svg"):
		c.Header("Content-Type", "image/svg+xml")
	default:
		c.String(http.StatusInternalServerError, "unsupported image type")
		return
	}

	modTime := time.Now()
	http.ServeContent(c.Writer, c.Request, "", modTime, bytes.NewReader(fileData))
}

func getAsset(c *gin.Context) {
	directory := c.Param("directory")
	filename := c.Param("filename")
	filePath := fmt.Sprintf("bundled/assets/%v/%v", directory, filename)

	fileData, err := bundled_files.ReadFile(filePath)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	switch {
	case strings.HasSuffix(filename, ".csv"):
		c.Header("Content-Type", "text/csv")
	case strings.HasSuffix(filename, ".eot"):
		c.Header("Content-Type", "application/vnd.ms-fontobject")
	case strings.HasSuffix(filename, ".epub"):
		c.Header("Content-Type", "application/epub+zip")
	case strings.HasSuffix(filename, ".gif"):
		c.Header("Content-Type", "image/gif")
	case strings.HasSuffix(filename, ".otf"):
		c.Header("Content-Type", "font/otf")
	case strings.HasSuffix(filename, ".pdf"):
		c.Header("Content-Type", "application/pdf")
	case strings.HasSuffix(filename, ".txt"):
		c.Header("Content-Type", "text/plain")
	case strings.HasSuffix(filename, ".js"):
		c.Header("Content-Type", "text/javascript")
	case strings.HasSuffix(filename, ".css"):
		c.Header("Content-Type", "text/css")
	case strings.HasSuffix(filename, ".woff"):
		c.Header("Content-Type", "font/woff")
	case strings.HasSuffix(filename, ".woff2"):
		c.Header("Content-Type", "font/woff2")
	case strings.HasSuffix(filename, ".ico"):
		c.Header("Content-Type", "image/x-icon")
	case strings.HasSuffix(filename, ".jpg"):
		c.Header("Content-Type", "image/jpeg")
	case strings.HasSuffix(filename, ".png"):
		c.Header("Content-Type", "image/png")
	case strings.HasSuffix(filename, ".svg"):
		c.Header("Content-Type", "image/svg+xml")
	case strings.HasSuffix(filename, ".map"):
		c.Header("Content-Type", "application/json")
	default:
		c.String(http.StatusInternalServerError, "unsupported image type")
		return
	}

	http.ServeContent(c.Writer, c.Request, "", time.Now(), bytes.NewReader(fileData))
}

func getSearch(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(*flag_i_search_timeout_seconds))
	defer cancel()
	query := c.DefaultQuery("query", "")
	log.Printf("getSearch using algorithm = %v ; query = %v", *flag_s_search_algorithm, query)

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

func find_pages_for_word(ctx context.Context, sch *go_smartchan.SmartChan, query string) error {
	var results = make(map[string]struct{})
	mu_word_pages.RLock()
	for word, pages := range m_word_pages {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		var distance float64
		var can_use_hamming bool = len(query) == len(word)
		if len(word) == 0 || len(query) == 0 {
			continue
		}
		if *flag_s_search_algorithm == "jaro" {
			distance = smetrics.Jaro(query, word)
			if distance >= *flag_f_search_jaro_threshold {
				for page_identifier, _ := range pages {
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
					}
					if sch.CanWrite() {
						err := sch.Write(page_identifier)
						if err != nil {
							return err
						}
					}
				}
			}
		} else if *flag_s_search_algorithm == "soundex" {
			query_soundex := smetrics.Soundex(query)
			word_soundex := smetrics.Soundex(word)
			if query_soundex == word_soundex {
				for page_identifier, _ := range pages {
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
					}
					if sch.CanWrite() {
						err := sch.Write(page_identifier)
						if err != nil {
							return err
						}
					}
				}
			}
		} else if *flag_s_search_algorithm == "ukkonen" {
			score := smetrics.Ukkonen(query, word, *flag_i_search_ukkonen_icost, *flag_i_search_ukkonen_dcost, *flag_i_search_ukkonen_scost)
			if score <= *flag_i_search_ukkonen_max_substitutions {
				for page_identifier, _ := range pages {
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
					}
					if sch.CanWrite() {
						err := sch.Write(page_identifier)
						if err != nil {
							return err
						}
					}
				}
			}
		} else if *flag_s_search_algorithm == "wagner_fischer" {
			score := smetrics.WagnerFischer(query, word, *flag_i_search_wagner_fischer_icost, *flag_i_search_wagner_fischer_dcost, *flag_i_search_wagner_fischer_scost)
			if score <= *flag_i_search_wagner_fischer_max_substitutions {
				for page_identifier, _ := range pages {
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
					}
					if sch.CanWrite() {
						err := sch.Write(page_identifier)
						if err != nil {
							return err
						}
					}
				}
			}
		} else if *flag_s_search_algorithm == "hamming" && can_use_hamming {
			substitutions, err := smetrics.Hamming(query, word)
			if err != nil {
				return fmt.Errorf("error received when performing Hamming analysis: %v", err)
			}
			if substitutions <= *flag_i_search_hamming_max_substitutions {
				for page_identifier, _ := range pages {
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
					}
					if sch.CanWrite() {
						err := sch.Write(page_identifier)
						if err != nil {
							return err
						}
					}
				}
			}
		} else { // use jaro_winkler
			distance = smetrics.JaroWinkler(query, word, *flag_f_search_jaro_winkler_boost_threshold, *flag_i_search_jaro_winkler_prefix_size)
			if distance >= *flag_f_search_jaro_winkler_threshold {
				for page_identifier, _ := range pages {
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
					}
					if sch.CanWrite() {
						err := sch.Write(page_identifier)
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}
	mu_word_pages.RUnlock()
	if len(results) == 0 {
		return fmt.Errorf("no results for %v", query)
	}
	return nil
}

func deliver_search_results(c *gin.Context, query string, analysis SearchAnalysis, inclusive map[string]struct{}, exclusive map[string]struct{}) {
	var page_identifiers []string
	for identifier, _ := range inclusive {
		_, excluded := exclusive[identifier]
		if !excluded {
			page_identifiers = append(page_identifiers, identifier)
		}
	}
	result := SearchResult{
		Query:    query,
		Analysis: analysis,
		Total:    len(page_identifiers),
		//Inclusive: inclusive_page_identifiers,
		//Exclusive: exclusive_page_identifiers,
		Results: page_identifiers,
	}

	marshal, err := json.Marshal(result)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	http.ServeContent(c.Writer, c.Request, "", time.Now(), bytes.NewReader(marshal))
	return
}

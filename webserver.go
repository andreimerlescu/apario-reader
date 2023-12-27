package main

import (
	"context"
	"crypto/tls"
	`errors`
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/didip/tollbooth_gin"
	"github.com/gin-gonic/gin"
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

		gin_func_map = template.FuncMap{
			"render_partial": render_partial_template,
		}
		default_gin_func_vars = gin.H{
			"company": *flag_s_site_company,
			"domain":  *flag_s_primary_domain,
		}
		gin_func_vars = map[string]gin.H{
			"head":   default_gin_func_vars,
			"header": default_gin_func_vars,
			"foot":   default_gin_func_vars,
			"footer": default_gin_func_vars,
		}

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
		r.GET("/", r_get_index)

		// Web App Routes
		r.GET("/search", r_get_search)

		r.GET("/waiting-room", gin_get_waiting_room)

		r.GET("/legal/community-standards", r_get_legal_community_standards)
		r.GET("/legal/coppa", r_get_legal_coppa)
		r.GET("/legal/gdpr", r_get_legal_gdpr)
		r.GET("/legal/privacy", r_get_legal_privacy_policy)
		r.GET("/legal/terms", r_get_legal_terms)
		r.GET("/contact", r_get_contact_us)
		r.GET("/status", r_get_status)
		r.GET("/documents", r_get_documents)
		r.GET("/document/:identifier", r_get_view_document)
		r.GET("/file/:filename", r_get_view_file)
		r.GET("/gematria/:type/:number", r_get_gematria)
		r.GET("/page/:identifier", r_get_page)
		r.GET("/words", r_get_words)
		r.GET("/word/:word", r_get_word)
		r.GET("/stumbleinto", r_get_stumble_into)
		r.GET("/search-results", r_get_search_results)

		r.GET("/dark", func(c *gin.Context) {
			c.SetCookie(*flag_s_dark_mode_cookie, strconv.Itoa(1), 31881600, "/", *flag_s_cookie_domain, false, true)
			c.SetCookie(*flag_s_dark_mode_cookie, strconv.Itoa(1), 31881600, "/", *flag_s_cookie_domain, true, true)
			c.Redirect(http.StatusTemporaryRedirect, "/")
		})

		r.GET("/light", func(c *gin.Context) {
			c.SetCookie(*flag_s_dark_mode_cookie, strconv.Itoa(0), 31881600, "/", *flag_s_cookie_domain, false, true)
			c.SetCookie(*flag_s_dark_mode_cookie, strconv.Itoa(0), 31881600, "/", *flag_s_cookie_domain, true, true)
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

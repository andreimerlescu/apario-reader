package main

import (
	"context"
	"crypto/tls"
	`errors`
	"html/template"
	`io`
	"log"
	"net/http"
	`os`
	"strconv"
	"time"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
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
			"render_partial":        render_partial_template,
			"render_document_card":  render_document_card,
			"render_page_card":      render_page_card,
			"render_page_card_from": render_page_card_from,
			"render_page_detail":    render_page_detail,
			"plus":                  f_i_plus,
			"minus":                 f_i_minus,
			"random_int":            f_i_random_int,
			"sequence":              f_i_sequence,
			"max":                   f_i_max,
			"min":                   f_i_min,
			"human_bytes":           f_s_human_bytes,
			"online_users":          f_i_online_users,
			"m_online_entry":        f_m_online_entry,
			"online_cache_delay":    f_i_online_cache_delay,
			"titleize":              f_s_titleize,
			"hits":                  f_i_hits,
			"get_pg_id_from_doc_id_def_id_and_cur_pg_num": f_s_get_page_identifier_from_document_identifier_default_identifier_and_current_page_number,
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
		var routeRateLimiter *limiter.Limiter
		if *flag_b_enable_rate_limiting {
			routeRateLimiter = tollbooth.NewLimiter(*flag_f_rate_limit, &limiter.ExpirableOptions{
				DefaultExpirationTTL: time.Duration(*flag_i_rate_limit_entry_ttl) * time.Second,
				ExpireJobInterval:    time.Duration(*flag_i_rate_limit_cleanup_delay) * time.Second,
			})
		}

		var assetRateLimiter *limiter.Limiter
		if *flag_b_enable_asset_rate_limiting {
			assetRateLimiter = tollbooth.NewLimiter(*flag_f_asset_rate_limit, &limiter.ExpirableOptions{
				DefaultExpirationTTL: time.Duration(*flag_i_asset_rate_limit_entry_ttl) * time.Second,
				ExpireJobInterval:    time.Duration(*flag_i_asset_rate_limit_cleanup_delay) * time.Second,
			})
		}

		var downloadRateLimiter *limiter.Limiter
		if *flag_b_enable_downloads_rate_limiting {
			downloadRateLimiter = tollbooth.NewLimiter(*flag_f_downloads_rate_limit, &limiter.ExpirableOptions{
				DefaultExpirationTTL: time.Duration(*flag_i_downloads_rate_limit_entry_ttl) * time.Second,
				ExpireJobInterval:    time.Duration(*flag_i_downloads_rate_limit_cleanup_delay) * time.Second,
			})
		}

		// gin logs
		f, f_err := os.OpenFile(*flag_s_gin_log_file, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0600)
		if f_err == nil { // no error received
			if *flag_b_gin_log_to_stdout { // logging to STDOUT + log file
				gin.DefaultWriter = io.MultiWriter(f, os.Stdout)
			} else {
				gin.DisableConsoleColor() // disable colors for logging to log file only
				gin.DefaultWriter = io.MultiWriter(f)
			}
		} // if there was an err with opening the gin log file, default to gin's default behavior

		// Web Server Configuration
		r := gin.New()      // don't use .Default() here since we want Recover() to be disabled manually
		r.Use(gin.Logger()) // Enable gin logging

		if *flag_s_environment == *flag_s_production_environment_label && len(*flag_s_production_environment_label) > 0 {
			gin.SetMode(gin.ReleaseMode)
			r.Use(gin.Recovery()) // enable recovery only in production mode
		} else {
			gin.SetMode(gin.DebugMode)
		}

		r.Use(middleware_database_loaded())

		// Middleware
		if *flag_b_enable_tls_handshake_error_check {
			r.Use(middleware_tls_handshake())
		}
		if *flag_b_enable_ip_ban_list {
			r.Use(middleware_enforce_ip_ban_list())
			go f_schedule_ip_ban_list_cleanup(ctx)
		}

		// Special Routes
		r.GET("/robots.txt", middleware_online_counter(), r_get_robots_txt)
		if *flag_b_enable_ads_txt {
			r.GET("/ads.txt", middleware_online_counter(), r_get_ads_txt)
		}

		if *flag_b_enable_security_txt {
			r.GET("/security.txt", middleware_online_counter(), r_get_security_txt)
		}

		if *flag_b_enable_ping {
			r.Any("/ping", middleware_online_counter(), r_get_ping)
		}

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
		if *flag_b_enable_asset_rate_limiting {
			r.GET("/assets/:directory/:filename", middleware_rate_limiter(assetRateLimiter), r_get_asset)
			r.GET("/covers/:document_identifier/:page_identifier/:size", middleware_rate_limiter(assetRateLimiter), r_get_database_page_image)
			r.GET("/pending-viewport-placeholder.svg", middleware_rate_limiter(assetRateLimiter), r_get_pending_viewport_placeholder_svg)
		} else {
			r.GET("/assets/:directory/:filename", r_get_asset)
			r.GET("/covers/:document_identifier/:page_identifier/:size", r_get_database_page_image)
			r.GET("/pending-viewport-placeholder.svg", r_get_pending_viewport_placeholder_svg)
		}

		if *flag_b_enable_downloads_rate_limiting {
			r.GET("/download/document/:document_identifier/:filename", middleware_rate_limiter(downloadRateLimiter), middleware_online_counter(), r_get_download_document)
			r.GET("/download/page/:page_identifier/:filename", middleware_rate_limiter(downloadRateLimiter), middleware_online_counter(), r_get_download_page)
		} else {
			r.GET("/download/document/:document_identifier/:filename", middleware_online_counter(), r_get_download_document)
			r.GET("/download/page/:page_identifier/:filename", middleware_online_counter(), r_get_download_page)
		}

		if *flag_b_enable_rate_limiting {
			r.Use(middleware_rate_limiter(routeRateLimiter))
		}
		if *flag_b_enable_csp {
			r.Use(middleware_content_security_policy())
		}
		if *flag_b_enable_cors {
			r.Use(middleware_cross_origin_request_scripts())
		}

		go clean_online_counter_scheduler(ctx)
		go load_online_counter_cache_scheduler(ctx)

		// Online hit counter
		r.Use(middleware_count_hits())
		go persist_hits_offline(ctx)

		// Routes
		r.GET("/", middleware_online_counter(), r_get_index)
		r.GET("/search", middleware_online_counter(), r_get_search)
		r.GET("/waiting-room", middleware_online_counter(), gin_get_waiting_room)
		r.GET("/legal/community-standards", middleware_online_counter(), r_get_legal_community_standards)
		r.GET("/legal/coppa", middleware_online_counter(), r_get_legal_coppa)
		r.GET("/legal/gdpr", middleware_online_counter(), r_get_legal_gdpr)
		r.GET("/legal/privacy", middleware_online_counter(), r_get_legal_privacy_policy)
		r.GET("/legal/terms", middleware_online_counter(), r_get_legal_terms)
		r.GET("/legal/license", middleware_online_counter(), r_get_legal_license)
		r.GET("/contact", middleware_online_counter(), r_get_contact_us)
		r.GET("/status", middleware_online_counter(), r_get_status)
		r.GET("/documents", middleware_online_counter(), r_get_documents)
		r.GET("/document/:identifier", middleware_online_counter(), r_get_view_document)
		r.GET("/gematria/:type/:number", middleware_online_counter(), r_get_gematria)
		r.GET("/page/:identifier", middleware_online_counter(), r_get_page)
		r.GET("/words", middleware_online_counter(), r_get_words)
		r.GET("/word/:word", middleware_online_counter(), r_get_word)
		r.GET("/stumbleinto", middleware_online_counter(), r_get_stumble_into)
		r.GET("/StumbleInto", middleware_online_counter(), r_get_stumble_into)
		r.GET("/dark", middleware_online_counter(), r_get_dark)
		r.GET("/light", middleware_online_counter(), r_get_light)

		//// devise inspired authentication
		//r.GET("/profile/:username", r_get_public_profile)
		//r_account_group := r.Group("/account")
		//{
		//	// added security
		//	r_account_group.Use(middleware_force_https())             // force https for all /account related actions
		//	r_account_group.Use(middleware_enforce_ip_ban_list())     // re-run for all /account related actions
		//	r_account_group.Use(middleware_use_authenticity_tokens()) // ensure that the authenticity token exists within the session for validation
		//
		//	// login
		//	r_account_group.GET("/login", r_get_login)
		//	r_account_group.POST("/login", middleware_enforce_authenticity_token(), r_post_login)
		//
		//	// log out
		//	r_account_group.GET("/logout", r_get_logout)
		//	r_account_group.DELETE("/logout", middleware_enforce_authenticity_token(), r_get_logout)
		//	r_account_group.DELETE("/session", middleware_enforce_authenticity_token(), r_get_logout)
		//
		//	// for the remainder of the routes, use this middleware
		//	r_account_group.Use(middleware_ensure_authenticated())
		//
		//	// update profile
		//	r_account_group.GET("/profile", r_get_manage_profile)
		//
		//	// new account
		//	r_account_group.GET("/register", r_get_register)
		//	r_account_group.POST("/register", middleware_enforce_authenticity_token(), r_post_register)
		//
		//	// change account email
		//	r_account_group.GET("/email", r_get_change_email)
		//	r_account_group.POST("/email", r_post_change_email)
		//
		//	// challenge user for account password with redirect
		//	r_account_group.GET("/challenge", r_get_challenge_password)
		//	r_account_group.POST("/challenge", middleware_enforce_authenticity_token(), r_post_challenge_password)
		//
		//	// enforce account bans
		//	r_account_group.GET("/banned", r_get_account_banned)
		//
		//	// enforce account locks
		//	r_account_group.GET("/locked", r_get_account_locked)
		//
		//	// change password
		//	r_account_group.GET("/reset", r_get_forgot_password)
		//	r_account_group.POST("/reset", middleware_enforce_authenticity_token(), r_post_forgot_password)
		//
		//	// download account data
		//	r_account_group.GET("/download", r_get_download_account_data)
		//	r_account_group.POST("/download", middleware_enforce_authenticity_token(), r_post_download_account_data)
		//
		//	// delete account data
		//	r_account_group.GET("/delete", r_get_request_account_deletion)
		//	r_account_group.POST("/delete", middleware_enforce_authenticity_token(), r_post_request_account_deletion)
		//	r_account_group.DELETE("/delete", middleware_enforce_authenticity_token(), r_get_request_account_deletion)
		//}

		r.NoRoute(handler_no_route_linter())

		// Start HTTP Server
		go func(r *gin.Engine) {
			if *flag_b_redirect_http_to_https {
				r.Use(middleware_force_https())
			}

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

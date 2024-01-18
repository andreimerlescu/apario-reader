package main

import (
	`fmt`
	`log`
	`net`
	`net/http`
	`net/url`
	"strings"
	`sync/atomic`
	`time`

	"github.com/gin-gonic/gin"
)

func handler_cross_origin_request_scripts(c *gin.Context) {
	var allow_creds string
	if *flag_s_cors_allow_credentials == true {
		allow_creds = "true"
	} else {
		allow_creds = "false"
	}
	c.Writer.Header().Set("Access-Control-Allow-Origin", *flag_s_cors_allow_origin)
	c.Writer.Header().Set("Access-Control-Allow-Methods", *flag_s_cors_allow_methods)
	c.Writer.Header().Set("Access-Control-Allow-Headers", *flag_s_cors_allow_headers)
	c.Writer.Header().Set("Access-Control-Allow-Credentials", allow_creds)

	if c.Request.Method == "OPTIONS" {
		c.AbortWithStatus(http.StatusOK)
		return
	}
}

func handler_wait_for_database(c *gin.Context) {
	if !a_b_database_loaded.Load() {
		c.AbortWithStatusJSON(http.StatusNotAcceptable, map[string]string{"error": "Please wait for the application to boot."})
	}
}

func handler_ensure_authenticated(c *gin.Context) {
	err := f_gin_check_authenticated_in_session(c)
	if err != nil {
		session := g_get_session(c)
		var last_failed_attempt time.Time
		_, last_failed_attempts_exist := session.Values["last_failed_login_attempt"]
		if !last_failed_attempts_exist {
			f_session_flash(session, NewSessionAlert("error", "Authentication is required to proceed.", time.Now().Add(45*time.Second), true))
			referer := c.GetHeader("Referer")
			if len(referer) > 0 {
				c.Redirect(http.StatusOK, fmt.Sprintf("/login?from=%v", url.PathEscape(referer)))
				return
			}
			c.Redirect(http.StatusOK, "/login")
			return
		}

		var valid_format bool
		last_failed_attempt, valid_format = session.Values["last_failed_login_attempt"].(time.Time)
		if !valid_format {
			session.Values["last_failed_login_attempt"] = time.Now().UTC()
		}

		var failed_attempts int
		var next_failed_attempts int
		_, failed_attempts_exists := session.Values["failed_attempts"]
		if failed_attempts_exists {
			last_value, valid_value := session.Values["failed_attempts"].(int)
			if !valid_value {
				session.Values["failed_attempts"] = 1
			} else {
				failed_attempts = last_value
				next_failed_attempts = last_value + 1
				session.Values["failed_attempts"] = next_failed_attempts
			}
		}

		maybe_username, username_exists := session.Values["username"]
		if !username_exists {
			f_session_flash(session, NewSessionAlert("error", "Authentication is required to proceed.", time.Now().Add(45*time.Second), true))
			referer := c.GetHeader("Referer")
			if len(referer) > 0 {
				c.Redirect(http.StatusOK, fmt.Sprintf("/login?from=%v", url.PathEscape(referer)))
				return
			}
			c.Redirect(http.StatusOK, "/login")
			return
		}

		username, is_username := maybe_username.(string)
		if !is_username {
			f_session_flash(session, NewSessionAlert("error", "Authentication is required to proceed.", time.Now().Add(45*time.Second), true))
			referer := c.GetHeader("Referer")
			if len(referer) > 0 {
				c.Redirect(http.StatusOK, fmt.Sprintf("/login?from=%v", url.PathEscape(referer)))
				return
			}
			c.Redirect(http.StatusOK, "/login")
			return
		}

		account, load_err := load_username_account(username)
		if load_err != nil {
			f_session_flash(session, NewSessionAlert("error", "Authentication is required to proceed.", time.Now().Add(45*time.Second), true))
			referer := c.GetHeader("Referer")
			if len(referer) > 0 {
				c.Redirect(http.StatusOK, fmt.Sprintf("/login?from=%v", url.PathEscape(referer)))
				return
			}
			c.Redirect(http.StatusOK, "/login")
			return
		}

		account.LastFailedLogin = time.Now().UTC()
		account.LastFailedLoginIP = net.IP(f_s_client_ip(c.Request))
		account.FailedLoginAttempts = failed_attempts
		err := store_username_account(account)
		if err != nil {
			log.Printf("failed to store_username_account for account %v", account)
		}

		if time.Since(last_failed_attempt) < time.Duration(fibonacci(failed_attempts))*time.Second*3 {
			if failed_attempts >= *flag_i_auth_max_failed_logins {
				account.Locked = true
				account.LockedAt = time.Now().UTC()
				account.LockedByIP = net.IP(f_s_client_ip(c.Request))
				err := store_username_account(account)
				if err != nil {
					log.Printf("failed to store_username_account for account %v", account)
				}
			}
			f_session_flash(session, NewSessionAlert("error", fmt.Sprintf("You must wait %.0f seconds before trying to sign in again.", time.Since(last_failed_attempt).Seconds()), time.Now().Add(45*time.Second), true))
			c.Redirect(http.StatusOK, "/login")
			return
		}

		f_session_flash(session, NewSessionAlert("error", "Authentication is required to proceed.", time.Now().Add(45*time.Second), true))
		referer := c.GetHeader("Referer")
		if len(referer) > 0 {
			c.Redirect(http.StatusOK, fmt.Sprintf("/login?from=%v", url.PathEscape(referer)))
			return
		}
		c.Redirect(http.StatusOK, "/login")
		return
	}
}

func handler_force_https(c *gin.Context) {
	if *flag_b_redirect_http_to_https {
		r_url := c.Request.URL
		r_url.Scheme = "https"
		r_url.Host = c.Request.Host
		c.Redirect(http.StatusMovedPermanently, r_url.String())
		return
	}
}

func handler_online_counter(c *gin.Context) {
	ip := f_s_filtered_ip(c)
	mu_online_list.RLock()
	entry, exists := m_online_list[ip]
	mu_online_list.RUnlock()
	if !exists {
		mu_online_list.Lock()
		m_online_list[ip] = online_entry{
			UserAgent:     c.Request.Header.Get("User-Agent"),
			IP:            net.IP(ip),
			FirstAction:   time.Now().UTC(),
			LastAction:    time.Now().UTC(),
			Hits:          &atomic.Int64{},
			LastPath:      c.Request.URL.Path,
			Authenticated: false,
			Administrator: false,
			Username:      "",
			Reputation:    0,
		}
		mu_online_list.Unlock()
		c.Next()
		return
	}

	entry.Hits.Add(1)
	entry.LastPath = c.Request.URL.Path
	entry.LastAction = time.Now().UTC()
	entry.UserAgent = c.Request.Header.Get("User-Agent")

	mu_online_list.Lock()
	m_online_list[ip] = entry
	mu_online_list.Unlock()
}

func handler_tls_handshake(c *gin.Context) {
	// Check if the client's TLS handshake is misconfigured
	if c.Request.TLS != nil && c.Request.TLS.HandshakeComplete == false {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "TLS handshake misconfiguration",
		})
		c.Abort()
		return
	}
}

func handler_content_security_policy(c *gin.Context) {
	// List of 1st party domains
	var domains []string
	if len(*flag_s_csp_domains_csv) > 1 {
		// parse the flag/config CSV values and sanitize the string
		_domains := strings.Split(*flag_s_csp_domains_csv, ",")
		for _, domain := range _domains {
			domain = strings.ReplaceAll(domain, " ", "")
			if len(domain) > 0 {
				domains = append(domains, domain)
			}
		}
	}
	// List of web socket domains
	var wsDomains []string
	if len(*flag_s_csp_websocket_domains_csv) > 1 {
		// parse the flag/config CSV values and sanitize the string
		_domains := strings.Split(*flag_s_csp_websocket_domains_csv, ",")
		for _, domain := range _domains {
			domain = strings.ReplaceAll(domain, " ", "")
			if len(domain) > 0 {
				wsDomains = append(wsDomains, domain)
			}
		}
	}
	// List of domains for third party styles
	var thirdPartyStyles []string
	if len(*flag_s_csp_thirdparty_styles_csv) > 1 {
		// parse the flag/config CSV values and sanitize the string
		_domains := strings.Split(*flag_s_csp_thirdparty_styles_csv, ",")
		for _, domain := range _domains {
			domain = strings.ReplaceAll(domain, " ", "")
			if len(domain) > 0 {
				thirdPartyStyles = append(thirdPartyStyles, domain)
			}
		}
	}
	// List of domains allowed for thirdParty usage
	var thirdParty []string
	if len(*flag_s_csp_thirdparty_csv) > 1 {
		// parse the flag/config CSV values and sanitize the string
		_domains := strings.Split(*flag_s_csp_thirdparty_csv, ",")
		for _, domain := range _domains {
			domain = strings.ReplaceAll(domain, " ", "")
			if len(domain) > 0 {
				thirdParty = append(thirdParty, domain)
			}
		}
	}

	// Script execution protection policy defaults
	script_unsafe_inline := ""
	script_unsafe_eval := ""
	child_unsafe_inline := ""
	style_unsafe_inline := ""
	upgrade_insecure := ""
	block_mixed := ""

	// Depending on config flags, set the policy defaults
	if *flag_b_csp_script_enable_unsafe_inline {
		script_unsafe_inline = "'unsafe-inline'"
	}
	if *flag_b_csp_script_enable_unsafe_eval {
		script_unsafe_eval = "'unsafe-eval'"
	}
	if *flag_b_csp_child_src_enable_unsafe_inline {
		child_unsafe_inline = "'unsafe-inline'"
	}
	if *flag_b_csp_style_src_enable_unsafe_inline {
		style_unsafe_inline = "'unsafe-inline'"
	}
	if *flag_b_csp_upgrade_unsecure_requests {
		upgrade_insecure = "upgrade-insecure-requests;"
	}
	if *flag_b_csp_block_mixed_content {
		block_mixed = "block-all-mixed-content;"
	}

	c.Writer.Header().Set("Content-Security-Policy",
		"default-src 'self' "+strings.Join(domains, " ")+" "+strings.Join(thirdParty, " ")+"; "+
			"font-src 'self' data: "+strings.Join(domains, " ")+"; "+
			"img-src 'self' data: blob: "+strings.Join(domains, " ")+" "+strings.Join(thirdParty, " ")+"; "+
			"object-src 'self' "+strings.Join(domains, " ")+" "+strings.Join(thirdParty, " ")+"; "+
			"script-src 'self' "+script_unsafe_inline+" "+script_unsafe_eval+" "+strings.Join(domains, " ")+" "+strings.Join(thirdParty, " ")+"; "+
			"frame-src 'self' "+strings.Join(domains, " ")+" "+strings.Join(thirdParty, " ")+";"+
			"child-src 'self' "+child_unsafe_inline+" blob: data: "+strings.Join(domains, " ")+" "+strings.Join(thirdParty, " ")+"; "+
			"style-src data: "+style_unsafe_inline+" "+strings.Join(domains, " ")+" "+strings.Join(thirdPartyStyles, " ")+"; "+
			"connect-src 'self' blob: "+strings.Join(domains, " ")+" "+strings.Join(wsDomains, " ")+"; "+
			"report-uri "+*flag_s_csp_report_uri+"; "+
			upgrade_insecure+
			block_mixed)

	// Process the next Gin middleware
}

func handler_enforce_ip_ban_list(c *gin.Context) {
	ip := f_s_filtered_ip(c)
	if len(ip) == 0 {
		log.Printf("invalid ip address detected")
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if f_ip_in_ban_list(net.ParseIP(ip)) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
}

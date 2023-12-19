package main

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func middleware_content_security_policy(c *gin.Context) {
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
	c.Next()
}

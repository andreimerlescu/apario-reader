package main

import (
	`fmt`
	`html/template`
	`log`
	`net`
	`net/http`
	`os`
	`strconv`
	`strings`

	`github.com/gin-gonic/gin`
)

func r_render_static(path string, c *gin.Context) (string, error) {
	filename := fmt.Sprintf("bundled/assets/html/%v.html", path)
	data, err := bundled_files.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to load %v due to %v", path, err)
	}

	tmpl := template.Must(template.New(path).Funcs(gin_func_map).Parse(string(data)))

	if a_i_total_documents.Load() == 0 {
		a_i_total_documents.Store(int64(len(m_document_total_pages)))
	}

	if a_i_total_pages.Load() == 0 {
		a_i_total_pages.Store(int64(len(m_page_identifier_document)))
	}

	var htmlBuilder strings.Builder
	if err := tmpl.Execute(&htmlBuilder, gin.H{
		"title":           *flag_s_site_title,
		"company":         *flag_s_site_company,
		"domain":          *flag_s_primary_domain,
		"total_documents": human_int(a_i_total_documents.Load()),
		"total_pages":     human_int(a_i_total_pages.Load() - 1),
		"dark_mode":       gin_is_dark_mode(c),
	}); err != nil {
		log.Println(err)
		return "", fmt.Errorf("error executing template: %v", err)
	}
	return htmlBuilder.String(), nil
}

func r_get_ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func r_get_robots_txt(c *gin.Context) {
	var contents []byte
	path := *flag_s_robots_txt_path
	if len(path) > 0 {
		file_bytes, file_err := os.ReadFile(path)
		if file_err != nil {
			log.Printf("/robots.txt served - failed to load path %v due to err %v", path, file_err)
			contents = []byte(c_s_default_robots_txt)
		} else {
			contents = file_bytes
			file_bytes = []byte{} // flush this out of memory early
		}
	} else {
		contents = []byte(c_s_default_robots_txt)
	}
	c.Data(http.StatusOK, "text/plain", contents)
}

func r_get_ads_txt(c *gin.Context) {
	var contents []byte
	path := *flag_s_ads_txt_path
	if len(path) > 0 {
		file_bytes, file_err := os.ReadFile(path)
		if file_err != nil {
			log.Printf("/ads.txt served - failed to load path %v due to err %v", path, file_err)
			contents = []byte(c_s_default_ads_txt)
		} else {
			contents = file_bytes
			file_bytes = []byte{} // flush this out of memory early
		}
	} else {
		contents = []byte(c_s_default_ads_txt)
	}
	c.Data(http.StatusOK, "text/plain", contents)
}

func r_get_security_txt(c *gin.Context) {
	var contents []byte
	path := *flag_s_security_txt_path
	if len(path) > 0 {
		file_bytes, file_err := os.ReadFile(path)
		if file_err != nil {
			log.Printf("/security.txt served - failed to load path %v due to err %v", path, file_err)
			contents = []byte(c_s_default_security_txt)
		} else {
			contents = file_bytes
			file_bytes = []byte{} // flush this out of memory early
		}
	} else {
		contents = []byte(c_s_default_security_txt)
	}
	c.Data(http.StatusOK, "text/plain", contents)
}

func handler_no_route_linter() gin.HandlerFunc {
	return r_any_no_route_linter
}

func r_any_no_route_linter(c *gin.Context) {
	requestedURL := c.Request.URL.Path

	var endpoint_is []string
	if len(*flag_s_no_route_path_watch_list) > 2 {
		endpoint_is = strings.Split(*flag_s_no_route_path_watch_list, "|")
	}

	var endpoint_contains []string
	if len(*flag_s_no_route_path_contains_watch_list) > 2 {
		endpoint_contains = strings.Split(*flag_s_no_route_path_contains_watch_list, "|")
	}

	ip := f_s_filtered_ip(c)
	if len(ip) == 0 {
		c.Data(http.StatusNotFound, "text/plain", []byte("The truth can never be concealed forever."))
		return
	}

	nip := net.ParseIP(ip)

	if f_ip_in_ban_list(nip) {
		c.Data(http.StatusForbidden, "text/plain", []byte("403"))
		return
	}

	for _, endpoint := range endpoint_is {
		if requestedURL == endpoint {
			f_add_ip_to_watch_list(nip)
			c.Data(http.StatusNotFound, "text/plain", []byte("404"))
			return
		}
	}

	for _, endpoint := range endpoint_contains {
		if strings.Contains(requestedURL, endpoint) {
			f_add_ip_to_watch_list(nip)
			c.Data(http.StatusNotFound, "text/plain", []byte("404"))
			return
		}
	}

	c.Data(http.StatusNotFound, "text/plain", []byte("404"))
	return
}

func r_get_index(c *gin.Context) {
	body, err := r_render_static("index", c)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load index.html due to err: %v", err)
		return
	}
	c.Header("Content-Type", "text/html; charset=UTF-8")
	c.String(http.StatusOK, body)
}

func r_get_dark(c *gin.Context) {
	c.SetCookie(*flag_s_dark_mode_cookie, strconv.Itoa(1), 31881600, "/", *flag_s_cookie_domain, false, true)
	c.SetCookie(*flag_s_dark_mode_cookie, strconv.Itoa(1), 31881600, "/", *flag_s_cookie_domain, true, true)
	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func r_get_light(c *gin.Context) {
	c.SetCookie(*flag_s_dark_mode_cookie, strconv.Itoa(0), 31881600, "/", *flag_s_cookie_domain, false, true)
	c.SetCookie(*flag_s_dark_mode_cookie, strconv.Itoa(0), 31881600, "/", *flag_s_cookie_domain, true, true)
	c.Redirect(http.StatusTemporaryRedirect, "/")
}

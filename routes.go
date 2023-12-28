package main

import (
	`fmt`
	`html/template`
	`log`
	`net/http`
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

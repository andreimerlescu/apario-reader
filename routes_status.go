package main

import (
	`fmt`
	`html/template`
	`log`
	`net/http`
	`strings`

	`github.com/gin-gonic/gin`
)

func r_get_status(c *gin.Context) {
	data, err := bundled_files.ReadFile("bundled/assets/html/status.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load status.html")
		return
	}

	tmpl := template.Must(template.New("status").Funcs(gin_func_map).Parse(string(data)))

	var htmlBuilder strings.Builder
	if err := tmpl.Execute(&htmlBuilder, gin.H{
		"title":             fmt.Sprintf("%v - Application Status", *flag_s_site_title),
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

package main

import (
	`net/http`

	`github.com/gin-gonic/gin`
)

func r_get_contact_us(c *gin.Context) {
	body, err := r_render_static("contact-us", c)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load due to err: %v", err)
		return
	}
	c.Header("Content-Type", "text/html; charset=UTF-8")
	c.String(http.StatusOK, body)
}

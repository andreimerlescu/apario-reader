package main

import (
	`net/http`

	`github.com/gin-gonic/gin`
)

func r_get_legal_community_standards(c *gin.Context) {
	body, err := r_render_static("legal-community-standards", c)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load due to err: %v", err)
		return
	}
	c.Header("Content-Type", "text/html; charset=UTF-8")
	c.String(http.StatusOK, body)
}

func r_get_legal_coppa(c *gin.Context) {
	body, err := r_render_static("legal-coppa", c)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load due to err: %v", err)
		return
	}
	c.Header("Content-Type", "text/html; charset=UTF-8")
	c.String(http.StatusOK, body)
}

func r_get_legal_gdpr(c *gin.Context) {
	body, err := r_render_static("legal-gdpr", c)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load due to err: %v", err)
		return
	}
	c.Header("Content-Type", "text/html; charset=UTF-8")
	c.String(http.StatusOK, body)
}

func r_get_legal_privacy_policy(c *gin.Context) {
	body, err := r_render_static("legal-privacy", c)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load due to err: %v", err)
		return
	}
	c.Header("Content-Type", "text/html; charset=UTF-8")
	c.String(http.StatusOK, body)
}

func r_get_legal_terms(c *gin.Context) {
	body, err := r_render_static("legal-terms", c)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load due to err: %v", err)
		return
	}
	c.Header("Content-Type", "text/html; charset=UTF-8")
	c.String(http.StatusOK, body)
}

func r_get_legal_license(c *gin.Context) {
	body, err := r_render_static("legal-license", c)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load due to err: %v", err)
		return
	}
	c.Header("Content-Type", "text/html; charset=UTF-8")
	c.String(http.StatusOK, body)
}

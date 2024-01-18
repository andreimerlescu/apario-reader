package main

import (
	`github.com/didip/tollbooth/limiter`
	`github.com/didip/tollbooth_gin`
	`github.com/gin-gonic/gin`
)

func middleware_database_loaded() gin.HandlerFunc {
	return func(c *gin.Context) {
		handler_wait_for_database(c)
		c.Next()
	}
}

func middleware_enforce_authenticity_token() gin.HandlerFunc {
	return func(c *gin.Context) {
		handler_enforce_authenticity_token(c)
		c.Next()
	}
}

func middleware_ensure_authenticated() gin.HandlerFunc {
	return func(c *gin.Context) {
		handler_ensure_authenticated(c)
		c.Next()
	}
}

func middleware_content_security_policy() gin.HandlerFunc {
	return func(c *gin.Context) {
		handler_content_security_policy(c)
		c.Next()
	}
}

func middleware_rate_limiter(lim *limiter.Limiter) gin.HandlerFunc {
	return tollbooth_gin.LimitHandler(lim)
}

func middleware_online_counter() gin.HandlerFunc {
	return func(c *gin.Context) {
		handler_online_counter(c)
		c.Next()
	}
}

func middleware_force_https() gin.HandlerFunc {
	return func(c *gin.Context) {
		handler_force_https(c)
		c.Next()
	}
}

func middleware_tls_handshake() gin.HandlerFunc {
	return func(c *gin.Context) {
		handler_tls_handshake(c)
		c.Next()
	}
}

func middleware_enforce_ip_ban_list() gin.HandlerFunc {
	return func(c *gin.Context) {
		handler_enforce_ip_ban_list(c)
		c.Next()
	}
}

func middleware_cross_origin_request_scripts() gin.HandlerFunc {
	return func(c *gin.Context) {
		handler_cross_origin_request_scripts(c)
		c.Next()
	}
}

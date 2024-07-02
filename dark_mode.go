package main

import (
	`github.com/gin-gonic/gin`
)

func gin_is_dark_mode(c *gin.Context) string {
	// 0 = light mode ; 1 = dark mode
	dark_mode, dark_mode_err := c.Cookie(*flag_s_dark_mode_cookie)
	if dark_mode_err != nil {
		return "0"
	} else {
		if dark_mode == "0" || dark_mode == "1" {
			return dark_mode
		} else {
			return "0"
		}
	}
}

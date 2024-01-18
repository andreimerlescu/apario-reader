package main

import (
	`net/http`

	`github.com/gin-gonic/gin`
	`github.com/gorilla/sessions`
)

const session_name = "GOSESSID"

func g_get_session(c *gin.Context) *sessions.Session {
	session, session_err := session_store.Get(c.Request, session_name)
	if session_err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return nil
	}
	return session
}

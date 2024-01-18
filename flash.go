package main

import (
	`time`

	`github.com/gorilla/sessions`
)

type TSSessionAlert struct {
	Kind        string    `json:"kind"`
	Message     string    `json:"message"`
	ExpiresAt   time.Time `json:"expires_at"`
	ClearOnRead bool      `json:"clear_on_read"`
}

func NewSessionAlert(kind string, message string, expires_at time.Time, clear_on_read bool) *TSSessionAlert {
	return &TSSessionAlert{
		Kind:        kind,
		Message:     message,
		ExpiresAt:   expires_at,
		ClearOnRead: clear_on_read,
	}
}

type TSFlashMessages map[time.Time]*TSSessionAlert

func f_session_flash(session *sessions.Session, alert *TSSessionAlert) {
	session.Values["flash"] = TSFlashMessages{time.Now().UTC(): alert}
}

package main

import (
	`fmt`
	`time`

	`github.com/gin-gonic/gin`
)

func f_username_to_identifier(username string) ([64]byte, error) {
	identifier := Sha256(username)
	if len(identifier) != 64 {
		return [64]byte{}, fmt.Errorf("identifier must be exactly 64 characters long")
	}

	var arr [64]byte
	copy(arr[:], username)
	return arr, nil
}

func f_gin_check_authenticated_in_session(c *gin.Context) error {
	ip := f_s_client_ip(c.Request)
	session := g_get_session(c)
	authenticated, authenticated_value_set := session.Values["authenticated"].(bool)
	last_sign_in_ip, last_sign_in_ip_value_set := session.Values["last_sign_in_ip"].(string)
	_, last_sign_in_at_value_set := session.Values["last_sign_in_at"].(time.Time)
	_, username_value_set := session.Values["username"].(string)

	if authenticated_value_set && last_sign_in_ip_value_set && last_sign_in_at_value_set && username_value_set {
		if authenticated && last_sign_in_ip == ip {
			return nil
		}
	}
	return fmt.Errorf("session not authenticated")
}

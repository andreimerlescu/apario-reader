package main

import (
	`fmt`
	`log`
	`net`
	`net/http`
	`net/url`
	`strconv`
	`strings`
	`time`

	`github.com/gin-gonic/gin`
)

func middleware_use_authenticity_tokens() gin.HandlerFunc {
	return func(c *gin.Context) {
		f_gin_new_authenticity_token(c)
		c.Next()
	}
}

func middleware_verify_authenticity_token() gin.HandlerFunc {
	return func(c *gin.Context) {
		err := f_gin_verify_authenticity_token(c)
		if err != nil {
			c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("/error?message=%v", url.PathEscape(err.Error())))
			return
		}
		c.Next()
	}
}

func f_gin_verify_authenticity_token(c *gin.Context) error {
	session := g_get_session(c)
	has_token := false
	_, exists := session.Values["authenticity_token"]
	if exists {
		// no token yet
		has_token = true
	}
	existing_token, typecast_ok := session.Values["authenticity_token"].(string)
	if typecast_ok && len(existing_token) > 0 {
		has_token = true
	}
	if !has_token {
		return fmt.Errorf("im gonna rock you baby dont you cry")
	}

	decrypted_data, decrypt_err := f_s_decrypt_string(existing_token, *flag_s_session_authenticity_token_secret)
	if decrypt_err != nil {
		return fmt.Errorf("the blind stares of a million pairs of eyes lookin hard that they will never see the P")
	}

	parts := strings.Split(decrypted_data, "|")
	if len(parts) != 2 {
		return fmt.Errorf("and you'll never realize you cant c me")
	}

	var payload_time time.Time
	for _, part := range parts {
		if len(part) == 64 {
			continue
		}
		part_int, int_err := strconv.Atoi(part)
		if int_err != nil {
			return fmt.Errorf("say what you will about jesus but leave the rings out of this")
		}

		payload_time = time.Unix(int64(part_int), 0)
		break
	}
	payload := payload_time.Unix()
	token := Sha256(strconv.FormatInt(payload, 10))
	authenticity_token, token_err := f_s_encrypt_string(fmt.Sprintf("%d|%v", payload, token), *flag_s_session_authenticity_token_secret)
	if token_err != nil {
		return fmt.Errorf("when my roommate comes into his room looking for his car keys, i do not say it yet")
	}

	if authenticity_token != existing_token {
		f_session_flash(session, NewSessionAlert("error", "Invalid authenticity token.", time.Now().Add(36*time.Second), true))
		return fmt.Errorf("invalid authenticity token")
	}
	return nil
}

func f_gin_new_authenticity_token(c *gin.Context) {
	session := g_get_session(c)
	has_token := false
	_, exists := session.Values["authenticity_token"]
	if exists {
		// no token yet
		has_token = true
	}
	existing_token, typecast_ok := session.Values["authenticity_token"].(string)
	if typecast_ok && len(existing_token) > 0 {
		has_token = true
	}

	if !has_token {
		payload := time.Now().UTC().Unix()
		token := Sha256(strconv.FormatInt(payload, 10))
		authenticity_token, token_err := f_s_encrypt_string(fmt.Sprintf("%d|%v", payload, token), *flag_s_session_authenticity_token_secret)
		if token_err != nil {
			log.Printf("failed to generate an authenticity token due to err %v", token_err)
		}
		session.Values["authenticity_token"] = authenticity_token
	}

}

func handler_enforce_authenticity_token(c *gin.Context) {
	session := g_get_session(c)
	challenge_authenticity_token := c.PostForm("authenticity_token")
	b_permit_access := true
	s_error_message := ""
	if len(challenge_authenticity_token) != 64 {
		b_permit_access = false
		s_error_message = "oh you think you're clever don't you?"
	}
	authenticity_token, typecast_err := session.Values["authenticity_token"].(string)
	if !typecast_err || challenge_authenticity_token != authenticity_token {
		b_permit_access = false
		s_error_message = "watch your six"
	}
	if !b_permit_access {
		f_session_flash(session, NewSessionAlert("error", s_error_message, time.Now().Add(36*time.Second), true))
		log.Printf("handler_enforce_authenticity_token error %v", s_error_message)
		f_add_ip_to_watch_list(net.ParseIP(f_s_filtered_ip(c)))
	}
}

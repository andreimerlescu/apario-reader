package main

import (
	`fmt`
	`net/http`
	`path/filepath`
	`time`

	`github.com/gin-gonic/gin`
	`golang.org/x/crypto/bcrypt`
)

func r_get_login(c *gin.Context) {

}

func r_post_login(c *gin.Context) {
	session := g_get_session(c)
	username := c.PostForm("username")
	identifier, identifier_err := f_username_to_identifier(username)
	if identifier_err != nil {
		f_session_flash(session, NewSessionAlert("error", "Cannot validate username.", time.Now().Add(36*time.Second), true))
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		return
	}

	account, user_load_err := load_username_account(username)
	if user_load_err != nil {
		f_session_flash(session, NewSessionAlert("error", "No such username.", time.Now().Add(36*time.Second), true))
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		return
	}

	raw_password := c.PostForm("password")
	err := bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(raw_password))
	if err != nil {
		f_session_flash(session, NewSessionAlert("error", "Invalid password.", time.Now().Add(36*time.Second), true))
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		return
	}
	session.Values["authenticated"] = true
	session.Values["last_sign_in_ip"] = f_s_client_ip(c.Request)
	session.Values["last_sign_in_at"] = time.Now().UTC()
	session.Values["username"] = account.Username
	checksum_path, path_err := checksum_to_path(string(identifier[:]))
	if path_err != nil {
		f_session_flash(session, NewSessionAlert("error", "Encountered a problem.", time.Now().Add(45*time.Second), true))
		c.Redirect(http.StatusTemporaryRedirect, "/login")
	}
	session.Values["account_db_path"] = filepath.Join(*flag_s_users_database_directory, checksum_path, account_database_filename)
	f_session_flash(session, NewSessionAlert("success", fmt.Sprintf("Welcome back %v", account.Username), time.Now().Add(63*time.Second), true))
	referer := c.GetHeader("Referer")
	if len(referer) > 0 {
		c.Redirect(http.StatusOK, referer)
	}
	c.Redirect(http.StatusOK, "/")
	return
}

func r_get_logout(c *gin.Context) {

}

func r_get_register(c *gin.Context) {

}

func r_post_register(c *gin.Context) {

}

func r_get_challenge_password(c *gin.Context) {

}

func r_post_challenge_password(c *gin.Context) {

}

func r_get_forgot_password(c *gin.Context) {

}

func r_post_forgot_password(c *gin.Context) {

}

func r_get_change_email(c *gin.Context) {

}

func r_post_change_email(c *gin.Context) {

}

func r_get_manage_profile(c *gin.Context) {

}

func r_get_public_profile(c *gin.Context) {

}

func r_get_account_locked(c *gin.Context) {

}

func r_get_account_banned(c *gin.Context) {

}

func r_get_download_account_data(c *gin.Context) {

}

func r_post_download_account_data(c *gin.Context) {

}

func r_get_request_account_deletion(c *gin.Context) {

}

func r_post_request_account_deletion(c *gin.Context) {

}

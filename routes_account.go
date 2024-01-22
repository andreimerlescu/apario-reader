package main

import (
	`fmt`
	`log`
	`net`
	`net/http`
	`time`

	`github.com/gin-gonic/gin`
	`golang.org/x/crypto/bcrypt`
)

func r_get_login(c *gin.Context) {

}

func r_post_login(c *gin.Context) {
	session := g_get_session(c)
	username := c.PostForm("username")

	account, username_err := GetAccount(username)
	if username_err != nil {
		f_session_flash(session, NewSessionAlert("error", "Cannot get username.", time.Now().Add(36*time.Second), true))
		c.Redirect(http.StatusOK, "/login")
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
	session.Values["account_identifier"] = account.Identifier
	account.LastLogin = time.Now().UTC()
	account.LastLoginIP = net.ParseIP(f_s_client_ip(c.Request))
	err = account.Save()
	if err != nil {
		f_session_flash(session, NewSessionAlert("error", "Successful Login", time.Now().Add(36*time.Second), true))
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		return
	}

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
	username := c.PostForm("username")

	account, err := GetAccount(username)
	if err != nil {
		log.Printf("account not found")
	} else {
		log.Printf(fmt.Sprintf("account found %v", account))
	}

	password := c.PostForm("password")
	password_confirmation := c.PostForm("password-confirmation")
	email := c.PostForm("email")
	firstname := c.PostForm("first_name")
	lastname := c.PostForm("last_name")
	country := c.PostForm("country")

	log.Printf(password, password_confirmation, email, firstname, lastname, country)

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

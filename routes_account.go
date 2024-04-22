package main

import (
	`fmt`
	`net/http`
	`time`

	`github.com/gin-gonic/gin`
)

func r_get_login(c *gin.Context) {

}

func r_post_login(c *gin.Context) {
	session := g_get_session(c)
	username := c.PostForm("username")
	f_session_flash(session, NewSessionAlert("success", fmt.Sprintf("Welcome %v", username), time.Now().Add(63*time.Second), true))
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
	session := g_get_session(c)
	username := c.PostForm("username")
	f_session_flash(session, NewSessionAlert("success", fmt.Sprintf("Signed out as %v", username), time.Now().Add(63*time.Second), true))
	referer := c.GetHeader("Referer")
	if len(referer) > 0 {
		c.Redirect(http.StatusOK, referer)
	}
	c.Redirect(http.StatusOK, "/")
	return
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

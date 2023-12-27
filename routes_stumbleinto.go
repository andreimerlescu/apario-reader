package main

import (
	`fmt`
	`net/http`

	`github.com/gin-gonic/gin`
)

func stumbleinto_identifier(c *gin.Context) string {
	random_page_index := int64(f_i_random_int(len(m_index_page_identifier)))
	random_identifier, exists := m_index_page_identifier[random_page_index]
	if !exists {
		return stumbleinto_identifier(c)
	}
	return random_identifier
}

func r_get_stumble_into(c *gin.Context) {
	c.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("/page/%v?from=stumbleinto", stumbleinto_identifier(c)))
	return
}

package main

import (
	`fmt`
	`html/template`
	`log`
	`net/http`
	`strconv`
	`strings`

	`github.com/gin-gonic/gin`
)

func r_get_gematria(c *gin.Context) {
	data, err := bundled_files.ReadFile("bundled/assets/html/view-gematria.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load view-gematria.html")
		return
	}

	gematria_type := c.Param("type")
	score_str := c.Param("number")

	score, score_err := strconv.Atoi(score_str)
	if score_err != nil {
		log.Printf("score_err = %v", score_err)
		c.String(http.StatusForbidden, "invalid score")
		return
	}

	template_vars := gin.H{
		"title":             fmt.Sprintf("%v - Gematria", *flag_s_site_title),
		"company":           *flag_s_site_company,
		"domain":            *flag_s_primary_domain,
		"active_searches":   human_int(int64(sem_concurrent_searches.Len())),
		"i_active_searches": int64(sem_concurrent_searches.Len()),
		"max_searches":      human_int(int64(*flag_i_concurrent_searches)),
		"i_max_searches":    int64(*flag_i_concurrent_searches),
		"in_waiting_room":   human_int(a_i_waiting_room.Load()),
		"i_in_waiting_room": a_i_waiting_room.Load(),
		"dark_mode":         gin_is_dark_mode(c),
		"type":              gematria_type,
		"i_score":           score,
		"s_score":           score_str,
	}

	var tmpl *template.Template

	mu_page_gematria_english.RLock()
	map_words_pages, is_defined := m_page_gematria_english[uint(score)] // map[GemScore.English]map[PageIdentifier]map[word]struct{}
	mu_page_gematria_english.RUnlock()
	if !is_defined {
		// TODO No such results, render out to new GUI.
		missing_data, missing_err := bundled_files.ReadFile("bundled/assets/html/missing-page.html")
		if missing_err != nil {
			c.String(http.StatusInternalServerError, "Failed to load missing missing-page.html")
			return
		}
		tmpl = template.Must(template.New("view-gematria").Funcs(gin_func_map).Parse(string(missing_data)))
	} else {
		tmpl = template.Must(template.New("view-gematria").Funcs(gin_func_map).Parse(string(data)))
	}

	type result struct {
		Word       string
		Pages      uint
		GemScore   GemScore
		URL        string
		Identifier string
	}

	var (
		english_results []result
		jewish_results  []result
		simple_results  []result
	)

	// map_words_pages = map[PageIdentifier]map[word]struct{}
	switch gematria_type {
	case "english":
		for page_identifier, matching_words := range map_words_pages {
			for word, _ := range matching_words {
				result := result{
					Word:       word,
					Identifier: page_identifier,
					GemScore:   NewGemScore(word),
					URL:        "/search?query=" + word,
				}
				mu_page_gematria_english.RLock()
				_, found := m_page_gematria_english[uint(score)]
				mu_page_gematria_english.RUnlock()
				if found {
					result.Pages = uint(len(m_page_gematria_english[uint(score)]))
				}
				english_results = append(english_results, result)
			}
		}
	case "simple":
		for page_identifier, matching_words := range map_words_pages {
			for word, _ := range matching_words {
				result := result{
					Word:       word,
					Identifier: page_identifier,
					GemScore:   NewGemScore(word),
					URL:        "/search?query=" + word,
				}
				mu_page_gematria_simple.RLock()
				_, found := m_page_gematria_simple[uint(score)]
				mu_page_gematria_simple.RUnlock()
				if found {
					result.Pages = uint(len(m_page_gematria_simple[uint(score)]))
				}
				simple_results = append(simple_results, result)
			}
		}
	case "jewish":
		for page_identifier, matching_words := range map_words_pages {
			for word, _ := range matching_words {
				result := result{
					Word:       word,
					Identifier: page_identifier,
					GemScore:   NewGemScore(word),
					URL:        "/search?query=" + word,
				}
				mu_page_gematria_jewish.RLock()
				_, found := m_page_gematria_jewish[uint(score)]
				mu_page_gematria_jewish.RUnlock()
				if found {
					result.Pages = uint(len(m_page_gematria_jewish[uint(score)]))
				}
				jewish_results = append(jewish_results, result)
			}
		}
	default:
	}

	template_vars["english_results"] = english_results
	template_vars["simple_results"] = simple_results
	template_vars["jewish_results"] = jewish_results

	var htmlBuilder strings.Builder
	if err := tmpl.Execute(&htmlBuilder, template_vars); err != nil {
		c.String(http.StatusInternalServerError, "error executing template", err)
		log.Println(err)
		return
	}
	c.Header("Content-Type", "text/html; charset=UTF-8")
	c.String(http.StatusOK, htmlBuilder.String())
}

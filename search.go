package main

import (
	`context`
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync/atomic"

	go_smartchan `github.com/andreimerlescu/go-smartchan`
	`github.com/xrash/smetrics`
)

func AnalyzeQuery(q string) (sa SearchAnalysis) {

	q = strings.Replace(q, " && ", " and ", -1)
	q = strings.Replace(q, " !", " not ", -1)
	q = strings.Replace(q, ", ", " or ", -1)
	q = strings.Replace(q, ",", " or ", -1)
	q = strings.Replace(q, "||", " or ", -1)
	q = strings.Replace(q, "|", " or ", -1)
	q = strings.Replace(q, `{`, `(`, -1)
	q = strings.Replace(q, `}`, `)`, -1)
	q = strings.Replace(q, `[`, `(`, -1)
	q = strings.Replace(q, `]`, `)`, -1)

	q = fmt.Sprintf("and %v", q)

	printDoneLine := func(line string) {
		fmt.Printf("#DONE=%v⬆\n", strings.Repeat(`=`, len(line)-6))
	}

	printInsideBlock := func(str string) {
		padding := 166 - len(str)
		pad := ""
		if padding%2 == 0 {
			side := padding / 2
			pad = strings.Repeat(` `, side)
		} else {
			side := (padding + 1) / 2
			pad = strings.Repeat(` `, side)
		}
		fmt.Printf("#%v%v%v|\n", pad[0:len(pad)-1], str, pad)
	}

	log.Println(fmt.Sprintf("RunQuery( `%v` )", q))

	sa = SearchAnalysis{
		Ors:  map[uint]string{},
		Ands: []string{},
		Nots: []string{},
	}

	orCounter := new(atomic.Uint64)

	expr := `(?:(?:not )|(?:(?:and |or )))?\((?:[^()]+|\([^()]*\))+\)`
	re, err := regexp.Compile(expr)
	if err != nil {
		log.Println(fmt.Sprintf("failed to compile regex: %v", err))
		return sa
	}

	matches := re.FindAllStringSubmatch(q, -1)
	if len(matches) > 1 {
		log.Println("FOUND MATCHES!")
	}

	for idx, m := range matches {
		match := m[0]
		log.Println(fmt.Sprintf("match #%d = %#v", idx, match))
		if len(match) < 2 {
			continue
		}
		if match[0] == '(' {
			sa.Ands = append(sa.Ands, match)
			fmt.Println("adding to andWords")
			continue
		}

		firstTwo := match[0:2]
		verb := ""
		if firstTwo == `an` {
			// and
			sa.Ands = append(sa.Ands, match)
			verb = "and"
			fmt.Println("adding to andWords")
		} else if firstTwo == `no` {
			// not
			log.Println("found a match to add to notWords of ", match)
			sa.Nots = append(sa.Nots, match)
			verb = "not"
			fmt.Println("adding to notWords")
		} else {
			fmt.Println("SKIPPING")
			continue
		}

		if strings.Contains(match, " or ") {
			orCounter.Add(1)
			orId := orCounter.Load()
			sa.Ors[uint(orId)] = match
			q = strings.Replace(q, match, fmt.Sprintf("%v OR_%d", verb, orId), -1)
			continue
		}

		newMatch := strings.ReplaceAll(match, `(`, ``)
		newMatch = strings.ReplaceAll(match, `)`, ``)
		q = strings.Replace(q, match, newMatch, -1)
	}

	ands := strings.Split(q, "and")
	for _, and := range ands {
		fmt.Println(fmt.Sprintf("%#v", and))
	}

	log.Println(fmt.Sprintf("query = %#v", q))
	log.Println(fmt.Sprintf("orCounter = %d", orCounter.Load()))
	log.Println(fmt.Sprintf("sa.Ors  = %#v", sa.Ors))
	log.Println(fmt.Sprintf("sa.Ands = %#v", sa.Ands))
	log.Println(fmt.Sprintf("sa.Nots = %#v\n\n", sa.Nots))

	q = strings.Replace(q, " and not ", " not ", len(q)) // not always refers to the right side

	words := strings.Split(q, " ")
	wordLen := len(words)

	prevLoopBufferedWord := words[0]

	var addToAnd = new(atomic.Bool)
	var addToNot = new(atomic.Bool)

	var forceNextToAnd = new(atomic.Bool)
	var forceNextToNot = new(atomic.Bool)

	forceNextToAnd.Store(true)
	forceNextToNot.Store(false)

	log.Println("switching to AND...")
	for i := 1; i < wordLen; i++ {
		fmt.Printf("\n#\n##\n###==%v-=> Loop Iteration #%d\n", strings.Repeat(`-`, 162), i)

		if forceNextToAnd.Load() {
			printInsideBlock(fmt.Sprintf("FORCING PARTIAL WORD #%d `%v` TO BELONG TO        ✅ addToAnd = TRUE    &&    addToNot = FALSE ❌ ", i, prevLoopBufferedWord))
			addToAnd.Store(true)  // flip the force
			addToNot.Store(false) // flip the force
		}

		if forceNextToNot.Load() {
			printInsideBlock(fmt.Sprintf("FORCING PARTIAL WORD #%d `%v` TO BELONG TO ❌ addToAnd = FALSE and addToNot = TRUE ✅ ", i, prevLoopBufferedWord))
			addToAnd.Store(false) // flip the force
			addToNot.Store(true)  // flip the force
		}

		word := words[i]

		if len(word) == 0 {
			continue
		}

		if word[0] == ' ' {
			word = word[1 : len(word)-1]
		}

		if word[len(word)-1] == ' ' {
			word = word[:len(word)]
		}
		padding := strings.Repeat(` `, 71)
		line := fmt.Sprintf("%vANALYZING WORD #%d %#v%v|", padding, i, word, padding)
		fmt.Printf("#%v\n#%v⬇︎\n", line, strings.Repeat(`-`, len(line)-1))

		switch word {
		case "or":
			log.Println("switching to OR...")
		case "not":
			log.Println("switching to NOT...")
			log.Println(fmt.Sprintf("prevLoopBufferedWord = %v", prevLoopBufferedWord))
			log.Println(fmt.Sprintf("Word = %v (i=%d)", word, i))
			log.Println(fmt.Sprintf("Query = %v", q))
			// the word is not; therefore it looks like this\
			// top secret and communist not oswald
			// 0   1      2   3         4   5
			//                i-1       ^   i+1
			// going from idx 2 (and) to idx 4 (not)
			if addToAnd.Load() { // last was and
				log.Println("\t addToAnd = TRUE")
				if len(prevLoopBufferedWord) > 0 {
					log.Println("\t\t because \t `len(prevLoopBufferedWord) > 0` \t caused \t `forceNextToNot = TRUE`")
					forceNextToNot.Store(true)
					forceNextToAnd.Store(false)
				}
			} else {
				log.Println("\t addToAnd = FALSE")
			}
		case "and":
			log.Println("switching to AND...")
			// the word is not; therefore it looks like this
			// top secret and communist not oswald
			// 0   1      2   3         4   5
			//                i-1       ^   i+1
			if !addToAnd.Load() { // last was not
				log.Println("\t addToAnd = FALSE")
				// addToAnd.Store(true) // switch to and
				// addToNot.Store(false)

				forceNextToAnd.Store(true)
				forceNextToNot.Store(false)
			} else {
				log.Println("\t addToAnd = TRUE")
			}

			log.Println("RECEIVED AN 'AND' - SETTING THE addToAnd = true")
		default:
			// regular word
			prevLoopBufferedWord = fmt.Sprintf("%v %v", prevLoopBufferedWord, word)
			prevLoopBufferedWord = strings.Replace(prevLoopBufferedWord, "and ", "", -1)
			prevLoopBufferedWord = strings.Replace(prevLoopBufferedWord, "not ", "", -1)
			prevLoopBufferedWord = strings.Replace(prevLoopBufferedWord, "  ", " ", len(prevLoopBufferedWord))
			printInsideBlock(fmt.Sprintf("\t adding word `%v` to prevLoopBufferedWord as `%v` for word #%d - GOING TO NEXT WORD", word, prevLoopBufferedWord, i))

			if i == wordLen-1 {
				printInsideBlock(fmt.Sprintf("\t `i == wordLen-1` == `TRUE` \t\t `i` = `%d` \t\t `wordLen` = `%d-1` (%d)", i, wordLen, wordLen-1))
				break
			}
			printInsideBlock(fmt.Sprintf("addToAnd = %#v %T", addToAnd.Load(), addToAnd.Load()))
			printInsideBlock(fmt.Sprintf("addToNot = %#v %T", addToNot.Load(), addToNot.Load()))
			printInsideBlock(fmt.Sprintf("forceNextToAnd = %#v %T", forceNextToAnd.Load(), forceNextToAnd.Load()))
			printInsideBlock(fmt.Sprintf("forceNextToNot = %#v %T", forceNextToNot.Load(), forceNextToNot.Load()))
			printDoneLine(line)
			continue
		}

		// clean up formatting of the partial word to remove extra whitespace
		if len(prevLoopBufferedWord) == 0 {
			printDoneLine(line)
			continue
		}

		if prevLoopBufferedWord[0] == ' ' {
			log.Println(fmt.Sprintf("\t unclean prevLoopBufferedWord[0] == ' ' \t %#v", prevLoopBufferedWord))
			prevLoopBufferedWord = prevLoopBufferedWord[1:len(prevLoopBufferedWord)]
			log.Println(fmt.Sprintf("\t cleaned prevLoopBufferedWord[0] == ' ' \t %#v", prevLoopBufferedWord))
		}

		if prevLoopBufferedWord[len(prevLoopBufferedWord)-1] == ' ' {
			log.Println(fmt.Sprintf("\t prevLoopBufferedWord[len(prevLoopBufferedWord)-1 => %d-1 = %d] == ' ' \t %#v", len(prevLoopBufferedWord), len(prevLoopBufferedWord)-1, prevLoopBufferedWord))
			prevLoopBufferedWord = prevLoopBufferedWord[:len(prevLoopBufferedWord)-1]
			log.Println(fmt.Sprintf("\t prevLoopBufferedWord[len(prevLoopBufferedWord)-1 => %d-1 = %d] == ' ' \t %#v", len(prevLoopBufferedWord), len(prevLoopBufferedWord)-1, prevLoopBufferedWord))
		}

		// add the partial word to the andWords or notWords slices
		if addToAnd.Load() {
			log.Println(fmt.Sprintf("\t ADDING %v TO andWords(%d)", prevLoopBufferedWord, len(sa.Ands)))
			sa := *sa.findOrsInAnds(prevLoopBufferedWord)
			log.Println(fmt.Sprintf("\t ADDED %v TO andWords(%d)", prevLoopBufferedWord, len(sa.Ands)))
			// reset the partial word
			prevLoopBufferedWord = ""
			continue
		} else if addToNot.Load() {
			log.Println(fmt.Sprintf("\t ADDING WORD \t `%v` \t TO \t `notWords(%d)` ... ", prevLoopBufferedWord, len(sa.Nots)))
			oldLen := len(sa.Nots)
			sa = *sa.findOrsInNots(prevLoopBufferedWord)
			log.Println(fmt.Sprintf("\t ADDED WORD \t `%v` \t TO \t `notWords(%d)` ✅ COMPLETED! (now %d sa.Nots)", prevLoopBufferedWord, oldLen, len(sa.Nots)))
			// reset the partial word
			prevLoopBufferedWord = ""
			continue
		}

		if forceNextToAnd.Load() {
			forceNextToAnd.Store(false) // turn off the force
		}

		if forceNextToNot.Load() {
			forceNextToNot.Store(false) // turn off the force
		}

		printDoneLine(line)
	}

	log.Println(fmt.Sprintf("\n\n\n\t SUMMARY \n\nquery = %#v", q))
	log.Println(fmt.Sprintf("orCounter = %d", orCounter.Load()))
	log.Println(fmt.Sprintf("sa.Ors  = %#v", sa.Ors))
	log.Println(fmt.Sprintf("sa.Ands = %#v", sa.Ands))
	log.Println(fmt.Sprintf("sa.Nots = %#v\n\n", sa.Nots))

	log.Printf("%#v", sa)

	return

}

func find_pages_for_word(ctx context.Context, sch *go_smartchan.SmartChan, query string) error {
	var results = make(map[string]struct{})
	mu_word_pages.RLock()
	for word, pages := range m_word_pages {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		var distance float64
		var can_use_hamming bool = len(query) == len(word)
		if len(word) == 0 || len(query) == 0 {
			continue
		}
		if *flag_s_search_algorithm == "jaro" {
			distance = smetrics.Jaro(query, word)
			if distance >= *flag_f_search_jaro_threshold {
				for page_identifier, _ := range pages {
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
					}
					if sch.CanWrite() {
						err := sch.Write(page_identifier)
						if err != nil {
							return err
						}
					}
				}
			}
		} else if *flag_s_search_algorithm == "soundex" {
			query_soundex := smetrics.Soundex(query)
			word_soundex := smetrics.Soundex(word)
			if query_soundex == word_soundex {
				for page_identifier, _ := range pages {
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
					}
					if sch.CanWrite() {
						err := sch.Write(page_identifier)
						if err != nil {
							return err
						}
					}
				}
			}
		} else if *flag_s_search_algorithm == "ukkonen" {
			score := smetrics.Ukkonen(query, word, *flag_i_search_ukkonen_icost, *flag_i_search_ukkonen_dcost, *flag_i_search_ukkonen_scost)
			if score <= *flag_i_search_ukkonen_max_substitutions {
				for page_identifier, _ := range pages {
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
					}
					if sch.CanWrite() {
						err := sch.Write(page_identifier)
						if err != nil {
							return err
						}
					}
				}
			}
		} else if *flag_s_search_algorithm == "wagner_fischer" {
			score := smetrics.WagnerFischer(query, word, *flag_i_search_wagner_fischer_icost, *flag_i_search_wagner_fischer_dcost, *flag_i_search_wagner_fischer_scost)
			if score <= *flag_i_search_wagner_fischer_max_substitutions {
				for page_identifier, _ := range pages {
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
					}
					if sch.CanWrite() {
						err := sch.Write(page_identifier)
						if err != nil {
							return err
						}
					}
				}
			}
		} else if *flag_s_search_algorithm == "hamming" && can_use_hamming {
			substitutions, err := smetrics.Hamming(query, word)
			if err != nil {
				return fmt.Errorf("error received when performing Hamming analysis: %v", err)
			}
			if substitutions <= *flag_i_search_hamming_max_substitutions {
				for page_identifier, _ := range pages {
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
					}
					if sch.CanWrite() {
						err := sch.Write(page_identifier)
						if err != nil {
							return err
						}
					}
				}
			}
		} else { // use jaro_winkler
			distance = smetrics.JaroWinkler(query, word, *flag_f_search_jaro_winkler_boost_threshold, *flag_i_search_jaro_winkler_prefix_size)
			if distance >= *flag_f_search_jaro_winkler_threshold {
				for page_identifier, _ := range pages {
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
					}
					if sch.CanWrite() {
						err := sch.Write(page_identifier)
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}
	mu_word_pages.RUnlock()
	if len(results) == 0 {
		return fmt.Errorf("no results for %v", query)
	}
	return nil
}

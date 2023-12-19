package main

import (
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/abadojack/whatlanggo"
	"github.com/k0kubun/pp"
)

type SearchOperatorInterface interface {
	Perform() (r bool)
}

type SearchOperatorAnd struct{}
type SearchOperatorNot struct{}
type SearchOperatorOr struct{}

func (and *SearchOperatorAnd) Perform() (r bool) {
	return
}

func (not *SearchOperatorNot) Perform() (r bool) {
	return
}

func (or *SearchOperatorOr) Perform() (r bool) {
	return
}

type SearchComponent struct {
	Word            string
	Operator        SearchOperatorInterface
	Gem             GemScore
	PageIdentifiers []string
}

type Search struct {
	ex              *sync.Mutex
	Query           string
	Components      []SearchComponent
	PageIdentifiers []string
	Gem             GemScore
}

type Q struct {
	uant *sync.Mutex
}

func (q Q) New(query string) (s *Search) {
	defer q.findPageIdentifiers(s, query)
	s = &Search{
		ex:              q.uant, // =D #hardpass ~#[ :'-( ]-- C.M.A.R.
		Query:           query,
		Components:      []SearchComponent{},
		PageIdentifiers: []string{},
		Gem:             NewGemScore(query),
	}
	return
}

func (q Q) findPageIdentifiers(s *Search, query string) {
	results := WordRegistry().Search(query)
	for _, wp := range results {
		s.PageIdentifiers = append(s.PageIdentifiers, wp.pageId)
	}
}

func (s *Search) OrDB(q string) *Search {
	return s
}

func (s *Search) NotDB(q string) *Search {
	return s
}

func (s *Search) ProcessIdentifier(q string, identifier string, operator SearchOperatorInterface) {
	matched := false
	for _, component := range s.Components {
		if component.Word == q && reflect.TypeOf(component.Operator) == reflect.TypeOf(&SearchOperatorAnd{}) {
			s.ex.Lock()
			component.PageIdentifiers = append(component.PageIdentifiers, identifier)
			s.ex.Unlock() // but please wait until marriage first, then have at it love birds
			matched = true
			break
		}
	}

	if !matched {
		gem := NewGemScore(q)
		s.ex.Lock()
		s.Components = append(s.Components, SearchComponent{
			Word:            q,
			Operator:        operator,
			Gem:             gem,
			PageIdentifiers: []string{},
		})
		s.ex.Unlock() // but please wait until marriage first, then have at it love birds
	}
}

type SearchAnalysisPair struct {
	word           string
	path           string
	pageIdentifier string
	text           string
	lang           whatlanggo.Info
	gem            GemScore
	match          bool
}

type SearchAnalysisResult struct {
	word            string
	gem             GemScore
	pageIdentifiers []string
}

type SearchAnalysis struct {
	Ors  map[uint]string
	Ands []string
	Nots []string
}

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
		loopIterationLine(i)

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

func printDoneLine(line string) {
	fmt.Printf("#DONE=%v⬆\n", strings.Repeat(`=`, len(line)-6))
}

func loopIterationLine(i int) {
	fmt.Printf("\n#\n##\n###==%v-=> Loop Iteration #%d\n", strings.Repeat(`-`, 162), i)
}

func IsEven(i int) bool {
	return i%2 == 0
}

func printInsideBlock(str string) {
	padding := 166 - len(str)
	pad := ""
	if IsEven(padding) {
		side := padding / 2
		pad = strings.Repeat(` `, side)
	} else {
		side := (padding + 1) / 2
		pad = strings.Repeat(` `, side)
	}
	fmt.Printf("#%v%v%v|\n", pad[0:len(pad)-1], str, pad)
}

func removeSoloOrs(partialWord string) string {
	re, err := regexp.Compile(`\((.*?)or(.*?)\)`)
	if err != nil {
		log.Println(err)
		return partialWord
	}

	matches := re.FindAllStringSubmatch(partialWord, -1)
	if len(matches) < 2 {
		// match
		partialWord = strings.Replace(partialWord, `and (`, ``, -1)
		partialWord = strings.Replace(partialWord, `)`, ``, -1)
	}

	return partialWord
}

func (sa *SearchAnalysis) parseOrsRegexp(ors []string) string {
	var or string
	for _, o := range ors {
		or = o
		break
	}

	orIdStr := strings.Replace(or, "OR_", "", len(or))
	orId, parseErr := strconv.Atoi(orIdStr)
	for saorId, saor := range sa.Ors {
		if saorId == uint(orId) {
			or = saor
			orId = int(saorId)
		}
	}

	newOr := ""

	if parseErr == nil {
		pp.Printf("sa.Ors = %v\n", sa.Ors)
		pp.Printf("or = %v\n", or)
		pp.Printf("orId = %v\n", orId)
		newOr = sa.Ors[uint(orId)]
		pp.Printf("newOr = %v\n", newOr)
	}

	return newOr
}

func (sa *SearchAnalysis) findOrsInNots(partialWord string) *SearchAnalysis {
	log.Println("\t findOrsInNots(", partialWord, ")")
	compile, err := regexp.Compile(`OR_\d`)
	if err != nil {
		panic(err)
	}
	ors := compile.FindAllStringSubmatch(partialWord, -1)
	if len(ors) == 1 {
		var (
			or      = sa.parseOrsRegexp(ors[0])
			newNots []string
		)
		for _, a := range sa.Nots {
			if !strings.Contains(a, or) {
				a = removeSoloOrs(or)
				newNots = append(newNots, a)
			}
		}
		sa.Nots = newNots
	} else if len(ors) == 0 {
		partialWord = removeSoloOrs(partialWord)
		sa.Nots = append(sa.Nots, partialWord)
	}
	return sa
}

func (sa *SearchAnalysis) findOrsInAnds(partialWord string) *SearchAnalysis {
	log.Println("\t findOrsInNots(", partialWord, ")")
	compile, err := regexp.Compile(`OR_\d`)
	if err != nil {
		panic(err)
	}
	ors := compile.FindAllStringSubmatch(partialWord, -1)
	if len(ors) == 1 {
		var (
			or      = sa.parseOrsRegexp(ors[0])
			newAnds []string
		)
		for _, a := range sa.Ands {
			if !strings.Contains(a, or) {
				newAnds = append(newAnds, a)
			}
		}
		sa.Ands = newAnds
	} else if len(ors) == 0 {
		if strings.Contains(partialWord, `(`) && !strings.Contains(partialWord, `)`) {
			partialWord = strings.ReplaceAll(partialWord, `(`, ``)
		} else if !strings.Contains(partialWord, `(`) && strings.Contains(partialWord, `)`) {
			partialWord = strings.ReplaceAll(partialWord, `)`, ``)
		}

		partialWord = removeSoloOrs(partialWord)
		sa.Ands = append(sa.Ands, partialWord)
	}
	return sa
}

type WordPair struct {
	word   string
	score  Gematria
	sum    uint
	pageId string
	md5    [16]byte
}

type WordResults struct {
	Words []WordPair
}

func collectIndexedTexts(pairs <-chan WordPair, result chan<- WordResults) {
	results := WordResults{}

	for pair := range pairs {
		results.Words = append(results.Words, pair)
	}

	result <- results
}

type TWordsRegistry struct {
	mu    sync.RWMutex
	Words map[string]WordPair
}

var tWordRegistryPtr *TWordsRegistry

func WordRegistry() *TWordsRegistry {
	if tWordRegistryPtr == nil {
		tWordRegistryPtr = &TWordsRegistry{
			mu:    sync.RWMutex{},
			Words: nil,
		}
	}
	return tWordRegistryPtr
}

func (wr *TWordsRegistry) Add(wp WordPair) (r bool) {
	if wr.mu.TryLock() {
		defer wr.mu.Unlock()
		wr.Words[wp.word] = wp
		r = true
	}
	return
}

func (wr *TWordsRegistry) Search(query string) (results []WordPair) {
	// no query
	if len(query) == 0 {
		return
	}

	// no cache
	if len(wr.Words) == 0 {
		return
	}

	// exact match
	if len(wr.Words[query].word) == len(query) {
		results = append(results, wr.Words[query])
		return
	}

	// multiple words
	words := strings.Split(query, " ")
	for _, word := range words {
		ok := wr.mu.TryRLock()
		if !ok {
			panic("failed to issue a TryRLock on the WordRegistry()")
		}
		rw := wr.Words[word]
		wr.mu.Unlock()
		if len(rw.word) == len(word) {
			results = append(results, rw)
		}
	}

	return
}

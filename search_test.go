package main

import (
	"fmt"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	Q1 = iota + 1
	Q2
	Q3
	Q4
	Q5
	Q6
	Q7
	Q8
	Q9
	Q10
	Q11
	Q12
)

func TestAnalyzeQuery(t *testing.T) {
	var (
		ok bool
		sa SearchAnalysis
		qs = map[uint]string{
			Q1:  "(top secret or confidential) and communist and oswald not thought",
			Q2:  "top secret and oswald",
			Q3:  "top secret and communist not oswald",
			Q4:  "(top secret or confidential or classified) and (assassin or murder or kill) and (kennedy or president) and (communi or infiltrat) not (cover page or blank page or unclassified)",
			Q5:  "top secret and communist and not oswald",
			Q6:  "[communism,communist] and (top secret , confidential) && {communist} not kevin bacon",
			Q7:  "(top secret or confidential or classified) and (communist or communism or commie) and (assassinated or killed or died or murdered)",
			Q8:  "(top secret or confidential or classified) and (communist or communism or commie) not (assassinated or killed or died or murdered) and bacon not sausage and lettuce not mustard",
			Q9:  "(top secret or confidential or classified) && (communist or communism or commie) !(assassinated or killed or died or murdered) !mustard !sausage && bacon && lettuce",
			Q10: "(top secret,confidential , classified) && (communist,communism,commie) !(assassinated|killed||died,murdered) !mustard !sausage && bacon && lettuce",
			Q11: "(secret or confidential or classified) not (cover or intentionally left blank) and (President Kennedy or John F Kennedy or President JFK or POTUS JFK or POTUS 35)",
			Q12: "(orange juice or coffee or apple juice or tomato juice) and (sunny side up or over easy or scrambled or omelet) not alcohol and jesus and (toast or fruit bowl or french crepe)",
		}
	)

	for qno, q := range qs {
		switch qno {
		case Q1:
			log.Println(fmt.Sprintf("PROCESSING QUERY #%02d `%#v`", qno, q))
			if sa = AnalyzeQuery(q); len(sa.Ands) > 0 {
				log.Println(fmt.Sprintf("%#v", sa))
				ok = assert.NotEmpty(t, sa.Ors)
				ok = assert.NotEmpty(t, sa.Ands)
				ok = assert.NotEmpty(t, sa.Nots)
				ok = assert.NotContains(t, sa.Ands, "thought")
				ok = assert.Contains(t, sa.Ands, "oswald")
				ok = assert.Contains(t, sa.Nots, "thought")
				ok = assert.NotContains(t, sa.Nots, "oswald")
				ok = assert.Len(t, sa.Ors, 1)
				ok = assert.Len(t, sa.Ands, 2)
				ok = assert.Len(t, sa.Nots, 1)
				log.Println(fmt.Sprintf("\nTEST %2d COMPLETED => successful? == %#v\n", qno+1, ok))
			}
		case Q2:
			log.Println(fmt.Sprintf("PROCESSING QUERY #%2d `%#v`", qno, q))
			if sa = AnalyzeQuery(q); len(sa.Ands) > 0 {
				log.Println(fmt.Sprintf("%#v", sa))
				ok = assert.Empty(t, sa.Ors)
				ok = assert.NotEmpty(t, sa.Ands)
				ok = assert.Empty(t, sa.Nots)
				ok = assert.Contains(t, sa.Ands, "oswald")
				ok = assert.Contains(t, sa.Ands, "top secret")
				ok = assert.Len(t, sa.Ors, 0)
				ok = assert.Len(t, sa.Ands, 2)
				ok = assert.Len(t, sa.Nots, 0)
				log.Println(fmt.Sprintf("\nTEST %2d COMPLETED => successful? == %#v\n", qno+1, ok))
			}
		case Q3:
			log.Println(fmt.Sprintf("PROCESSING QUERY #%2d `%#v`", qno, q))
			if sa = AnalyzeQuery(q); len(sa.Ands) > 0 {
				log.Println(fmt.Sprintf("%#v", sa))
				ok = assert.Empty(t, sa.Ors)
				ok = assert.NotEmpty(t, sa.Ands)
				ok = assert.NotEmpty(t, sa.Nots)
				ok = assert.Contains(t, sa.Ands, "top secret")
				ok = assert.Contains(t, sa.Ands, "communist")
				ok = assert.Contains(t, sa.Nots, "oswald")
				ok = assert.Len(t, sa.Ors, 0)
				ok = assert.Len(t, sa.Ands, 2)
				ok = assert.Len(t, sa.Nots, 1)
				log.Println(fmt.Sprintf("\nTEST %2d COMPLETED => successful? == %#v\n", qno+1, ok))
			}
		case Q4:
			log.Println(fmt.Sprintf("PROCESSING QUERY #%2d `%#v`", qno, q))
			if sa = AnalyzeQuery(q); len(sa.Ands) > 0 {
				log.Println(fmt.Sprintf("%#v", sa))
				ok = assert.Empty(t, sa.Ors)
				ok = assert.NotEmpty(t, sa.Ands)
				ok = assert.NotEmpty(t, sa.Nots)
				ok = assert.Contains(t, sa.Ands, "top secret")
				ok = assert.Contains(t, sa.Ands, "communist")
				ok = assert.Contains(t, sa.Nots, "oswald")
				ok = assert.Len(t, sa.Ors, 0)
				ok = assert.Len(t, sa.Ands, 2)
				ok = assert.Len(t, sa.Nots, 1)
				log.Println(fmt.Sprintf("\nTEST %2d COMPLETED => successful? == %#v\n", qno+1, ok))
			}
		case Q5:
			log.Println(fmt.Sprintf("PROCESSING QUERY #%2d `%#v`", qno, q))
			if sa = AnalyzeQuery(q); len(sa.Ands) > 0 {
				log.Println(fmt.Sprintf("%#v", sa))
				ok = assert.Empty(t, sa.Ors)
				ok = assert.NotEmpty(t, sa.Ands)
				ok = assert.NotEmpty(t, sa.Nots)
				ok = assert.Len(t, sa.Ors, 0)
				ok = assert.Len(t, sa.Ands, 2)
				ok = assert.Len(t, sa.Nots, 1)
				log.Println(fmt.Sprintf("\nTEST %2d COMPLETED => successful? == %#v\n", qno+1, ok))
			}
		case Q6:
			log.Println(fmt.Sprintf("PROCESSING QUERY #%2d `%#v`", qno, q))
			if sa = AnalyzeQuery(q); len(sa.Ands) > 0 {
				log.Println(fmt.Sprintf("%#v", sa))
				ok = assert.NotEmpty(t, sa.Ors)
				ok = assert.NotEmpty(t, sa.Ands)
				ok = assert.NotEmpty(t, sa.Nots)
				ok = assert.Contains(t, sa.Ands, "communist")
				ok = assert.NotContains(t, sa.Ands, "kevin bacon")
				ok = assert.Contains(t, sa.Nots, "kevin bacon")
				ok = assert.Len(t, sa.Ors, 2)
				ok = assert.Len(t, sa.Ands, 2)
				ok = assert.Len(t, sa.Nots, 1)
				log.Println(fmt.Sprintf("\nTEST %2d COMPLETED => successful? == %#v\n", qno+1, ok))
			}
		case Q7:
			log.Println(fmt.Sprintf("PROCESSING QUERY #%2d `%#v`", qno, q))
			if sa = AnalyzeQuery(q); len(sa.Ands) > 0 {
				log.Println(fmt.Sprintf("%#v", sa))
				ok = assert.Empty(t, sa.Ors)
				ok = assert.Empty(t, sa.Ands)
				ok = assert.Empty(t, sa.Nots)
				ok = assert.Contains(t, sa.Ors, "")
				ok = assert.Contains(t, sa.Ands, "")
				ok = assert.Contains(t, sa.Nots, "")
				ok = assert.Len(t, sa.Ors, 0)
				ok = assert.Len(t, sa.Ands, 0)
				ok = assert.Len(t, sa.Nots, 0)
				log.Println(fmt.Sprintf("\nTEST %2d COMPLETED => successful? == %#v\n", qno+1, ok))
			}
		case Q8:
		case Q9:
		case Q10:
			log.Println(fmt.Sprintf("PROCESSING QUERY #%2d `%#v`", qno, q))
			if sa = AnalyzeQuery(q); len(sa.Ands) > 0 || len(sa.Nots) > 0 || len(sa.Ors) > 0 {
				log.Println(fmt.Sprintf("%#v", sa))
				ok = assert.NotEmpty(t, sa.Ors)
				ok = assert.NotEmpty(t, sa.Ands)
				ok = assert.NotEmpty(t, sa.Nots)
				ok = assert.Contains(t, sa.Ands, "bacon")
				ok = assert.Contains(t, sa.Ands, "lettuce")
				ok = assert.Contains(t, sa.Nots, "sausage")
				ok = assert.Contains(t, sa.Nots, "mustard")
				ok = assert.Len(t, sa.Ors, 3)
				ok = assert.Len(t, sa.Ands, 2)
				ok = assert.Len(t, sa.Nots, 2)
				log.Println(fmt.Sprintf("\nTEST %2d COMPLETED => successful? == %#v\n", qno+1, ok))
			}
		case Q11:
			log.Println(fmt.Sprintf("PROCESSING QUERY #%2d `%#v`", qno, q))
			if sa = AnalyzeQuery(q); len(sa.Ands) > 0 {
				log.Println(fmt.Sprintf("%#v", sa))
				ok = assert.Empty(t, sa.Ors)
				ok = assert.Empty(t, sa.Ands)
				ok = assert.Empty(t, sa.Nots)
				ok = assert.Contains(t, sa.Ors, "")
				ok = assert.Contains(t, sa.Ands, "")
				ok = assert.Contains(t, sa.Nots, "")
				ok = assert.Len(t, sa.Ors, 0)
				ok = assert.Len(t, sa.Ands, 0)
				ok = assert.Len(t, sa.Nots, 0)
				log.Println(fmt.Sprintf("\nTEST %2d COMPLETED => successful? == %#v\n", qno+1, ok))
			}
		case Q12:
			log.Println(fmt.Sprintf("PROCESSING QUERY #%2d `%#v`", qno, q))
			if sa = AnalyzeQuery(q); len(sa.Ands) > 0 {
				log.Println(fmt.Sprintf("%#v", sa))
				ok = assert.NotEmpty(t, sa.Ors)
				ok = assert.NotEmpty(t, sa.Ands)
				ok = assert.NotEmpty(t, sa.Nots)
				ok = assert.Len(t, sa.Ors, 3)
				ok = assert.Len(t, sa.Ands, 1)
				ok = assert.Len(t, sa.Nots, 1)
				log.Println(fmt.Sprintf("\nTEST %2d COMPLETED => successful? == %#v\n", qno+1, ok))
			}
		default:
			continue
		}
		//time.Sleep(time.Millisecond * 12_369)
	}

}

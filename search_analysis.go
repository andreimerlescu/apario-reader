package main

import (
	`log`
	`regexp`
	`strconv`
	`strings`

	`github.com/k0kubun/pp`
)

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

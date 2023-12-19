/*
Project Apario is the World's Truth Repository that was invented and started by Andrei Merlescu in 2020.
Copyright (C) 2023  Andrei Merlescu

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"
)

func InitGematria() {
	m_gcm_jewish["A"], m_gcm_jewish["B"], m_gcm_jewish["C"], m_gcm_jewish["D"], m_gcm_jewish["E"], m_gcm_jewish["F"] = 1, 2, 3, 4, 5, 6
	m_gcm_jewish["G"], m_gcm_jewish["H"], m_gcm_jewish["I"], m_gcm_jewish["J"], m_gcm_jewish["K"], m_gcm_jewish["L"] = 7, 8, 9, 600, 10, 20
	m_gcm_jewish["M"], m_gcm_jewish["N"], m_gcm_jewish["O"], m_gcm_jewish["P"], m_gcm_jewish["Q"], m_gcm_jewish["R"] = 30, 40, 50, 60, 70, 80
	m_gcm_jewish["S"], m_gcm_jewish["T"], m_gcm_jewish["U"], m_gcm_jewish["V"], m_gcm_jewish["W"], m_gcm_jewish["X"] = 90, 100, 200, 700, 900, 300
	m_gcm_jewish["Y"], m_gcm_jewish["Z"] = 400, 500

	m_gcm_english["A"], m_gcm_english["B"], m_gcm_english["C"], m_gcm_english["D"], m_gcm_english["E"], m_gcm_english["F"] = 6, 12, 18, 24, 30, 36
	m_gcm_english["G"], m_gcm_english["H"], m_gcm_english["I"], m_gcm_english["J"], m_gcm_english["K"], m_gcm_english["L"] = 42, 48, 54, 60, 66, 72
	m_gcm_english["M"], m_gcm_english["N"], m_gcm_english["O"], m_gcm_english["P"], m_gcm_english["Q"], m_gcm_english["R"] = 78, 84, 90, 96, 102, 108
	m_gcm_english["S"], m_gcm_english["T"], m_gcm_english["U"], m_gcm_english["V"], m_gcm_english["W"], m_gcm_english["X"] = 114, 120, 126, 132, 138, 144
	m_gcm_english["Y"], m_gcm_english["Z"] = 150, 156

	m_gcm_simple["A"], m_gcm_simple["B"], m_gcm_simple["C"], m_gcm_simple["D"], m_gcm_simple["E"], m_gcm_simple["F"] = 1, 2, 3, 4, 5, 6
	m_gcm_simple["G"], m_gcm_simple["H"], m_gcm_simple["I"], m_gcm_simple["J"], m_gcm_simple["K"], m_gcm_simple["L"] = 7, 8, 9, 10, 11, 12
	m_gcm_simple["M"], m_gcm_simple["N"], m_gcm_simple["O"], m_gcm_simple["P"], m_gcm_simple["Q"], m_gcm_simple["R"] = 13, 14, 15, 16, 17, 18
	m_gcm_simple["S"], m_gcm_simple["T"], m_gcm_simple["U"], m_gcm_simple["V"], m_gcm_simple["W"], m_gcm_simple["X"] = 19, 20, 21, 22, 23, 24
	m_gcm_simple["Y"], m_gcm_simple["Z"] = 25, 26

	//

	m_gcm_jewish["a"], m_gcm_jewish["b"], m_gcm_jewish["c"], m_gcm_jewish["d"], m_gcm_jewish["e"], m_gcm_jewish["f"] = 1, 2, 3, 4, 5, 6
	m_gcm_jewish["g"], m_gcm_jewish["h"], m_gcm_jewish["i"], m_gcm_jewish["j"], m_gcm_jewish["k"], m_gcm_jewish["l"] = 7, 8, 9, 600, 10, 20
	m_gcm_jewish["m"], m_gcm_jewish["n"], m_gcm_jewish["o"], m_gcm_jewish["p"], m_gcm_jewish["q"], m_gcm_jewish["r"] = 30, 40, 50, 60, 70, 80
	m_gcm_jewish["s"], m_gcm_jewish["t"], m_gcm_jewish["u"], m_gcm_jewish["v"], m_gcm_jewish["w"], m_gcm_jewish["x"] = 90, 100, 200, 700, 900, 300
	m_gcm_jewish["y"], m_gcm_jewish["z"] = 400, 500

	m_gcm_english["a"], m_gcm_english["b"], m_gcm_english["c"], m_gcm_english["d"], m_gcm_english["e"], m_gcm_english["f"] = 6, 12, 18, 24, 30, 36
	m_gcm_english["g"], m_gcm_english["h"], m_gcm_english["i"], m_gcm_english["j"], m_gcm_english["k"], m_gcm_english["l"] = 42, 48, 54, 60, 66, 72
	m_gcm_english["m"], m_gcm_english["n"], m_gcm_english["o"], m_gcm_english["p"], m_gcm_english["q"], m_gcm_english["r"] = 78, 84, 90, 96, 102, 108
	m_gcm_english["s"], m_gcm_english["t"], m_gcm_english["u"], m_gcm_english["v"], m_gcm_english["w"], m_gcm_english["x"] = 114, 120, 126, 132, 138, 144
	m_gcm_english["y"], m_gcm_english["z"] = 150, 156

	m_gcm_simple["a"], m_gcm_simple["b"], m_gcm_simple["c"], m_gcm_simple["d"], m_gcm_simple["e"], m_gcm_simple["f"] = 1, 2, 3, 4, 5, 6
	m_gcm_simple["g"], m_gcm_simple["h"], m_gcm_simple["i"], m_gcm_simple["j"], m_gcm_simple["k"], m_gcm_simple["l"] = 7, 8, 9, 10, 11, 12
	m_gcm_simple["m"], m_gcm_simple["n"], m_gcm_simple["o"], m_gcm_simple["p"], m_gcm_simple["q"], m_gcm_simple["r"] = 13, 14, 15, 16, 17, 18
	m_gcm_simple["s"], m_gcm_simple["t"], m_gcm_simple["u"], m_gcm_simple["v"], m_gcm_simple["w"], m_gcm_simple["x"] = 19, 20, 21, 22, 23, 24
	m_gcm_simple["y"], m_gcm_simple["z"] = 25, 26
}

func charGematrix(str string) (uint, uint, uint) {
	if len(str) != 1 {
		log.Println("fatal charGematrix(", str, ")")
		return 0, 0, 0
	}
	j, e, s := m_gcm_jewish[str], m_gcm_english[str], m_gcm_simple[str]
	return j, e, s
}

func NewGemScore(data string) GemScore {
	if !a_b_gematria_loaded.Load() {
		InitGematria()
		a_b_gematria_loaded.Store(true)
	}
	re, err := regexp.Compile(`[^a-zA-Z\d.\s]`)
	if err != nil {
		log.Fatal(err)
	}
	data = re.ReplaceAllString(data, "")
	data = strings.TrimLeft(data, "")
	dataBytes := []byte(data)
	var letters []GemScore
	for i := 0; i < len(dataBytes); i++ {
		wat := strings.ToUpper(string(dataBytes[i]))
		if len(wat) == 1 && wat != "" && wat != " " {
			j, e, s := charGematrix(wat)
			letters = append(letters, GemScore{
				Jewish:  j,
				English: e,
				Simple:  s,
			})
		}
	}
	var jf, ef, sf uint
	for _, gs := range letters {
		jf += gs.Jewish
		ef += gs.English
		sf += gs.Simple
	}
	return GemScore{Jewish: jf, English: ef, Simple: sf}
}

func (s GemScore) String() string {
	output := "\t"
	output += fmt.Sprintf("%s = %d \t", "Jewish", s.Jewish)
	output += fmt.Sprintf("%s = %d \t", "English", s.English)
	output += fmt.Sprintf("%s = %d \t", "Simple", s.Simple)
	return output
}

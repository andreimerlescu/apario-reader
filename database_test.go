package main

import (
	`fmt`
	`regexp`
	`strings`
	`testing`
)

const testString string = `Approved For Release 2008/04/01 : NSA-RDP96X00790R000100040024-8

| . a
eX ~ te ws in we:
ve a * Pe Aide
-t; 4. 3 swage ~
ei bts . x 1 Fang te Te be i
Fee Pe ye a” gt ts oh See oye ere “Eng ses
They fT Bag Te aes * -24 ein PRR
. re o- “sua . — “at! : CRS : 2%
“SER: ty ee a oe ede .
-* “ " way eae eS yietés jh
with regitd 20. the “first category, basic knowledze might be gained in
*“ 18s = eat

‘wee.’
two ways: che “acquisition of new scientific information and the improved

 

application of scientific methodology to deal more effectively vith irregular
phenomena such as ESP, PK, and other psi phenomena. For example, in the

area of education, the ability of the human mind to obtain informaticen purpose-—
fully in a heretofore “intuitive” (i.e. "gut-feeling"’) manner could have a
significance impact on decisionmaking theory and practice and on traditional

approaches t2 training the human mind in educational and vocational settings.

 

a . ee 5. .

with regard so the second category, fotential applications o% pat? oo Bia

= . Lia we i"

enly be srecuiated on conservatively, given che uncertain and relatively: Le 7 ae
. 1 oe
irreplicable status of psi funccions thus far subjected to Systematic séru~ 7 ot Rags.
ee EE,
tiny. It seems Likely to expect that such functions as remote percertion, worry et.
precognition, and vetrocognition are of potential interesr to inteliigence wee

agencies, law enfircerent units, and activities involving surveillance.
For example, “emational imerints"” have already been used to trace past
events in archeslicisal ani police inavesrigacions. at
Dowsing or divining for water, oil, and other sinerais:- is an estaslished
practice in this couniry and abroad, particutar!y ia the Sovier Union. Ac-
corcgingly, further investization into the nature and sossibdle mechanisms cf
this phenomenon, ehich appears to ancorperate both natura! and paranorna!

aspects, would seem to be warranted.

In the area of health, the coupling of tradicional medicine wich mind-

initiated healing is being investigated and could be advanced. Recent work

ai
G
sy
Oo

22/ Jahn, Tt, op. cit.

 

Approved For Release 2008/04/01 : NSA-RDP96X00790R000100040024-8
`

// Benchmark using strings.Fields
func BenchmarkStringFields(b *testing.B) {
	for i := 0; i < b.N; i++ {
		words := strings.Fields(testString) // 15435 ns/op
		if i == 0 {
			fmt.Printf("%d\n", len(words)) // 370 words
		}
		if i == b.N-1 {
			fmt.Printf("%d\n", len(words))
		}
	}
}

// Benchmark using regular expressions
func BenchmarkRegexp(b *testing.B) {
	wordRegexp := regexp.MustCompile(`\b[\p{L}\d]+\b`)
	for i := 0; i < b.N; i++ {
		words := wordRegexp.FindAllString(testString, -1) // 112311 ns/op
		if i == 0 {
			fmt.Printf("%d\n", len(words)) // 356 words
		}
		if i == b.N-1 {
			fmt.Printf("%d\n", len(words))
		}
	}
}

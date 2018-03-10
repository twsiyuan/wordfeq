package wordfreq

import (
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/reiver/go-porterstemmer"
)

type Options struct {
	Languages          []string // Default: ['chinese', 'english']
	StopWordSets       []string // Default: ['cjk', 'english1', 'english2']
	StopWords          []string // Default: []
	NoFilterSubstring  bool     // Default: false
	MaxiumPhraseLength int      // Default: 8
	MinimumCount       int      // Default: 2
}

func New(ops Options) (*WordFeq, error) {
	if ops.Languages == nil {
		ops.Languages = []string{"chinese", "english"}
	}

	if ops.StopWordSets == nil {
		ops.StopWordSets = []string{"cjk", "english1", "english2"}
	}

	if ops.StopWords == nil {
		ops.StopWords = []string{}
	}

	if ops.MaxiumPhraseLength <= 0 {
		ops.MaxiumPhraseLength = 8
	}

	if ops.MinimumCount <= 0 {
		ops.MinimumCount = 2
	}

	ops.StopWords = append(ops.StopWords, stopWordsFromSets(ops.StopWordSets)...)

	return &WordFeq{
		options: ops,
		terms:   make(map[string]int),
		list:    make([]Term, 0),
	}, nil
}

type WordFeq struct {
	options Options
	terms   map[string]int
	list    []Term
}

type Term struct {
	Term  string
	Count int
}

func (w *WordFeq) Process(text string) []Term {

	pushTerm := func(term string, count int) {
		if n, ok := w.terms[term]; ok {
			w.terms[term] = n + count
		} else {
			w.terms[term] = count
		}
	}

	for _, lang := range w.options.Languages {
		switch lang {
		case "english":
			processEnglish(text, w.options.StopWords, pushTerm)
			break
		case "chinese":
			processChinese(text, w.options.StopWords, w.options.MaxiumPhraseLength, w.options.NoFilterSubstring, pushTerm)
			break
		}
	}

	w.list = w.list[:0]
	for term, termCount := range w.terms {
		if termCount < w.options.MinimumCount {
			continue
		}
		w.list = append(w.list, Term{term, termCount})
	}
	sort.Sort(byTerm(w.list))

	return w.list
}

func (w *WordFeq) Empty() {
	w.list = w.list[:0]
	w.terms = make(map[string]int)
}

func (w WordFeq) List() []Term {
	return w.list
}

type byTerm []Term

func (s byTerm) Len() int {
	return len(s)
}
func (s byTerm) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s byTerm) Less(i, j int) bool {
	t1 := s[i]
	t2 := s[j]
	if t1.Count == t2.Count {
		return t1.Term < t2.Term
	} else {
		return t1.Count > t2.Count
	}
}

type stemWord struct {
	Word  string
	Count int
}

var (
	engSplit = regexp.MustCompile("[^A-Za-zéÉ'’_\\-0-9@\\.]+")
	engR1    = regexp.MustCompile("\\.+")                      // replace multiple full stops
	engR2    = regexp.MustCompile("(.{3,})\\.$")               // replace single trailing stop
	engR3    = regexp.MustCompile("(?i)n[\\'’]t\b")            // get rid of ~n't
	engR4    = regexp.MustCompile("(?i)[\\'’](s|ll|d|ve)?\\b") // get rid of ’ and '
	engTest  = regexp.MustCompile("^[0-9\\.@\\-]+$")
)

func processEnglish(text string, stopWords []string, pushTerm func(string, int)) {

	// For English, we count "stems" instead of words,
	// and decide how to represent that stem at the end
	// according to the counts.
	stems := make(map[string]*stemWord)

	// say bye bye to characters that is not belongs to a word
	words := engSplit.Split(text, -1)

	for _, word := range words {
		word = engR1.ReplaceAllString(word, ".")
		word = engR2.ReplaceAllString(word, "$1")
		word = engR3.ReplaceAllString(word, "")
		word = engR4.ReplaceAllString(word, "")

		// skip if the word is shorter than two characters
		// (i.e. exactly one letter)
		if utf8.RuneCountInString(word) <= 2 {
			continue
		}

		if engTest.MatchString(word) {
			continue
		}

		// stopwords test
		ok := true
		for _, stopWord := range stopWords {
			if stopWord == word {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}

		stem := strings.ToLower(porterstemmer.StemString(word))

		// count++ for the stem
		wc, ok := stems[stem]
		if !ok {
			wc = &stemWord{word, 0}
			stems[stem] = wc
		}
		wc.Count += 1

		// if the current word representing the stem is longer than
		// this one, use this word instead (booking -> book)
		if utf8.RuneCountInString(word) < utf8.RuneCountInString(wc.Word) {
			wc.Word = word
		}

		// if the current word representing the stem is of the same
		// length but with different form,
		// use the lower-case representation (Book -> book)
		if utf8.RuneCountInString(word) == utf8.RuneCountInString(wc.Word) &&
			word != wc.Word {
			wc.Word = strings.ToLower(word)
		}
	}

	// Push each "stem" into terms as word
	for _, stem := range stems {
		pushTerm(stem.Word, stem.Count)
	}
}

var (
	chReplace = regexp.MustCompile("[^\u4E00-\u9FFF\u3400-\u4DBF]+")
	chTest    = regexp.MustCompile("^[\u4E00-\u9FFF\u3400-\u4DBF]+$")
	chLines   = regexp.MustCompile("\n+")
)

func processChinese(text string, stopWords []string, maxPhrashLength int, noFilterSubstring bool, pushTerm func(string, int)) {
	// Chinese is a language without word boundary.
	// We must use N-gram here to extract meaningful terms.

	// say good bye to non-Chinese (Kanji) characters
	// TBD: Cannot match CJK characters beyond BMP,
	// e.g. \u20000-\u2A6DF at plane B.

	// Han: \u4E00-\u9FFF\u3400-\u4DBF
	// Kana: \u3041-\u309f\u30a0-\u30ff
	text = chReplace.ReplaceAllString(text, "\n")

	// Use the stop words as separators -- replace them.
	for _, stopWord := range stopWords {
		// Not handling that stop word if it's not a Chinese word.
		if !chTest.MatchString(stopWord) {
			continue
		}

		text = strings.Replace(text, stopWord, stopWord+"\n", -1)
	}

	chunks := chLines.Split(text, -1)
	pendingTerms := make(map[string]int)

	// counts all the chunks (and it's substrings) in pendingTerms
	for _, chunk := range chunks {
		if utf8.RuneCountInString(chunk) <= 1 {
			continue
		}

		substrings := getAllSubStrings(chunk, maxPhrashLength)
		for _, substring := range substrings {
			if utf8.RuneCountInString(substring) <= 1 {
				continue
			}

			if n, ok := pendingTerms[substring]; !ok {
				pendingTerms[substring] = 1
			} else {
				pendingTerms[substring] = n + 1
			}
		}
	}

	// if filterSubstring is true, remove the substrings with the exact
	// same count as the longer term (implying they are only present in
	// the longer terms)
	if !noFilterSubstring {
		for term, termCount := range pendingTerms {
			var substrings = getAllSubStrings(term, maxPhrashLength)
			for _, substring := range substrings {
				if term == substring {
					continue
				}

				if subTermCount, ok := pendingTerms[substring]; ok {
					if subTermCount == termCount {
						delete(pendingTerms, substring)
					}
				}
			}
		}
	}

	// add the pendingTerms into terms
	for term, termCount := range pendingTerms {
		pushTerm(term, termCount)
	}
}

// Return all possible substrings of a give string in an array
// If there is no maxLength is unrestricted, array will contain
// (2 * str.length) substrings.
func getAllSubStrings(str string, maxLength int) []string {
	runes := []rune(str)
	result := make([]string, 0, 2*len(runes))
	i := maxLength
	c := len(runes)
	if c < maxLength {
		i = c
	}

	for ; i >= 1; i-- {
		result = append(result, string(runes[0:i]))
	}

	if c > 1 {
		result = append(result, getAllSubStrings(string(runes[1:]), maxLength)...)
	}

	return result
}

// Default stop words from set
func stopWordsFromSets(sets []string) []string {
	words := make([]string, 0)
	for _, set := range sets {
		switch set {
		case "cjk":
			words = append(words,
				"\u3092",       /* Japanese wo */
				"\u3067\u3059", /* Japanese desu */
				"\u3059\u308b", /* Japanese suru */
				"\u306e",       /* Japanese no */
				"\u308c\u3089", /* Japanese rera */
			)
			break
		case "english1":
			words = append(words, "i", "a", "about", "an", "and", "are", "as", "at",
				"be", "been", "by", "com", "for", "from", "how", "in",
				"is", "it", "not", "of", "on", "or", "that",
				"the", "this", "to", "was", "what", "when", "where", "which",
				"who", "will", "with", "www", "the")
			break
		case "english2":
			words = append(words, "we", "us", "our", "ours",
				"they", "them", "their", "he", "him", "his",
				"she", "her", "hers", "it", "its", "you", "yours", "your",
				"has", "have", "would", "could", "should", "shall",
				"can", "may", "if", "then", "else", "but",
				"there", "these", "those")
			break
		}
	}
	return words
}

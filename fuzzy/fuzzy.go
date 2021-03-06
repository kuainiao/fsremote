package main

import (
	"fmt"
	"index/suffixarray"
	"strings"
	"sync"
)

type Pair struct {
	str1 string
	str2 string
}

type Potential struct {
	Term   string
	Score  int
	Leven  int
	Method int // 0 - is word, 1 - suggest maps to input, 2 - input delete maps to dictionary, 3 - input delete maps to suggest
}

type Model struct {
	Data            map[string]int      `json:"data"`
	Maxcount        int                 `json:"maxcount"`
	Suggest         map[string][]string `json:"suggest"`
	Depth           int                 `json:"depth"`
	Threshold       int                 `json:"threshold"`
	SuffDivergence  int                 `json:"-"`
	SuffixArr       *suffixarray.Index  `json:"-"`
	SuffixArrConcat string              `json:"-"`
	sync.RWMutex
}

// For sorting autocomplete suggestions
// to bias the most popular first
type Autos struct {
	Results []string
	Model   *Model
}

func (a Autos) Len() int      { return len(a.Results) }
func (a Autos) Swap(i, j int) { a.Results[i], a.Results[j] = a.Results[j], a.Results[i] }

func (a Autos) Less(i, j int) bool {
	icount := a.Model.Data[a.Results[i]]
	jcount := a.Model.Data[a.Results[j]]
	if icount == jcount {
		return a.Results[i] > a.Results[j]
	}
	return icount > jcount
}

// Create and initialise a new model
func NewModel() *Model {
	model := new(Model)
	return model.Init()
}

func (model *Model) Init() *Model {
	model.Data = make(map[string]int)
	model.Suggest = make(map[string][]string)
	model.Depth = 2
	model.Threshold = 3 // Setting this to 1 is most accurate, but "1" is 5x more memory and 30x slower processing than "4". This is a big performance tuning knob
	return model
}

// Change the default depth value of the model. This sets how many
// character differences are indexed. The default is 2.
func (model *Model) SetDepth(val int) {
	model.Lock()
	model.Depth = val
	model.Unlock()
}

// Change the default threshold of the model. This is how many times
// a term must be seen before suggestions are created for it
func (model *Model) SetThreshold(val int) {
	model.Lock()
	model.Threshold = val
	model.Unlock()
}

// Calculate the Levenshtein distance between two strings
func LevenshteinR(a, b []rune) int {
	la := len(a)
	lb := len(b)
	d := make([]int, la+1)
	var lastdiag, olddiag, temp int

	for i := 1; i <= la; i++ {
		d[i] = i
	}
	for i := 1; i <= lb; i++ {
		d[0] = i
		lastdiag = i - 1
		for j := 1; j <= la; j++ {
			olddiag = d[j]
			min := d[j] + 1
			if (d[j-1] + 1) < min {
				min = d[j-1] + 1
			}
			if a[j-1] == b[i-1] {
				temp = 0
			} else {
				temp = 1
			}
			if (lastdiag + temp) < min {
				min = lastdiag + temp
			}
			d[j] = min
			lastdiag = olddiag
		}
	}
	return d[la]
}

// Add an array of words to train the model in bulk
func (model *Model) Train(terms []string) {
	for _, term := range terms {
		model.TrainWord(term)
	}

}

// Manually set the count of a word. Optionally trigger the
// creation of suggestion keys for the term. This function lets
// you build a model from an existing dictionary with word popularity
// counts without needing to run "TrainWord" repeatedly
func (model *Model) SetCount(term string, count int, suggest bool) {
	model.Lock()
	model.Data[term] = count
	if suggest {
		model.createSuggestKeys(term)
	}
	model.Unlock()
}

// Train the model word by word
func (model *Model) TrainWord(term string) {
	model.Lock()
	model.Data[term]++
	// Set the max
	if model.Data[term] > model.Maxcount {
		model.Maxcount = model.Data[term]
		model.SuffDivergence++
	}
	// If threshold is triggered, store delete suggestion keys
	if model.Data[term] == model.Threshold {
		model.createSuggestKeys(term)
	}
	model.Unlock()
}

// For a given term, create the partially deleted lookup keys
func (model *Model) createSuggestKeys(term string) {
	edits := model.EditsMulti(term, model.Depth)
	for _, edit := range edits {
		skip := false
		for _, hit := range model.Suggest[edit] {
			if hit == term {
				// Already know about this one
				skip = true
				continue
			}
		}
		if !skip && len(edit) > 1 {
			model.Suggest[edit] = append(model.Suggest[edit], term)
		}
	}
}

// Edits at any depth for a given term. The depth of the model is used
func (model *Model) EditsMulti(term string, depth int) []string {
	edits := Edits1(term)
	for {
		depth--
		if depth <= 0 {
			break
		}
		for _, edit := range edits {
			edits2 := Edits1(edit)
			for _, edit2 := range edits2 {
				edits = append(edits, edit2)
			}
		}
	}
	return edits
}

// Edits1 creates a set of terms that are 1 char delete from the input term
func Edits1(word string) []string {
	rword := []rune(word)
	splits := []Pair{}
	for i := 0; i <= len(rword); i++ {
		splits = append(splits, Pair{string(rword[:i]), string(rword[i:])})
	}

	total_set := []string{}
	for _, elem := range splits {
		rstr2 := []rune(elem.str2)
		//deletion
		if len(rstr2) > 0 {
			total_set = append(total_set, string(append([]rune(elem.str1), rstr2[1:]...)))
		} else {
			total_set = append(total_set, elem.str1)
		}

	}
	return total_set
}

func (model *Model) score(input string) int {
	if score, ok := model.Data[input]; ok {
		return score
	}
	return 0
}

// From a group of potentials, work out the most likely result
func best(in string, potential map[string]*Potential) string {
	rinput := []rune(in)
	best := ""
	bestcalc := 0
	for i := 0; i < 4; i++ {
		for _, pot := range potential {
			if pot.Leven == 0 {
				return pot.Term
			} else if pot.Leven == i {
				if pot.Score > bestcalc {
					bestcalc = pot.Score
					// If the first letter is the same, that's a good sign. Bias these potentials
					if []rune(pot.Term)[0] == rinput[0] {
						bestcalc += bestcalc * 100
					}

					best = pot.Term
				}
			}
		}
		if bestcalc > 0 {
			return best
		}
	}

	return best
}

// Test an input, if we get it wrong, look at why it is wrong. This
// function returns a bool indicating if the guess was correct as well
// as the term it is suggesting. Typically this function would be used
// for testing, not for production
func (model *Model) CheckKnown(input string, correct string) bool {
	model.RLock()
	defer model.RUnlock()
	suggestions := model.suggestPotential(input, true)
	best := best(input, suggestions)
	if best == correct {
		// This guess is correct
		fmt.Printf("Input correctly maps to correct term")
		return true
	}
	if pot, ok := suggestions[correct]; !ok {

		if model.score(correct) > 0 {
			fmt.Printf("\"%v\" - %v (%v) not in the suggestions. (%v) best option.\n", input, correct, model.score(correct), best)
			for _, sugg := range suggestions {
				fmt.Printf("	%v\n", sugg)
			}
		} else {
			fmt.Printf("\"%v\" - Not in dictionary\n", correct)
		}
	} else {
		fmt.Printf("\"%v\" - (%v) suggested, should however be (%v).\n", input, suggestions[best], pot)
	}
	return false
}

// For a given input term, suggest some alternatives. If exhaustive, each of the 4
// cascading checks will be performed and all potentials will be sorted accordingly
func (model *Model) suggestPotential(input string, exhaustive bool) map[string]*Potential {
	input = strings.ToLower(input)
	suggestions := make(map[string]*Potential, 20)

	// 0 - If this is a dictionary term we're all good, no need to go further
	if model.score(input) > model.Threshold {
		suggestions[input] = &Potential{Term: input, Score: model.score(input), Leven: 0, Method: 0}
		if !exhaustive {
			return suggestions
		}
	}

	// 1 - See if the input matches a "suggest" key
	if sugg, ok := model.Suggest[input]; ok {
		for _, pot := range sugg {
			if _, ok := suggestions[pot]; !ok {
				suggestions[pot] = &Potential{Term: pot, Score: model.score(pot), Leven: LevenshteinR([]rune(input), []rune(pot)), Method: 1}
			}
		}

		if !exhaustive {
			return suggestions
		}
	}

	// 2 - See if edit1 matches input
	max := 0
	edits := model.EditsMulti(input, model.Depth)
	for _, edit := range edits {
		score := model.score(edit)
		if score > 0 && len(edit) > 2 {
			if _, ok := suggestions[edit]; !ok {
				suggestions[edit] = &Potential{Term: edit, Score: score, Leven: LevenshteinR([]rune(input), []rune(edit)), Method: 2}
			}
			if score > max {
				max = score
			}
		}
	}
	if max > 0 {
		if !exhaustive {
			return suggestions
		}
	}

	// 3 - No hits on edit1 distance, look for transposes and replaces
	// Note: these are more complex, we need to check the guesses
	// more thoroughly, e.g. levals=[valves] in a raw sense, which
	// is incorrect
	for _, edit := range edits {
		if sugg, ok := model.Suggest[edit]; ok {
			// Is this a real transpose or replace?
			for _, pot := range sugg {
				lev := LevenshteinR([]rune(input), []rune(pot))
				if lev <= model.Depth+1 { // The +1 doesn't seem to impact speed, but has greater coverage when the depth is not sufficient to make suggestions
					if _, ok := suggestions[pot]; !ok {
						suggestions[pot] = &Potential{Term: pot, Score: model.score(pot), Leven: lev, Method: 3}
					}
				}
			}
		}
	}
	return suggestions
}

// Return the raw potential terms so they can be ranked externally
// to this package
func (model *Model) Potentials(input string, exhaustive bool) map[string]*Potential {
	model.RLock()
	defer model.RUnlock()
	return model.suggestPotential(input, exhaustive)
}

// For a given input string, suggests potential replacements
func (model *Model) Suggestions(input string, exhaustive bool) []string {
	model.RLock()
	suggestions := model.suggestPotential(input, exhaustive)
	model.RUnlock()
	output := make([]string, 0, 10)
	for _, suggestion := range suggestions {
		output = append(output, suggestion.Term)
	}
	return output
}

// Return the most likely correction for the input term
func (model *Model) SpellCheck(input string) string {
	model.RLock()
	suggestions := model.suggestPotential(input, false)
	model.RUnlock()
	return best(input, suggestions)
}

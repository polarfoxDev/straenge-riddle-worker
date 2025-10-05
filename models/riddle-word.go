package models

import "strings"

type RiddleWord struct {
	Word            string `json:"word"`
	IsSuperSolution bool   `json:"isSuperSolution"`
	Color           string `json:"color"`
	Used            bool   `json:"used"`
	letters         []rune
	cachedWord      string
}

func MakeWordSafe(word string) string {
	word = strings.ToUpper(word)
	word = strings.ReplaceAll(word, " ", "")
	word = strings.ReplaceAll(word, "-", "")
	word = strings.ReplaceAll(word, "ß", "ẞ")
	return word
}

func (word *RiddleWord) ensureLetters() {
	if word == nil {
		return
	}
	if word.cachedWord != word.Word {
		word.letters = []rune(word.Word)
		word.cachedWord = word.Word
	}
}

func (word *RiddleWord) Length() int {
	word.ensureLetters()
	return len(word.letters)
}

func (word *RiddleWord) RuneAt(index int) rune {
	word.ensureLetters()
	return word.letters[index]
}

func (word *RiddleWord) Letters() []rune {
	word.ensureLetters()
	return word.letters
}

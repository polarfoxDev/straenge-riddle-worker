package models

import (
	"strconv"
	"strings"
)

type RiddleWord struct {
	Word            string `json:"word"`
	IsSuperSolution bool   `json:"isSuperSolution"`
	Color           string `json:"color"`
	Used            bool   `json:"used"`
}

var specialCharacterMap = []string{"Ä", "Ö", "Ü", "ẞ"}

func MakeWordSafe(word string) string {
	word = strings.ToUpper(word)
	word = strings.ReplaceAll(word, " ", "")
	word = strings.ReplaceAll(word, "-", "")
	word = strings.ReplaceAll(word, "ß", "ẞ")
	for i, specialCharacter := range specialCharacterMap {
		word = strings.ReplaceAll(word, specialCharacter, strconv.Itoa(i))
	}
	return word
}

func MakeWordUnsafe(word string) string {
	for i, specialCharacter := range specialCharacterMap {
		word = strings.ReplaceAll(word, strconv.Itoa(i), specialCharacter)
	}
	return word
}

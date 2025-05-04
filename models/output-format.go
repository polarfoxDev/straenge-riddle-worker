package models

type RiddleConfig struct {
	ConfigVersion int              `json:"configVersion"`
	Theme         string           `json:"theme"`
	Letters       [][]string       `json:"letters"`
	Solutions     []SolutionConfig `json:"solutions"`
}

type RiddleConfigFromFile struct {
	FilePath     string
	RiddleConfig *RiddleConfig
}

type SolutionConfig struct {
	Word            string           `json:"_generator_word"`
	Locations       []LetterLocation `json:"locations"`
	IsSuperSolution bool             `json:"isSuperSolution"`
}

type LetterLocation struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

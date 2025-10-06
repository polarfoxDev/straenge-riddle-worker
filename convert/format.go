package convert

import (
	"straenge-riddle-worker/m/models"
)

func TransformToOutputFormat(riddle *models.Riddle, theme string) *models.RiddleConfig {
	var riddleConfig = models.RiddleConfig{
		ConfigVersion: 3,
		Theme:         theme,
		Letters:       make([][]string, models.RiddleHeight),
		Solutions:     []models.SolutionConfig{},
	}
	for i := 0; i < models.RiddleHeight; i++ {
		riddleConfig.Letters[i] = make([]string, models.RiddleWidth)
		for j := 0; j < models.RiddleWidth; j++ {
			var node = riddle.GetNode(i, j)
			if node.RiddleWord == nil {
				riddleConfig.Letters[i][j] = " "
			} else {
				if node.RiddleWordIndex == -1 {
					riddleConfig.Letters[i][j] = "?"
				} else {
					riddleConfig.Letters[i][j] = string(node.RiddleWord.RuneAt(node.RiddleWordIndex))
				}
			}
		}
	}
	for _, word := range riddle.Words {
		if !word.Used {
			continue
		}
		var edges = riddle.GetEdgesForWord(word)
		var locations = []models.LetterLocation{}
		for _, edge := range edges {
			locations = append(locations, models.LetterLocation{Row: edge.Node1.Row, Col: edge.Node1.Col})
		}
		locations = append(locations, models.LetterLocation{Row: edges[len(edges)-1].Node2.Row, Col: edges[len(edges)-1].Node2.Col})
		riddleConfig.Solutions = append(riddleConfig.Solutions, models.SolutionConfig{
			Locations:       locations,
			IsSuperSolution: word.IsSuperSolution,
			Word:            word.Word,
		})
	}
	return &riddleConfig
}

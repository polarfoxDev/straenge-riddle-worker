package models

type LetterEdge struct {
	Word  *RiddleWord `json:"word"`
	Node1 *Node       `json:"node1"`
	Node2 *Node       `json:"node2"`
}

package models

type Node struct {
	Row             int         `json:"row"`
	Col             int         `json:"col"`
	RiddleWord      *RiddleWord `json:"riddleWord"`
	RiddleWordIndex int         `json:"riddleWordIndex"`
}

func (node *Node) isEmpty() bool {
	return node.RiddleWord == nil
}

func ContainsNodePosition(nodes []*Node, node *Node) bool {
	for _, n := range nodes {
		if n.Row == node.Row && n.Col == node.Col {
			return true
		}
	}
	return false
}

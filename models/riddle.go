package models

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"straenge-riddle-worker/m/defaults/colors"
	"strconv"

	"github.com/sirupsen/logrus"
)

const (
	RiddleWidth  int = 6
	RiddleHeight int = 8
)

type Riddle struct {
	Nodes []*Node       `json:"nodes"`
	Words []*RiddleWord `json:"words"`
	Edges []*LetterEdge `json:"edges"`
}

func NewRiddleFromConfig(riddleConfig *RiddleConfig) *Riddle {
	availableColors := []string{colors.Blue, colors.Cyan, colors.Gray, colors.Green, colors.Magenta, colors.Red, colors.Yellow}
	var riddle = &Riddle{
		Nodes: make([]*Node, RiddleWidth*RiddleHeight),
		Words: []*RiddleWord{},
		Edges: []*LetterEdge{},
	}
	for i := 0; i < RiddleHeight; i++ {
		for j := 0; j < RiddleWidth; j++ {
			riddle.Nodes[i*RiddleWidth+j] = &Node{
				Row: i,
				Col: j,
			}
		}
	}
	for _, solution := range riddleConfig.Solutions {
		var color = colors.White
		if !solution.IsSuperSolution {
			color = availableColors[rand.Intn(len(availableColors))]
		}
		// can't use solutionWord := solution.Word because some solutions don't have a word
		// instead, get it by the solution.Locations and the letters in the riddle
		var solutionWord string
		for _, location := range solution.Locations {
			solutionWord += riddleConfig.Letters[location.Row][location.Col]
		}
		var word = &RiddleWord{
			Word:            MakeWordSafe(solutionWord),
			IsSuperSolution: solution.IsSuperSolution,
			Color:           color,
			Used:            true,
		}
		riddle.Words = append(riddle.Words, word)
		for i, location := range solution.Locations {
			riddle.FillNode(location.Row, location.Col, word, i)
		}
		// make edges
		for i := 0; i < len(solution.Locations)-1; i++ {
			riddle.Edges = append(riddle.Edges, &LetterEdge{
				Word:  word,
				Node1: riddle.GetNode(solution.Locations[i].Row, solution.Locations[i].Col),
				Node2: riddle.GetNode(solution.Locations[i+1].Row, solution.Locations[i+1].Col),
			})
		}
	}
	return riddle
}

func NewRiddle(superSolution string, words []string) (*Riddle, error) {
	if len(superSolution) < 6 {
		return nil, &RiddleError{ErrType: ErrWordLength, Message: "Super solution word too short"}
	}
	var riddle = &Riddle{
		Nodes: make([]*Node, RiddleWidth*RiddleHeight),
		Words: []*RiddleWord{},
		Edges: []*LetterEdge{},
	}
	riddle.Words = append(riddle.Words, &RiddleWord{
		Word:            MakeWordSafe(superSolution),
		IsSuperSolution: true,
		Color:           colors.White,
		Used:            true,
	})
	for index, word := range words {
		// color: get from colors and begin from the beginning if overflown
		colors := []string{colors.Blue, colors.Cyan, colors.Gray, colors.Green, colors.Magenta, colors.Red, colors.Yellow}
		color := colors[index%len(colors)]
		riddle.Words = append(riddle.Words, &RiddleWord{
			Word:            MakeWordSafe(word),
			IsSuperSolution: false,
			Color:           color,
			Used:            false,
		})
	}
	for row := 0; row < RiddleHeight; row++ {
		for col := 0; col < RiddleWidth; col++ {
			riddle.Nodes[row*RiddleWidth+col] = &Node{
				Row: row,
				Col: col,
			}
		}
	}
	riddle, error := riddle.FillWord(riddle.Words[0], riddle.Nodes)
	return riddle, error
}

func (riddle *Riddle) Copy() *Riddle {
	var newRiddle = &Riddle{
		Nodes: make([]*Node, RiddleWidth*RiddleHeight),
		Words: make([]*RiddleWord, len(riddle.Words)),
		Edges: riddle.Edges,
	}
	for i, node := range riddle.Nodes {
		// log the node
		newRiddle.Nodes[i] = &Node{
			Row:             node.Row,
			Col:             node.Col,
			RiddleWord:      node.RiddleWord,
			RiddleWordIndex: node.RiddleWordIndex,
		}
	}
	for i, word := range riddle.Words {
		newRiddle.Words[i] = &RiddleWord{
			Word:            word.Word,
			IsSuperSolution: word.IsSuperSolution,
			Color:           word.Color,
			Used:            word.Used,
		}
	}
	return newRiddle
}

func (riddle *Riddle) FillWithWords() (*Riddle, error) {
	subgraphsToFill := riddle.GetAllSubgraphs()
	// sort subgraphs by size ascending
	for i := 0; i < len(subgraphsToFill); i++ {
		for j := i + 1; j < len(subgraphsToFill); j++ {
			if len(subgraphsToFill[i]) > len(subgraphsToFill[j]) {
				subgraphsToFill[i], subgraphsToFill[j] = subgraphsToFill[j], subgraphsToFill[i]
			}
		}
	}

	updatedRiddle := riddle.Copy()
	logrus.Debug("[FillWithWords] Subgraphs to fill: ", len(subgraphsToFill))
	for index, subgraph := range subgraphsToFill {
		logrus.Debug("[FillWithWords] Filling subgraph " + strconv.Itoa(index) + "/" + strconv.Itoa(len(subgraphsToFill)-1))
		riddleWithFilledSubgraph, error := updatedRiddle.fillSubgraphRecursive(0, subgraph)
		if error != nil {
			return nil, error
		}
		updatedRiddle = riddleWithFilledSubgraph
	}
	return updatedRiddle, nil
}

func (riddle *Riddle) fillSubgraphRecursive(depth int, subgraph []*Node) (*Riddle, error) {
	logrus.Debug("[fillSubgraphRecursive("+strconv.Itoa(depth)+")] Trying to fill subgraph with length ", len(subgraph))
	availableWords := []*RiddleWord{}
	for _, word := range riddle.Words {
		if !word.Used && (len(word.Word) <= len(subgraph)-4 || len(word.Word) == len(subgraph)) {
			availableWords = append(availableWords, word)
		}
	}
	if len(availableWords) == 0 {
		return nil, &RiddleError{ErrType: ErrWordFill, Message: "No available words to fill subgraph of size " + strconv.Itoa(len(subgraph))}
	}
	// randomize order of available words
	for i := range availableWords {
		j := rand.Intn(i + 1)
		availableWords[i], availableWords[j] = availableWords[j], availableWords[i]
	}
	logrus.Debug("[fillSubgraphRecursive("+strconv.Itoa(depth)+")] availableWord count: ", len(availableWords))
	for _, word := range availableWords {
		logrus.Debug("[fillSubgraphRecursive("+strconv.Itoa(depth)+")] trying with word: ", word.Word)
		riddleWithWordFilled, err := riddle.FillWord(word, subgraph)
		if err != nil {
			logrus.Debug("[fillSubgraphRecursive("+strconv.Itoa(depth)+")] failed to fill word: ", word.Word)
			logrus.Debug(err)
		}
		if err == nil {
			for _, riddleWord := range riddleWithWordFilled.Words {
				if riddleWord.Word == word.Word {
					riddleWord.Used = true
					break
				}
			}
			riddleWithWordFilled.Render(true)
			subgraphs := riddleWithWordFilled.GetAllSubgraphs()

			// filter subgraphs: only keep those that are part of the original subgraph
			var filteredSubgraphs [][]*Node
			for _, subgraphToCheck := range subgraphs {
				if ContainsNodePosition(subgraph, subgraphToCheck[0]) {
					filteredSubgraphs = append(filteredSubgraphs, subgraphToCheck)
				}
			}
			if len(filteredSubgraphs) == 1 {
				return riddleWithWordFilled.fillSubgraphRecursive(depth+1, filteredSubgraphs[0])
			}
			// if there are multiple subgraphs, try to fill them all
			currentRiddleCopy := riddleWithWordFilled.Copy()
			for _, filteredSubgraph := range filteredSubgraphs {
				var nextRiddleTry *Riddle
				nextRiddleTry, err = currentRiddleCopy.fillSubgraphRecursive(depth+1, filteredSubgraph)
				if err == nil {
					currentRiddleCopy = nextRiddleTry
				} else {
					break
				}
			}
			if err == nil {
				return currentRiddleCopy, nil
			}
		}
	}
	return nil, &RiddleError{ErrType: ErrWordFill, Message: "No possible fill found for subgraph of size " + strconv.Itoa(len(subgraph))}
}

func (riddle *Riddle) FillWord(word *RiddleWord, subgraph []*Node) (*Riddle, error) {
	return riddle.fillWordRecursive(0, word, subgraph, 0, nil, nil, false)
	// forbiddenPaths := [][]*LetterEdge{}
	// isAmbiguous := true
	// for i := 0; i < 10; i++ {
	// 	fmt.Println(i)
	// 	newRiddle, err := riddle.fillWordRecursive(0, word, subgraph, 0, nil, nil, false, forbiddenPaths)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	isAmbiguous, _ = newRiddle.CheckForAmbiguity()
	// 	if isAmbiguous {
	// 		edges := newRiddle.GetEdgesForWord(word)
	// 		// add pseudo edge for the last node
	// 		edges = append(edges, &LetterEdge{
	// 			Word:  word,
	// 			Node1: edges[len(edges)-1].Node2,
	// 			Node2: &Node{Row: -1, Col: -1},
	// 		})
	// 		forbiddenPaths = append(forbiddenPaths, edges)
	// 		continue
	// 	}
	// 	return newRiddle, nil
	// }
	// fmt.Println("Not able to fill word " + word.Word + " without ambiguity")
	// return nil, &RiddleError{ErrType: ErrAmbiguity, Message: "Not able to fill word without ambiguity"}
}

func isEdgeReachable(node *Node, rowToReach, colToReach int, remainingSteps int) bool {
	if (rowToReach == -1 || node.Row == rowToReach) && (colToReach == -1 || node.Col == colToReach) {
		return true
	}
	if remainingSteps == 0 {
		return false
	}
	if rowToReach != -1 {
		// we need to reach rowToReach in the remaining steps, so the difference between the current row and rowToReach must be less or equal to the remaining steps
		return int(math.Abs(float64(node.Row-rowToReach))) <= remainingSteps
	} else {
		// we need to reach colToReach in the remaining steps, so the difference between the current col and colToReach must be less or equal to the remaining steps
		return int(math.Abs(float64(node.Col-colToReach))) <= remainingSteps
	}
}

func (riddle *Riddle) fillWordRecursive(depth int, word *RiddleWord, subgraph []*Node, index int, firstNode *Node, previousNode *Node, touchedOppositeEdge bool) (*Riddle, error) {
	logrus.Debug("[fillWordRecursive("+strconv.Itoa(depth)+")] Trying to fill word ", word.Word, "(l=", len(word.Word), ")[i=", index, "] into subgraph with length ", len(subgraph))
	if index == len(word.Word) {
		return riddle, nil
	}
	if len(subgraph) < len(word.Word)-index {
		return nil, &RiddleError{ErrType: ErrWordFill, Message: "Subgraph too small"}
	}
	if firstNode == nil {
		firstNode = previousNode
	}
	if word.IsSuperSolution && index != 0 && (previousNode.Row == 0 && firstNode.Row == RiddleHeight-1 || previousNode.Row == RiddleHeight-1 && firstNode.Row == 0 || previousNode.Col == 0 && firstNode.Col == RiddleWidth-1 || previousNode.Col == RiddleWidth-1 && firstNode.Col == 0) {
		touchedOppositeEdge = true
		logrus.Debug("Touched opposite edge")
	}
	var possibleNodes []*Node = []*Node{}
	minimumRemainingSubgraphSize := 4
	remainingLetterCount := len(word.Word) - index
	if remainingLetterCount == len(subgraph) {
		minimumRemainingSubgraphSize = remainingLetterCount - 1
	}
	if index == 0 {
		for _, node := range subgraph {
			if riddle.NodeCanBeFilled(word, node, nil, minimumRemainingSubgraphSize) {
				if word.IsSuperSolution {
					if !(node.Row == 0 || node.Row == RiddleHeight-1 || node.Col == 0 || node.Col == RiddleWidth-1) {
						continue
					}
				}
				// // if forbiddenPaths contains an edge that would be drawn by filling this node, skip it
				// forbiddenPath := false
				// for _, path := range forbiddenPaths {
				// 	if path[index].Node1.Row == node.Row && path[index].Node1.Col == node.Col {
				// 		forbiddenPath = true
				// 		break
				// 	}
				// }
				// if forbiddenPath {
				// 	continue
				// }

				possibleNodes = append(possibleNodes, node)
			}
		}
		// fmt.Println("First node")
		// fmt.Println(possibleNodes)
	} else {
		for _, node := range riddle.GetAvailableAdjacentNodes(previousNode.Row, previousNode.Col, minimumRemainingSubgraphSize) {
			if word.IsSuperSolution {
				if !touchedOppositeEdge {
					var rowToReach, colToReach int = -1, -1
					if firstNode.Row == 0 {
						rowToReach = RiddleHeight - 1
					} else if firstNode.Row == RiddleHeight-1 {
						rowToReach = 0
					} else if firstNode.Col == 0 {
						colToReach = RiddleWidth - 1
					} else if firstNode.Col == RiddleWidth-1 {
						colToReach = 0
					}
					logrus.Debug("isEdgeReachable(", node.Row, ",", node.Col, ",", rowToReach, ",", colToReach, ",", remainingLetterCount-1, ") = ", isEdgeReachable(node, rowToReach, colToReach, remainingLetterCount-1))
					if !isEdgeReachable(node, rowToReach, colToReach, remainingLetterCount-1) {
						continue
					}
				}
			}
			// // if forbiddenPaths contains an edge that would be drawn by filling this node, skip it
			// forbiddenPath := false
			// for _, path := range forbiddenPaths {
			// 	if path[index].Node1.Row == node.Row && path[index].Node1.Col == node.Col {
			// 		forbiddenPath = true
			// 		break
			// 	}
			// }
			// if forbiddenPath {
			// 	continue
			// }
			possibleNodes = append(possibleNodes, node)
		}
	}
	if len(possibleNodes) == 0 {
		return nil, &RiddleError{ErrType: ErrWordFill, Message: "No possible nodes to fill word"}
	}
	// randomize order of possible nodes
	for i := range possibleNodes {
		j := rand.Intn(i + 1)
		possibleNodes[i], possibleNodes[j] = possibleNodes[j], possibleNodes[i]
	}
	// depth first try to fill the word with possible nodes
	var lastErr error
	for _, node := range possibleNodes {
		logrus.Debug("[fillWordRecursive("+strconv.Itoa(depth)+")] Trying to use node ", node.Row, ",", node.Col)
		riddleCopy := riddle.Copy()
		riddleCopy.FillNode(node.Row, node.Col, word, index)
		if previousNode != nil {
			var drawnEdge = &LetterEdge{
				Word:  word,
				Node1: previousNode,
				Node2: node,
			}
			riddleCopy.Edges = append(riddleCopy.Edges, drawnEdge)
		}
		// fill the rest of the word
		var nextSubgraph []*Node = []*Node{}
		for _, subgraphNode := range subgraph {
			if subgraphNode.Row != node.Row || subgraphNode.Col != node.Col {
				nextSubgraph = append(nextSubgraph, subgraphNode)
			}
		}
		riddleCopy, lastErr = riddleCopy.fillWordRecursive(depth+1, word, nextSubgraph, index+1, firstNode, node, touchedOppositeEdge)
		if lastErr == nil {
			logrus.Debug("[fillWordRecursive("+strconv.Itoa(depth)+")] Successfully filled word ", word.Word, "(l=", len(word.Word), ") into subgraph with length ", len(subgraph))
			return riddleCopy, nil
		}
		logrus.Debug("[fillWordRecursive("+strconv.Itoa(depth)+")] Failed using this node because: ", lastErr.Error())
	}
	return nil, &RiddleError{ErrType: ErrWordFill, Message: "No possible fill path found, inner error: " + lastErr.Error()}
}

func (riddle *Riddle) FillNode(row, col int, riddleWord *RiddleWord, riddleWordIndex int) {
	riddle.GetNode(row, col).RiddleWord = riddleWord
	riddle.GetNode(row, col).RiddleWordIndex = riddleWordIndex
}

func (riddle *Riddle) GetNode(row, col int) *Node {
	return riddle.Nodes[row*RiddleWidth+col]
}

func (riddle *Riddle) NodeIsInBounds(row, col int) bool {
	return row >= 0 && row < RiddleHeight && col >= 0 && col < RiddleWidth
}

func (riddle *Riddle) GetAllSubgraphs() [][]*Node {
	var subgraphs [][]*Node = [][]*Node{}
	for _, node := range riddle.Nodes {
		if !node.isEmpty() {
			continue
		}
		nodeAlreadyInSubgraphs := false
		for _, subgraph := range subgraphs {
			if ContainsNodePosition(subgraph, node) {
				nodeAlreadyInSubgraphs = true
				break
			}
		}
		if !nodeAlreadyInSubgraphs {
			subgraph := riddle.GetConnectedSubgraph(node)
			subgraphs = append(subgraphs, subgraph)
		}
	}
	return subgraphs
}

func (riddle *Riddle) GetConnectedSubgraph(node *Node) []*Node {
	var connectedNodes = make(map[*Node]bool)
	var nodesToCheck = []*Node{node}
	for len(nodesToCheck) > 0 {
		var currentNode = nodesToCheck[0]
		nodesToCheck = nodesToCheck[1:]
		connectedNodes[currentNode] = true
		emptyAdjacentNodes := riddle.GetEmptyAdjacentNodes(currentNode)
		for _, adjacentNode := range emptyAdjacentNodes {
			if !connectedNodes[adjacentNode] && !HasOverlappingEdges(append(riddle.Edges, &LetterEdge{Node1: currentNode, Node2: adjacentNode})) {
				nodesToCheck = append(nodesToCheck, adjacentNode)
			}
		}
	}
	var connectedNodesSlice []*Node
	for node := range connectedNodes {
		connectedNodesSlice = append(connectedNodesSlice, node)
	}
	return connectedNodesSlice
}

func (riddle *Riddle) GetConnectedSubgraphSize(node *Node) int {
	return len(riddle.GetConnectedSubgraph(node))
}

func (riddle *Riddle) DoesNotOverlapWithEdges(node *Node, next *Node) bool {
	var newEdge = &LetterEdge{
		Node1: node,
		Node2: next,
	}
	return !HasOverlappingEdges(append(riddle.Edges, newEdge))
}

func (riddle *Riddle) GetEmptyAdjacentNodes(node *Node) []*Node {
	var allAdjacentNodes = riddle.GetAdjacentNodes(node, false)
	var emptyAdjacentNodes []*Node
	for _, adjacentNode := range allAdjacentNodes {
		if adjacentNode.isEmpty() {
			emptyAdjacentNodes = append(emptyAdjacentNodes, adjacentNode)
		}
	}
	return emptyAdjacentNodes
}

func (riddle *Riddle) getAdjacentNodesWithLetter(node *Node, letter byte, nodesToIgnore []*Node) []*Node {
	var allAdjacentNodes = riddle.GetAdjacentNodes(node, true)
	var matchingAdjacentNodes []*Node
	for _, adjacentNode := range allAdjacentNodes {
		for _, ignoredNode := range nodesToIgnore {
			if adjacentNode.Row == ignoredNode.Row && adjacentNode.Col == ignoredNode.Col {
				continue
			}
		}
		if adjacentNode.RiddleWord != nil && adjacentNode.RiddleWord.Word[adjacentNode.RiddleWordIndex] == letter {
			matchingAdjacentNodes = append(matchingAdjacentNodes, adjacentNode)
		}
	}
	return matchingAdjacentNodes
}

func (riddle *Riddle) GetAdjacentNodes(node *Node, ignoreEdges bool) []*Node {
	var row, col = node.Row, node.Col
	var adjacentNodes []*Node
	if riddle.NodeIsInBounds(row-1, col) {
		var potentialNode = riddle.GetNode(row-1, col)
		if ignoreEdges || riddle.DoesNotOverlapWithEdges(node, potentialNode) {
			adjacentNodes = append(adjacentNodes, potentialNode)
		}
	}
	if riddle.NodeIsInBounds(row+1, col) {
		var potentialNode = riddle.GetNode(row+1, col)
		if ignoreEdges || riddle.DoesNotOverlapWithEdges(node, potentialNode) {
			adjacentNodes = append(adjacentNodes, potentialNode)
		}
	}
	if riddle.NodeIsInBounds(row, col-1) {
		var potentialNode = riddle.GetNode(row, col-1)
		if ignoreEdges || riddle.DoesNotOverlapWithEdges(node, potentialNode) {
			adjacentNodes = append(adjacentNodes, potentialNode)
		}
	}
	if riddle.NodeIsInBounds(row, col+1) {
		var potentialNode = riddle.GetNode(row, col+1)
		if ignoreEdges || riddle.DoesNotOverlapWithEdges(node, potentialNode) {
			adjacentNodes = append(adjacentNodes, potentialNode)
		}
	}
	if riddle.NodeIsInBounds(row-1, col-1) {
		var potentialNode = riddle.GetNode(row-1, col-1)
		if ignoreEdges || riddle.DoesNotOverlapWithEdges(node, potentialNode) {
			adjacentNodes = append(adjacentNodes, potentialNode)
		}
	}
	if riddle.NodeIsInBounds(row-1, col+1) {
		var potentialNode = riddle.GetNode(row-1, col+1)
		if ignoreEdges || riddle.DoesNotOverlapWithEdges(node, potentialNode) {
			adjacentNodes = append(adjacentNodes, potentialNode)
		}
	}
	if riddle.NodeIsInBounds(row+1, col-1) {
		var potentialNode = riddle.GetNode(row+1, col-1)
		if ignoreEdges || riddle.DoesNotOverlapWithEdges(node, potentialNode) {
			adjacentNodes = append(adjacentNodes, potentialNode)
		}
	}
	if riddle.NodeIsInBounds(row+1, col+1) {
		var potentialNode = riddle.GetNode(row+1, col+1)
		if ignoreEdges || riddle.DoesNotOverlapWithEdges(node, potentialNode) {
			adjacentNodes = append(adjacentNodes, potentialNode)
		}
	}
	return adjacentNodes
}

func (riddle *Riddle) NodeCanBeFilled(riddleWord *RiddleWord, node *Node, comingFrom *Node, minimumRemainingSubgraphSizeForCurrentNode int) bool {
	// rules:
	// - node is empty
	// - does not create isolated nodes or unconnected subgraphs with fewer than 4 nodes
	//   (only relevant if those islands are empty)
	// - does not create overlapping edges with the word itself or other words (e.g. 1,1<->2,2 and 2,1<-->1,2 are not allowed at the same time)

	if !node.isEmpty() {
		return false
	}

	riddleCopy := riddle.Copy()
	riddleCopy.Nodes[node.Row*RiddleWidth+node.Col].RiddleWord = riddleWord

	if comingFrom != nil {
		riddleCopy.Edges = append(riddleCopy.Edges, &LetterEdge{
			Word:  riddleWord,
			Node1: comingFrom,
			Node2: node,
		})
	}

	emptyAdjacentNodes := riddleCopy.GetEmptyAdjacentNodes(riddleCopy.GetNode(node.Row, node.Col))

	if len(emptyAdjacentNodes) > 0 {
		// fmt.Printf("Checking subgraph size for node %d,%d\n", node.Row, node.Col)
		for _, adjacentNode := range emptyAdjacentNodes {
			subgraphSize := riddleCopy.GetConnectedSubgraphSize(adjacentNode)
			// fmt.Printf("Subgraph size: %d\n", subgraphSize)
			if subgraphSize < minimumRemainingSubgraphSizeForCurrentNode {
				return false
			}
		}
	}

	return true
}

func (riddle *Riddle) GetAvailableAdjacentNodes(row, col int, minimumRemainingSubgraphSize int) []*Node {
	node := riddle.GetNode(row, col)
	var availableAdjacentNodes []*Node
	adjacentNodes := riddle.GetEmptyAdjacentNodes(node)
	for _, adjacentNode := range adjacentNodes {
		if riddle.NodeCanBeFilled(node.RiddleWord, adjacentNode, node, minimumRemainingSubgraphSize) {
			availableAdjacentNodes = append(availableAdjacentNodes, adjacentNode)
		}
	}
	return availableAdjacentNodes
}

func (riddle *Riddle) GetEdgesForWord(word *RiddleWord) []*LetterEdge {
	var edges []*LetterEdge
	for _, edge := range riddle.Edges {
		if edge.Word.Word == word.Word {
			edges = append(edges, edge)
		}
	}
	return edges
}

func (riddle *Riddle) CheckForAmbiguity() (bool, [][]*LetterEdge) {
	// for each word, check if it has more than one way to be filled on the board
	for _, word := range riddle.Words {
		if !word.Used && !word.IsSuperSolution {
			continue
		}
		var wordNodes = []*Node{}
		for _, node := range riddle.Nodes {
			if node.RiddleWord != nil && node.RiddleWord.Word == word.Word {
				wordNodes = append(wordNodes, node)
			}
		}
		var problematicSolutions = [][]*LetterEdge{}
		// start with first letter and find all nodes that have this letter
		var startingNodes []*Node
		for _, node := range riddle.Nodes {
			if node.RiddleWord != nil && node.RiddleWord.Word[node.RiddleWordIndex] == word.Word[0] {
				startingNodes = append(startingNodes, node)
			}
		}
		// for each starting node, find all possible paths that would fill the word
		// only increase solutionCount if the path continues nodes outside of the actual word nodes
		// this allows for words to be filled in multiple ways but not in a way that would create ambiguity
		for _, startingNode := range startingNodes {
			var paths = getPossiblePaths(riddle, word, startingNode, 0, []*Node{})
			for _, path := range paths {
				if len(path) < len(word.Word)-1 {
					continue
				}
				// check if path continues outside of the word nodes
				for _, edge := range path {
					// check if edge.Node1 or edge.Node2 is not in wordNodes
					if !ContainsNodePosition(wordNodes, edge.Node1) || !ContainsNodePosition(wordNodes, edge.Node2) {
						problematicSolutions = append(problematicSolutions, path)
						break
					}
				}
			}
		}
		if len(problematicSolutions) > 0 {
			return true, problematicSolutions
		}
	}
	return false, nil
}

func getPossiblePaths(riddle *Riddle, word *RiddleWord, node *Node, index int, nodesToIgnore []*Node) [][]*LetterEdge {
	if index+1 == len(word.Word) {
		return [][]*LetterEdge{}
	}
	nodesToIgnore = append(nodesToIgnore, node)
	var nextLetter = word.Word[index+1]
	var nextNodes = riddle.getAdjacentNodesWithLetter(node, nextLetter, nodesToIgnore)
	var paths = [][]*LetterEdge{}
	for _, nextNode := range nextNodes {
		pathToNextNode := []*LetterEdge{{
			Word:  word,
			Node1: node,
			Node2: nextNode,
		}}
		followingPaths := getPossiblePaths(riddle, word, nextNode, index+1, nodesToIgnore)
		if len(followingPaths) == 0 {
			paths = append(paths, pathToNextNode)
		} else {
			for _, followingPath := range followingPaths {
				paths = append(paths, append(pathToNextNode, followingPath...))
			}
		}
	}
	return paths
}

func (riddle *Riddle) Render(debugOnly bool) {
	if debugOnly && logrus.GetLevel() != logrus.DebugLevel {
		return
	}
	for i := 0; i < RiddleHeight; i++ {
		for j := 0; j < RiddleWidth; j++ {
			var node = riddle.GetNode(i, j)
			if node.RiddleWord == nil {
				fmt.Print(" ")
			} else {
				if node.RiddleWordIndex == -1 {
					fmt.Print(node.RiddleWord.Color + "?" + colors.Reset)
				} else {
					fmt.Print(node.RiddleWord.Color + string(node.RiddleWord.Word[node.RiddleWordIndex]) + colors.Reset)
				}
			}
			fmt.Print("|") // Add vertical grid line after each cell
		}
		fmt.Println()
		for j := 0; j < RiddleWidth; j++ {
			fmt.Print("-") // Add horizontal grid line below each cell
			fmt.Print("+") // Add intersection grid line
		}
		fmt.Println()
	}
	for _, word := range riddle.Words {
		if !word.Used {
			continue
		}
		fmt.Printf("Word: %s\n", word.Word)
		var edges = riddle.GetEdgesForWord(word)
		// sort edges by word index
		sort.Slice(edges, func(i, j int) bool {
			return edges[i].Node1.RiddleWordIndex < edges[j].Node1.RiddleWordIndex
		})
		for _, edge := range edges {
			fmt.Printf("Edge: %d,%d -> %d,%d\n", edge.Node1.Row, edge.Node1.Col, edge.Node2.Row, edge.Node2.Col)
		}
	}
}

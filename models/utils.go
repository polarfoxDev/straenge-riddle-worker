package models

func HasOverlappingEdges(edges []*LetterEdge) bool {
	for i, edge1 := range edges {
		for j, edge2 := range edges {
			if i != j && EdgesCross(edge1, edge2) {
				return true
			}
		}
	}
	return false
}

func EdgesCross(edge1, edge2 *LetterEdge) bool {
	// edges cross if they would cross drawn on the grid
	// e.g. 1,1<->2,2 and 2,1<-->1,2 are not allowed at the same time
	p1, q1 := edge1.Node1, edge1.Node2
	p2, q2 := edge2.Node1, edge2.Node2

	dir1 := GetDirection(p1, q1)
	dir2 := GetDirection(p2, q2)

	if dir1 == dir2 {
		return false
	}

	if dir1 == "horizontal" || dir2 == "horizontal" || dir1 == "vertical" || dir2 == "vertical" {
		return false
	}

	if dir1 == "diagonal-type2-b" {
		// p1, q1 = q1, p1
		p1 = q1
		dir1 = "diagonal-type2-a"
	}
	if dir1 == "diagonal-type1-b" {
		// p1, q1 = q1, p1
		p1 = q1
		dir1 = "diagonal-type1-a"
	}

	if dir2 == "diagonal-type2-b" {
		// p2, q2 = q2, p2
		p2 = q2
		dir2 = "diagonal-type2-a"
	}
	if dir2 == "diagonal-type1-b" {
		// p2, q2 = q2, p2
		p2 = q2
		dir2 = "diagonal-type1-a"
	}

	if dir1 == "diagonal-type1-a" {
		if dir2 == "diagonal-type2-a" && p1.Row+1 == p2.Row && p1.Col == p2.Col {
			return true
		}
	}

	if dir1 == "diagonal-type2-a" {
		if dir2 == "diagonal-type1-a" && p1.Row-1 == p2.Row && p1.Col == p2.Col {
			return true
		}
	}

	return false
}

func GetDirection(p, q *Node) string {
	if p.Row == q.Row {
		return "vertical"
	}
	if p.Col == q.Col {
		return "horizontal"
	}
	if p.Row < q.Row && p.Col < q.Col {
		return "diagonal-type1-a"
	}
	if p.Row < q.Row && p.Col > q.Col {
		return "diagonal-type2-b"
	}
	if p.Row > q.Row && p.Col < q.Col {
		return "diagonal-type2-a"
	}
	if p.Row > q.Row && p.Col > q.Col {
		return "diagonal-type1-b"
	}
	return "unknown"
}

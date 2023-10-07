package gasonn

import (
	"strconv"
)

type Asonn struct {
	Nodes []*Node
}

type Node struct {
	Value 		interface{}
	Connections	[]*Node
}

func NewNode(value interface{}) Node {
	return Node{Value: value}
}

func BuildAgds(x [][]string, y []string) (Asonn){
	asonn := Asonn{}
	for _, value := range x[0] {
		newNode := NewNode(value)
		asonn.Nodes = append(asonn.Nodes, &newNode)
	}
	var classNodes []*Node
	for i, row := range x {
		if i == 0 || y[i] == "" {
			continue // Feature names in first row, skip data with no class
		}
		objectNode := NewNode("O" + strconv.Itoa(i))
		for j, value := range row {
			newNode, reused := tryToReuseNode(value, asonn.Nodes, j)
			addConnection(newNode, &objectNode)
			if !reused {
				addConnection(newNode, asonn.Nodes[j])
				asonn.Nodes = append(asonn.Nodes, newNode)
			}
		}
		classNode, reused := tryToReuseClassNode(y[i], classNodes)
		addConnection(classNode, &objectNode)
		if !reused {
			asonn.Nodes = append(asonn.Nodes, &objectNode)
			classNodes = append(classNodes, classNode)
		}
		asonn.Nodes = append(asonn.Nodes, &objectNode)
	}
	asonn.Nodes = append(asonn.Nodes, classNodes...)
	return asonn
}

func addConnection(first *Node, second *Node) {
	first.Connections = append(first.Connections, second)
	second.Connections = append(second.Connections, first)
}

func areConnected(first *Node, second *Node) (bool) {
	for _, node := range first.Connections {
		if node == second {
			return true
		}
	}

	return false
}

func tryToReuseNode(value string, nodes []*Node, i int) (*Node, bool) {
	for _, node := range nodes {
		if node.Value == value && areConnected(node, nodes[i]) {
			return node, true
		}
	}
	newNode := NewNode(value)
	return &newNode, false
}

func tryToReuseClassNode(value string, nodes []*Node) (*Node, bool) {
	for _, node := range nodes {
		if node.Value == value {
			return node, true
		}
	}
	newNode := NewNode(value)
	return &newNode, false
}

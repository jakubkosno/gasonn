package gasonn

import (
	"strconv"
)

type Asonn struct {
	Nodes []*Node
}

type Node struct {
	Value 		interface{}
	Connections	[]Connection
}

func NewNode(value interface{}) Node {
	return Node{Value: value}
}

type Connection struct {
	Node	*Node
	Weight	float32
}

func NewConnection(node *Node, weight float32) Connection {
	return Connection{Node: node, Weight: weight}
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
		for j, strValue := range row {
			value := convertToCorrectType(strValue)
			newNode, reused := tryToReuseNode(value, asonn.Nodes, j)
			addConnection(newNode, &objectNode)
			if !reused {
				addConnection(newNode, asonn.Nodes[j])
				asonn.Nodes = append(asonn.Nodes, newNode)
			}
		}
		value := convertToCorrectType(y[i])
		classNode, reused := tryToReuseClassNode(value, classNodes)
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
	first.Connections = append(first.Connections, NewConnection(second, 1))
	second.Connections = append(second.Connections, NewConnection(first, 1))
}

func areConnected(first *Node, second *Node) (bool) {
	for _, connection := range first.Connections {
		if connection.Node == second {
			return true
		}
	}

	return false
}

func tryToReuseNode(value interface{}, nodes []*Node, i int) (*Node, bool) {
	for _, node := range nodes {
		if node.Value == value && areConnected(node, nodes[i]) {
			return node, true
		}
	}
	newNode := NewNode(value)
	return &newNode, false
}

func tryToReuseClassNode(value interface{}, nodes []*Node) (*Node, bool) {
	for _, node := range nodes {
		if node.Value == value {
			return node, true
		}
	}
	newNode := NewNode(value)
	return &newNode, false
}

func convertToCorrectType(value string) interface{} {
	if intValue, err := strconv.Atoi(value); err == nil {
		return intValue
	}
	if floatValue, err := strconv.ParseFloat(value, 32); err == nil {
		return floatValue
	}
	return value
}

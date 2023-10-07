package gasonn

import (
	"sort"
	"strconv"
)

type Asonn struct {
	Nodes []*Node
}

type Node struct {
	Value       interface{}
	Connections ConnectionSlice
	Type        string
}

const (
	Value   = "Value"
	Object  = "Object"
	Feature = "Feature"
	Class   = "Class"
)

func NewNode(value interface{}, nodeType string) Node {
	return Node{Value: value, Type: nodeType}
}

type ConnectionSlice []Connection

// Implement Len from sort.Interface for ConnectionSlice
func (connectionSlice ConnectionSlice) Len() int {
	return len(connectionSlice)
}

// Implement Swap from sort.Interface for ConnectionSlice
func (connectionSlice ConnectionSlice) Swap(i, j int) {
	connectionSlice[i], connectionSlice[j] = connectionSlice[j], connectionSlice[i]
}

// Implement Less from sort.Interface for ConnectionSlice
func (connectionSlice ConnectionSlice) Less(i, j int) bool {
	firstVal, firstOk := connectionSlice[i].Node.Value.(float32)
	secondVal, secondOk := connectionSlice[j].Node.Value.(float32)
	if firstOk && secondOk {
		return firstVal > secondVal
	}
	return true // For non numeric types order doesn't matter
}

func (node Node) sortConnections() {
	sort.Sort(node.Connections)
}

type Connection struct {
	Node   *Node
	Weight float32
}

func NewConnection(node *Node, weight float32) Connection {
	return Connection{Node: node, Weight: weight}
}

func BuildAgds(x [][]string, y []string) Asonn {
	asonn := Asonn{}
	for _, value := range x[0] {
		newNode := NewNode(value, Feature)
		asonn.Nodes = append(asonn.Nodes, &newNode)
	}
	var classNodes []*Node
	for i, row := range x {
		if i == 0 || y[i] == "" {
			continue // Feature names in first row, skip data with no class
		}
		objectNode := NewNode("O"+strconv.Itoa(i), Object)
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
	for _, node := range asonn.Nodes {
		if node.Type == Feature {
			node.sortConnections()
		}
	}
	return asonn
}

func addConnection(first *Node, second *Node) {
	first.Connections = append(first.Connections, NewConnection(second, 1))
	second.Connections = append(second.Connections, NewConnection(first, 1))
}

func areConnected(first *Node, second *Node) bool {
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
	newNode := NewNode(value, Value)
	return &newNode, false
}

func tryToReuseClassNode(value interface{}, nodes []*Node) (*Node, bool) {
	for _, node := range nodes {
		if node.Value == value {
			return node, true
		}
	}
	newNode := NewNode(value, Class)
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

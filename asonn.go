package gasonn

import (
	"sort"
	"strconv"
)

type Asonn struct {
	Nodes []*Node
}

func (asonn Asonn) addAsimAndAdefConnections() {
	for i := range asonn.Nodes {
		if asonn.Nodes[i].Type == Feature {
			// Check if value is numeric
			maxVal, maxOk := asonn.Nodes[i].Connections[len(asonn.Nodes[i].Connections)-1].Node.Value.(float64)
			minVal, minOk := asonn.Nodes[i].Connections[0].Node.Value.(float64)
			if minOk && maxOk {
				valRange := maxVal - minVal
				for j := range asonn.Nodes[i].Connections {
					if j == len(asonn.Nodes[i].Connections)-1 {
						break
					}
					weight := (valRange - (asonn.Nodes[i].Connections[j+1].Node.Value.(float64) - asonn.Nodes[i].Connections[j].Node.Value.(float64))) / valRange
					addConnection(asonn.Nodes[i].Connections[j].Node, asonn.Nodes[i].Connections[j+1].Node, weight)
				}
			}
			if asonn.Nodes[i].Type == Object {
				denominator := 0.0
				for j := range asonn.Nodes[i].Connections {
					if asonn.Nodes[i].Connections[j].Node.Type == Value {
						denominator += float64(countObjectConnectionsFromClass(asonn.Nodes[i].Connections[j].Node, getClassOfObject(asonn.Nodes[i]))) / float64(countObjectConnections(asonn.Nodes[i].Connections[j].Node))
					}
				}
				for j := range asonn.Nodes[i].Connections {
					if asonn.Nodes[i].Connections[j].Node.Type == Value {
						weight := (float64(countObjectConnectionsFromClass(asonn.Nodes[i].Connections[j].Node, getClassOfObject(asonn.Nodes[i]))) / float64(countObjectConnections(asonn.Nodes[i].Connections[j].Node))) / denominator
						asonn.Nodes[i].Connections[j].Weight = weight
					}
				}
			}
		}
	}
}

func countObjectConnections(node *Node) int {
	counter := 0
	for i := range node.Connections {
		if node.Connections[i].Node.Type == Object {
			counter += 1
		}
	}
	return counter
}

func countObjectConnectionsFromClass(node *Node, class string) int {
	counter := 0
	for i := range node.Connections {
		if node.Connections[i].Node.Type == Object {
			for j := range node.Connections[i].Node.Connections {
				if node.Connections[i].Node.Connections[j].Node.Type == Class && node.Connections[i].Node.Connections[j].Node.Value == class {
					counter += 1
				}
			}
		}
	}
	return counter
}

func getClassOfObject(node *Node) string {
	for i := range node.Connections {
		if node.Connections[i].Node.Type == Class {
			return node.Connections[i].Node.Value.(string)
		}
	}
	return ""
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
	firstVal, firstOk := connectionSlice[i].Node.Value.(float64)
	secondVal, secondOk := connectionSlice[j].Node.Value.(float64)
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
	Weight float64
}

func NewConnection(node *Node, weight float64) Connection {
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
			addConnection(newNode, &objectNode, 1)
			if !reused {
				addConnection(newNode, asonn.Nodes[j], 1)
				asonn.Nodes = append(asonn.Nodes, newNode)
			}
		}
		value := convertToCorrectType(y[i])
		classNode, reused := tryToReuseClassNode(value, classNodes)
		addConnection(classNode, &objectNode, 1)
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
	asonn.addAsimAndAdefConnections()
	return asonn
}

func addConnection(first *Node, second *Node, weight float64) {
	first.Connections = append(first.Connections, NewConnection(second, weight))
	second.Connections = append(second.Connections, NewConnection(first, weight))
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
	if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
		return floatValue
	}
	return value
}

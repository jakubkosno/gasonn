package gasonn

import (
	"errors"
	"sort"
	"strconv"
)

type Asonn struct {
	Nodes []*Node
}

func BuildAsonn(x [][]string, y []string) Asonn {
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
	asonn.addCombinations()
	return asonn
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

func (asonn Asonn) addCombinations() {
	i := 0
	for asonn.countNotRepresentedObjects() > 0 {
		combinationSeed := asonn.getMostOutCorrelatedObjectNode()
		combinationNode := NewNode("C" + strconv.Itoa(i), Combination)
		for j := range combinationSeed.Connections {
			if combinationSeed.Connections[j].Node.Type == Value {
				var valRange  []interface{}
				valRange = append(valRange, combinationSeed.Connections[j].Node.Value)
				rangeNode := NewNode(valRange, Range)
				addConnection(&combinationNode, &rangeNode, 1)
				asonn.Nodes = append(asonn.Nodes, &rangeNode)
			} else if combinationSeed.Connections[j].Node.Type == Class {
				addConnection(&combinationNode, combinationSeed.Connections[j].Node, 1)
			}
		}
		asonn.Nodes = append(asonn.Nodes, &combinationNode)
		i++
	}
}

func (asonn Asonn) countNotRepresentedObjects() int {
	counter := 0
	for i := range asonn.Nodes {
		if asonn.Nodes[i].Type != Object {
			break
		}
		isConnected := false
		for j := range asonn.Nodes[i].Connections {
			if asonn.Nodes[i].Connections[j].Node.Type == Combination {
				isConnected = true
				break
			}
		}
		if !isConnected {
			counter++
		}
	}
	return counter
}

func (asonn Asonn) getMostOutCorrelatedObjectNode() *Node {
	maxIndex := 0
	var maxOutCorrelations []int
	changed := false
	for i := range asonn.Nodes {
		for j := range asonn.Nodes[i].Connections {
			if asonn.Nodes[i].Connections[j].Node.Type == Combination {
				continue
			}
		}
		if asonn.Nodes[i].Type == Object {
			outCorrelations, err := asonn.calculateObjectNodeOutCorrelation(asonn.Nodes[i])
			if err == nil {
				maxOutCorrelations, changed = getBiggerCorrelation(maxOutCorrelations, outCorrelations)
				if changed {
					maxIndex = i
				}
			}
			changed = false
		}
	}
	return asonn.Nodes[maxIndex]
}

func (asonn Asonn) calculateObjectNodeOutCorrelation(node *Node) ([]int, error) {
	if node.Type != Object {
		return nil, errors.New("Not an object node")
	}
	length := node.countValueConnections() + 1
	var outCorrelations []int
	outCorrelations = append(outCorrelations, make([]int, length)...)
	for i := range asonn.Nodes {
		if asonn.Nodes[i].Type == Object {
			commmonFeatures, err := countCommonFeatures(node, asonn.Nodes[i])
			if err == nil {
				outCorrelations[length - commmonFeatures - 1]++
			}
		}
	}
	return outCorrelations, nil
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

func getBiggerCorrelation(currentMax []int, pretender []int) ([]int, bool) {
	for i := range currentMax {
		if currentMax[i] > pretender[i] {
			return currentMax, false
		} else if pretender[i] > currentMax[i] {
			return pretender, true
		}
	}
	return currentMax, false
}

type Node struct {
	Value       interface{}
	Connections ConnectionSlice
	Type        string
}

const (
	Value   	= "Value"
	Object  	= "Object"
	Feature 	= "Feature"
	Class   	= "Class"
	Range   	= "Range"
	Combination = "Combination"
)

func NewNode(value interface{}, nodeType string) Node {
	return Node{Value: value, Type: nodeType}
}

func (node Node) countValueConnections() int {
	counter := 0
	for i := range node.Connections {
		if node.Connections[i].Node.Type == Value {
			counter++
		}
	}
	return counter
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

func countCommonFeatures(first *Node, second *Node) (int, error) {
	if first.Type != Object || second.Type != Object {
		return 0, errors.New("Not an object node")
	}
	counter := 0
	for i := range first.Connections {
		if first.Connections[i].Node.Type == Value {
			for j := range second.Connections {
				if second.Connections[j].Node.Type == Value && first.Connections[i].Node.Value == second.Connections[j].Node.Value {
					counter++
				}
			}
		}
	}
	return counter, nil
}

func getFeatureType(node *Node) (interface{}, error) {
	if node.Type != Value {
		return nil, errors.New("Not a value node")
	}
	for i := range node.Connections {
		if node.Connections[i].Node.Type == Feature {
			return node.Connections[i].Node.Value, nil
		}
	}
	return nil, errors.New("No connection to feature node")
}

package gasonn

import (
	"errors"
	"math"
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
		value := y[i]
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
	asonn.removeValueAndObjectNodes()
	return asonn
}

func (asonn Asonn) Predict(test [][]string) []float64 {
	var results []float64
	features := test[0]
	values := test[1:]
	for i := range values {
		for j := range values[i] {
			val := convertToCorrectType(values[i][j])
			asonn.activate(val, features[j])
		}
	}
	return results
}

func (asonn Asonn) activate(value interface{}, feature string) {
	for i := range asonn.Nodes {
		if asonn.Nodes[i].Type == Feature && asonn.Nodes[i].Value == feature {
			for j := range asonn.Nodes[i].Connections {
				if asonn.Nodes[i].Connections[j].Node.Type == Range {
					asonn.Nodes[i].Connections[j].Node.activate(value)
				}
			}
		}
	}
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

func (asonn *Asonn) addCombinations() {
	i := 0
	for asonn.countNotRepresentedObjects() > 0 {
		combinationSeed := asonn.getMostOutCorrelatedObjectNode()
		combinationNode := NewNode("C"+strconv.Itoa(i), Combination)
		addConnection(combinationSeed, &combinationNode, 1)
		for j := range combinationSeed.Connections {
			if combinationSeed.Connections[j].Node.Type == Value {
				var valRange []interface{}
				valRange = append(valRange, combinationSeed.Connections[j].Node.Value)
				rangeNode := NewNode(valRange, Range)
				addConnection(&combinationNode, &rangeNode, 1)
				addConnection(&rangeNode, combinationSeed.Connections[j].Node, 1)
				featureNode, _ := getFeatureConnection(combinationSeed.Connections[j].Node)
				addConnection(&rangeNode, featureNode, 1)
				asonn.Nodes = append(asonn.Nodes, &rangeNode)
			} else if combinationSeed.Connections[j].Node.Type == Class {
				addConnection(&combinationNode, combinationSeed.Connections[j].Node, 1)
			}
		}
		asonn.Nodes = append(asonn.Nodes, &combinationNode)
		asonn.expandCombination(&combinationNode)
		i++
	}
}

func (asonn Asonn) countNotRepresentedObjects() int {
	counter := 0
	for i := range asonn.Nodes {
		if asonn.Nodes[i].Type == Object {
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
	}
	return counter
}

func (asonn Asonn) getMostOutCorrelatedObjectNode() *Node {
	maxIndex := asonn.getFirstNotRepresentedObjectIndex()
	length := asonn.Nodes[maxIndex].countValueConnections() + 1
	var maxOutCorrelations []int
	maxOutCorrelations = append(maxOutCorrelations, make([]int, length)...)
	changed := false
	for i := range asonn.Nodes {
		represented := false
		for j := range asonn.Nodes[i].Connections {
			if asonn.Nodes[i].Connections[j].Node.Type == Combination {
				represented = true
				break
			}
		}
		if represented {
			continue
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
				outCorrelations[length-commmonFeatures-1]++
			}
		}
	}
	return outCorrelations, nil
}

func (asonn Asonn) expandCombination(node *Node) error {
	if node.Type != Combination {
		return errors.New("Tried to expand non combination node")
	}
	shouldContinue := true
	for shouldContinue {
		weedlessExtensionNotDone := true
		for weedlessExtensionNotDone {
			weedlessExtensionNotDone = false
			possibleExpansions, err := asonn.getPossibleExpansions(node)
			if err != nil {
				return err
			}
			for i := range possibleExpansions {
				canExpand := true
				if possibleExpansions[i].Smaller != nil {
					for j := range possibleExpansions[i].Range.Value.([]interface{}) {
						if possibleExpansions[i].Range.Value.([]interface{})[j].(float64) == possibleExpansions[i].Smaller.Value {
							possibleExpansions[i].Range.Value = append(possibleExpansions[i].Range.Value.([]interface{})[:j], possibleExpansions[i].Range.Value.([]interface{})[j+1:]...)
						}
					}
					for j := range possibleExpansions[i].Smaller.Connections {
						if possibleExpansions[i].Smaller.Connections[j].Node.Type == Object {
							if getClassOfObject(possibleExpansions[i].Smaller.Connections[j].Node) != getClassOfObject(node) {
								canExpand = false
								break
							}
						}
					}
					if canExpand {
						possibleExpansions[i].Range.Value = append(possibleExpansions[i].Range.Value.([]interface{}), possibleExpansions[i].Smaller.Value)
						addConnection(possibleExpansions[i].Range, possibleExpansions[i].Smaller, 1)
						weedlessExtensionNotDone = true
					}
				}
				canExpand = true
				if possibleExpansions[i].Bigger != nil {
					for j := range possibleExpansions[i].Range.Value.([]interface{}) {
						if possibleExpansions[i].Range.Value.([]interface{})[j].(float64) == possibleExpansions[i].Bigger.Value {
							possibleExpansions[i].Range.Value = append(possibleExpansions[i].Range.Value.([]interface{})[:j], possibleExpansions[i].Range.Value.([]interface{})[j+1:]...)
						}
					}
					for j := range possibleExpansions[i].Bigger.Connections {
						if possibleExpansions[i].Bigger.Connections[j].Node.Type == Object {
							if getClassOfObject(possibleExpansions[i].Bigger.Connections[j].Node) != getClassOfObject(node) {
								canExpand = false
								break
							}
						}
					}
					if canExpand {
						possibleExpansions[i].Range.Value = append(possibleExpansions[i].Range.Value.([]interface{}), possibleExpansions[i].Bigger.Value)
						addConnection(possibleExpansions[i].Range, possibleExpansions[i].Bigger, 1)
						weedlessExtensionNotDone = true
					}
				}
			}
			if weedlessExtensionNotDone {
				asonn.addRepresentedObjects(node)
			}
		}
		possibleExpansions, err := asonn.getPossibleExpansions(node)
		if err != nil {
			return err
		}
		var nodeToAdd *Node
		var rangeToExpand *Node
		nodeToAdd = nil
		maxCoeff := math.Inf(-1)
		shouldContinue = false
		for i := range possibleExpansions {
			if possibleExpansions[i].Smaller != nil {
				coeff, _ := asonn.calculateMinusCoefficient(possibleExpansions[i].Range)
				if maxCoeff < coeff {
					nodeToAdd = possibleExpansions[i].Smaller
					rangeToExpand = possibleExpansions[i].Range
				}
			}
			if possibleExpansions[i].Bigger != nil {
				coeff, _ := asonn.calculatePlusCoefficient(possibleExpansions[i].Range)
				if maxCoeff < coeff {
					nodeToAdd = possibleExpansions[i].Bigger
					rangeToExpand = possibleExpansions[i].Range
				}
			}
		}
		if nodeToAdd != nil && rangeToExpand != nil {
			if shouldContinue = asonn.expandWith(nodeToAdd, rangeToExpand, node); shouldContinue {
				asonn.addRepresentedObjects(node)
			}
		}
	}
	for i := range node.Connections {
		if node.Connections[i].Node.Type == Range {
			reduceRange(node.Connections[i].Node)
		}
	}
	return nil
}

func (asonn Asonn) getPossibleExpansions(node *Node) ([]Expansions, error) {
	var expansionOptions []Expansions
	for i := range node.Connections {
		if node.Connections[i].Node.Type == Range {
			expansions := Expansions{Range: node.Connections[i].Node, Smaller: nil, Bigger: nil}
			for j := range node.Connections[i].Node.Connections {
				valRange, ok := node.Connections[i].Node.Value.([]interface{})
				if !ok {
					return nil, errors.New("Range doesn't store []interface{}")
				}
				if node.Connections[i].Node.Connections[j].Node.Type == Value {
					minVal, maxVal, err := minMax(valRange)
					if err != nil {
						return nil, err
					}
					if node.Connections[i].Node.Connections[j].Node.Value == minVal {
						nextSmaller, reachedMin, err := getNextSmaller(node.Connections[i].Node.Connections[j].Node)
						if err != nil {
							return nil, err
						}
						if !reachedMin {
							expansions.Smaller = nextSmaller
						}
					} else if node.Connections[i].Node.Connections[j].Node.Value == maxVal {
						nextBigger, reachedMax, err := getNextBigger(node.Connections[i].Node.Connections[j].Node)
						if err != nil {
							return nil, err
						}
						if !reachedMax {
							expansions.Bigger = nextBigger
						}
					}
				}
			}
			expansionOptions = append(expansionOptions, expansions)
		}
	}
	return expansionOptions, nil
}

func (asonn Asonn) addRepresentedObjects(node *Node) {
	var represented []*Node
	var connected []*Node
	for i := range node.Connections {
		if node.Connections[i].Node.Type == Object {
			represented = append(represented, node.Connections[i].Node)
		} else if node.Connections[i].Node.Type == Range {
			for j := range node.Connections[i].Node.Connections {
				if node.Connections[i].Node.Connections[j].Node.Type == Value {
					for k := range node.Connections[i].Node.Connections[j].Node.Connections {
						if node.Connections[i].Node.Connections[j].Node.Connections[k].Node.Type == Object {
							connected = append(connected, node.Connections[i].Node.Connections[j].Node.Connections[k].Node)
						}
					}
				}
			}
		}
	}
	features := represented[0].countValueConnections()
	connectionsMap := make(map[*Node]int)
	for _, connectedNode := range connected {
		connectionsMap[connectedNode]++
	}
	for connectedNode, count := range connectionsMap {
		canAdd := true
		if count == features {
			for i := range connected {
				if connected[i] == connectedNode {
					canAdd = false
					break
				}
				if canAdd {
					addConnection(node, connectedNode, 1)
				}
			}
		}
	}
}

func (asonn Asonn) calculateMinusCoefficient(node *Node) (float64, error) {
	if node.Type != Range {
		return 0.0, errors.New("Not a range node")
	}
	var smallerValues []*Node
	for i := range node.Connections {
		if node.Connections[i].Node.Type == Feature {
			for j := range node.Connections[i].Node.Connections {
				nodeVal, _ := convertToFloat64(node.Connections[i].Node.Connections[j].Node.Value)
				rangeMin, _, _ := minMax(node.Value.([]interface{}))
				rangeMinFloat, _ := convertToFloat64(rangeMin)
				if node.Connections[i].Node.Connections[j].Node.Type == Value && nodeVal < rangeMinFloat {
					smallerValues = append(smallerValues, node.Connections[i].Node.Connections[j].Node)
				}
			}
		}
	}
	coefficient := 0.0
	for i := range smallerValues {
		coefficient += asonn.calculate_7_16(smallerValues[i], node)
	}
	return coefficient, nil
}

func (asonn Asonn) calculatePlusCoefficient(node *Node) (float64, error) {
	if node.Type != Range {
		return 0.0, errors.New("Not a range node")
	}
	var biggerValues []*Node
	for i := range node.Connections {
		if node.Connections[i].Node.Type == Feature {
			for j := range node.Connections[i].Node.Connections {
				nodeVal, _ := convertToFloat64(node.Connections[i].Node.Connections[j].Node.Value)
				_, rangeMax, _ := minMax(node.Value.([]interface{}))
				rangeMaxFloat, _ := convertToFloat64(rangeMax)
				if node.Connections[i].Node.Connections[j].Node.Type == Value && nodeVal > rangeMaxFloat {
					biggerValues = append(biggerValues, node.Connections[i].Node.Connections[j].Node)
				}
			}
		}
	}
	coefficient := 0.0
	for i := range biggerValues {
		coefficient += asonn.calculate_7_17(biggerValues[i], node)
	}
	return coefficient, nil
}

func (asonn Asonn) expandWith(node *Node, rangeNode *Node, combinationNode *Node) bool {
	var objectWeeds []*Node
	for i := range node.Connections {
		if node.Connections[i].Node.Type == Object {
			if getClassOfObject(node.Connections[i].Node) != getClassOfObject(combinationNode) {
				objectWeeds = append(objectWeeds, node.Connections[i].Node)
			}
		}
	}
	canAdd := true
	for i := range objectWeeds {
		counter := 0.0
		for j := range objectWeeds[i].Connections {
			var feature interface{}
			if objectWeeds[i].Connections[j].Node.Type == Value {
				feature, _ = getFeatureType(objectWeeds[i].Connections[j].Node)
			} else {
				continue
			}
			for k := range combinationNode.Connections {
				if combinationNode.Connections[k].Node.Type == Range {
					if rangeFeature, _ := getFeatureType(combinationNode.Connections[k].Node); rangeFeature == feature {
						rangeMin, rangeMax, _ := minMax(combinationNode.Connections[k].Node.Value.([]interface{}))
						rangeMinFloat, _ := convertToFloat64(rangeMin)
						rangeMaxFloat, _ := convertToFloat64(rangeMax)
						if val, _ := convertToFloat64(objectWeeds[i].Connections[j].Node.Value); val <= rangeMaxFloat && val >= rangeMinFloat {
							counter += 1
							if counter == asonn.getFeaturesNumber()-1 {
								canAdd = false
							}
						}
					}
				}
			}
		}
	}
	if canAdd {
		rangeNode.Value = append(rangeNode.Value.([]interface{}), node.Value)
		addConnection(rangeNode, node, 1)
		return true
	}
	return false
}

func (asonn *Asonn) removeValueAndObjectNodes() {
	var filtered []*Node
	for i := range asonn.Nodes {
		if asonn.Nodes[i].Type != Value && asonn.Nodes[i].Type != Object {
			filtered = append(filtered, asonn.Nodes[i])
		}
	}
	asonn.Nodes = filtered
}

func (asonn Asonn) calculate_7_16(node *Node, rangeNode *Node) float64 {
	return asonn.calculate_7_18(node, rangeNode) *
		(asonn.calculate_7_20(node) - asonn.calculate_7_21(node, rangeNode))
}

func (asonn Asonn) calculate_7_17(node *Node, rangeNode *Node) float64 {
	return asonn.calculate_7_19(node, rangeNode) *
		(asonn.calculate_7_20(node) - asonn.calculate_7_21(node, rangeNode))
}

func (asonn Asonn) calculate_7_18(node *Node, rangeNode *Node) float64 {
	nodeValue, _ := convertToFloat64(node.Value)
	rangeMin, _, _ := minMax(rangeNode.Value.([]interface{}))
	minVal, _ := convertToFloat64(rangeMin)
	featureRange, _ := getFeatureRange(rangeNode)
	return math.Pow(1-(nodeValue-minVal)/featureRange, 2)
}

func (asonn Asonn) calculate_7_19(node *Node, rangeNode *Node) float64 {
	nodeValue, _ := convertToFloat64(node.Value)
	_, rangeMax, _ := minMax(rangeNode.Value.([]interface{}))
	maxVal, _ := convertToFloat64(rangeMax)
	featureRange, _ := getFeatureRange(rangeNode)
	return math.Pow(1-(maxVal-nodeValue)/featureRange, 2)
}

func getFeatureRange(node *Node) (float64, error) {
	if node.Type != Value && node.Type != Range {
		return 0, errors.New("Not a value or range node")
	}
	for i := range node.Connections {
		if node.Connections[i].Node.Type == Feature {
			minVal, _ := convertToFloat64(node.Connections[i].Node.Connections[0].Node.Value)
			maxVal, _ := convertToFloat64(node.Connections[i].Node.Connections[0].Node.Value)
			for j := range node.Connections[i].Node.Connections {
				newVal, _ := convertToFloat64(node.Connections[i].Node.Connections[j].Node.Value)
				if newVal < minVal {
					minVal = newVal
				} else if newVal > maxVal {
					maxVal = newVal
				}
			}
			return maxVal - minVal, nil
		}
	}
	return 0, errors.New("Range not found")
}

func (asonn Asonn) calculate_7_20(node *Node) float64 {
	return math.Pow(1/(1+asonn.countCombinationConnections(node)), 2)
}

func (asonn Asonn) calculate_7_21(node *Node, rangeNode *Node) float64 {
	result := math.Pow((1+asonn.calculate_7_25(node, rangeNode))/asonn.getFeaturesNumber(), 1)
	return result
}

func (asonn Asonn) getFeaturesNumber() float64 {
	counter := 0.0
	for i := range asonn.Nodes {
		if asonn.Nodes[i].Type == Feature {
			counter += 1
		}
	}
	return counter
}

func (asonn Asonn) countCombinationConnections(node *Node) float64 {
	counter := 0.0
	for i := range node.Connections {
		if node.Connections[i].Node.Type == Combination {
			counter += 1
		}
	}
	return counter
}

func (asonn Asonn) calculate_7_25(node *Node, rangeNode *Node) float64 {
	counter := 0.0
	// node should be Object type
	// rangeNode should be Range type
	for i := range node.Connections {
		if node.Connections[i].Node.Type == Value {
			val, _ := convertToFloat64(node.Connections[i].Node.Value)
			minVal, maxVal, _ := minMax(rangeNode.Value.([]interface{}))
			minValFloat, _ := convertToFloat64(minVal)
			maxValFloat, _ := convertToFloat64(maxVal)
			if val <= maxValFloat && val >= minValFloat {
				counter += 1
			}
		}
	}
	return counter
}

func min(values []float64) float64 {
	minValue := values[0]
	for _, value := range values {
		if value < minValue {
			minValue = value
		}
	}
	return minValue
}

func max(values []float64) float64 {
	maxValue := values[0]
	for _, value := range values {
		if value > maxValue {
			maxValue = value
		}
	}
	return maxValue
}

func (asonn Asonn) getFirstNotRepresentedObjectIndex() int {
	for i := range asonn.Nodes {
		if asonn.Nodes[i].Type == Object {
			for j := range asonn.Nodes[i].Connections {
				if asonn.Nodes[i].Connections[j].Node.Type == Combination {
					break
				}
			}
			return i
		}
	}
	return -1
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
		if pretender[i] > currentMax[i] {
			return pretender, true
		}
	}
	return currentMax, false
}

func reduceRange(node *Node) {
	if node.Type == Range {
		min, max, err := minMax(node.Value.([]interface{}))
		if err == nil {
			node.Value = [2]interface{}{min, max}
		}
	}
}

func minMax(slice []interface{}) (interface{}, interface{}, error) {
	if len(slice) == 0 {
		return nil, nil, errors.New("Empty slice")
	}
	if !isNumeric(slice[0]) {
		return nil, nil, errors.New("Not a numeric type")
	}
	min, minErr := convertToFloat64(slice[0])
	max, maxErr := convertToFloat64(slice[0])
	if minErr != nil || maxErr != nil {
		return nil, nil, errors.New("Conversion unsuccessful")
	}
	for i := range slice {
		if val, _ := convertToFloat64(slice[i]); val < min {
			min = val
		}
		if val, _ := convertToFloat64(slice[i]); val > max {
			max = val
		}
	}
	return min, max, nil
}

func isNumeric(value interface{}) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return true // It's a numeric value
	default:
		return false // It's not a numeric value
	}
}

func convertToFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	default:
		return 0, errors.New("Unsupported type")
	}
}

func getNextSmaller(node *Node) (*Node, bool, error) {
	if node.Type != Value {
		return nil, false, errors.New("Not a value node")
	}
	for i := range node.Connections {
		if node.Connections[i].Node.Type == Value && node.Connections[i].Node.Value.(float64) < node.Value.(float64) {
			return node.Connections[i].Node, false, nil
		}
	}
	return nil, true, nil
}

func getNextBigger(node *Node) (*Node, bool, error) {
	if node.Type != Value {
		return nil, false, errors.New("Not a value node")
	}
	for i := range node.Connections {
		if node.Connections[i].Node.Type == Value && node.Connections[i].Node.Value.(float64) > node.Value.(float64) {
			return node.Connections[i].Node, false, nil
		}
	}
	return nil, true, nil
}

type Node struct {
	Value       interface{}
	Connections ConnectionSlice
	Type        string
	Activation  float64
}

const (
	Value       = "Value"
	Object      = "Object"
	Feature     = "Feature"
	Class       = "Class"
	Range       = "Range"
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

func (node Node) activate(value interface{}) {
	if node.Type == Range {
		node.Activation = node.getActivation(value)
	}
}

func (node Node) getActivation(value interface{}) float64 {
	activation := 0.0
	val, ok := value.(float64)
	if !ok {
		valInt := value.(int)
		val = float64(valInt)
	}
	if node.Type == Range {
		min, _ := node.Value.([2]interface{})[0].(float64)
		max, _ := node.Value.([2]interface{})[1].(float64)
		if val >= min && val <= max {
			activation = 1.0
		} else {
			activation = math.Pow(math.E, (1-math.Pow((2*val-max-min)/(max-min), 2))/2)
		}
	}
	return activation
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

type Expansions struct {
	Range   *Node
	Smaller *Node
	Bigger  *Node
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
			firstFeature, firstErr := getFeatureType(first.Connections[i].Node)
			for j := range second.Connections {
				secondFeature, secondErr := getFeatureType(second.Connections[j].Node)
				if !(firstErr == nil) || !(secondErr == nil) {
					break
				}
				if second.Connections[j].Node.Type == Value && first.Connections[i].Node.Value == second.Connections[j].Node.Value && firstFeature == secondFeature {
					counter++
				}
			}
		}
	}
	return counter, nil
}

func getFeatureType(node *Node) (interface{}, error) {
	if node.Type != Value && node.Type != Range {
		return nil, errors.New("Not a value or range node")
	}
	for i := range node.Connections {
		if node.Connections[i].Node.Type == Feature {
			return node.Connections[i].Node.Value, nil
		}
	}
	return nil, errors.New("No connection to feature node")
}

func getFeatureConnection(node *Node) (*Node, error) {
	if node.Type != Value {
		return nil, errors.New("Not a value node")
	}
	for i := range node.Connections {
		if node.Connections[i].Node.Type == Feature {
			return node.Connections[i].Node, nil
		}
	}
	return nil, errors.New("No connection to feature node")
}

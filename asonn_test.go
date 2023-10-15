package gasonn

import (
	"fmt"
	"sort"
	"testing"

	"github.com/jakubkosno/pmlb"
)

type buildAsonnTestData struct {
	datasetName string
	nodesNumber int
}

var buildAsonnTests = []buildAsonnTestData{
	{"iris", 627},
	{"ecoli", 2628},
	{"monk1", 3900},
}

func TestBuildAsonn(t *testing.T) {
	for _, testData := range buildAsonnTests {
		x, y, err := pmlb.FetchXYData(testData.datasetName)
		if err != nil {
			fmt.Println(err)
		}
		asonn := BuildAsonn(x, y)
		if len(asonn.Nodes) != testData.nodesNumber {
			t.Errorf("Created %d nodes instead of %d", len(asonn.Nodes), testData.nodesNumber)
		}
		for _, node := range asonn.Nodes {
			if node.Type == Feature {
				var numbers []float32
				for _, connection := range node.Connections {
					val, ok := connection.Node.Value.(float32)
					if ok {
						numbers = append(numbers, val)
					}
				}
				sortedNumbers := make([]float32, len(numbers))
				copy(sortedNumbers, numbers)
				sort.Sort(Float32Slice(sortedNumbers))
				if !areEqual(sortedNumbers, numbers) {
					t.Errorf("Connections not sorted")
				}
			}
			if node.Type == Value {
				for _, connection := range node.Connections {
					if connection.Weight < 0 || connection.Weight > 1 {
						t.Errorf("Incorrect connection weight")
					}
				}
			}
		}
	}
}

type Float32Slice []float32

// Implement Len from sort.Interface for Float32Slice
func (s Float32Slice) Len() int {
	return len(s)
}

// Implement Less from sort.Interface for Float32Slice
func (s Float32Slice) Less(i, j int) bool {
	return s[i] < s[j]
}

// Implement Swap from sort.Interface for Float32Slice
func (s Float32Slice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func areEqual(slice1, slice2 []float32) bool {
	if len(slice1) != len(slice2) {
		return false
	}
	for i := range slice1 {
		if slice1[i] != slice2[i] {
			return false
		}
	}
	return true
}

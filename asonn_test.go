package gasonn

import (
	"fmt"
	"github.com/jakubkosno/pmlb"
	"sort"
	"testing"
)

type buildAgdsTestData struct {
	datasetName string
	nodesNumber int
}

var buildAgdsTests = []buildAgdsTestData{
	buildAgdsTestData{"iris", 283},
	buildAgdsTestData{"car", 1763},
	buildAgdsTestData{"chess", 3309},
	buildAgdsTestData{"ecoli", 702},
	buildAgdsTestData{"monk1", 583},
}

func TestBuildAgds(t *testing.T) {
	for _, testData := range buildAgdsTests {
		x, y, err := pmlb.FetchXYData(testData.datasetName)
		if err != nil {
			fmt.Println(err)
		}
		agds := BuildAgds(x, y)
		if len(agds.Nodes) != testData.nodesNumber {
			t.Errorf("Created %d nodes instead of %d", len(agds.Nodes), testData.nodesNumber)
		}
		for _, node := range agds.Nodes {
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

package gasonn

import (
	"fmt"
	"testing"
	"github.com/jakubkosno/pmlb"
)

type buildAgdsTestData struct {
	datasetName string
	nodesNumber int
}

var buildAgdsTests = []buildAgdsTestData {
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
	}
}

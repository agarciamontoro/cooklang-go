package canonical_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/aquilax/cooklang-go"
)

type Result struct {
	Steps    [][]ResultStep    `json:"steps"`
	Metadata map[string]string `json:"metadata"`
}

type ResultStep struct {
	Type     string      `json:"type"`
	Value    string      `json:"value,omitempty"`
	Name     string      `json:"name,omitempty"`
	Quantity interface{} `json:"quantity,omitempty"` // it could be string, int or float
	Units    string      `json:"units,omitempty"`
}

type TestCase struct {
	Source string `json:"source"`
	Result Result `json:"result"`
}

type SpecTests struct {
	Version int                 `json:"version"`
	Tests   map[string]TestCase `json:"tests"`
}

const specFileName = "canonical.json"

func loadSpecs(fileName string) (*SpecTests, error) {
	var err error
	var jsonFile *os.File
	jsonFile, err = os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	b, _ := ioutil.ReadAll(jsonFile)

	var result *SpecTests
	err = json.Unmarshal(b, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func contains(s []string, searchterm string) bool {
	i := sort.SearchStrings(s, searchterm)
	return i < len(s) && s[i] == searchterm
}

type IndexedData struct {
	Idx  int
	Type string
	Data interface{}
}

func orderedIndexes(step cooklang.Step) []IndexedData {
	indexes := []IndexedData{}
	for _, i := range step.Ingredients {
		indexes = append(indexes, IndexedData{i.Idx, "ingredient", i})
	}
	for _, t := range step.Timers {
		indexes = append(indexes, IndexedData{t.Idx, "timer", t})
	}
	for _, c := range step.Cookware {
		indexes = append(indexes, IndexedData{c.Idx, "cookware", c})
	}

	sort.Slice(indexes, func(i, j int) bool { return indexes[i].Idx < indexes[j].Idx })
	return indexes
}

func toSpecResultStep(step cooklang.Step) ([]ResultStep, error) {
	indexes := orderedIndexes(step)

	// Only text
	if len(indexes) == 0 {
		if step.Directions != "" {
			return []ResultStep{
				{
					Type:  "text",
					Value: step.Directions,
				},
			}, nil
		} else {
			return []ResultStep{}, nil
		}
	}

	resultSteps := []ResultStep{}
	i := 0
	for _, data := range indexes {
		text := step.Directions[i:data.Idx]
		if text != "" {
			newStep := ResultStep{
				Type:  "text",
				Value: text,
			}
			resultSteps = append(resultSteps, newStep)
		}

		newStep := ResultStep{
			Type: data.Type,
		}
		switch data.Type {
		case "ingredient":
			i := data.Data.(cooklang.Ingredient)
			newStep.Name = i.Name
			newStep.Quantity = i.Amount.QuantityRaw
			newStep.Units = i.Amount.Unit
		case "timer":
			t := data.Data.(cooklang.Timer)
			newStep.Name = t.Name
			newStep.Quantity = t.Duration
			newStep.Units = t.Unit
		case "cookware":
			c := data.Data.(cooklang.Cookware)
			newStep.Name = c.Name
			newStep.Quantity = c.QuantityRaw
			// TODO: newStep.Units = c.Unit
		}

		resultSteps = append(resultSteps, newStep)
		i += len(newStep.Name)
	}

	return resultSteps, nil
}

func toSpecResult(original cooklang.Recipe) (Result, error) {
	var steps [][]ResultStep
	for _, s := range original.Steps {
		resultSteps, err := toSpecResultStep(s)
		if err != nil {
			return Result{}, fmt.Errorf("unable to convert original step %v into ResultStep: %w", s, err)
		}
		steps = append(steps, resultSteps)
	}

	return Result{
		Steps:    steps,
		Metadata: original.Metadata,
	}, nil
}

func compareResult(got *cooklang.Recipe, want Result) error {
	// To do check results
	return nil
}

func TestCanonical(t *testing.T) {
	specs, err := loadSpecs(specFileName)
	if err != nil {
		panic(err)
	}
	skipCases := []string{}
	sort.Strings(skipCases)
	for name, spec := range (*specs).Tests {
		name := name
		spec := spec
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if contains(skipCases, name) {
				t.Skip(name)
			}
			r, err := cooklang.ParseString(spec.Source)
			if err != nil {
				t.Errorf("%s ParseString returned %v", name, err)
			}
			if err = compareResult(r, spec.Result); err != nil {
				t.Errorf("parseString() got = %v, want %v", r, spec.Result)
			}
		})
	}
}

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hadesgate/hadesgate/internal/anonymizer"
	"github.com/hadesgate/hadesgate/internal/anonymizer/vault"
)

type expectedEntity struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type corpusCase struct {
	ID       string           `json:"id"`
	Text     string           `json:"text"`
	Expected []expectedEntity `json:"expected"`
}

func main() {
	corpusPath := flag.String("corpus", "benchmarks/pii-accuracy/corpus.json", "accuracy corpus JSON")
	minPrecision := flag.Float64("min-precision", 0.90, "minimum accepted precision")
	minRecall := flag.Float64("min-recall", 0.90, "minimum accepted recall")
	flag.Parse()

	body, err := os.ReadFile(*corpusPath)
	if err != nil {
		fatal(err)
	}
	var corpus []corpusCase
	if err := json.Unmarshal(body, &corpus); err != nil {
		fatal(err)
	}

	memoryVault := vault.NewMemoryVault(time.Minute)
	defer memoryVault.Close()
	engine := anonymizer.NewEngine(anonymizer.DefaultConfig(), memoryVault)
	tp, fp, fn := 0, 0, 0
	failedCases := make([]string, 0)

	for index, testCase := range corpus {
		sessionID := fmt.Sprintf("benchmark-%d", index)
		result := engine.Anonymize(sessionID, testCase.Text)
		matched := make([]bool, len(testCase.Expected))
		caseFailed := false
		for _, detected := range result.Entities {
			matchIndex := -1
			for expectedIndex, expected := range testCase.Expected {
				if !matched[expectedIndex] && strings.EqualFold(string(detected.Type), expected.Type) && detected.Original == expected.Text {
					matchIndex = expectedIndex
					break
				}
			}
			if matchIndex >= 0 {
				matched[matchIndex] = true
				tp++
			} else {
				fp++
				caseFailed = true
			}
		}
		for _, found := range matched {
			if !found {
				fn++
				caseFailed = true
			}
		}
		if caseFailed {
			failedCases = append(failedCases, testCase.ID)
		}
		memoryVault.DeleteSession(sessionID)
	}

	precision := ratio(tp, tp+fp)
	recall := ratio(tp, tp+fn)
	report := map[string]interface{}{
		"cases": len(corpus), "true_positives": tp, "false_positives": fp,
		"false_negatives": fn, "precision": precision, "recall": recall,
		"failed_cases": failedCases,
	}
	encoded, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(encoded))
	if precision < *minPrecision || recall < *minRecall {
		os.Exit(1)
	}
}

func ratio(numerator, denominator int) float64 {
	if denominator == 0 {
		return 1
	}
	return float64(numerator) / float64(denominator)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}

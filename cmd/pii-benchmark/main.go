package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
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
	Tags     []string         `json:"tags,omitempty"`
	Language string           `json:"language,omitempty"`
}

type typeMetrics struct {
	TruePositives  int     `json:"true_positives"`
	FalsePositives int     `json:"false_positives"`
	FalseNegatives int     `json:"false_negatives"`
	Precision      float64 `json:"precision"`
	Recall         float64 `json:"recall"`
}

type caseFailure struct {
	ID         string           `json:"id"`
	Missing    []expectedEntity `json:"missing,omitempty"`
	Unexpected []expectedEntity `json:"unexpected,omitempty"`
}

func main() {
	corpusPath := flag.String("corpus", "benchmarks/pii-accuracy/corpus.json", "accuracy corpus JSON")
	minPrecision := flag.Float64("min-precision", 0.90, "minimum accepted precision")
	minRecall := flag.Float64("min-recall", 0.90, "minimum accepted recall")
	minTypePrecision := flag.Float64("min-type-precision", 0.90, "minimum accepted precision for each entity type")
	minTypeRecall := flag.Float64("min-type-recall", 0.90, "minimum accepted recall for each entity type")
	minCases := flag.Int("min-cases", 100, "minimum corpus size")
	minNegativeCases := flag.Int("min-negative-cases", 25, "minimum number of negative-control cases")
	minEntityTypes := flag.Int("min-entity-types", 20, "minimum number of expected entity types")
	flag.Parse()

	body, err := os.ReadFile(*corpusPath)
	if err != nil {
		fatal(err)
	}
	var corpus []corpusCase
	if err := json.Unmarshal(body, &corpus); err != nil {
		fatal(err)
	}
	if err := validateCorpus(corpus, *minCases, *minNegativeCases, *minEntityTypes); err != nil {
		fatal(err)
	}

	memoryVault := vault.NewMemoryVault(time.Minute)
	defer memoryVault.Close()
	engine := anonymizer.NewEngine(anonymizer.DefaultConfig(), memoryVault)
	tp, fp, fn := 0, 0, 0
	failedCases := make([]string, 0)
	failureDetails := make([]caseFailure, 0)
	perType := make(map[string]*typeMetrics)
	negativeCases := 0

	for index, testCase := range corpus {
		sessionID := fmt.Sprintf("benchmark-%d", index)
		result := engine.Anonymize(sessionID, testCase.Text)
		matched := make([]bool, len(testCase.Expected))
		caseFailed := false
		failure := caseFailure{ID: testCase.ID}
		if len(testCase.Expected) == 0 {
			negativeCases++
		}
		for _, detected := range result.Entities {
			detectedType := strings.ToUpper(string(detected.Type))
			metric := metricFor(perType, detectedType)
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
				metric.TruePositives++
			} else {
				fp++
				metric.FalsePositives++
				caseFailed = true
				failure.Unexpected = append(failure.Unexpected, expectedEntity{Type: detectedType, Text: detected.Original})
			}
		}
		for expectedIndex, found := range matched {
			if !found {
				fn++
				expected := testCase.Expected[expectedIndex]
				metricFor(perType, strings.ToUpper(expected.Type)).FalseNegatives++
				caseFailed = true
				failure.Missing = append(failure.Missing, expected)
			}
		}
		if caseFailed {
			failedCases = append(failedCases, testCase.ID)
			failureDetails = append(failureDetails, failure)
		}
		memoryVault.DeleteSession(sessionID)
	}

	precision := ratio(tp, tp+fp)
	recall := ratio(tp, tp+fn)
	for _, metric := range perType {
		metric.Precision = ratio(metric.TruePositives, metric.TruePositives+metric.FalsePositives)
		metric.Recall = ratio(metric.TruePositives, metric.TruePositives+metric.FalseNegatives)
	}
	failedEntityTypes := typesBelowThreshold(perType, *minTypePrecision, *minTypeRecall)
	report := map[string]interface{}{
		"cases": len(corpus), "true_positives": tp, "false_positives": fp,
		"false_negatives": fn, "precision": precision, "recall": recall,
		"positive_cases": len(corpus) - negativeCases, "negative_cases": negativeCases,
		"entity_types": len(perType), "per_type": perType,
		"failed_cases": failedCases, "failure_details": failureDetails,
		"failed_entity_types": failedEntityTypes,
	}
	encoded, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(encoded))
	if precision < *minPrecision || recall < *minRecall || len(failedEntityTypes) > 0 {
		os.Exit(1)
	}
}

func validateCorpus(corpus []corpusCase, minCases, minNegativeCases, minEntityTypes int) error {
	if len(corpus) < minCases {
		return fmt.Errorf("corpus has %d cases; minimum is %d", len(corpus), minCases)
	}
	ids := make(map[string]bool, len(corpus))
	entityTypes := make(map[string]bool)
	negativeCases := 0
	for index, testCase := range corpus {
		if strings.TrimSpace(testCase.ID) == "" {
			return fmt.Errorf("corpus case %d has an empty id", index)
		}
		if ids[testCase.ID] {
			return fmt.Errorf("duplicate corpus case id %q", testCase.ID)
		}
		ids[testCase.ID] = true
		if strings.TrimSpace(testCase.Text) == "" {
			return fmt.Errorf("corpus case %q has empty text", testCase.ID)
		}
		if len(testCase.Expected) == 0 {
			negativeCases++
		}
		seenExpected := make(map[string]bool)
		for _, expected := range testCase.Expected {
			expected.Type = strings.ToUpper(strings.TrimSpace(expected.Type))
			if expected.Type == "" || expected.Text == "" {
				return fmt.Errorf("corpus case %q has an incomplete expected entity", testCase.ID)
			}
			if !strings.Contains(testCase.Text, expected.Text) {
				return fmt.Errorf("corpus case %q expected text %q is not present in input", testCase.ID, expected.Text)
			}
			key := expected.Type + "\x00" + expected.Text
			if seenExpected[key] {
				return fmt.Errorf("corpus case %q duplicates expected entity %s %q", testCase.ID, expected.Type, expected.Text)
			}
			seenExpected[key] = true
			entityTypes[expected.Type] = true
		}
	}
	if negativeCases < minNegativeCases {
		return fmt.Errorf("corpus has %d negative cases; minimum is %d", negativeCases, minNegativeCases)
	}
	if len(entityTypes) < minEntityTypes {
		return fmt.Errorf("corpus covers %d entity types; minimum is %d", len(entityTypes), minEntityTypes)
	}
	return nil
}

func metricFor(metrics map[string]*typeMetrics, entityType string) *typeMetrics {
	metric := metrics[entityType]
	if metric == nil {
		metric = &typeMetrics{}
		metrics[entityType] = metric
	}
	return metric
}

func ratio(numerator, denominator int) float64 {
	if denominator == 0 {
		return 1
	}
	return float64(numerator) / float64(denominator)
}

func typesBelowThreshold(metrics map[string]*typeMetrics, minPrecision, minRecall float64) []string {
	failed := make([]string, 0)
	for entityType, metric := range metrics {
		if metric.Precision < minPrecision || metric.Recall < minRecall {
			failed = append(failed, entityType)
		}
	}
	sort.Strings(failed)
	return failed
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}

package metricscalculator

import (
	"fmt"
	"strings"

	"github.com/texttheater/golang-levenshtein/levenshtein"
)

// CalculateWER calculates the Word Error Rate (WER).
// WER = (Substitutions + Insertions + Deletions) / Number of words in reference
func CalculateWER(groundTruth string, recognizedText string) (float64, error) {
	if groundTruth == "" && recognizedText == "" {
		return 0.0, nil // Both empty, 0 errors
	}
	if groundTruth == "" { // All recognized words are insertions
		if recognizedText == "" { // Should be caught by above, but defensive
			return 0.0, nil
		}
		wordsRecognized := strings.Fields(recognizedText)
		return 1.0, fmt.Errorf("ground truth is empty, cannot normalize WER (recognized: %d words, treated as 100%% error)", len(wordsRecognized)) // Or return len(wordsRecognized) as edit distance
	}

	wordsGroundTruth := strings.Fields(groundTruth)
	wordsRecognized := strings.Fields(recognizedText)

	nGroundTruth := len(wordsGroundTruth)
	if nGroundTruth == 0 { // Should be caught by groundTruth == ""
		if len(wordsRecognized) == 0 {
			return 0.0, nil
		}
		return 1.0, fmt.Errorf("ground truth has 0 words after tokenization, cannot normalize WER (recognized: %d words, treated as 100%% error)", len(wordsRecognized))
	}

	// Levenshtein distance options for WER (words are items)
	options := levenshtein.Options{
		InsCost: 1,
		DelCost: 1,
		SubCost: 1,
		Matches: func(sourceItem, targetItem interface{}) bool {
			return sourceItem.(string) == targetItem.(string)
		},
	}

	// Convert word slices to []interface{} for Levenshtein function
	gtInterface := make([]interface{}, len(wordsGroundTruth))
	for i, v := range wordsGroundTruth {
		gtInterface[i] = v
	}
	recInterface := make([]interface{}, len(wordsRecognized))
	for i, v := range wordsRecognized {
		recInterface[i] = v
	}

	distance := levenshtein.DistanceForMatrix(gtInterface, recInterface, options)
	wer := float64(distance) / float64(nGroundTruth)

	return wer, nil
}

// CalculateCER calculates the Character Error Rate (CER).
// CER = (Substitutions + Insertions + Deletions) / Number of characters in reference
func CalculateCER(groundTruth string, recognizedText string) (float64, error) {
	if groundTruth == "" && recognizedText == "" {
		return 0.0, nil
	}
	if groundTruth == "" { // All recognized characters are insertions
		if recognizedText == "" { // Should be caught by above
			return 0.0, nil
		}
		// For CER, if ground truth is empty, any recognized text means 100% error rate.
		// The normalization by length of ground truth would be division by zero.
		// Some definitions might count insertions against a zero-length reference differently.
		// Here, we'll consider it 1.0 (100% error) if recognizedText is not empty.
		return 1.0, fmt.Errorf("ground truth is empty, cannot normalize CER (recognized: %d chars, treated as 100%% error)", len(recognizedText))
	}

	// For CER, we operate on runes (characters)
	runesGroundTruth := []rune(groundTruth)
	runesRecognized := []rune(recognizedText)

	nGroundTruth := len(runesGroundTruth)
	if nGroundTruth == 0 { // Should be caught by groundTruth == ""
		if len(runesRecognized) == 0 {
			return 0.0, nil
		}
		return 1.0, fmt.Errorf("ground truth has 0 characters after tokenization, cannot normalize CER (recognized: %d chars, treated as 100%% error)", len(runesRecognized))
	}
	
	// Levenshtein distance options for CER (runes are items)
	options := levenshtein.Options{
		InsCost: 1,
		DelCost: 1,
		SubCost: 1,
		Matches: func(sourceItem, targetItem interface{}) bool {
			return sourceItem.(rune) == targetItem.(rune)
		},
	}

	// Convert rune slices to []interface{}
	gtInterface := make([]interface{}, len(runesGroundTruth))
	for i, v := range runesGroundTruth {
		gtInterface[i] = v
	}
	recInterface := make([]interface{}, len(runesRecognized))
	for i, v := range runesRecognized {
		recInterface[i] = v
	}

	distance := levenshtein.DistanceForMatrix(gtInterface, recInterface, options)
	cer := float64(distance) / float64(nGroundTruth)

	return cer, nil
}

// CalculateLatency simply returns the duration in milliseconds.
// Actual timing logic (start/stop) will be in the evaluation engine.
func CalculateLatency(durationMs int64) int64 {
	return durationMs
}

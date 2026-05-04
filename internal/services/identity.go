package services

import (
	"math"
)

// Completion types
const (
	CompletionTypeStandard  = "COMPLETED_STANDARD"
	CompletionTypeEmergency = "COMPLETED_EMERGENCY"
	CompletionTypeMissed    = "MISSED"
	CompletionTypePaused    = "PAUSED"
)

// UpdateIdentityScore calculates the new identity score based on the completion type of a habit.
// The score is clamped between 0.0 and 100.0.
func UpdateIdentityScore(currentScore float64, completionType string) float64 {
	var modifier float64

	switch completionType {
	case CompletionTypeStandard:
		modifier = 2.0
	case CompletionTypeEmergency:
		modifier = 0.5
	case CompletionTypeMissed:
		modifier = -1.0
	case CompletionTypePaused:
		modifier = 0.0
	default:
		// Unknown or PENDING, no change
		modifier = 0.0
	}

	newScore := currentScore + modifier

	// Clamp the score between 0.0 and 100.0
	if newScore > 100.0 {
		newScore = 100.0
	} else if newScore < 0.0 {
		newScore = 0.0
	}

	// Round to two decimal places to handle float precision issues
	return math.Round(newScore*100) / 100
}

package services

import (
	"math"

	"zenith/internal/models"
)

// UpdateIdentityScore calculates the new identity score based on the completion
// type of a habit log. The resulting score is clamped between 0.0 and 100.0.
func UpdateIdentityScore(currentScore float64, ct models.CompletionType) float64 {
	newScore := currentScore + GetModifier(ct)
	return Clamp(newScore)
}

// GetModifier returns the raw score modifier for a given CompletionType.
// Exported so that handlers can revert a previous score adjustment without
// duplicating the business logic.
func GetModifier(ct models.CompletionType) float64 {
	switch ct {
	case models.CompletionTypeCompletedStandard:
		return 2.0
	case models.CompletionTypeCompletedEmergency:
		return 0.5
	case models.CompletionTypeMissed:
		return -1.0
	case models.CompletionTypePaused:
		return 0.0
	default:
		// PENDING or unknown — no change
		return 0.0
	}
}

// Clamp restricts a score to the [0.0, 100.0] range and rounds to two
// decimal places to eliminate floating-point drift.
func Clamp(score float64) float64 {
	if score > 100.0 {
		score = 100.0
	} else if score < 0.0 {
		score = 0.0
	}
	return math.Round(score*100) / 100
}

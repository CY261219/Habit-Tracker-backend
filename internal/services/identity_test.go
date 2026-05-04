package services

import (
	"math"
	"testing"

	"zenith/internal/models"
)

func TestUpdateIdentityScore(t *testing.T) {
	tests := []struct {
		name           string
		currentScore   float64
		completionType models.CompletionType
		expectedScore  float64
	}{
		{
			name:           "Standard completion adds 2.0",
			currentScore:   50.0,
			completionType: models.CompletionTypeCompletedStandard,
			expectedScore:  52.0,
		},
		{
			name:           "Emergency completion adds 0.5",
			currentScore:   50.0,
			completionType: models.CompletionTypeCompletedEmergency,
			expectedScore:  50.5,
		},
		{
			name:           "Missed subtracts 1.0",
			currentScore:   50.0,
			completionType: models.CompletionTypeMissed,
			expectedScore:  49.0,
		},
		{
			name:           "Paused does not change score",
			currentScore:   50.0,
			completionType: models.CompletionTypePaused,
			expectedScore:  50.0,
		},
		{
			name:           "Pending does not change score",
			currentScore:   50.0,
			completionType: models.CompletionTypePending,
			expectedScore:  50.0,
		},
		{
			name:           "Score does not exceed 100.0 (Standard)",
			currentScore:   99.0,
			completionType: models.CompletionTypeCompletedStandard,
			expectedScore:  100.0,
		},
		{
			name:           "Score does not exceed 100.0 (Emergency)",
			currentScore:   99.8,
			completionType: models.CompletionTypeCompletedEmergency,
			expectedScore:  100.0,
		},
		{
			name:           "Score does not drop below 0.0",
			currentScore:   0.5,
			completionType: models.CompletionTypeMissed,
			expectedScore:  0.0,
		},
		{
			name:           "Handles floating point precision correctly",
			currentScore:   50.1,
			completionType: models.CompletionTypeCompletedEmergency,
			expectedScore:  50.6,
		},
		{
			name:           "Handles precision edge case at boundary",
			currentScore:   99.9,
			completionType: models.CompletionTypeCompletedEmergency,
			expectedScore:  100.0,
		},
		{
			name:           "Score is already maxed",
			currentScore:   100.0,
			completionType: models.CompletionTypeCompletedStandard,
			expectedScore:  100.0,
		},
		{
			name:           "Score is already minimum",
			currentScore:   0.0,
			completionType: models.CompletionTypeMissed,
			expectedScore:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UpdateIdentityScore(tt.currentScore, tt.completionType)
			if math.Abs(got-tt.expectedScore) > 1e-9 {
				t.Errorf("UpdateIdentityScore() = %v, want %v", got, tt.expectedScore)
			}
		})
	}
}

func TestGetModifier(t *testing.T) {
	tests := []struct {
		ct       models.CompletionType
		expected float64
	}{
		{models.CompletionTypeCompletedStandard, 2.0},
		{models.CompletionTypeCompletedEmergency, 0.5},
		{models.CompletionTypeMissed, -1.0},
		{models.CompletionTypePaused, 0.0},
		{models.CompletionTypePending, 0.0},
		{"UNKNOWN", 0.0},
	}
	for _, tt := range tests {
		got := GetModifier(tt.ct)
		if got != tt.expected {
			t.Errorf("GetModifier(%q) = %v, want %v", tt.ct, got, tt.expected)
		}
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{50.0, 50.0},
		{100.0, 100.0},
		{0.0, 0.0},
		{101.0, 100.0},
		{-1.0, 0.0},
		{100.001, 100.0},
	}
	for _, tt := range tests {
		got := Clamp(tt.input)
		if math.Abs(got-tt.expected) > 1e-9 {
			t.Errorf("Clamp(%v) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

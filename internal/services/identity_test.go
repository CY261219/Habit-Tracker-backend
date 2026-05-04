package services

import (
	"testing"
)

func TestUpdateIdentityScore(t *testing.T) {
	tests := []struct {
		name           string
		currentScore   float64
		completionType string
		expectedScore  float64
	}{
		{
			name:           "Standard completion adds 2.0",
			currentScore:   50.0,
			completionType: CompletionTypeStandard,
			expectedScore:  52.0,
		},
		{
			name:           "Emergency completion adds 0.5",
			currentScore:   50.0,
			completionType: CompletionTypeEmergency,
			expectedScore:  50.5,
		},
		{
			name:           "Missed subtracts 1.0",
			currentScore:   50.0,
			completionType: CompletionTypeMissed,
			expectedScore:  49.0,
		},
		{
			name:           "Paused does not change score",
			currentScore:   50.0,
			completionType: CompletionTypePaused,
			expectedScore:  50.0,
		},
		{
			name:           "Unknown type does not change score",
			currentScore:   50.0,
			completionType: "PENDING",
			expectedScore:  50.0,
		},
		{
			name:           "Score does not exceed 100.0 (Standard)",
			currentScore:   99.0,
			completionType: CompletionTypeStandard,
			expectedScore:  100.0,
		},
		{
			name:           "Score does not exceed 100.0 (Emergency)",
			currentScore:   99.8,
			completionType: CompletionTypeEmergency,
			expectedScore:  100.0,
		},
		{
			name:           "Score does not drop below 0.0",
			currentScore:   0.5,
			completionType: CompletionTypeMissed,
			expectedScore:  0.0,
		},
		{
			name:           "Handles floating point precision correctly",
			currentScore:   50.1,
			completionType: CompletionTypeEmergency,
			expectedScore:  50.6, // 50.1 + 0.5
		},
		{
			name:           "Handles more precision edge cases",
			currentScore:   99.9,
			completionType: CompletionTypeEmergency,
			expectedScore:  100.0,
		},
		{
			name:           "Score is already maxed",
			currentScore:   100.0,
			completionType: CompletionTypeStandard,
			expectedScore:  100.0,
		},
		{
			name:           "Score is already minimum",
			currentScore:   0.0,
			completionType: CompletionTypeMissed,
			expectedScore:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := UpdateIdentityScore(tt.currentScore, tt.completionType)
			if got != tt.expectedScore {
				t.Errorf("UpdateIdentityScore() = %v, want %v", got, tt.expectedScore)
			}
		})
	}
}

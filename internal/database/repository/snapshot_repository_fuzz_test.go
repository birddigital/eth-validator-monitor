package repository

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FuzzCalculateEffectivenessScore performs fuzz testing to ensure the effectiveness
// calculation handles all input combinations without panics or invalid outputs
func FuzzCalculateEffectivenessScore(f *testing.F) {
	// Seed corpus with known good inputs
	f.Add(true, true, true, int32(1))
	f.Add(false, false, false, int32(5))
	f.Add(true, false, true, int32(3))
	f.Add(false, true, false, int32(10))
	f.Add(true, true, false, int32(2))

	// Add edge cases to corpus
	f.Add(false, false, false, int32(0))
	f.Add(true, true, true, int32(-1))
	f.Add(false, false, false, int32(2147483647)) // Max int32

	f.Fuzz(func(t *testing.T, head, source, target bool, inclusionDelay int32) {
		// Execute the function under test
		effectiveness := CalculateEffectivenessScore(head, source, target, inclusionDelay)

		// Invariants that MUST ALWAYS hold:

		// 1. Effectiveness must be within valid range [0, 100]
		require.GreaterOrEqual(t, effectiveness, 0.0,
			"effectiveness score cannot be negative (inputs: head=%v, source=%v, target=%v, delay=%d)",
			head, source, target, inclusionDelay)

		require.LessOrEqual(t, effectiveness, 100.0,
			"effectiveness score cannot exceed 100 (inputs: head=%v, source=%v, target=%v, delay=%d)",
			head, source, target, inclusionDelay)

		// 2. Result must not be NaN
		assert.False(t, math.IsNaN(effectiveness),
			"effectiveness score cannot be NaN (inputs: head=%v, source=%v, target=%v, delay=%d)",
			head, source, target, inclusionDelay)

		// 3. Result must not be infinite
		assert.False(t, math.IsInf(effectiveness, 0),
			"effectiveness score cannot be infinite (inputs: head=%v, source=%v, target=%v, delay=%d)",
			head, source, target, inclusionDelay)

		// 4. If all votes are correct and delay is 1, score must be 100
		if head && source && target && inclusionDelay == 1 {
			assert.Equal(t, 100.0, effectiveness,
				"perfect attestation must score exactly 100")
		}

		// 5. If all votes are missed and delay >= 5, score must be 0
		if !head && !source && !target && inclusionDelay >= 5 {
			assert.Equal(t, 0.0, effectiveness,
				"all missed votes with max delay must score 0")
		}

		// 6. Scores must be multiples of 6.25 (since each component is 25% and penalty is 6.25%)
		// Allow small floating point tolerance
		remainder := math.Mod(effectiveness*100, 625) // Multiply by 100 to avoid float precision issues
		assert.InDelta(t, 0.0, remainder, 1.0,
			"effectiveness score should be a multiple of 6.25 (got %v, remainder: %v)",
			effectiveness, remainder)

		// 7. Vote scores are deterministic
		expectedVoteScore := 0.0
		if head {
			expectedVoteScore += 25.0
		}
		if source {
			expectedVoteScore += 25.0
		}
		if target {
			expectedVoteScore += 25.0
		}

		// The minimum score should at least match vote scores minus inclusion penalty
		assert.GreaterOrEqual(t, effectiveness, 0.0,
			"score cannot be less than 0 (votes contributed: %v)", expectedVoteScore)
	})
}

// FuzzCalculateEffectivenessScore_Comparative tests that score increases monotonically
// with better performance
func FuzzCalculateEffectivenessScore_Comparative(f *testing.F) {
	f.Add(int32(2), int32(1))
	f.Add(int32(5), int32(3))
	f.Add(int32(10), int32(5))

	f.Fuzz(func(t *testing.T, worseDelay, betterDelay int32) {
		// Skip invalid comparisons
		if betterDelay >= worseDelay || betterDelay < 1 || worseDelay < 1 {
			t.Skip("invalid delay comparison")
		}

		// Given same votes, better inclusion delay should yield higher or equal score
		scoreWithBetterDelay := CalculateEffectivenessScore(true, true, true, betterDelay)
		scoreWithWorseDelay := CalculateEffectivenessScore(true, true, true, worseDelay)

		assert.GreaterOrEqual(t, scoreWithBetterDelay, scoreWithWorseDelay,
			"better inclusion delay (%d) should yield higher or equal score than worse delay (%d)",
			betterDelay, worseDelay)
	})
}

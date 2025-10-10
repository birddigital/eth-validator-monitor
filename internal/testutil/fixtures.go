package testutil

import (
	"time"

	"github.com/birddigital/eth-validator-monitor/internal/database/models"
)

// ValidatorFixture creates a test validator
func ValidatorFixture(index int64) *models.Validator {
	activationEpoch := int64(0)
	activationEligibilityEpoch := int64(0)

	return &models.Validator{
		ValidatorIndex:              index,
		Pubkey:                      "0x1234567890abcdef" + string(rune(index)),
		WithdrawalCredentials:       ptr("0x00" + string(rune(index))),
		EffectiveBalance:            32000000000,
		Slashed:                     false,
		ActivationEpoch:             &activationEpoch,
		ActivationEligibilityEpoch:  &activationEligibilityEpoch,
		ExitEpoch:                   nil,
		WithdrawableEpoch:           nil,
		Name:                        nil,
		Tags:                        []string{},
		Monitored:                   true,
		CreatedAt:                   FixedTime(),
		UpdatedAt:                   FixedTime(),
	}
}

// ValidatorSnapshotFixture creates a test validator snapshot
func ValidatorSnapshotFixture(validatorIndex int64, t time.Time) *models.ValidatorSnapshot {
	effectiveness := 98.5
	inclusionDelay := int32(1)
	headVote := true
	sourceVote := true
	targetVote := true
	dailyIncome := int64(10000000)
	apr := 4.5

	return &models.ValidatorSnapshot{
		Time:                          t,
		ValidatorIndex:                validatorIndex,
		Balance:                       32100000000,
		EffectiveBalance:              32000000000,
		AttestationEffectiveness:      &effectiveness,
		AttestationInclusionDelay:     &inclusionDelay,
		AttestationHeadVote:           &headVote,
		AttestationSourceVote:         &sourceVote,
		AttestationTargetVote:         &targetVote,
		ProposalsScheduled:            0,
		ProposalsExecuted:             0,
		ProposalsMissed:               0,
		SyncCommitteeParticipation:    true,
		Slashed:                       false,
		IsOnline:                      true,
		ConsecutiveMissedAttestations: 0,
		DailyIncome:                   &dailyIncome,
		APR:                           &apr,
	}
}

// ptr returns a pointer to the given value
func ptr[T any](v T) *T {
	return &v
}

// AlertFixture creates a test alert
func AlertFixture(validatorIndex int64) *models.Alert {
	return &models.Alert{
		ID:             1,
		ValidatorIndex: &validatorIndex,
		AlertType:      "missed_attestation",
		Severity:       models.SeverityWarning,
		Title:          "Test Alert",
		Message:        "Test alert message",
		Details:        models.JSONB{},
		Status:         models.AlertStatusActive,
		AcknowledgedAt: nil,
		ResolvedAt:     nil,
		CreatedAt:      FixedTime(),
		UpdatedAt:      FixedTime(),
	}
}

// MultipleValidatorFixtures creates multiple test validators
func MultipleValidatorFixtures(count int) []*models.Validator {
	validators := make([]*models.Validator, count)
	for i := 0; i < count; i++ {
		validators[i] = ValidatorFixture(int64(i))
	}
	return validators
}

// MultipleSnapshotFixtures creates multiple test snapshots
func MultipleSnapshotFixtures(validatorIndex int64, count int) []*models.ValidatorSnapshot {
	snapshots := make([]*models.ValidatorSnapshot, count)
	baseTime := FixedTime()

	for i := 0; i < count; i++ {
		t := baseTime.Add(time.Duration(i) * time.Hour)
		snapshots[i] = ValidatorSnapshotFixture(validatorIndex, t)
	}

	return snapshots
}

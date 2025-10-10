package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"

	"github.com/birddigital/eth-validator-monitor/pkg/types"
)

// BeaconClientImpl implements the BeaconClient interface
type BeaconClientImpl struct {
	baseURL       string
	httpClient    *http.Client
	retryClient   *RetryableHTTPClient
	timeout       time.Duration
	useRetry      bool
}

// NewBeaconClient creates a new beacon chain client with retry logic
func NewBeaconClient(baseURL string, timeout time.Duration) *BeaconClientImpl {
	retryConfig := DefaultRetryConfig()

	return &BeaconClientImpl{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		retryClient: NewRetryableHTTPClient(timeout, retryConfig),
		timeout:     timeout,
		useRetry:    true,
	}
}

// NewBeaconClientWithoutRetry creates a beacon client without retry logic (for testing)
func NewBeaconClientWithoutRetry(baseURL string, timeout time.Duration) *BeaconClientImpl {
	return &BeaconClientImpl{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout:  timeout,
		useRetry: false,
	}
}

// doRequest executes an HTTP request with optional retry logic
func (c *BeaconClientImpl) doRequest(req *http.Request) (*http.Response, error) {
	if c.useRetry && c.retryClient != nil {
		return c.retryClient.Do(req)
	}
	return c.httpClient.Do(req)
}

// GetValidator retrieves validator information by index
func (c *BeaconClientImpl) GetValidator(ctx context.Context, index int) (*types.ValidatorData, error) {
	url := fmt.Sprintf("%s/eth/v1/beacon/states/head/validators/%d", c.baseURL, index)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for validator %d: %w", index, err)
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request for validator %d: %w", index, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d for validator %d: %s", resp.StatusCode, index, string(body))
	}

	var result struct {
		Data types.ValidatorData `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result.Data, nil
}

// GetValidatorBalance retrieves the balance for a validator at a specific epoch
func (c *BeaconClientImpl) GetValidatorBalance(ctx context.Context, index int, epoch int) (*big.Int, error) {
	stateID := "head"
	if epoch > 0 {
		stateID = fmt.Sprintf("%d", epoch)
	}

	url := fmt.Sprintf("%s/eth/v1/beacon/states/%s/validators/%d", c.baseURL, stateID, index)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for validator %d balance at epoch %d: %w", index, epoch, err)
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request for validator %d balance at epoch %d: %w", index, epoch, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d for validator %d balance at epoch %d: %s", resp.StatusCode, index, epoch, string(body))
	}

	var result struct {
		Data types.ValidatorData `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Data.Balance, nil
}

// GetValidatorByPubkey retrieves validator information by public key
func (c *BeaconClientImpl) GetValidatorByPubkey(ctx context.Context, pubkey string) (*types.ValidatorData, error) {
	url := fmt.Sprintf("%s/eth/v1/beacon/states/head/validators/%s", c.baseURL, pubkey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for validator pubkey %s: %w", pubkey[:10]+"...", err)
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request for validator pubkey %s: %w", pubkey[:10]+"...", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d for validator pubkey %s: %s", resp.StatusCode, pubkey[:10]+"...", string(body))
	}

	var result struct {
		Data types.ValidatorData `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result.Data, nil
}

// GetAttestations retrieves attestations for a specific epoch
func (c *BeaconClientImpl) GetAttestations(ctx context.Context, epoch int) ([]types.Attestation, error) {
	// Calculate slot range for the epoch (32 slots per epoch)
	startSlot := epoch * 32
	endSlot := startSlot + 31

	var allAttestations []types.Attestation

	// Fetch attestations for each slot in the epoch
	for slot := startSlot; slot <= endSlot; slot++ {
		url := fmt.Sprintf("%s/eth/v1/beacon/blocks/%d/attestations", c.baseURL, slot)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request for slot %d: %w", slot, err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			// Skip slots without blocks or network errors
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}

		var result struct {
			Data []types.Attestation `json:"data"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		allAttestations = append(allAttestations, result.Data...)
	}

	return allAttestations, nil
}

// GetProposals retrieves block proposals for a specific epoch
func (c *BeaconClientImpl) GetProposals(ctx context.Context, epoch int) ([]types.Proposal, error) {
	// Calculate slot range for the epoch
	startSlot := epoch * 32
	endSlot := startSlot + 31

	var proposals []types.Proposal

	// Fetch proposals for each slot in the epoch
	for slot := startSlot; slot <= endSlot; slot++ {
		url := fmt.Sprintf("%s/eth/v2/beacon/blocks/%d", c.baseURL, slot)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			continue
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			continue
		}

		var result struct {
			Data struct {
				Message struct {
					Slot          string `json:"slot"`
					ProposerIndex string `json:"proposer_index"`
					StateRoot     string `json:"state_root"`
				} `json:"message"`
			} `json:"data"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		var slotNum, proposerNum int
		fmt.Sscanf(result.Data.Message.Slot, "%d", &slotNum)
		fmt.Sscanf(result.Data.Message.ProposerIndex, "%d", &proposerNum)

		proposals = append(proposals, types.Proposal{
			Slot:      slotNum,
			Proposer:  proposerNum,
			BlockRoot: result.Data.Message.StateRoot,
			Timestamp: time.Now(), // Would be calculated from genesis time + slot
		})
	}

	return proposals, nil
}

// SubscribeToHeadEvents subscribes to new beacon chain head events
func (c *BeaconClientImpl) SubscribeToHeadEvents(ctx context.Context) (<-chan types.HeadEvent, error) {
	eventChan := make(chan types.HeadEvent, 100)

	go func() {
		defer close(eventChan)

		url := fmt.Sprintf("%s/eth/v1/events?topics=head", c.baseURL)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return
		}

		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("Cache-Control", "no-cache")
		req.Header.Set("Connection", "keep-alive")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return
		}

		// Read Server-Sent Events stream
		buf := make([]byte, 4096)
		var data string

		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := resp.Body.Read(buf)
				if err != nil {
					if err != io.EOF {
						return
					}
					return
				}

				chunk := string(buf[:n])
				lines := parseSSEChunk(chunk)

				for _, line := range lines {
					if len(line) > 6 && line[:5] == "data:" {
						data = line[6:]

						var event struct {
							Slot  string `json:"slot"`
							Block string `json:"block"`
							State string `json:"state"`
						}

						if err := json.Unmarshal([]byte(data), &event); err != nil {
							continue
						}

						var slot int
						fmt.Sscanf(event.Slot, "%d", &slot)

						select {
						case eventChan <- types.HeadEvent{
							Slot:      slot,
							Block:     event.Block,
							State:     event.State,
							Timestamp: time.Now(),
						}:
						case <-ctx.Done():
							return
						}
					}
				}
			}
		}
	}()

	return eventChan, nil
}

// SubscribeToHead is an alias for SubscribeToHeadEvents (for compatibility)
func (c *BeaconClientImpl) SubscribeToHead(ctx context.Context) (<-chan types.HeadEvent, error) {
	return c.SubscribeToHeadEvents(ctx)
}

// parseSSEChunk parses Server-Sent Events chunk into lines
func parseSSEChunk(chunk string) []string {
	lines := []string{}
	current := ""

	for _, char := range chunk {
		if char == '\n' {
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}

	if current != "" {
		lines = append(lines, current)
	}

	return lines
}

// GetCurrentEpoch retrieves the current epoch number
func (c *BeaconClientImpl) GetCurrentEpoch(ctx context.Context) (int, error) {
	slot, err := c.GetCurrentSlot(ctx)
	if err != nil {
		return 0, err
	}
	// Each epoch is 32 slots
	return slot / 32, nil
}

// GetCurrentSlot retrieves the current slot number
func (c *BeaconClientImpl) GetCurrentSlot(ctx context.Context) (int, error) {
	url := fmt.Sprintf("%s/eth/v1/beacon/headers/head", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request for current slot: %w", err)
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return 0, fmt.Errorf("failed to execute request for current slot: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("unexpected status code %d for current slot: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data struct {
			Header struct {
				Message struct {
					Slot string `json:"slot"`
				} `json:"message"`
			} `json:"header"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	var slot int
	_, err = fmt.Sscanf(result.Data.Header.Message.Slot, "%d", &slot)
	if err != nil {
		return 0, fmt.Errorf("failed to parse slot: %w", err)
	}

	return slot, nil
}

// GetNetworkStats retrieves network-wide statistics
func (c *BeaconClientImpl) GetNetworkStats(ctx context.Context) (*types.NetworkStats, error) {
	// Get current epoch and slot
	currentEpoch, err := c.GetCurrentEpoch(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current epoch: %w", err)
	}

	currentSlot, err := c.GetCurrentSlot(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current slot: %w", err)
	}

	// Get validator set at current state
	url := fmt.Sprintf("%s/eth/v1/beacon/states/head/validators", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for network stats: %w", err)
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request for network stats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d for network stats: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []types.ValidatorData `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Calculate statistics
	var (
		totalValidators   int
		activeValidators  int
		pendingValidators int
		exitingValidators int
		slashedValidators int
		totalBalance      = big.NewInt(0)
	)

	for _, validator := range result.Data {
		totalValidators++

		switch validator.Status {
		case types.StatusActive:
			activeValidators++
		case types.StatusPending:
			pendingValidators++
		case types.StatusExiting:
			exitingValidators++
		case types.StatusSlashed:
			slashedValidators++
		}

		if validator.Validator.Slashed {
			slashedValidators++
		}

		if validator.Balance != nil {
			totalBalance.Add(totalBalance, validator.Balance)
		}
	}

	averageBalance := big.NewInt(0)
	if totalValidators > 0 {
		averageBalance.Div(totalBalance, big.NewInt(int64(totalValidators)))
	}

	return &types.NetworkStats{
		CurrentEpoch:      currentEpoch,
		CurrentSlot:       currentSlot,
		TotalValidators:   totalValidators,
		ActiveValidators:  activeValidators,
		PendingValidators: pendingValidators,
		ExitingValidators: exitingValidators,
		SlashedValidators: slashedValidators,
		AverageBalance:    averageBalance,
		TotalStaked:       totalBalance,
		ParticipationRate: float64(activeValidators) / float64(totalValidators),
		Timestamp:         time.Now(),
	}, nil
}

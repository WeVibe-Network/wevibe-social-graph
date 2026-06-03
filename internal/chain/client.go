package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type Client struct {
	restURL string
	http    *http.Client
}

func New(restURL string) *Client {
	if restURL == "" {
		restURL = "http://wevibe-chain:1317"
	}

	for len(restURL) > 0 && restURL[len(restURL)-1] == '/' {
		restURL = restURL[:len(restURL)-1]
	}

	return &Client{
		restURL: restURL,
		http: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type ContributorStats struct {
	Pubkey         string
	Contributions  uint64
	Serves         uint64
	SelfServes     uint64
	ReputationXP   uint64
	ServeXP        uint64
	OrgBreadth     uint64
	FirstSeenEpoch uint64
}

type contributorStatsResponse struct {
	ContributorID  string          `json:"contributor_id"`
	XP             string          `json:"xp"`
	ServeXP        string          `json:"serve_xp"`
	MemoryCount    string          `json:"memory_count"`
	ServeCount     string          `json:"serve_count"`
	SelfServeCount string          `json:"self_serve_count"`
	OrgBreadth     string          `json:"org_breadth"`
	FirstSeenEpoch string          `json:"first_seen_epoch"`
	Code           json.RawMessage `json:"code"`
	Message        string          `json:"message"`
}

func (c *Client) GetContributorStats(ctx context.Context, pubkey string) (*ContributorStats, error) {
	if c == nil || c.http == nil {
		return nil, fmt.Errorf("chain client is not initialized")
	}

	endpoint := fmt.Sprintf("%s/wevibe/reputation/v1/contributor/%s?epoch=0", c.restURL, pubkey)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build contributor stats request: %w", err)
	}

	response, err := c.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request contributor stats: %w", err)
	}
	defer response.Body.Close()

	var payload contributorStatsResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("contributor stats request failed: status %d", response.StatusCode)
		}
		return nil, fmt.Errorf("decode contributor stats response: %w", err)
	}

	errorCode, hasErrorCode := parseErrorCode(payload.Code)
	if response.StatusCode != http.StatusOK || hasErrorCode {
		if isNotFoundError(errorCode, payload.Message) {
			return &ContributorStats{Pubkey: pubkey}, nil
		}

		if hasErrorCode {
			return nil, fmt.Errorf("contributor stats response error (code=%d): %s", errorCode, payload.Message)
		}

		return nil, fmt.Errorf("contributor stats request failed: status %d", response.StatusCode)
	}

	contributions, err := parseUintField(payload.MemoryCount, "memory_count")
	if err != nil {
		return nil, err
	}
	serves, err := parseUintField(payload.ServeCount, "serve_count")
	if err != nil {
		return nil, err
	}
	selfServes, err := parseUintField(payload.SelfServeCount, "self_serve_count")
	if err != nil {
		return nil, err
	}
	reputationXP, err := parseUintField(payload.XP, "xp")
	if err != nil {
		return nil, err
	}
	serveXP, err := parseUintField(payload.ServeXP, "serve_xp")
	if err != nil {
		return nil, err
	}
	orgBreadth, err := parseUintField(payload.OrgBreadth, "org_breadth")
	if err != nil {
		return nil, err
	}
	firstSeenEpoch, err := parseUintField(payload.FirstSeenEpoch, "first_seen_epoch")
	if err != nil {
		return nil, err
	}

	resultPubkey := payload.ContributorID
	if resultPubkey == "" {
		resultPubkey = pubkey
	}

	return &ContributorStats{
		Pubkey:         resultPubkey,
		Contributions:  contributions,
		Serves:         serves,
		SelfServes:     selfServes,
		ReputationXP:   reputationXP,
		ServeXP:        serveXP,
		OrgBreadth:     orgBreadth,
		FirstSeenEpoch: firstSeenEpoch,
	}, nil
}

func parseUintField(raw string, field string) (uint64, error) {
	if raw == "" {
		return 0, nil
	}

	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", field, err)
	}

	return value, nil
}

func parseErrorCode(raw json.RawMessage) (uint64, bool) {
	if len(raw) == 0 || isJSONNull(raw) {
		return 0, false
	}

	var numericCode uint64
	if err := json.Unmarshal(raw, &numericCode); err == nil {
		return numericCode, true
	}

	var stringCode string
	if err := json.Unmarshal(raw, &stringCode); err == nil {
		if stringCode == "" {
			return 0, false
		}
		parsed, parseErr := strconv.ParseUint(stringCode, 10, 64)
		if parseErr == nil {
			return parsed, true
		}
	}

	return 0, true
}

func isJSONNull(raw json.RawMessage) bool {
	return len(raw) == 4 && raw[0] == 'n' && raw[1] == 'u' && raw[2] == 'l' && raw[3] == 'l'
}

func isNotFoundError(code uint64, message string) bool {
	if code == 2 {
		return true
	}

	if containsFoldASCII(message, "not found") {
		return true
	}

	return containsFoldASCII(message, "no stats")
}

func containsFoldASCII(value string, needle string) bool {
	if len(needle) == 0 {
		return true
	}

	if len(needle) > len(value) {
		return false
	}

	for start := 0; start <= len(value)-len(needle); start++ {
		matched := true
		for offset := 0; offset < len(needle); offset++ {
			if lowerASCII(value[start+offset]) != lowerASCII(needle[offset]) {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}

	return false
}

func lowerASCII(ch byte) byte {
	if ch >= 'A' && ch <= 'Z' {
		return ch + ('a' - 'A')
	}
	return ch
}

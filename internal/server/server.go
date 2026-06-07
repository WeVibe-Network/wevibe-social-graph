package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/wevibe-network/wevibe-social-graph/internal/chain"
	"github.com/wevibe-network/wevibe-social-graph/internal/store"
)

type Server struct {
	store *store.Store
	chain *chain.Client
}

func New(profileStore *store.Store, chainClient *chain.Client) *Server {
	return &Server{store: profileStore, chain: chainClient}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/health", s.handleHealth)
	mux.HandleFunc("/v1/profiles/batch", s.handleBatchProfiles)
	mux.HandleFunc("/v1/profiles", s.handleProfiles)
	mux.HandleFunc("/v1/profiles/", s.handleProfileByWallet)
	mux.HandleFunc("/v1/stats/contributor/", s.handleContributorStats)
	return corsMiddleware(mux)
}

type createProfileRequest struct {
	Wallet      string  `json:"wallet"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

type patchProfileRequest struct {
	DisplayName     *string `json:"display_name,omitempty"`
	AvatarURL       *string `json:"avatar_url,omitempty"`
	WalletPubkey    string  `json:"wallet_pubkey"`
	WalletSignature string  `json:"wallet_signature"`
}

type contributorStatsResponse struct {
	Pubkey            string `json:"pubkey"`
	DisplayName       string `json:"display_name,omitempty"`
	Contributions     uint64 `json:"contributions"`
	Serves            uint64 `json:"serves"`
	SelfServes        uint64 `json:"self_serves"`
	ReputationXP      uint64 `json:"reputation_xp"`
	ServeXP           uint64 `json:"serve_xp"`
	OrgBreadth        uint64 `json:"org_breadth"`
	FirstSeenEpoch    uint64 `json:"first_seen_epoch"`
	PendingWithdrawal uint64 `json:"pending_withdrawal"`
	AllTimeEarnings   uint64 `json:"all_time_earnings"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req createProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Wallet == "" || strings.TrimSpace(req.DisplayName) == "" {
		respondError(w, http.StatusBadRequest, "wallet and display_name are required")
		return
	}

	if err := validateAvatarURL(req.AvatarURL); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	profile, err := s.store.CreateProfile(r.Context(), req.Wallet, req.DisplayName, req.AvatarURL)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrProfileExists):
			respondError(w, http.StatusConflict, "profile already exists")
		default:
			respondError(w, http.StatusInternalServerError, "failed to create profile")
		}
		return
	}

	respondJSON(w, http.StatusCreated, profile)
}

func (s *Server) handleProfileByWallet(w http.ResponseWriter, r *http.Request) {
	wallet := strings.TrimPrefix(r.URL.Path, "/v1/profiles/")
	wallet = strings.TrimSpace(wallet)
	if wallet == "" || strings.Contains(wallet, "/") {
		respondError(w, http.StatusBadRequest, "wallet is required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		profile, err := s.store.GetProfile(r.Context(), wallet)
		if err != nil {
			switch {
			case errors.Is(err, store.ErrProfileNotFound):
				respondError(w, http.StatusNotFound, "profile not found")
			default:
				respondError(w, http.StatusInternalServerError, "failed to load profile")
			}
			return
		}
		respondJSON(w, http.StatusOK, profile)
	case http.MethodPatch:
		var req patchProfileRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.DisplayName == nil && req.AvatarURL == nil {
			respondError(w, http.StatusBadRequest, "no profile fields to update")
			return
		}

		if req.WalletPubkey == "" || req.WalletSignature == "" {
			respondError(w, http.StatusUnauthorized, "wallet ownership proof is required")
			return
		}

		if err := validateAvatarURL(req.AvatarURL); err != nil {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}

		canonical := buildPatchCanonical(wallet, req.DisplayName, req.AvatarURL)
		if err := verifyCosmosArbitrarySignature(wallet, []byte(canonical), req.WalletPubkey, req.WalletSignature); err != nil {
			respondError(w, http.StatusUnauthorized, "wallet signature verification failed")
			return
		}

		profile, err := s.store.UpdateProfile(r.Context(), wallet, req.DisplayName, req.AvatarURL)
		if err != nil {
			switch {
			case errors.Is(err, store.ErrProfileNotFound):
				respondError(w, http.StatusNotFound, "profile not found")
			default:
				respondError(w, http.StatusBadRequest, err.Error())
			}
			return
		}

		respondJSON(w, http.StatusOK, profile)
	default:
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleBatchProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	walletQuery := strings.TrimSpace(r.URL.Query().Get("wallets"))
	if walletQuery == "" {
		respondJSON(w, http.StatusOK, []store.Profile{})
		return
	}

	wallets := make([]string, 0)
	for _, wallet := range strings.Split(walletQuery, ",") {
		trimmed := strings.TrimSpace(wallet)
		if trimmed != "" {
			wallets = append(wallets, trimmed)
		}
	}

	profiles, err := s.store.GetProfilesByWallets(r.Context(), wallets)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to load profiles")
		return
	}

	respondJSON(w, http.StatusOK, profiles)
}

func (s *Server) handleContributorStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	pubkey := strings.TrimPrefix(r.URL.Path, "/v1/stats/contributor/")
	pubkey = strings.TrimSpace(pubkey)
	if pubkey == "" || strings.Contains(pubkey, "/") {
		respondError(w, http.StatusBadRequest, "pubkey is required")
		return
	}

	stats, err := s.chain.GetContributorStats(r.Context(), pubkey)
	if err != nil {
		respondError(w, http.StatusBadGateway, "failed to load contributor stats")
		return
	}

	response := contributorStatsResponse{
		Pubkey:         stats.Pubkey,
		Contributions:  stats.Contributions,
		Serves:         stats.Serves,
		SelfServes:     stats.SelfServes,
		ReputationXP:   stats.ReputationXP,
		ServeXP:        stats.ServeXP,
		OrgBreadth:     stats.OrgBreadth,
		FirstSeenEpoch: stats.FirstSeenEpoch,
	}

	if response.Pubkey == "" {
		response.Pubkey = pubkey
	}

	pending, allTime, rewardErr := s.chain.GetContributorReward(r.Context(), pubkey)
	if rewardErr == nil {
		response.PendingWithdrawal = pending
		response.AllTimeEarnings = allTime
	}

	profile, profileErr := s.store.GetProfile(r.Context(), pubkey)
	if profileErr == nil && profile != nil {
		response.DisplayName = profile.DisplayName
	}

	respondJSON(w, http.StatusOK, response)
}

func buildPatchCanonical(wallet string, displayName, avatarURL *string) string {
	displayNameValue := ""
	if displayName != nil {
		displayNameValue = strings.TrimSpace(*displayName)
	}

	avatarURLValue := ""
	if avatarURL != nil {
		avatarURLValue = strings.TrimSpace(*avatarURL)
	}

	return strings.Join([]string{
		"wevibe.social_graph.update_profile.v1",
		fmt.Sprintf("wallet:%s", wallet),
		fmt.Sprintf("display_name:%s", displayNameValue),
		fmt.Sprintf("avatar_url:%s", avatarURLValue),
	}, "\n")
}

func validateAvatarURL(avatarURL *string) error {
	if avatarURL == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*avatarURL)
	if trimmed == "" {
		return nil
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return fmt.Errorf("invalid avatar_url")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("avatar_url must use http or https")
	}
	if parsed.Host == "" {
		return fmt.Errorf("invalid avatar_url")
	}

	return nil
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

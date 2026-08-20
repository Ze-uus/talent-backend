package response

// Machine-readable error codes returned in the Response.Message field.
// All values are snake_case strings — the frontend maps these to user-facing text.

const (
	// Auth
	ErrInvalidCredentials = "invalid_credentials"
	ErrTOTPRequired       = "totp_required"
	ErrInvalidTOTP        = "invalid_totp_code"
	ErrSessionExpired     = "session_expired"
	ErrSessionInactive    = "session_inactive"
	ErrAccountPending     = "account_pending_approval"
	ErrAccountSuspended   = "account_suspended"
	ErrAccountBanned      = "account_banned"
	ErrAccountDeleted     = "account_deleted"
	ErrAccountRejected    = "account_rejected"
	ErrAccountInvited     = "account_invited"
	ErrInvalidInvite      = "invalid_or_expired_invite"
	ErrViewerInvalid      = "invalid_viewer_credentials"
	ErrMissingToken       = "missing_token"
	ErrInsufficientRole   = "insufficient_role"

	// Validation
	ErrValidationFailed     = "validation_failed"
	ErrNotFound             = "not_found"
	ErrConflict             = "conflict"
	ErrInternal             = "internal_error"
	ErrInvalidReference     = "invalid_reference"
	ErrNumericOutOfRange    = "numeric_out_of_range"
	ErrRequiredFieldMissing = "required_field_missing"

	// Campaign / cycle
	ErrCycleBudgetExceeds     = "cycle_budget_exceeds_remaining_campaign_budget"
	ErrInvalidCycleTransition = "invalid_cycle_status_transition"
	ErrCycleLengthInvalid     = "cycle_length_must_be_5_7_or_10"
	ErrNoRemainingBudget      = "no_remaining_budget_in_cycle"

	// Payout
	ErrInvalidPDCReference = "invalid_pdc_reference"
	ErrPerformanceBelow50  = "performance_below_50_percent"

	// General
	ErrRateLimited        = "rate_limit_exceeded"
	ErrStreamNotSupported = "streaming_not_supported"
)

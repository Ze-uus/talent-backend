package response

import (
	"errors"
	"log/slog"
	"net/http"

	storepg "github.com/Ze-uus/talent-backend/internal/store/pgerr"
)

type ClientError struct {
	Status int
	Code   string
	cause  error
}

func (e *ClientError) Error() string {
	if e.Code == "" {
		return ErrValidationFailed
	}
	return e.Code
}

func (e *ClientError) Unwrap() error { return e.cause }

func NewClientError(status int, code string) error {
	return &ClientError{Status: status, Code: code}
}

func WrapClientError(err error, status int, code string) error {
	return &ClientError{Status: status, Code: code, cause: err}
}

func Validation(code string) error {
	return NewClientError(http.StatusUnprocessableEntity, code)
}

func NotFound(code string) error {
	return NewClientError(http.StatusNotFound, code)
}

func Conflict(code string) error {
	return NewClientError(http.StatusConflict, code)
}

func Unauthorized(code string) error {
	return NewClientError(http.StatusUnauthorized, code)
}

func Forbidden(code string) error {
	return NewClientError(http.StatusForbidden, code)
}

var legacyValidationCodes = map[string]struct{}{
	"assignment_not_active":                          {},
	"content_html_not_allowed":                       {},
	"content_image_required":                         {},
	"content_images_required":                        {},
	"content_item_id_required":                       {},
	"content_link_id_required":                       {},
	"cycle_budget_below_max_cpa":                     {},
	"cycle_budget_below_target_cpa":                  {},
	"cycle_budget_exceeds_remaining_campaign_budget": {},
	"cycle_campaign_mismatch":                        {},
	"cycle_length_must_be_between_5_and_10":          {},
	"cycle_not_active":                               {},
	"duplicate_content_item_id":                      {},
	"duplicate_content_link_id":                      {},
	"event_type_required":                            {},
	"idempotency_key_required":                       {},
	"invalid_active_transition":                      {},
	"invalid_campaign_cpa":                           {},
	"invalid_campaign_type":                          {},
	"invalid_content_image_url":                      {},
	"invalid_content_link_url":                       {},
	"invalid_cycle_status_transition":                {},
	"invalid_pdc_reference":                          {},
	"max_contacts_reached":                           {},
	"max_cpa_exceeds_total_budget":                   {},
	"no_eligible_budget_slots":                       {},
	"performance_below_50_percent":                   {},
	"target_cpa_exceeds_max_cpa":                     {},
	"tier_value_exceeds_supported_range":             {},
	"too_many_content_images":                        {},
	"too_many_content_items":                         {},
	"too_many_content_links":                         {},
	"total_budget_required":                          {},
}

func MapError(err error) (int, string) {
	if err == nil {
		return http.StatusInternalServerError, ErrInternal
	}
	var clientErr *ClientError
	if errors.As(err, &clientErr) {
		code := clientErr.Code
		if code == "" {
			code = ErrValidationFailed
		}
		return clientErr.Status, code
	}
	if translated, ok := storepg.As(err); ok {
		switch translated.Kind {
		case storepg.KindNotFound:
			return http.StatusNotFound, translated.Code
		case storepg.KindConflict:
			return http.StatusConflict, translated.Code
		case storepg.KindValidation:
			return http.StatusUnprocessableEntity, translated.Code
		}
	}
	code := err.Error()
	switch code {
	case "not_found", "cycle_not_assigned", "invalid_token", "presentation_unavailable":
		return http.StatusNotFound, ErrNotFound
	case "forbidden", ErrInsufficientRole:
		return http.StatusForbidden, code
	case "invalid_password", ErrInvalidCredentials, ErrInvalidTOTP, ErrInvalidInvite:
		return http.StatusUnauthorized, code
	case ErrAccountPending, ErrAccountSuspended, ErrAccountBanned, ErrAccountDeleted,
		ErrAccountRejected, ErrAccountInvited:
		return http.StatusForbidden, code
	case "duplicate_event", "already_exists", "superadmin_already_exists":
		return http.StatusConflict, code
	}
	if _, ok := legacyValidationCodes[code]; ok {
		return http.StatusUnprocessableEntity, code
	}
	return http.StatusInternalServerError, ErrInternal
}

func FromError(err error) (int, Response) {
	status, code := MapError(err)
	logUnexpected(err, status)
	return status, Fail(code)
}

func ErrorStatus(err error) int {
	status, _ := MapError(err)
	logUnexpected(err, status)
	return status
}

func ErrorCode(err error) string {
	_, code := MapError(err)
	return code
}

func WriteError(w http.ResponseWriter, err error) {
	status, body := FromError(err)
	WriteJSON(w, status, body)
}

func logUnexpected(err error, status int) {
	if status == http.StatusInternalServerError && err != nil {
		slog.Error("request failed", "error", err)
	}
}

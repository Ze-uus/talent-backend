package pgerr

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Kind string

const (
	KindNotFound   Kind = "not_found"
	KindConflict   Kind = "conflict"
	KindValidation Kind = "validation"
)

type Error struct {
	Kind       Kind
	Code       string
	Constraint string
	cause      error
}

func (e *Error) Error() string { return e.Code }
func (e *Error) Unwrap() error { return e.cause }

var constraintCodes = map[string]string{
	"users_email_key":                           "email_already_exists",
	"users_google_id_key":                       "google_account_already_exists",
	"campaigns_human_id_key":                    "campaign_human_id_conflict",
	"brands_shortcode_key":                      "brand_shortcode_conflict",
	"cycles_campaign_id_cycle_number_key":       "cycle_number_conflict",
	"talent_assignments_talent_id_cycle_id_key": "talent_already_assigned",
	"tracking_links_token_key":                  "tracking_token_conflict",
	"payout_records_talent_id_cycle_id_key":     "payout_already_exists",
	"manager_campaign_assignments_pkey":         "manager_already_assigned",
	"conversion_events_idempotency_key_key":     "duplicate_event",
}

func Translate(err error) error {
	if err == nil {
		return nil
	}
	var translated *Error
	if errors.As(err, &translated) {
		return err
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return &Error{Kind: KindNotFound, Code: "not_found", cause: err}
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case "23505":
		code := constraintCodes[pgErr.ConstraintName]
		if code == "" {
			code = "conflict"
		}
		return &Error{Kind: KindConflict, Code: code, Constraint: pgErr.ConstraintName, cause: err}
	case "23503":
		return &Error{Kind: KindValidation, Code: "invalid_reference", Constraint: pgErr.ConstraintName, cause: err}
	case "23514":
		return &Error{Kind: KindValidation, Code: "validation_failed", Constraint: pgErr.ConstraintName, cause: err}
	case "23502":
		return &Error{Kind: KindValidation, Code: "required_field_missing", Constraint: pgErr.ConstraintName, cause: err}
	case "22003":
		return &Error{Kind: KindValidation, Code: "numeric_out_of_range", Constraint: pgErr.ConstraintName, cause: err}
	case "22P02":
		return &Error{Kind: KindValidation, Code: "invalid_value", Constraint: pgErr.ConstraintName, cause: err}
	default:
		return err
	}
}

func As(err error) (*Error, bool) {
	err = Translate(err)
	var translated *Error
	ok := errors.As(err, &translated)
	return translated, ok
}

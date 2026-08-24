package pgerr_test

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Ze-uus/talent-backend/internal/store/pgerr"
)

func TestTranslate(t *testing.T) {
	tests := []struct {
		name string
		err  error
		kind pgerr.Kind
		code string
	}{
		{"not found", pgx.ErrNoRows, pgerr.KindNotFound, "not_found"},
		{"known unique", &pgconn.PgError{Code: "23505", ConstraintName: "users_email_key"}, pgerr.KindConflict, "email_already_exists"},
		{"generic unique", &pgconn.PgError{Code: "23505"}, pgerr.KindConflict, "conflict"},
		{"foreign key", &pgconn.PgError{Code: "23503"}, pgerr.KindValidation, "invalid_reference"},
		{"check", &pgconn.PgError{Code: "23514"}, pgerr.KindValidation, "validation_failed"},
		{"not null", &pgconn.PgError{Code: "23502"}, pgerr.KindValidation, "required_field_missing"},
		{"overflow", &pgconn.PgError{Code: "22003"}, pgerr.KindValidation, "numeric_out_of_range"},
		{"invalid value", &pgconn.PgError{Code: "22P02"}, pgerr.KindValidation, "invalid_value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			translated, ok := pgerr.As(tt.err)
			if !ok {
				t.Fatal("expected translated error")
			}
			if translated.Kind != tt.kind || translated.Code != tt.code {
				t.Fatalf("got kind=%s code=%s", translated.Kind, translated.Code)
			}
			if !errors.Is(translated, tt.err) {
				t.Fatal("translated error must wrap cause")
			}
		})
	}
}

func TestTranslateLeavesUnknownError(t *testing.T) {
	if _, ok := pgerr.As(errors.New("network_failed")); ok {
		t.Fatal("unexpected translation")
	}
}

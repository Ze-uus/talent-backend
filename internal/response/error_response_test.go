package response_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Ze-uus/talent-backend/internal/response"
)

func TestMapError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"typed validation", response.Validation("bad_budget"), 422, "bad_budget"},
		{"typed conflict", response.Conflict("already_added"), 409, "already_added"},
		{"not found", pgx.ErrNoRows, 404, "not_found"},
		{"overflow", &pgconn.PgError{Code: "22003"}, 422, "numeric_out_of_range"},
		{"unique", &pgconn.PgError{Code: "23505", ConstraintName: "users_email_key"}, 409, "email_already_exists"},
		{"legacy validation", errors.New("target_cpa_exceeds_max_cpa"), 422, "target_cpa_exceeds_max_cpa"},
		{"unknown hidden", errors.New("connection password leaked"), 500, "internal_error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, code := response.MapError(tt.err)
			if status != tt.status || code != tt.code {
				t.Fatalf("got %d/%s want %d/%s", status, code, tt.status, tt.code)
			}
		})
	}
}

func TestFromError(t *testing.T) {
	status, body := response.FromError(response.Validation("invalid_value"))
	if status != http.StatusUnprocessableEntity || body.Message != "invalid_value" || body.Success {
		t.Fatalf("status=%d body=%+v", status, body)
	}
}

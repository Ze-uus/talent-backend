## What does this PR do?

<!-- One or two sentences. What changed and why. -->

## Type of change

- [ ] Bug fix
- [ ] New feature / endpoint
- [ ] Algorithm change
- [ ] Refactor (no behaviour change)
- [ ] Migration
- [ ] Config / infra
- [ ] Documentation

## Related issue

Closes #

---

## Checklist

### Code
- [ ] `make ci` passes locally (`make lint` + `make test-coverage` + `make build`)
- [ ] All new service logic has unit tests (mock store)
- [ ] No business logic in handlers — services only
- [ ] No DB calls in `internal/algo/` — pure functions only
- [ ] PATCH endpoints use pointer-field input structs

### API
- [ ] New endpoints registered with Huma (visible at `/docs`)
- [ ] Response envelope used on all handlers (`data`, `message`, `status`, `success`)
- [ ] Naming follows `snake_case` throughout (fields, JSON keys, constants)

### Data
- [ ] Migration added if schema changed (`migrations/NNN_description.sql`)
- [ ] `make migrate-up` applied cleanly
- [ ] No raw SQL outside `internal/store/postgres/`

### Security
- [ ] Admin override actions write to `audit_log`
- [ ] No secrets or credentials in code or test fixtures
- [ ] New env vars added to `.env.local.example` and `README.md`

### Real-time
- [ ] Conversion events push to `conversion_ch` after DB write
- [ ] Anomaly events push to `anomaly_ch` from daily compute job
- [ ] WS messages use the standard `WSMessage` type

---

## Testing notes

<!-- How did you test this? What edge cases were checked? -->

## Migration notes

<!-- If a migration is included, describe what it changes and whether it's reversible. -->

## Screenshots / output

<!-- For new endpoints, paste a sample request/response. For algo changes, paste test output. -->
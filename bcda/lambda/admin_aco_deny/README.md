The ACO Deny administrative task lambda will add an existing ACO to our deny list.  It should be called via AWS's lambda interface (see: https://confluence.cms.gov/display/BCDA/How+To+deny+an+ACO+From+Generating+Credentials).

## Payload Specification

The Lambda expects a JSON payload with the following fields:

| Field | Type | Required | Description |
|---|---|---|---|
| `deny_aco_ids` | `[]string` | **Yes** | List of ACO CMS IDs to deny. Must not be empty. |
| `cutoff_date` | `string` (RFC3339) | No | Date and time after which the ACO will have no access and requests will be blocked (denylisted). Defaults to current time (`time.Now()`) if omitted. |
| `termination_date` | `string` (RFC3339) | No | Date and time when the ACO moved from full to limited access. Defaults to `cutoff_date` if omitted. Cannot be after `cutoff_date`. If `termination_date` is in the future, a `cutoff_date` is required. |

### Timestamp Format Requirement

> **IMPORTANT:** Date fields (`cutoff_date` and `termination_date`) MUST be formatted in standard RFC3339 / ISO 8601 format including time and timezone (e.g., `"2026-12-31T23:59:59Z"`). Plain date strings such as `"2026-12-31"` are not supported and will result in a JSON unmarshaling error.

### Example Payloads

#### 1. Immediate Deny (default behavior)
```json
{
  "deny_aco_ids": ["A9994", "A9995"]
}
```

#### 2. Scheduled Future Cutoff Date
```json
{
  "deny_aco_ids": ["A9994"],
  "cutoff_date": "2026-12-31T23:59:59Z"
}
```

#### 3. Custom Termination and Cutoff Dates
```json
{
  "deny_aco_ids": ["A9994"],
  "termination_date": "2026-10-01T00:00:00Z",
  "cutoff_date": "2026-12-31T23:59:59Z"
}
```

## Testing & Deployment

You can run the unit test suite from the base dir (bcda-app) using the following command:

```bash
make test-path TEST_PATH="bcda/lambda/admin_aco_deny/*.go"
```
(You might have to make load-fixtures first). It also has an integration test run via github actions (see .github/workflows/admin-aco-deny-integration-test.yml).

The lambda is deployed (or promoted in the case of prod) using github actions (see .github/workflows/admin-aco-deny-lambda-{env}-deploy.yml files).

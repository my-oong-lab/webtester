# Security Audit Report - WebTester

**Date:** 2026-11-16
**Methodology:** Cloudflare Security Audit Skill
**Scope:** Full codebase security review

---

## Executive Summary

The WebTester application contains **2 Critical/High severity vulnerabilities** and **several Medium/Low issues**. The most significant issues are:

1. **Hardcoded JWT signing secrets** - allows complete authentication bypass
2. **SSRF via user-controlled URLs** - multiple endpoints allow scanning internal networks

All SQL queries use parameterized statements (no SQL injection found). Authentication uses bcrypt for password hashing and JWT with proper expiration.

---

## Critical & High Severity Findings

### WEB-005: Hardcoded JWT Signing Secrets (CRITICAL)

**File:** `internal/auth/token.go:28`

The `DefaultTokenConfig()` function returns hardcoded JWT signing secrets:
```go
func DefaultTokenConfig() TokenConfig {
	return TokenConfig{
		AccessSecret:  "change-me-in-production-access-secret",
		RefreshSecret: "change-me-in-production-refresh-secret",
		...
	}
}
```

**Impact:** If deployed without overriding these secrets, attackers can forge arbitrary JWT tokens for any user, achieving complete authentication bypass.

**Remediation:** Load secrets from environment variables or a secrets manager. Fail startup if secrets are not configured.

---

### WEB-001: SSRF via target_url in Scan Creation (HIGH)

**File:** `internal/services/scan_service.go:46`

The `CreateScan` endpoint accepts `target_url` from the user without validating that it points to an external resource. The server makes HTTP requests to this URL via:
1. ZAP Docker container (`internal/clients/zap/docker.go`)
2. Passive header checks (`internal/checks/passive.go`)

**Impact:** Attackers can use the webtester instance as a proxy to scan internal networks and access cloud metadata endpoints (169.254.169.254).

**Remediation:** Validate `target_url` against an allowlist of external hosts. Block private IP ranges, localhost, and link-local addresses.

---

### WEB-002: SSRF via Passive Header Checks (HIGH)

**File:** `internal/checks/passive.go:19`

`PassiveHeaderChecks` makes an HTTP request to a user-controlled target URL without validation.

**Remediation:** Validate and sanitize the target URL before making requests.

---

### WEB-003: SSRF via BOLA Probe URLs (HIGH)

**File:** `internal/checks/bola.go:19`

`BolaProbe` fetches user-supplied probe URLs with user-supplied auth tokens. No URL validation is performed.

**Remediation:** Validate probe URLs against target_url domain. Block internal/private IPs.

---

### WEB-004: SSRF via Load Test Target URL (HIGH)

**File:** `internal/loadtest/runner.go:52`

Load test runner makes HTTP requests to a user-controlled target URL without validation. Concurrency and duration can be set arbitrarily high.

**Remediation:** Validate target URL. Enforce reasonable limits on duration and concurrency.

---

## Medium Severity Findings

### WEB-006: No Rate Limiting on Authentication Endpoints (MEDIUM)

**File:** `internal/routes/routes.go:29`

Login, Setup, and Refresh endpoints have no rate limiting, enabling brute-force password attacks and account enumeration.

**Remediation:** Apply rate limiting middleware to auth endpoints. The Redis `RateLimit` method exists in `internal/auth/store.go` but is unused.

---

### WEB-007: ZAP Docker Image Configurable Without Validation (MEDIUM)

**File:** `internal/clients/zap/docker.go:18`

The ZAP Docker image name is loaded from config/environment without validation.

**Remediation:** Validate image name against an allowlist. Use a pinned image digest.

---

### WEB-010: No CORS Configuration (MEDIUM)

**File:** `internal/routes/routes.go:17`

No CORS middleware is configured. If the frontend is served from a different origin, cross-origin requests may be unrestricted.

**Remediation:** Configure CORS with explicit allowed origins.

---

## Low Severity Findings

### WEB-008: No Security Headers (LOW)

**File:** `internal/routes/routes.go:17`

The application does not set security headers like X-Content-Type-Options, X-Frame-Options, or Content-Security-Policy.

**Remediation:** Add security headers middleware.

---

### WEB-011: Docker Socket Mounted in Production Container (LOW)

**File:** `docker-compose.yml:18`

The docker-compose.yml mounts `/var/run/docker.sock` into the app container, giving the app full control over the Docker host.

**Remediation:** Use a Docker socket proxy to restrict API access.

---

## Informational Findings

### WEB-009: Potential Account Enumeration (NEEDS VALIDATION)

**File:** `internal/services/auth_service.go:51`

Login returns "invalid credentials" for both wrong password and non-existent user, but timing differences (bcrypt comparison vs immediate not-found) may leak user existence.

**Status:** Needs validation - whether the timing difference is measurable over the network.

---

## Rejected Findings

### WEB-012: SQL Injection in Scan Store

**File:** `internal/stores/scan_store.go:80`

All SQL queries use parameterized statements with `?` placeholders. No string concatenation in SQL. **No SQL injection risk found.**

---

## Coverage Summary

| Category | Units | Coverage |
|----------|-------|----------|
| Authentication | 5 | Partial - token handling reviewed, setup/login reviewed |
| Input Validation | 6 | Partial - URL validation missing |
| Data Storage | 4 | Full - parameterized queries, no injection |
| Container Execution | 1 | Partial - image validation missing |
| Network/Config | 3 | Partial - CORS, security headers missing |

**Overall:** 12 units audited, 11 confirmed findings (1 rejected), 1 needs validation.

---

## Recommendations Summary

1. **Immediate:** Rotate and externalize JWT secrets (WEB-005)
2. **High Priority:** Implement URL validation for all external requests (WEB-001-004)
3. **Medium Priority:** Add rate limiting to auth endpoints (WEB-006)
4. **Low Priority:** Add security headers and CORS configuration (WEB-008, WEB-010)

---

*Report generated by Cloudflare Security Audit Skill methodology.*
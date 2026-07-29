# Security Standards

These standards are based on the OWASP Cheat Sheet Series and should be followed for application code, APIs, and supporting infrastructure.

## Core Principles

- Assume all external input is untrusted.
- Validate early, validate on the server, and validate again where needed.
- Deny by default and allow explicitly.
- Minimize privileges, secrets, and exposed attack surface.
- Prefer secure defaults over custom security behavior.

## Input Validation

- Validate all data from users, services, files, and integrations.
- Use allowlists for structure, format, and allowed values.
- Enforce both syntactic and semantic validation.
- Reject malformed data before it reaches persistence or business logic.
- Do not rely on client-side validation for security.
- Treat validation as complementary to encoding and parameterization, not a replacement.

## Authentication And Sessions

- Use strong authentication mechanisms appropriate to the application.
- Store session identifiers securely and make them hard to predict.
- Protect session cookies with `HttpOnly`, `Secure`, and an appropriate `SameSite` setting.
- Prefer `__Host-` cookie prefixes for session cookies when possible.
- Regenerate session identifiers after authentication or privilege changes.
- Expire sessions and tokens when they are no longer needed.

## Authorization

- Check authorization on the server for every sensitive action and every data access path.
- Do not trust hidden UI state, client-side route guards, or front-end assumptions.
- Use least privilege for service accounts, database users, and infrastructure roles.
- Deny access by default when authorization logic fails or is ambiguous.
- Test access control paths explicitly.

## Output Encoding And XSS

- Encode output for the correct context before rendering untrusted content.
- Prefer framework-safe rendering patterns.
- Avoid raw HTML rendering unless the content has been sanitized and the risk is understood.
- Never treat untrusted data as script, markup, CSS, or a URL without proper protection.
- Be especially careful with escape hatches such as direct DOM manipulation.

## CSRF

- Protect state-changing browser requests with CSRF defenses where cookies are used for auth.
- Do not assume same-origin alone is sufficient.
- Require the right HTTP method and verify tokens or equivalent protections for unsafe actions.

## Secrets And Credentials

- Never commit secrets, private keys, tokens, or production credentials to the repository.
- Keep secrets out of logs, screenshots, and error messages.
- Rotate credentials if exposure is suspected.
- Use environment-specific secret storage or a vault instead of source control.

## Collector v2 Local Operations

- Keep SNMP community strings as environment references; never place production values in static or managed inventory.
- Bind collector status/control and TUI mutation operations to a Unix socket or localhost with OS access controls; do not add a public management HTTP endpoint.
- Require an explicit local confirmation and secret-free audit record for inventory, discovery-acceptance, threshold, or dependency mutations.
- Redact SNMP communities, MQTT credentials, TLS material, raw telemetry bodies, environment values, and secret-derived data from collector metrics, logs, diagnostics, and local messages.
- Limit discovery to configured CIDR allowlists and bounded target/rate/burst/concurrency settings.

## Database And Injection Safety

- Use parameterized queries for all application-generated database access.
- Never build SQL by concatenating untrusted strings.
- Keep database permissions scoped to the minimum required.
- Use database constraints as a defense-in-depth layer, not as the only validation.

## File Uploads And Untrusted Content

- Validate file type, size, and destination before accepting uploads.
- Store uploaded files under application-controlled names.
- Scan or inspect untrusted files when the threat model requires it.
- Never execute uploaded content.

## Logging And Error Handling

- Log security-relevant events such as authentication failures, authorization failures, and suspicious input.
- Do not log secrets, session tokens, or full sensitive payloads.
- Return generic error messages to users when detailed output would reveal internals.
- Keep security logs actionable and consistent.

## Dependency And Supply Chain Hygiene

- Keep dependencies current enough to receive security fixes.
- Review new dependencies before adding them.
- Remove unused packages and tooling.
- Treat transitive dependency changes as part of the security surface.

## Practical App Rules

- Do not add `dangerouslySetInnerHTML` unless there is a reviewed, sanitized use case.
- Do not rely on obscurity or front-end-only controls.
- Use TLS for network traffic that crosses trust boundaries.
- Apply rate limiting or abuse controls where repeated unauthenticated requests are possible.

## References

- OWASP Cheat Sheet Series: https://cheatsheetseries.owasp.org
- Authentication Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html
- Input Validation Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Input_Validation_Cheat_Sheet.html
- Cross-Site Scripting Prevention Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Cross_Site_Scripting_Prevention_Cheat_Sheet.html
- Session Management Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html
- Authorization Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html
- CSRF Prevention Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Cross_Site_Request_Forgery_Prevention_Cheat_Sheet.html

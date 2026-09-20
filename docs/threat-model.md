# Threat model

## Assets

TypeSafe and Twilio credentials, caller speech, forwarding numbers, service availability, and routing integrity.

## Trust boundaries and mitigations

- **Caller speech is untrusted.** The JEV request explicitly treats it as data and asks only a bounded Choice question. Routing remains simple Go code. Adversarial prompt-injection cases have opt-in live evaluations.
- **Public webhooks are untrusted.** Twilio HMAC signature validation is enabled by default. Production deployments must use HTTPS and restrict request/body sizes at their proxy.
- **Upstream services can fail.** Timeouts and bounded retries cover transient JEV failures. The default routing policy fails open; malformed output cannot silently reject a call.
- **Secrets can leak through logs or source.** Credentials come only from environment variables, `.env` is ignored, HTTP error bodies are not logged, and transcripts are not logged by default.
- **Caller data is sensitive.** The service does not record calls. The V1 in-memory store retains only the current session's transcripts and deletes them when routing completes. Operators should control process and log access.
- **Resource exhaustion is possible.** HTTP server timeouts and REST body limits are set. Deployments should add rate limiting at a trusted reverse proxy; the in-memory session store is intentionally single-instance V1 infrastructure.

## Out of scope

The service cannot verify whether a caller's factual claims are true, prevent abuse at the carrier layer, or replace jurisdiction-specific legal review. Users are responsible for applicable call-handling, disclosure, privacy, and telephony laws.

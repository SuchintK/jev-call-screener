# JEV Call Screener

An open-source call screening backend that asks callers why
they're calling, uses JEV to make a calibrated routing decision,
and forwards or rejects the call.

Twilio is currently supported as a telephony adapter, but the
screening engine itself is provider-independent.

```text
Caller → Twilio / future provider → Speech-to-text → ScreeningService → JEV
Promotional ──→ Reject
Wanted ───────→ Forward
Uncertain ────→ Clarify → Forward if still uncertain
```

The defaults are deliberately fail-open: classifier errors and uncertainty after one clarification forward the call rather than silently discard it.

## Demo

Run without a telephony provider, then classify caller text over HTTP:

```bash
curl -X POST http://localhost:8080/api/v1/classify \
  -H 'Content-Type: application/json' \
  -d '{"transcript":"I am calling to offer you a lifetime free credit card"}'
```

The JSON response includes JEV's category, confidence, probabilities, model, and the Go-owned routing action.

## How it works

1. A provider asks the caller why they are calling and transcribes the answer.
2. `ScreeningService` sends the untrusted transcript to JEV as a bounded Choice question.
3. Go routing policy rejects only high-confidence promotional calls, forwards high-confidence wanted calls, and otherwise asks one deterministic clarification.
4. A second uncertain result is forwarded. JEV errors fail open by default.

JEV classifies; it does not generate dialogue or control the call.

## Architecture

The core depends on ordinary Go ports, not Twilio or JEV SDK types:

```text
Twilio adapter ─┐
REST simulator ─┼→ ScreeningService → Classifier port → JEV HTTP adapter
Future adapter ─┘
```

See [architecture](docs/architecture.md) and the [threat model](docs/threat-model.md).

## Quick start

Requires Go 1.22 or newer.

```bash
git clone https://github.com/SuchintK/jev-call-screener.git
cd jev-call-screener
cp .env.example .env
# Edit .env. For REST-only development set TELEPHONY_PROVIDER=none.
make run
```

`make run` loads `.env`. Alternatively, export the variables and run `go run ./cmd/server`.

## Get a TypeSafe API key

Create a TypeSafe account and obtain an API key with access to the System One endpoint. Put it in `TYPESAFE_API_KEY`. Keys are sent as Bearer credentials only to the configured, pinned JEV endpoint and must never be committed.

## Configure environment variables

Important settings are shown in `.env.example`:

| Variable | Default | Purpose |
|---|---:|---|
| `PORT` | `8080` | HTTP listen port |
| `PUBLIC_BASE_URL` | — | Exact public HTTPS origin used for Twilio signatures |
| `TYPESAFE_API_KEY` | — | Required JEV credential |
| `JEV_MODEL` | `jev-1.13.0` | Pinned model version |
| `JEV_TIMEOUT_MS` | `3000` | Per-request HTTP timeout |
| `PROMOTIONAL_REJECT_THRESHOLD` | `0.90` | Minimum confidence to reject |
| `WANTED_FORWARD_THRESHOLD` | `0.75` | Minimum confidence to forward as wanted |
| `MAX_CLARIFICATION_TURNS` | `1` | Deterministic clarification turns |
| `FORWARD_ON_ERROR` | `true` | Fail-open classifier errors |
| `TELEPHONY_PROVIDER` | `twilio` | `twilio` or `none` |
| `FORWARD_TO_NUMBER` | — | Destination for accepted calls |
| `TWILIO_VALIDATE_SIGNATURE` | `true` | Verify public webhook requests |
| `LOG_TRANSCRIPTS` | `false` | Explicitly opt into sensitive transcript logs |
| `SCREENING_CONFIG_PATH` | — | Optional YAML rules/messages path |

Credentials are read only from environment variables, never YAML.

## Test without Twilio

Set `TELEPHONY_PROVIDER=none`, provide `TYPESAFE_API_KEY`, start the server, and use `POST /api/v1/classify` as in the demo. `GET /healthz` returns only `{"status":"ok"}`.

## Configure Twilio

Set `TELEPHONY_PROVIDER=twilio`, `PUBLIC_BASE_URL`, `TWILIO_AUTH_TOKEN`, and `FORWARD_TO_NUMBER`. Point the number's incoming Voice webhook at:

```text
POST https://YOUR_DOMAIN/webhooks/twilio/incoming
```

Full instructions are in [docs/twilio-setup.md](docs/twilio-setup.md). Startup never changes your Twilio account.

## Configuration

Routing rules are intentionally explicit:

```text
promotional and confidence ≥ promotional threshold → reject
wanted and confidence ≥ wanted threshold             → forward
clarification turns remain                            → clarify
otherwise                                             → forward
```

Set `FORWARD_ON_ERROR=false` only if you intentionally want classifier failures to clarify and eventually fail closed.

## Custom screening rules

Copy `config/screening.example.yaml`, edit category descriptions and caller messages, then set:

```dotenv
SCREENING_CONFIG_PATH=config/screening.yaml
```

This lets users define what they consider important without source changes. Do not put credentials in YAML.

## Privacy

The application does not record calls. Its V1 in-memory session store keeps only a session ID, turn count, caller responses, and creation time, then deletes the session after forwarding or rejection. Transcripts are not logged unless `LOG_TRANSCRIPTS=true`.

You are responsible for compliance with laws applicable to call handling, disclosure, recording, privacy, and telephony in your jurisdiction.

## Security

Twilio signature validation defaults to enabled, production webhooks must use HTTPS, and API/error logs omit credentials and upstream response bodies. Review [SECURITY.md](SECURITY.md) for private vulnerability reporting and [the threat model](docs/threat-model.md) before deployment.

## Docker

```bash
docker build -t jev-call-screener .
docker run --env-file .env -p 8080:8080 jev-call-screener
```

The final image runs as a non-root user and includes a health check.

## Testing

```bash
make test
make vet
make build
```

Unit tests use deterministic mock classifiers and no secrets. Live model evaluations are opt-in:

```bash
RUN_LIVE_JEV_TESTS=true TYPESAFE_API_KEY=... go test ./internal/adapters/jev -run Live
```

Public CI never runs live JEV or Twilio calls.

## Adding another telephony provider

Implement webhook verification, request normalization, and call-control rendering in a new adapter, then call the existing `ScreeningService`. Provider credentials remain in configuration and adapter packages. Exotel is a likely future option; no domain or routing changes should be needed. See [architecture](docs/architecture.md).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) and follow the [Code of Conduct](CODE_OF_CONDUCT.md).

## License

[MIT](LICENSE)

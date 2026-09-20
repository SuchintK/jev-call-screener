# Architecture

JEV Call Screener uses ports and adapters so call routing does not depend on a telephony vendor.

```text
Caller → telephony adapter → ScreeningService → Classifier port → JEV adapter
                                  ↓
                            RoutingPolicy
```

The domain contains only `Classification`, `Decision`, and call-session types. `ScreeningService` asks a `ports.Classifier` for a bounded classification; `RoutingPolicy` owns all forwarding, rejection, and clarification thresholds. JEV never generates caller dialogue.

Twilio webhook handlers translate form fields and TwiML at the edge. The in-memory store maps Twilio's Call SID to the provider-independent session ID and retains only turn count, transcripts, and creation time. Sessions are deleted after forwarding or rejection. Process restarts lose active sessions; those calls fail open.

## Adding another telephony provider

```text
New provider webhook / call-control adapter
        ↓
Normalize provider request into an internal call session
        ↓
ScreeningService
        ↓
Translate Decision into provider-specific call-control response
```

Add provider credentials, webhook verification, speech-field normalization, and call-control rendering inside a new adapter. Do not import provider SDK types into `domain`, `application`, or `ports`. Exotel is a likely future adapter.

# Twilio setup

1. Obtain a Twilio voice-capable number.
2. Copy `.env.example` to `.env` and set the TypeSafe and Twilio values, including `FORWARD_TO_NUMBER`.
3. Expose the application over HTTPS. For local testing, use a trusted HTTPS tunnel and set `PUBLIC_BASE_URL` to the exact public origin.
4. In the Twilio number's Voice configuration, set **A call comes in** to:

   ```text
   POST https://YOUR_DOMAIN/webhooks/twilio/incoming
   ```

5. Call the Twilio number.
6. Twilio asks why the caller is calling and transcribes their speech.
7. The backend invokes JEV and applies the Go routing policy.
8. Twilio forwards, asks one clarification, or terminates a high-confidence promotional call.

Keep `TWILIO_VALIDATE_SIGNATURE=true` in production. Signature validation uses `PUBLIC_BASE_URL`, the webhook path, and your auth token, so proxy URL rewriting must preserve the configured public URL. This application does not mutate your Twilio account or record calls.

You are responsible for compliance with laws applicable to call handling, automated disclosure, recording, privacy, and telephony in your jurisdiction.

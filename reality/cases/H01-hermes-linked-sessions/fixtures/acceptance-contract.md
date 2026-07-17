# Acceptance contract

1. Execute session A through the official Hermes CLI with a real
   OpenAI-compatible model provider.
2. Persist the completed turn through the Vermory Hermes lifecycle adapter.
3. Confirm the exact user observation that names `thesis-defense-v7.zip` as
   current and `thesis-defense-v6.zip` as obsolete.
4. Start a brand-new session B whose transcript contains neither filename.
5. Link session A and B explicitly through Vermory. Similar wording alone must
   not create the link.
6. Ask session B for the exact current upload bundle. The answer and injected
   delivery must contain `thesis-defense-v7.zip` and must not present
   `thesis-defense-v6.zip` as current.
7. Verify the delivery excludes unrelated Hermes and OpenClaw continuities and
   does not include raw session A transcript text.
8. Reverse the link and call the prepare endpoint directly for session B. The
   fresh delivery must no longer contain either thesis filename from session A.
9. Stop Vermory temporarily and run a separate Hermes turn. Hermes must still
   return a visible model answer, while the evidence must not claim that the
   turn was persisted by Vermory.
10. Keep credentials process-local. Evidence may record provider, model,
    response hashes, session identifiers, and lifecycle receipts, but never API
    keys or full environment dumps.

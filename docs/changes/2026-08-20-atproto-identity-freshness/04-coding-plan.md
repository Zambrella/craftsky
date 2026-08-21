# Coding Plan: AT Protocol Identity Freshness

1. Retain both the hardened `BaseDirectory` and its `CacheDirectory` wrapper in
   application composition; wire cached display and authoritative OAuth paths.
2. Add purpose-named resolver fields at the app/routes composition boundary.
3. Make exact mention and deletion intent resolution authoritative and
   write-through to `atproto_identity_cache`.
4. Route handle-targeted mutations through the authoritative resolver.
5. Add a bounded persistent-index refresh processor with durable retry timing,
   configuration, startup/shutdown wiring, and non-PII observations.
6. Verify focused behaviour first, then required-PostgreSQL/race/static gates.
7. Extend the durable Tap identity transaction to enqueue a DID-keyed immediate
   refresh with source event ordering and same-event redelivery idempotency.
8. Inject a local Indigo invalidation capability into Tap ingestion and the
   refresh processor. Purge DID, old handle, and new handle entries without
   performing remote work in the acknowledgement path.
9. Keep Tap handles out of the persistent index until the existing
   authoritative refresh worker verifies the current bidirectional mapping;
   retain periodic reconciliation as fallback.

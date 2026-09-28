# Subscription billing identity recovery

Craftsky's billing account belongs to a Craftsky DID in AppView. AppView stores a random `revenuecat_app_user_id` for that DID; each installation stores a copy of the DID and ID in its secure session registry. RevenueCat is identified with that AppView ID, not a device identifier. AppView's subscription access endpoint remains the authority for access.

## Expected new-device path

On a new installation, the user signs in with the original Craftsky account and explicitly selects it as the billing owner. `PUT /v1/billing/account` is idempotent by owner DID: it returns the existing ID if AppView still has the billing account. The app persists the returned ID before identifying an anonymous RevenueCat session. Confirm the existing license and assignment in AppView and the account's self-access after setup. A new installation must not be treated as a request to transfer a subscription to another DID.

Before enabling production purchases, run the new-device/reinstall test with a sandbox purchase on both supported stores: sign in as the original owner on a second installation, confirm the same AppView billing ID is returned, verify RevenueCat identifies as that ID, and verify the existing assignment and self-access. Test a different DID and a RevenueCat session already identified as a different user: neither may take over the original account. The wrong-owner store-restore test also requires the RevenueCat restore ownership policy to be configured.

## Triage a reported lock

1. Ask the customer to sign in again if the app says their sign-in expired (`401`). This is an authentication failure, not proof of a missing billing account.
2. For a billing lock, verify the authenticated owner's DID and check the AppView billing account for that DID using an authorized, read-only lookup. Check whether it is absent, inactive, or has an ID different from the one saved on the device. Do not collect session tokens or ask the customer to send the stored ID in a support message.
3. Compare the AppView ID with the RevenueCat customer under the correct project and environment, using authorized support tooling. Determine whether any store purchases or licenses exist and whether their assignment and AppView self-access agree. Treat a pre-existing *different* identified RevenueCat user on the device separately from a missing AppView account.
4. Record which check failed and the relevant request ID in the private support incident. Do not treat an empty subscription list or a fresh install as evidence that an existing billing identity can be replaced.

## Recovery boundary

If AppView still has the original billing account and the new installation is anonymous, the normal same-DID setup path is sufficient. If the saved ID differs, the billing account is missing, or RevenueCat is identified as someone else, leave purchases, restore, and assignment disabled. Do not clear the customer's app data, call `ensure` after a completed local binding, call RevenueCat `logOut`/`logIn` to override a mismatch, insert a replacement billing row, or reassign a receipt as a quick fix.

There is **no approved automated or operator mutation for those mismatches yet**. Escalate with verified account ownership, store transaction ownership where applicable, AppView database/backup history, and RevenueCat customer history. Design and review an explicit recovery operation and its audit trail before attempting a production repair. If AppView data was lost, consult the database recovery procedure in [production-render.md](production-render.md); inspect a backup in isolation first, since restoring the entire production database can roll back other customers. Recheck billing state, license assignment, and self-access after any approved recovery. Account deletion, original-owner loss, and transfer to a different DID require separate decisions.

# Browser notifications

Browser push is optional and uses the existing AppView Firebase Cloud Messaging
delivery pipeline. No PDS/OAuth credential is exposed to Firebase or browser code.
The browser stores only its FCM SDK registration and opaque per-account routing
bindings; it does not persist notification contents.

Set these **public build-time** environment variables before building the web
client. SvelteKit 3 declares them as optional static variables in `src/env.ts`:

```
PUBLIC_FIREBASE_API_KEY=
PUBLIC_FIREBASE_APP_ID=
PUBLIC_FIREBASE_PROJECT_ID=
PUBLIC_FIREBASE_MESSAGING_SENDER_ID=
PUBLIC_FIREBASE_VAPID_KEY=
PUBLIC_FIREBASE_AUTH_DOMAIN=
```

The first five are required to enable the settings control. `AUTH_DOMAIN` is
optional. Use a Firebase web application in the same project configured by the
backend's existing `FIREBASE_PROJECT_ID` and `PUSH_ENABLED`. Configure Web Push
certificates in that project's Cloud Messaging settings and supply its **public**
VAPID key. Firebase manages the private VAPID key. Never put a Firebase service
account or any private key in a `PUBLIC_` variable.

The user explicitly enables notifications in Settings → Notifications. Browser
permission is requested only from that button. On later visits, already-enabled
accounts renew their registration when permission is still granted. Disabling
one account removes only its server subscription. FCM's browser token is removed
when no enabled account remains. Server logout/account deletion already cancels
the associated subscriptions through owner lifecycle cleanup.

`src/service-worker/index.ts` is bundled with the application. Configure
SvelteKit's `serviceWorker.register` as `false`; the push client registers it
explicitly when needed. The worker has no fetch handler or application cache.
FCM displays each background notification once and opens the server-generated
HTTPS link. The shell resolves that link's subscription against local account
bindings, selects the authenticated account, then opens activity or resolves the
opaque chat reference through authenticated AppView APIs. An unknown binding
never falls back to the currently selected account's private message request.

Serve the web app over HTTPS. Install the PWA on the Home Screen for iOS/iPadOS
push. Browser support is checked with Firebase before registration. This requires
the manifest link and PNG icon in the document head. All SDK code is bundled;
remote Firebase scripts and `unsafe-eval` are not required. The existing HTTPS
`connect-src` permits Firebase registration requests; `worker-src 'self'` permits
the bundled service worker.

Apply AppView migration `000113_browser_push` before enabling web push. It adds
`web` to the installation platform constraint without changing existing native
registrations. The sender sets a Web Push TTL and HTTPS click URL, preserving
unique-event delivery with no collapse metadata.

Validation covers API platform admission, sender JSON/TTL/click routing, malformed
payloads, account binding isolation, and per-account registration replacement.
Actual Firebase delivery and browser permission/OS behavior still require a
configured HTTPS deployment and physical browser/device verification.

Official references:

- https://firebase.google.com/docs/cloud-messaging/web/get-started
- https://firebase.google.com/docs/cloud-messaging/web/receive-messages
- https://firebase.google.com/docs/reference/fcm/rest/v1/projects.messages

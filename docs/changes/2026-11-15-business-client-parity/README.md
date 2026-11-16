# Business editing and settings parity

Business subscribers manage public business details at **Settings → Business → Business details** on mobile and web (`/settings/business`). Personal Edit profile no longer includes business types, offerings, tagline, hours, service area, locality or primary action. Native reuses the versioned form, independent business save, product/catalogue preservation, explicit conflict recovery, and unsaved-work protection. The new route uses confirmed Business access and closes when that access lapses. Editor form state is keyed to its owner and purpose.

The web product manager allows a subscriber to add their first product without a separate declaration setup step, as mobile does. Its editor has a large desktop dialog and full-screen compact layout, explicit close, image preparation/cropping/description, optional amount and currency, and a persistent Save action. The event manager keeps Create event at the bottom, matching the mobile action placement. Product menus provide edit, move up/down and confirmed removal outside the editor; they cannot save an unfinished editor draft. A suggested currency does not create a price when amount is blank. Cancel and navigation protect unsaved edits. Failed list mutations restore accepted products; uncertain writes retain their original retry closure and operation key.

The web event editor uses native wording, role chips, optional media, explicit close, and a persistent Save action. Existing all-day and IANA timezone handling remains available. Owner management and public profiles use the same event card, including date/time, venue, mode and roles, with owner actions beside the title. Public product cards already used the native image/title/price arrangement; profile empty-state wording now matches mobile.

Business records, authorization, subscription ownership, API routes and Lexicons are unchanged. Business detail saves preserve current server products and unknown catalogue entries. Test fixture support for first-declaration creation does not change the server contract.

## Verification

- Inspected the running local web app and iPhone simulator; compared product/event editors and verified the dedicated settings page on both.
- 239 native business/profile/route tests passed, plus the focused settings visibility check. A final 53-test profile/route rerun passed after the editor-owner key was added. These tests cover independent saves, products/catalogue preservation, conflict recovery, discard, accessibility sizes and subscription access guards.
- Nine browser mutation/navigation cases and ten compact/desktop-dark layout/accessibility cases passed. Browser fixture tests cover business detail preservation, image preparation, event create/edit/delete, cancellation without writes, first product creation at desktop/compact widths, ordering and confirmed removal.
- Checked native analysis, web type checking, lint and build; applied native hot restart/reload.

These are local checks and fixture-backed mutation tests. No public business product or event was published for manual verification, and no deployment was performed.

An exploratory full `settings_page_test.dart` run still times out in the unrelated existing Free customisation paywall test, which expects the former intermediate subscription prompt. The changed Business settings visibility test passes.

## Business details composer parity follow-up

Removed the web form's outer card so fields sit directly on the page. Save is a text action in the sticky top-right app bar and submits the versioned business form; it remains disabled until there are changes. The top helper and selector guidance match Flutter exactly ("Choose up to 5." and "Choose up to 10."). The hours field is labelled "Hours" and the empty action is "No primary action".

Tagline, Hours, Service area and Town or locality display live counters with limits of 100, 300, 200 and 100 characters. They use the existing shared committed-edit input limiter to preserve complete emoji/combining characters, enforce the matching UTF-8 limits, and respect IME composition. Existing stored values are not truncated at mount; save-time validation remains in place. Country and locality are stacked as in the mobile form.

Browser coverage checks the bare form, exact helper copy, header Save location/disabled state, all four limits and Unicode counting at desktop and compact widths. The existing business save/reload test verifies that the header action still preserves products and unknown catalogue values.

## Business details field hints

Both clients now show matching examples for business types, offerings, tagline, hours, service area, country code and locality. The primary-action selector explains what the choice does. Its destination field suggests an HTTPS shop URL, switching to a mailto example when Email is selected. These are placeholders only; they do not prefill or save example values. Native copy is localized through the generated ARB resources.

The mobile primary-action field now initializes absent actions to the explicit "No primary action" option, matching web. Its separate clear button is disabled so it cannot return to an unlabeled null selection. Choosing an action and then No primary action returns an otherwise untouched form to its clean state. Shared selectors retain their existing clear behavior by default.

Placeholder wording is now identical across clients: text examples omit the "e.g." prefix, and dropdown prompts say "Select Business types", "Select Offerings" and "Select Primary action". Existing selected values, including "No primary action", remain visible when the selector is closed.

## Conditional primary-action destination

Both clients hide the destination while No primary action is selected. Selecting an action reveals a matching label and example: Shop URL, Patterns URL, Custom order URL, Booking URL, Event calendar URL, Email address, Website URL or Wholesale enquiries URL. Switching to No primary action clears the destination so it cannot later reappear or block a save through hidden validation.

Email editing uses a plain address and email keyboard/input. Existing mailto destinations are displayed without the scheme and are converted back when saving, preserving the public-record contract. Business detail removal still preserves products and unknown catalogue values. Regression checks cover hidden/visible transitions, invalid draft clearing, removal of an existing action, and email editing/save/reload on both clients.

## Business About presentation

The web profile's About section now follows Flutter's typography and chip styling: 12px semibold headings with 1.68px letter spacing, 14px body text with 1.5 line height, and 13px semibold pills with 1.5px ink borders and 8px/12px padding. Chip wrapping uses 8px horizontal and 4px vertical gaps. Section spacing remains 20px, with an 8px heading-to-chip gap and a 4px heading-to-body gap. Offerings and Joined use the same wording and heading treatment as mobile; multiline text retains its line breaks. These changes are scoped to About presentation, preserving the compact summary above the profile actions.

The web profile Business label now uses the primary colour. Native business-profile action buttons adopt the web CTA treatment: neutral ink text/icon, surface fill, a 1.5px pill outline, 44px visual minimum height, 8px/14px padding, 15px semibold text and an 18px icon, without elevation. The existing external-link confirmation flow remains in place.

## Events tab bar

The web Events manager follows Flutter's tab bar: equal-width 48px targets, 13px semibold labels, ink for the selected label and a 3px ink indicator that spans only its text. Inactive labels use muted ink. Tabs start immediately below the page header and have no bottom divider. Existing Upcoming/History filtering and pressed-state accessibility are preserved. Verified against the open iPhone simulator and web page, with browser checks for filtering/event management and compact/desktop-dark layout and accessibility.

## Event composer parity and validation

The web composer now follows the mobile header (close on the left), field spacing, outlined role group, required labels and single persistent Create event/Save event action. Bounded name, summary and venue fields show matching character counters. Both clients use the same example placeholders and Select dropdown prompts; web's empty date controls show the same date/time guidance while retaining browser date inputs. The shared image-description hint remains available in both clients.

Mobile roles now use a FormField validator rather than rendering an error whenever the selection is empty. A fresh composer has no role error; submission or interaction with an empty selection triggers validation, and selecting a role clears it. Both clients limit selection to four roles, and the web role error is attached to its field group and cleared on selection.

The existing event Lexicon explicitly supports `isAllDay`, `timeZone`, inclusive `startsAt` and exclusive `endsAt` at local midnight. No schema or API change was needed. Mobile now exposes the existing all-day capability with date-only pickers, date-only display and matching guidance that End is the first date after the event. Switching a same-day timed event to all-day advances the end to the following calendar date. UTC conversion remains in the existing timezone service, including daylight-saving transitions.

Validation: 28 native editor/draft tests passed, including initial role validation, date/time versus date-only picking, same-day conversion, exclusive end preservation and spring/fall DST. Eleven browser checks passed, covering event creation/edit/delete, hints, inline validation without writes, all-day ambiguous retries, compact draft protection and compact/desktop-dark accessibility. The open browser and simulator were compared and the mobile toggle was exercised without publishing an event. Native analysis, web type checking/lint/build and documentation checks passed; mobile was hot reloaded.

## Routed web event composer

Web event creation now opens `/compose/event`, and editing opens `/compose/event/edit?event=…`, following the New post page pattern instead of a modal. The authenticated shell presents a fixed header with Close, a scrolling form and a persistent bottom action. Events settings navigates to these routes; direct loads and refresh retrieve the edited event for the current account, retaining its CID. Missing events show a retryable load error rather than an empty create form. Business access checks still apply.

Close, browser Back and navigation protect unfinished drafts. Close and successful saves use the shell's return navigation to reach Events without leaving an obsolete composer in browser history. Successful ambiguous retries also return only after their frozen original write completes. Image editing retains its separate crop/review dialog. Flutter is unchanged by this web-only routing follow-up.

Validation: route/title unit checks, web type checking, lint/build and 12 event browser cases passed. Browser coverage includes create and edit refresh, Back/discard, missing-event handling, role validation, all-day ambiguous retry, compact action visibility and accessibility. Inspected the route in the open Browser panel without publishing a real event.

## Shared web event photos

The event composer uses the same `PhotoAttachments` component as post/project composition, with `maxImages=1` and Back to event wording. Post/project defaults remain eight photos. The shared picker, drag/drop, summary, photo-details navigation, crop/rotate, alt text, removal and clear action are reused. A full event slot hides the picker, multiple-file drops are rejected, and single-photo details omit reorder controls. Returning from details restores the event form's scroll position.

Photo selection and edits remain local until event submission. Existing images retain their blob reference when only the description changes; removing the photo omits it from the replacement event record. Prepared images upload once, and the resulting blob reference is cached by Blob identity so failed event writes and exact retries reuse it. Saving and navigation remain guarded during preparation/cropping. No schema, API or Flutter changes are required.

Validation: nine browser cases passed, including compact/desktop event photo selection, rejection of multiple-photo drops, crop/rotate, alt-text retention, browser Back, accessibility, single upload across event retries, existing-image description/removal without uploads, and the existing post/project and product image flows. Inspected the generated compact/desktop photo-details screenshots. Web type checking, lint, build and documentation checks passed.

## Shared Flutter event photo flow

Flutter events now use the shared post/project attachment widget's Photo details route instead of inline attachment controls. Selecting a photo opens details automatically; returning shows the thumbnail summary and described count. The limit remains one photo, with single-photo guidance and no reorder controls. Descriptions, removal, clear and replacement continue through the shared widget, and updates reach the details page while the event form is covered.

Newly selected event photos can open the same crop/rotate editor as posts, in composer mode. Edited output passes the existing image preparation and upload-size validation before replacing the pending photo. Cancel or failure retains the previous photo and description. Account/lifetime checks fence late editor results. Photo bytes remain local until event submission, and the existing upload/mutation retry behavior is retained.

Validation: 53 native event, shared-attachment and image-editor tests passed, including apply/cancel/failure, description retention on returning/reopening, one-photo controls, removal and upload retries. Native analysis passed and the running app was hot reloaded. Host image tests use the checked-in codec substitute; no event was published.

On the iPhone simulator, selecting a local photo opened Photo details automatically. Edit photo opened the shared native editor; Rotate 90° and Done exported successfully and returned the edited image to details. The temporary selection was removed after verification without submitting an event. Android device behavior was not manually exercised.

## Flutter event detail polish

Status/lifecycle pills place their icon and label in one content row, removing the Chip avatar slot's extra leading inset. Icons in the event information card and event list cards use the theme primary colour. Event website is a flat surface-filled outlined pill; Register uses the existing primary ChunkyButton with its hard-offset shadow. Both actions use an external-link icon, matching the supplied reference, and retain the existing external-destination confirmation flow. This follow-up is scoped to Flutter event details.

Validation: all 16 event-detail tests and 21 event-list/settings tests passed, including compact/large-text layouts, access controls, reporting and exact external action destinations. Native analysis passed and the running app was hot reloaded. Visually verified the status pills, primary information icons and both CTA styles on the open iPhone simulator event detail page.

## Event overview card clipping and alignment

Flutter's shared CraftskyCard now subtracts its 1.5px border inset from the inner Material clip radius. The image therefore meets the rounded ink border without the paper sliver caused by applying the outer radius again to the inset child. Outer radius, border, shadow, padding and explicit Clip.none behavior remain in place. Event overview metadata rows centre icons beside their text on both Flutter and web. Web metadata icons also use the primary colour to match mobile. Confirmed the corrected upper corners and alignment on the running iPhone simulator.

Validation: 39 native event-list, settings, details and product-card tests passed; four compact/desktop-dark web event layout/accessibility checks passed. Native analysis, web type checking/lint/build and documentation checks passed. Applied the final native hot reload.

## Mobile upcoming events footer

Removed the end-of-list message and standalone Refresh action from the mobile Upcoming Events tab. The footer now appears only for pagination progress, Load more or error feedback/retry. Profile pull-to-refresh remains available. Web is unchanged.

Validation: all seven upcoming-events widget tests and focused native analysis passed. Hot reloaded and confirmed the footer is absent on the iPhone simulator.


## Web event detail visual parity

Compared the open event detail page with the iPhone simulator. Web now uses the Flutter layout: a 16:9 photo header with centred title, top/bottom gradient and overlaid back/menu controls; scrolling pins the title on a paper app bar. Events without photos use the title app bar directly. Summary spacing, outlined status pills, native status/lifecycle/people/owner icons, heavier information titles, dividers and the bounded information card match mobile. Website and registration use the matching 48px pills, with a flat outlined website action and primary registration shadow. The redundant organiser link is removed; the back action still returns to the organiser's events. Existing owner-only metadata, menus, reporting and external-link confirmation remain in place. Flutter is unchanged.

Validation: compact and desktop-dark visual/accessibility cases passed, plus photo/no-photo interaction checks covering header collapse/restore, menus, external-link confirmation and organiser navigation. Web type checking (zero warnings), lint, formatting and production build passed. Visually inspected the running web page at compact and desktop sizes against the open iPhone simulator.


## Event creator context and owner actions

Both event detail clients now link directly to the creator's profile and show their public event roles to visitors, labelled Creator's roles; owners retain Your role and publication metadata. Mobile uses the typed profile route and web uses the creator DID's profile URL. Web's context menu adds Send via chat, passing the canonical event link to the existing recipient picker and draft-review flow.

Owners now have Edit event and Delete event menu actions in both clients. Edit reuses the existing composer. Deletion requires the existing destructive confirmation wording, passes the displayed CID and an idempotency key, and reconciles/navigates after acceptance. Native deletion captures and rechecks the active account around confirmation, uses the shared mutation controller, and displays failure/conflict feedback with Reload. Web retains the immutable delete request for Work's existing ambiguous retry behavior. Visitors keep reporting and cannot see owner mutation actions.

Validation: 18 native detail tests passed, including creator-profile navigation, visitor role visibility, owner editor opening, no visitor edit/delete, and cancelled/confirmed deletion with the displayed CID. The 14 events-settings tests also passed. Six web browser cases passed: compact/desktop-dark accessibility, header with/without photo, owner edit/save/delete using the refreshed CID and a visitor creator-link/roles/chat-draft flow. Web type checking, lint and build and focused native analysis passed. Hot reloaded mobile; verified the creator profile link and visitor menu on the iPhone simulator and the owner menu in the open web page. No real event was deleted and no chat message was sent; mutation checks used isolated test fixtures.


## Inline creator attribution

Visitor event details now replace Creator's roles with a 24px creator avatar and linked display name beneath the roles. Both clients trim the display name and fall back to @handle when it is missing or blank, using existing profile reads and avatar components. The duplicate standalone creator-profile action is omitted where the inline attribution is present. The owner's Your role label is retained. A profile read never prevents the event itself from rendering; the profile action remains available while the identity loads.

Validation: 19 native detail tests and four compact/desktop accessibility and creator-identity browser cases passed, including profile navigation, whitespace-name fallback and the web avatar's 24px size. Focused native analysis, web check/lint/build and diff checks passed; the running mobile app was hot reloaded.


## Canonical event timestamps and validation feedback

Investigated the local server's four 422 event-edit responses on 16 November. The web timezone formatter emitted millisecond UTC strings (`.000Z`), while AppView's existing event validator requires canonical whole-second UTC strings (`Z`). The temporal validation envelope serializes to the same 195 bytes recorded for the failures. The formatter now emits the server's canonical format for both creates and edits; timezone/DST conversion and exclusive all-day boundaries remain intact. The browser fixture now rejects noncanonical event instants, closing the gap that allowed previous mock-backed edits to pass. A server validator regression case covers zero-valued fractional seconds explicitly.

The Start/End input names now match startsAt/endsAt in the API error envelope so temporal validation failures can be displayed inline instead of falling back to a generic error. No production mutation or backend behavior change was needed. The user's real edit was not resubmitted as part of this server investigation.

Validation: server instant/all-day validator tests and all 15 web settings-model tests passed. Browser checks passed for create/edit/delete, owner-detail edit/delete, all-day exact retry, and inline server time errors/native required fields. Web type checking, lint/build and diff checks passed.


## Web event postpone and cancel actions

Owner menus on event details and Settings → Events now expose Postpone event and Cancel event, matching the existing native management actions. Each action is hidden when the event already has that status and blocked while another write or retry is pending. Status updates retain the public event fields and photo, use the displayed CID and one operation key, and reuse Work's failure/retry handling. Accepted responses update the displayed status/CID immediately; management removes cancelled/postponed events from Upcoming while retaining them in History. Visitor menus do not expose either action. Flutter is unchanged.

Validation: focused browser checks cover management postponement followed by detail cancellation using the accepted CID, preservation of optional fields/photo, redundant-action hiding, visitor restrictions and compact/desktop accessibility. Web check/lint/build and diff/documentation checks passed. No real event was changed during verification.


## Owner event-management sharing

Added Copy link and Send via chat to each owner context menu in the web Settings → Events list, above the editing/status/deletion group. Event details and management now share one canonical event-link helper. Copying uses the existing failure/notice handling; chat navigation opens the existing recipient picker and draft-review flow. Both new controls are blocked during pending management writes/retries. Native owner menus already included the shared link-sharing group and required no change.

Validation: the management browser test verifies the copied canonical URL, chat recipient-picker destination and subsequent lifecycle actions; the photo/no-photo event-detail checks also passed. Web type checking, lint/build, diff and documentation checks passed. Clipboard writes were intercepted in the isolated test; no real chat message was sent.


## Event composer optional labels and role dropdown

Removed the explicit (optional) suffix from Summary, Venue, Event link and Registration link in both clients. Your role now reuses the existing searchable multi-select input in each client instead of a standalone role chip group. It shows selected roles as removable tokens, retains existing/unknown selections, permits up to four roles, and validates the required selection after interaction/submission. Both selectors show Select roles guidance and the existing four-role helper. The shared selectors gained an opt-in required marker; other callers retain their existing defaults. Required indicators remain on the event's required fields.

Validation: native event-editor and shared multi-select tests passed (40 checks), covering required selection/removal, unknown-role correction, existing selected roles, compact/large-text layouts and photo flows. Thirteen event browser checks passed, including required/error recovery, labels/placeholders, compact accessibility, image flows, create/edit, all-day retries, routing and owner actions. Focused native analysis and web check/lint/build passed; mobile was hot reloaded. Browser fixtures now interact with the dropdown's transient checkboxes rather than the removed role-chip controls.


The web multi-select's required marker now shares the existing field-label baseline layout, 8px gap and required-note typography. This prevents settings label grid styles from moving the marker onto its own line. The rule is scoped to the shared component's heading label; option descriptions keep their own styling. Verified with the event-composer browser/layout/accessibility check and web check/lint/build.

# Business identity icons

Business accounts show a storefront immediately before the display name on
mobile and web, including compact identities and expanded profiles. Craft icons
remain after the name. The business marker is separate from the craft list and
its overflow count, so it stays visible when craft glyphs collapse into +N.
Business-only identities work without any crafts. Accessible labels identify
the marker as Business; regular or unknown account types have no marker.

Shared icon components compensate for the storefront glyph's internal padding
with 1.25 paint scaling. Compact markers retain a 16px layout slot and match the
painted craft size; expanded-name markers use a 24px slot. Native and browser
layout checks protect spacing, alignment and narrow/scaled text. Web no longer
shows the separate Business account text or Business craft chip. The web
Business details settings tile matches mobile's identification-badge icon.

The storefront uses the profile owner's selected colour, with the palette's
dark accent in dark mode and cobalt when no profile colour is available.
Shared compact and expanded identities receive the owner's customisation;
web post projections, settings and account-switcher summaries retain it too.

On mobile, the storefront has a separate hero paired with the existing profile
identity transition. It scales from the compact marker into the expanded marker
and returns on Back. Business status travels in the in-memory transition so the
loading profile provides a matching destination before the request completes.
A stable destination key survives loading-to-loaded replacement; flights retain
the owner's theme colour and respect reduced motion. Repeated identities keep
their separate source tags.

Native post authors, account summaries and search results decode the existing
AppView accountType field and preserve it through copies and summary projection.
Post, quote/composer, profile list, loaded settings and chat identities pass the
known status into the shared component. Web settings and account-switcher
projections retain their already-loaded accountType. No extra per-card requests,
API routes, Lexicon changes, PDS craft changes or migrations are required.
Unknown status is not inferred from a name, declaration or cached billing tier.

Validation for the final name-leading layout: 65 native profile/identity tests,
five web identity unit tests and four phone/desktop browser cases passed. Browser
checks measure the painted storefront path against a craft icon, confirm the
marker is before the name, and check the exact storefront/settings SVG paths.
Regular craft order and overflow remain covered. The web header screenshot was
inspected; native analysis, web type checking, lint and the production build
passed. The running iPhone simulator app was hot reloaded while preserving the
user's draft. No physical-device/Android manual check or deployment was performed.

Profile-colour follow-up validation: 65 native tests and all 599 web unit tests
passed, plus two browser cases checking a rose business author at phone and
desktop widths. Native analysis, web type checking, lint and build passed, and
the running simulator was hot reloaded.

Storefront hero follow-up: 36 mobile hero/compact-identity tests passed, covering
duplicate sources, cold loading, hydration during flight, Back, light/dark colour
and reduced motion. Static analysis passed and the simulator was hot reloaded.

Earlier metadata propagation validation passed 209 focused native identity,
profile/post/summary/settings/chat tests and the 600-test web unit suite before
the final marker-placement refinement. Generated native mappers were updated
while unrelated generated changes were restored.

# Flutter Image Caching Strategy

## Summary

Replace raw `NetworkImage` use in `ProfileAvatar` and `ProfileBanner` with `cached_network_image`, backed by two named `flutter_cache_manager` instances: a long-lived "profile" cache for avatars + banners (any user's), and a shorter-lived "feed" cache reserved for future post-image content. The split is by image *kind* (small + reused vs. large + sequentially consumed), not by *whose* image. Cache managers are exposed via Riverpod providers so test code can override them.

## Why now

The two image-rendering widgets in the app today (`profile_avatar.dart`, `profile_banner.dart`) use `DecorationImage(NetworkImage(url))`, which means:
- No disk caching — every screen mount re-downloads.
- No loading state — the user sees the swatch colour, then a hard pop.
- No error handling — a failed load throws a Flutter exception that paints onto the rendered tree.

The blob-upload work for post images is already flagged as deferred future work in [`2026-04-21-appview-api-architecture-design.md`](2026-04-21-appview-api-architecture-design.md). When it lands, the feed will need image caching too. This spec puts the infrastructure in place now and refactors the existing two call sites; adding feed-image call sites later is one line per call site.

## Key property: Bluesky CDN URLs are content-addressed

Avatar and banner URLs synthesised by the AppView take the form `https://cdn.bsky.app/img/{avatar|banner}/plain/{did}/{cid}@{ext}` (see the [profile-onboarding spec](2026-04-23-profile-onboarding-design.md), "Avatar/banner URL synthesis"). The `cid` is a hash of the image bytes — when the user changes their avatar, the URL changes too. **Cached entries can never go stale-but-wrong.** This makes long-lived caching safe and removes any need for cache-busting logic.

## Non-goals (v1)

- **Caching for post / feed images.** No call sites yet. The `FeedImageCacheManager` is set up so adding them later is a one-line change, but no UI work targets it in this spec.
- **Manual cache warming.** No precaching at app launch. First view of an image triggers the download.
- **Image transformation / resizing.** Bluesky's CDN serves appropriate variants per path (`/avatar/`, `/banner/`).
- **Tuning Flutter's in-memory `ImageCache`.** The framework's decoded-bitmap cache is orthogonal and works fine at defaults.
- **Sharing the cache between platforms or with web builds.** Web has its own caching story; this spec targets mobile and desktop.

## Architecture

### Two managers, one library

`flutter_cache_manager` exposes `BaseCacheManager` (abstract) and `CacheManager` (the standard implementation). Each manager subclass owns a SQLite-backed metadata store and a filesystem subdirectory keyed by `Config.cacheKey`. Two managers = full isolation: profile entries cannot be evicted by feed-image churn, and `stalePeriod` / `maxNrOfCacheObjects` are tuned independently.

| Cache | `cacheKey` | `stalePeriod` | `maxNrOfCacheObjects` | Contents |
|---|---|---|---|---|
| Profile (long) | `craftskyProfileImages` | 90 days | 500 | All avatars + banners (any user) |
| Feed (short) | `craftskyFeedImages` | 7 days | 300 | Feed post images (future, no v1 call sites) |

#### Why split by kind, not by user

Avatars and banners are small (~10–50 KB and ~100–300 KB respectively) and reused across screens — every PostCard, comment, notification, search result repeats the same avatar URLs. Feed post images, when they land, will be much larger and sequentially consumed. If both kinds shared one cache, the feed bulk would evict reusable identity images under LRU pressure and the user would pay network for the cheapest, most-reused content. Splitting by kind avoids that.

The CID-immutability property applies to anyone's avatar, not just the signed-in user's, so "long-lived" being a function of "the signed-in user" was a false constraint.

#### Disk math at the recommended numbers

- Profile cache worst case: 500 avatars × ~30 KB ≈ 15 MB, plus ~50 banners × ~200 KB ≈ 10 MB → **~25 MB**.
- Feed cache worst case: 300 post images × ~500 KB ≈ **~150 MB** (rough; actual size depends on the future post-media spec).

Both fit comfortably under typical device cache budgets.

### File layout

```
app/lib/shared/image/
  image_cache_managers.dart      # Singleton CacheManager subclasses
  image_cache_providers.dart     # Riverpod providers exposing them
  image_cache_providers.g.dart   # generated
  clear_image_cache_provider.dart    # Mutation that empties both caches
  clear_image_cache_provider.g.dart  # generated

app/lib/settings/widgets/
  clear_image_cache_tile.dart    # Settings list tile that triggers the mutation
```

`shared/` matches the existing convention (`shared/api/`, `shared/device/`, `shared/widgets/`) — image caching is infrastructure consumable by any feature, not feature-scoped. The settings tile lives next to the existing `sign_out_tile.dart`.

### Singleton + Riverpod-provider pattern

Cache managers are singleton classes with private constructors. Riverpod providers return the singleton instance.

```dart
class ProfileImageCacheManager extends CacheManager {
  static const _key = 'craftskyProfileImages';
  static final ProfileImageCacheManager _instance =
      ProfileImageCacheManager._();
  factory ProfileImageCacheManager() => _instance;

  ProfileImageCacheManager._() : super(
    Config(
      _key,
      stalePeriod: const Duration(days: 90),
      maxNrOfCacheObjects: 500,
    ),
  );
}

class FeedImageCacheManager extends CacheManager {
  static const _key = 'craftskyFeedImages';
  static final FeedImageCacheManager _instance = FeedImageCacheManager._();
  factory FeedImageCacheManager() => _instance;

  FeedImageCacheManager._() : super(
    Config(
      _key,
      stalePeriod: const Duration(days: 7),
      maxNrOfCacheObjects: 300,
    ),
  );
}
```

```dart
// image_cache_providers.dart
@riverpod
BaseCacheManager profileImageCacheManager(Ref ref) =>
    ProfileImageCacheManager();

@riverpod
BaseCacheManager feedImageCacheManager(Ref ref) => FeedImageCacheManager();
```

The singleton is required because `flutter_cache_manager` keys SQLite + the disk directory by `cacheKey`; instantiating two managers with the same key in one process is wasteful at best and racy at worst. The Riverpod provider gives us DI consistency with the rest of the app and makes test overrides trivial. The provider's return type is the abstract `BaseCacheManager` so test fakes do not need to subclass our singletons.

## Refactored widgets

### `ProfileAvatar`

Becomes a `ConsumerWidget`. The chunky border + drop shadow stay on the outer `Container`. A `ClipOval` holds either the existing initial-letter fallback (when `avatarUrl == null`) or `CachedNetworkImage` whose `placeholder` and `errorWidget` are *the same* fallback. The user sees the initial letter in all three "no-face" states (no URL, loading, error), with the real avatar fading in over the letter on first view.

The fallback is extracted to a private `_AvatarInitialFallback` widget within the same file so the three call sites share an identical implementation.

```dart
class ProfileAvatar extends ConsumerWidget {
  const ProfileAvatar({
    required this.seed,
    this.avatarUrl,
    this.size = ProfileAvatarSize.medium,
    super.key,
  });

  final String seed;
  final String? avatarUrl;
  final ProfileAvatarSize size;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final shadows = theme.extension<BrandShadowTheme>()!;
    final dimension = size.dimension;
    final borderWidth = size.borderWidth;

    final fallback = _AvatarInitialFallback(
      seed: seed,
      dimension: dimension,
      backgroundColor: swatches.butter,
      foregroundColor: theme.colorScheme.onSurface,
    );

    return Container(
      width: dimension,
      height: dimension,
      decoration: BoxDecoration(
        shape: BoxShape.circle,
        color: swatches.butter,
        border: Border.all(
          color: theme.colorScheme.onSurface,
          width: borderWidth,
        ),
        boxShadow: size.shadowsFrom(shadows),
      ),
      child: ClipOval(
        child: avatarUrl == null
            ? fallback
            : CachedNetworkImage(
                imageUrl: avatarUrl!,
                cacheManager: ref.watch(profileImageCacheManagerProvider),
                fit: BoxFit.cover,
                width: dimension,
                height: dimension,
                placeholder: (_, __) => fallback,
                errorWidget: (_, __, ___) => fallback,
              ),
      ),
    );
  }
}

class _AvatarInitialFallback extends StatelessWidget {
  const _AvatarInitialFallback({
    required this.seed,
    required this.dimension,
    required this.backgroundColor,
    required this.foregroundColor,
  });

  final String seed;
  final double dimension;
  final Color backgroundColor;
  final Color foregroundColor;

  @override
  Widget build(BuildContext context) {
    final initial = seed.isEmpty ? '?' : seed.characters.first.toUpperCase();
    return Container(
      color: backgroundColor,
      alignment: Alignment.center,
      child: Text(
        initial,
        style: Theme.of(context).textTheme.displaySmall?.copyWith(
          fontSize: dimension * 0.5,
          color: foregroundColor,
        ),
      ),
    );
  }
}
```

`fadeInDuration` keeps its 500ms default — a brief cross-fade from initial letter to face is the "graceful loading state" we want.

### `ProfileBanner`

Same pattern, simpler. No fallback widget needed because the colour swatch is the `Container`'s `color` and the image goes on top. During loading and on error, render `SizedBox.shrink()` so the colour shows through.

```dart
class ProfileBanner extends ConsumerWidget {
  const ProfileBanner({
    required this.color,
    this.bannerUrl,
    this.height = ProfileBanner.defaultHeight,
    super.key,
  });

  static const double defaultHeight = 160;

  final Color color;
  final String? bannerUrl;
  final double height;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Container(
      height: height,
      width: double.infinity,
      color: color,
      child: bannerUrl == null
          ? null
          : CachedNetworkImage(
              imageUrl: bannerUrl!,
              cacheManager: ref.watch(profileImageCacheManagerProvider),
              fit: BoxFit.cover,
              placeholder: (_, __) => const SizedBox.shrink(),
              errorWidget: (_, __, ___) => const SizedBox.shrink(),
            ),
    );
  }
}
```

### Public API preserved

- Constructor signatures of both widgets are unchanged. Call sites in `PostCard`, `ProfileSliverAppBar`, `EditProfileBannerAvatar`, etc. need zero modifications.
- `ProfileAvatarSize` enum, dimensions, border weights, shadow logic — preserved verbatim.
- `ProfileBanner.defaultHeight` and the `Color color` swatch parameter — preserved.

## "Clear image cache" settings tile

A new `ClearImageCacheTile` is added to `SettingsPageBody` next to `SignOutTile`. Tapping it empties **both** cache managers — one tile, one action. Splitting into per-cache tiles would expose internal infrastructure to the user without giving them a useful choice.

### UX

- Tile label: "Clear image cache".
- Leading icon: `Icons.cleaning_services_outlined`.
- Tap → fires the mutation. The tile is `enabled: false` while the operation is in flight (matches the `SignOutTile` pattern).
- On success: a `SnackBar` with "Image cache cleared".
- On failure: a `SnackBar` with the error.
- **No confirmation dialog.** The action is reversible — images re-download from the CDN on next view. A confirmation step would be friction without value.

### Provider shape

A standard mutation provider per the project's Riverpod conventions ([`.claude/rules/riverpod.md`](../../.claude/rules/riverpod.md), "Mutations"):

```dart
// clear_image_cache_provider.dart
@riverpod
class ClearImageCache extends _$ClearImageCache {
  @override
  FutureOr<void> build() => null;

  Future<void> clear() async {
    final profileCache = ref.read(profileImageCacheManagerProvider);
    final feedCache = ref.read(feedImageCacheManagerProvider);

    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      await Future.wait([
        profileCache.emptyCache(),
        feedCache.emptyCache(),
      ]);
    });
  }
}
```

`FutureOr<void> build() => null` keeps the notifier idle on init (no spurious loading transition that would fire the `ref.listen` block in the tile). `BaseCacheManager.emptyCache()` is the documented API for wiping a manager's disk store + SQLite metadata.

### Tile shape

```dart
class ClearImageCacheTile extends ConsumerWidget {
  const ClearImageCacheTile({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(clearImageCacheProvider);

    ref.listen(clearImageCacheProvider, (prev, state) {
      switch ((prev, state)) {
        case (AsyncLoading(), AsyncData()):
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('Image cache cleared')),
          );
        case (AsyncLoading(), AsyncError(:final error)):
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Could not clear cache: $error')),
          );
        case _:
          break;
      }
    });

    return ListTile(
      leading: const Icon(Icons.cleaning_services_outlined),
      title: const Text('Clear image cache'),
      enabled: state is! AsyncLoading,
      onTap: () => ref.read(clearImageCacheProvider.notifier).clear(),
    );
  }
}
```

### `SettingsPageBody` change

```dart
class SettingsPageBody extends ConsumerWidget {
  const SettingsPageBody({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return const Column(
      children: [
        ClearImageCacheTile(),
        SignOutTile(),
      ],
    );
  }
}
```

Strings stay hardcoded for now to match the precedent set by `SignOutTile` and the existing settings copy. Moving settings strings to l10n is a separate cleanup, out of scope here.

## Data flow

`CachedNetworkImage` resolution path on each render:

1. **Hit, fresh** (entry exists, age < `stalePeriod`): file returned from disk → decoded → painted. No network call. Steady-state fast path.
2. **Hit, stale** (age ≥ `stalePeriod`): file is *still* served immediately from disk, *and* a background revalidation kicks off (HTTP `If-Modified-Since` / `ETag`). The CDN responds 304; entry's `validTill` is bumped. The user sees an instant render. Harmless for us because URLs are CID-keyed.
3. **Miss**: HTTP GET → bytes saved under `<cache_dir>/<key>/<sha256_of_url>` → SQLite metadata row inserted/updated → bytes streamed to decoder → painted. `placeholder` shows during this time; `errorWidget` shows on HTTP failure or decode failure.

### Concurrency

`flutter_cache_manager` deduplicates concurrent requests for the same URL. If ten `PostCard`s by the same author mount in one frame, only one network request fires. We do not need to coalesce requests ourselves.

### Eviction

A periodic background sweep (scheduled by the library) evicts:
- entries past `stalePeriod` that have not been touched recently, **and**
- the oldest-touched entries beyond `maxNrOfCacheObjects`.

Eviction is best-effort and asynchronous; the cap can briefly overshoot between sweeps.

### App lifecycle

- Cache directories live under the OS standard cache directory (`getTemporaryDirectory()`). The OS may purge them under disk pressure; nothing in the app holds invariants that depend on cached bytes existing.
- The cache survives across app restarts. The CID-keyed URLs guarantee correctness regardless of how long the app was closed.
- We do not precache anything on startup. The user's own avatar is requested by the first widget that displays it (typically the profile tab) and cached after first view.

## Testing

### Unit tests

- `app/test/shared/image/image_cache_providers_test.dart` — verify each provider returns a non-null `BaseCacheManager`, and that repeat reads in the same `ProviderContainer` return the same instance (singleton invariant).
- `app/test/shared/image/clear_image_cache_provider_test.dart` — verify the mutation calls `emptyCache()` on both injected cache managers (override both providers with fakes that record calls); verify `state` transitions `AsyncLoading → AsyncData(null)` on success and `AsyncLoading → AsyncError` if either `emptyCache()` throws.

### Widget tests

A `_FakeCacheManager` test double (subclass of `BaseCacheManager`) returns canned responses for known URLs and throws for unknown ones. The fake lives under `app/test/fakes/`.

Override the provider in `ProviderScope`:

```dart
ProviderScope(
  overrides: [
    profileImageCacheManagerProvider.overrideWith((ref) => fakeCacheManager),
  ],
  child: ProfileAvatar(seed: 'A', avatarUrl: 'https://...'),
)
```

`ProfileAvatar` cases (`app/test/profile/widgets/profile_avatar_test.dart`):
- **null URL** → renders `_AvatarInitialFallback` directly. No cache calls.
- **valid URL, fake returns success** → renders the image once the fake's stream emits.
- **valid URL, fake throws** → renders the fallback via `errorWidget`.

`ProfileBanner` cases (`app/test/profile/widgets/profile_banner_test.dart`): same three states, with the loaded state showing the image and the loading/error states letting the colour swatch show through.

`ClearImageCacheTile` cases (`app/test/settings/widgets/clear_image_cache_tile_test.dart`):
- Idle → tap fires the mutation; tile re-renders disabled while `AsyncLoading`.
- Loading → success: snackbar "Image cache cleared" appears.
- Loading → error: snackbar with the error message appears.

## Implementation steps (high level)

The implementation plan will be authored separately via the writing-plans skill, but the rough shape is:

1. Add `cached_network_image` to `app/pubspec.yaml`.
2. Create `app/lib/shared/image/image_cache_managers.dart` with both singleton classes.
3. Create `app/lib/shared/image/image_cache_providers.dart` with both `@riverpod` providers; run `dart run build_runner build --delete-conflicting-outputs`.
4. Refactor `app/lib/profile/widgets/profile_avatar.dart` to `ConsumerWidget` + `CachedNetworkImage` + extracted `_AvatarInitialFallback`.
5. Refactor `app/lib/profile/widgets/profile_banner.dart` to `ConsumerWidget` + `CachedNetworkImage`.
6. Create `app/lib/shared/image/clear_image_cache_provider.dart` with the `ClearImageCache` mutation.
7. Create `app/lib/settings/widgets/clear_image_cache_tile.dart` and add it to `SettingsPageBody`.
8. Add unit tests for the providers and the clear-cache mutation.
9. Add widget tests for the two refactored widgets and the new tile, using a `_FakeCacheManager`.
10. Smoke-test on a real device: confirm avatars render from cache on second app launch (airplane mode after first launch); confirm the settings tile clears both caches and the next image load re-fetches from the network.

## Future work

- **Feed post-image call sites** — light up `FeedImageCacheManager` once the blob-upload spec lands and post media starts coming through the API.
- **Manual precaching of the user's own avatar** — only if metrics show the first-render delay is user-visible.
- **Per-cache "Clear" controls** — split the single tile into per-cache options once we have telemetry that shows users actually want that granularity.
- **Web platform support** — current spec targets mobile + desktop; if the Flutter web build becomes a target, revisit because `flutter_cache_manager` uses `path_provider` which has different semantics on web.

## References

- [`atproto-craft-social-app-reference.md`](../../atproto-craft-social-app-reference.md) — overall app shape.
- [API architecture spec](2026-04-21-appview-api-architecture-design.md) — flags blob upload as deferred future work.
- [Profile onboarding spec](2026-04-23-profile-onboarding-design.md) — documents the CID-keyed CDN URL synthesis.
- `cached_network_image` — https://pub.dev/packages/cached_network_image
- `flutter_cache_manager` — https://pub.dev/packages/flutter_cache_manager

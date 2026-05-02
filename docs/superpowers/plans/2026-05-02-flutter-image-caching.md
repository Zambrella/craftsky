# Flutter Image Caching Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace raw `NetworkImage` use in `ProfileAvatar` and `ProfileBanner` with `cached_network_image` backed by two named `flutter_cache_manager` instances (long-lived "profile" + reserved-for-feed "feed"), and add a "Clear image cache" tile to the settings page that empties both.

**Architecture:** Two singleton `CacheManager` subclasses live under `app/lib/shared/image/`, exposed via Riverpod providers (return type `BaseCacheManager` so tests can swap fakes). `ProfileAvatar` becomes a `ConsumerWidget` whose `placeholder` and `errorWidget` reuse a shared `_AvatarInitialFallback` so the initial-letter fallback is shown in all "no-face" states (null URL, loading, error). `ProfileBanner` does the same with `SizedBox.shrink()` letting the existing colour swatch show through. A `ClearImageCache` mutation calls `emptyCache()` on both managers in parallel; `ClearImageCacheTile` listens for the success/error transition and surfaces a snackbar.

**Tech Stack:** Flutter 3.11+, Riverpod 3 (`@riverpod` codegen), `cached_network_image` (which depends on `flutter_cache_manager`), `flutter_test` (for `Fake`).

**Spec:** [docs/superpowers/specs/2026-05-02-flutter-image-caching-design.md](../specs/2026-05-02-flutter-image-caching-design.md)

---

## Background reading for the implementer

Read these before starting. They're load-bearing.

- [`docs/superpowers/specs/2026-05-02-flutter-image-caching-design.md`](../specs/2026-05-02-flutter-image-caching-design.md) — the spec this plan implements. **Primary source of truth.**
- [`.claude/rules/flutter.md`](../../../.claude/rules/flutter.md) — widget-architecture rules (one class per widget, no `_build*` helpers; theme-driven; const constructors).
- [`.claude/rules/riverpod.md`](../../../.claude/rules/riverpod.md) — provider patterns. The clear-cache mutation follows the "Mutations" pattern; the cache managers follow the "Data Fetching" pattern (function-style provider returning a singleton).
- [`app/lib/profile/widgets/profile_avatar.dart`](../../../app/lib/profile/widgets/profile_avatar.dart) and [`profile_banner.dart`](../../../app/lib/profile/widgets/profile_banner.dart) — the widgets being refactored.
- [`app/lib/settings/widgets/sign_out_tile.dart`](../../../app/lib/settings/widgets/sign_out_tile.dart) — the existing settings tile pattern to mirror (`ConsumerWidget`, `ListTile`, `enabled: state is! AsyncLoading`).
- [`app/test/settings/sign_out_tile_test.dart`](../../../app/test/settings/sign_out_tile_test.dart) — widget-test pattern: `ProviderScope` with overrides, `tester.container().read(...)` to introspect fakes.
- [`app/lib/theme/app_theme.dart`](../../../app/lib/theme/app_theme.dart) — `AppTheme.lightThemeData` is what tests must pass to `MaterialApp(theme:)` so `theme.extension<BrandSwatchTheme>()!` resolves.

## Conventions this plan follows

- **TDD.** Every task that adds production code writes the failing test first, runs it to confirm it fails, writes the minimum to pass, runs it again, and commits. Don't batch.
- **One commit per task.** Tasks are small. Frequent commits make reverts cheap.
- **Test runner:** `cd app && flutter test <path>` for individual files; `cd app && flutter test` for the whole app suite.
- **Codegen:** after editing any file with `@riverpod` or `part 'foo.g.dart'`, run `cd app && dart run build_runner build --delete-conflicting-outputs`. Commit the regenerated `.g.dart` alongside the source.
- **Format:** `cd app && dart format .` before committing if you've touched `.dart` files.
- **No emojis in code or commit messages.**
- **Hardcoded strings are OK** for the new settings tile — matches the precedent set by `SignOutTile` ("Sign out") and the existing settings page header.

---

## File structure

All paths are relative to repo root.

### New files

```
app/lib/shared/image/
  image_cache_managers.dart           # ProfileImageCacheManager, FeedImageCacheManager singletons
  image_cache_providers.dart          # @riverpod functions returning BaseCacheManager
  image_cache_providers.g.dart        # generated
  clear_image_cache_provider.dart     # @riverpod ClearImageCache mutation
  clear_image_cache_provider.g.dart   # generated

app/lib/settings/widgets/
  clear_image_cache_tile.dart         # ListTile that triggers the mutation

app/test/fakes/
  image_cache_fakes.dart              # FakeBaseCacheManager (shared across test files)

app/test/shared/image/
  image_cache_providers_test.dart
  clear_image_cache_provider_test.dart

app/test/profile/widgets/
  profile_avatar_test.dart
  profile_banner_test.dart

app/test/settings/
  clear_image_cache_tile_test.dart
```

### Modified files

- `app/pubspec.yaml` — add `cached_network_image`.
- `app/lib/profile/widgets/profile_avatar.dart` — `StatelessWidget` → `ConsumerWidget`; replace `DecorationImage(NetworkImage(...))` with `ClipOval(child: CachedNetworkImage(...))`; extract `_AvatarInitialFallback`.
- `app/lib/profile/widgets/profile_banner.dart` — `StatelessWidget` → `ConsumerWidget`; replace `DecorationImage(NetworkImage(...))` with a `CachedNetworkImage` child of the existing coloured `Container`.
- `app/lib/settings/pages/settings_page.dart` — add `ClearImageCacheTile()` to the `Column` children inside `SettingsPageBody`.
- `app/test/settings/settings_page_test.dart` — assert the new tile is present (only if the existing test enumerates tiles; otherwise leave alone).

### Deleted files

None.

## Chunk boundaries

- **Chunk 1** — Add the dependency.
- **Chunk 2** — Cache managers + Riverpod providers + tests.
- **Chunk 3** — Shared test fake (`FakeBaseCacheManager`).
- **Chunk 4** — `ProfileAvatar` refactor (TDD).
- **Chunk 5** — `ProfileBanner` refactor (TDD).
- **Chunk 6** — `ClearImageCache` mutation provider (TDD).
- **Chunk 7** — `ClearImageCacheTile` + wiring into `SettingsPageBody` (TDD).
- **Chunk 8** — Manual smoke test on a device.

---

## Chunk 1: Add the dependency

### Task 1: Add `cached_network_image` to pubspec

**Files:**
- Modify: `app/pubspec.yaml`

- [ ] **Step 1: Edit `app/pubspec.yaml`** — add to the `dependencies:` block, alphabetised between `cupertino_icons` and `dart_mappable`:

```yaml
  cached_network_image: ^3.4.1
```

- [ ] **Step 2: Run `flutter pub get`**

```bash
cd app && flutter pub get
```

Expected: clean output, no version-conflict warnings. `cached_network_image` and `flutter_cache_manager` (transitive) appear in `pubspec.lock`.

- [ ] **Step 3: Commit**

```bash
git add app/pubspec.yaml app/pubspec.lock
git commit -m "feat(app): add cached_network_image dependency"
```

---

## Chunk 2: Cache managers + Riverpod providers

### Task 2: Write `image_cache_managers.dart`

This is plain Dart with no behaviour to test directly (the classes wrap `flutter_cache_manager`'s `Config` constructor). We test it indirectly through the provider tests in Task 4. So no test-first here — just write it.

**Files:**
- Create: `app/lib/shared/image/image_cache_managers.dart`

- [ ] **Step 1: Create the file** with this content:

```dart
import 'package:flutter_cache_manager/flutter_cache_manager.dart';

/// Long-lived disk cache for avatar and banner images. URLs from the
/// Bluesky CDN are content-addressed (CID-keyed), so cached entries can
/// never go stale-but-wrong; a long stale period and large object cap
/// are safe and avoid re-downloading reusable identity images under
/// LRU pressure from larger feed media.
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

/// Shorter-lived disk cache reserved for feed post images. No call sites
/// in v1 — wired up so adding feed-image call sites later is a one-line
/// change.
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

- [ ] **Step 2: Verify the analyser is clean**

```bash
cd app && dart analyze lib/shared/image/image_cache_managers.dart
```

Expected: `No issues found!`

- [ ] **Step 3: Format**

```bash
cd app && dart format lib/shared/image/image_cache_managers.dart
```

- [ ] **Step 4: Commit**

```bash
git add app/lib/shared/image/image_cache_managers.dart
git commit -m "feat(app): add Profile and Feed image cache managers"
```

---

### Task 3: Write the Riverpod providers (TDD)

**Files:**
- Create: `app/lib/shared/image/image_cache_providers.dart`
- Create: `app/test/shared/image/image_cache_providers_test.dart`

- [ ] **Step 1: Write the failing test** at `app/test/shared/image/image_cache_providers_test.dart`:

```dart
import 'package:craftsky_app/shared/image/image_cache_managers.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:flutter_cache_manager/flutter_cache_manager.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('profileImageCacheManagerProvider', () {
    test('returns a ProfileImageCacheManager instance', () {
      final container = ProviderContainer.test();
      final manager = container.read(profileImageCacheManagerProvider);

      expect(manager, isA<ProfileImageCacheManager>());
      expect(manager, isA<BaseCacheManager>());
    });

    test('returns the same singleton on repeat reads', () {
      final container = ProviderContainer.test();
      final first = container.read(profileImageCacheManagerProvider);
      final second = container.read(profileImageCacheManagerProvider);

      expect(identical(first, second), isTrue);
    });
  });

  group('feedImageCacheManagerProvider', () {
    test('returns a FeedImageCacheManager instance', () {
      final container = ProviderContainer.test();
      final manager = container.read(feedImageCacheManagerProvider);

      expect(manager, isA<FeedImageCacheManager>());
      expect(manager, isA<BaseCacheManager>());
    });

    test('returns the same singleton on repeat reads', () {
      final container = ProviderContainer.test();
      final first = container.read(feedImageCacheManagerProvider);
      final second = container.read(feedImageCacheManagerProvider);

      expect(identical(first, second), isTrue);
    });
  });
}
```

- [ ] **Step 2: Run the test to confirm it fails**

```bash
cd app && flutter test test/shared/image/image_cache_providers_test.dart
```

Expected: compile error — `image_cache_providers.dart` doesn't exist.

- [ ] **Step 3: Create `app/lib/shared/image/image_cache_providers.dart`**:

```dart
import 'package:craftsky_app/shared/image/image_cache_managers.dart';
import 'package:flutter_cache_manager/flutter_cache_manager.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'image_cache_providers.g.dart';

@riverpod
BaseCacheManager profileImageCacheManager(Ref ref) =>
    ProfileImageCacheManager();

@riverpod
BaseCacheManager feedImageCacheManager(Ref ref) => FeedImageCacheManager();
```

- [ ] **Step 4: Run codegen**

```bash
cd app && dart run build_runner build --delete-conflicting-outputs
```

Expected: writes `app/lib/shared/image/image_cache_providers.g.dart`.

- [ ] **Step 5: Run the test to confirm it passes**

```bash
cd app && flutter test test/shared/image/image_cache_providers_test.dart
```

Expected: all 4 tests pass.

- [ ] **Step 6: Format**

```bash
cd app && dart format lib/shared/image/ test/shared/image/
```

- [ ] **Step 7: Commit**

```bash
git add app/lib/shared/image/image_cache_providers.dart \
        app/lib/shared/image/image_cache_providers.g.dart \
        app/test/shared/image/image_cache_providers_test.dart
git commit -m "feat(app): expose image cache managers via Riverpod"
```

---

## Chunk 3: Shared test fake

A single `FakeBaseCacheManager` is reused by every widget and provider test that touches a cache manager. It uses `flutter_test`'s `Fake` mixin so any unstubbed `BaseCacheManager` method throws `UnimplementedError` rather than silently returning null.

### Task 4: Add `FakeBaseCacheManager`

**Files:**
- Create: `app/test/fakes/image_cache_fakes.dart`

No production code changes, so no failing test step — but the fake itself must be exercisable. We assert its behaviour from a tiny inline test below before committing.

- [ ] **Step 1: Create `app/test/fakes/image_cache_fakes.dart`**:

```dart
import 'dart:async';

import 'package:flutter_cache_manager/flutter_cache_manager.dart';
import 'package:flutter_test/flutter_test.dart';

/// Recording fake `BaseCacheManager` for tests. Any method not explicitly
/// overridden throws `UnimplementedError` (via [Fake]) so unintended
/// usages are loud, not silent.
///
/// Override `nextStream` per-test to control what `getFileStream` emits;
/// override `throwOnEmptyCache` to make `emptyCache` fail.
class FakeBaseCacheManager extends Fake implements BaseCacheManager {
  int emptyCacheCalls = 0;
  Object? throwOnEmptyCache;

  /// Stream returned by [getFileStream]. Default is an empty stream that
  /// stays open forever — `CachedNetworkImage` will sit on its
  /// placeholder.
  Stream<FileResponse> Function(String url)? nextStream;

  @override
  Future<void> emptyCache() async {
    emptyCacheCalls++;
    final err = throwOnEmptyCache;
    if (err != null) {
      throw err;
    }
  }

  @override
  Stream<FileResponse> getFileStream(
    String url, {
    String? key,
    Map<String, String>? headers,
    bool? withProgress,
  }) {
    final builder = nextStream;
    if (builder != null) {
      return builder(url);
    }
    // Default: a stream that emits nothing and never closes.
    return StreamController<FileResponse>().stream;
  }
}

/// Convenience builder: a stream that immediately errors. Use as
/// `fake.nextStream = (_) => erroringStream();` to drive `errorWidget`.
Stream<FileResponse> erroringStream([Object error = 'fake-cache-error']) {
  final controller = StreamController<FileResponse>();
  controller.addError(error);
  unawaited(controller.close());
  return controller.stream;
}
```

- [ ] **Step 2: Verify the analyser is clean**

```bash
cd app && dart analyze test/fakes/image_cache_fakes.dart
```

Expected: `No issues found!`

- [ ] **Step 3: Format**

```bash
cd app && dart format test/fakes/image_cache_fakes.dart
```

- [ ] **Step 4: Commit**

```bash
git add app/test/fakes/image_cache_fakes.dart
git commit -m "test(app): add FakeBaseCacheManager for image-cache tests"
```

---

## Chunk 4: ProfileAvatar refactor

Refactor the existing `ProfileAvatar` from a `StatelessWidget` using `DecorationImage(NetworkImage(...))` to a `ConsumerWidget` using `ClipOval(child: CachedNetworkImage(...))`, with the initial-letter fallback extracted to a private widget reused as `placeholder` and `errorWidget`.

### Task 5: Write the failing avatar tests

**Files:**
- Create: `app/test/profile/widgets/profile_avatar_test.dart`

- [ ] **Step 1: Create the test file**:

```dart
import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/image_cache_fakes.dart';

Widget _wrap(Widget child, {List<Override> overrides = const []}) {
  return ProviderScope(
    overrides: overrides,
    child: MaterialApp(
      theme: AppTheme.lightThemeData,
      home: Scaffold(body: Center(child: child)),
    ),
  );
}

void main() {
  group('ProfileAvatar', () {
    testWidgets('renders the initial-letter fallback when avatarUrl is null',
        (tester) async {
      await tester.pumpWidget(_wrap(const ProfileAvatar(seed: 'Alice')));

      expect(find.text('A'), findsOneWidget);
      expect(find.byType(CachedNetworkImage), findsNothing);
    });

    testWidgets('renders "?" when seed is empty and avatarUrl is null',
        (tester) async {
      await tester.pumpWidget(_wrap(const ProfileAvatar(seed: '')));

      expect(find.text('?'), findsOneWidget);
    });

    testWidgets(
        'mounts CachedNetworkImage with the profile cache manager '
        'when avatarUrl is set', (tester) async {
      final fake = FakeBaseCacheManager();

      await tester.pumpWidget(
        _wrap(
          const ProfileAvatar(
            seed: 'Bob',
            avatarUrl: 'https://example.test/b.jpg',
          ),
          overrides: [
            profileImageCacheManagerProvider.overrideWith((ref) => fake),
          ],
        ),
      );
      await tester.pump();

      final image =
          tester.widget<CachedNetworkImage>(find.byType(CachedNetworkImage));
      expect(image.imageUrl, 'https://example.test/b.jpg');
      expect(image.cacheManager, same(fake));
      expect(image.fit, BoxFit.cover);
    });

    testWidgets('shows the initial-letter placeholder while loading',
        (tester) async {
      // Default fake: getFileStream returns an empty, never-closing stream,
      // so CachedNetworkImage sits on its placeholder.
      final fake = FakeBaseCacheManager();

      await tester.pumpWidget(
        _wrap(
          const ProfileAvatar(
            seed: 'Cara',
            avatarUrl: 'https://example.test/c.jpg',
          ),
          overrides: [
            profileImageCacheManagerProvider.overrideWith((ref) => fake),
          ],
        ),
      );
      await tester.pump();

      // Initial letter visible during load.
      expect(find.text('C'), findsOneWidget);
    });

    testWidgets('shows the initial-letter on cache error', (tester) async {
      final fake = FakeBaseCacheManager()
        ..nextStream = (_) => erroringStream();

      await tester.pumpWidget(
        _wrap(
          const ProfileAvatar(
            seed: 'Dan',
            avatarUrl: 'https://example.test/d.jpg',
          ),
          overrides: [
            profileImageCacheManagerProvider.overrideWith((ref) => fake),
          ],
        ),
      );
      // Pump twice: once to mount, once to let the error propagate.
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      expect(find.text('D'), findsOneWidget);
    });

    testWidgets('preserves chunky border + butter background', (tester) async {
      await tester.pumpWidget(_wrap(const ProfileAvatar(seed: 'E')));

      // Find the outermost Container (the one decorated with the circle
      // shape + border + shadow). It is the first Container ancestor
      // of the rendered Text 'E' that has a circular BoxDecoration.
      final containers = tester.widgetList<Container>(find.byType(Container));
      final circle = containers.firstWhere((c) {
        final d = c.decoration;
        return d is BoxDecoration && d.shape == BoxShape.circle;
      });
      final decoration = circle.decoration! as BoxDecoration;

      // Butter background present, ink border applied.
      final swatches = AppTheme.lightThemeData.extension<BrandSwatchTheme>()!;
      expect(decoration.color, swatches.butter);
      expect(decoration.border, isNotNull);
      expect(decoration.boxShadow, isNotEmpty);
    });
  });
}
```

- [ ] **Step 2: Run the test to confirm it fails**

```bash
cd app && flutter test test/profile/widgets/profile_avatar_test.dart
```

Expected: tests for the `CachedNetworkImage` flow fail because the current `ProfileAvatar` uses `DecorationImage(NetworkImage(...))` directly. The "initial-letter" test for the null case may pass against the current code (it already renders the initial); that's fine.

- [ ] **Step 3: Refactor `app/lib/profile/widgets/profile_avatar.dart`** to:

```dart
import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Circular paper-cutout avatar with the chunky 1.5px ink border and
/// hard-offset drop shadow that are signature to the design system.
/// Reused on profile headers, post cards, comments, search results —
/// anywhere we render someone's face.
///
/// Falls back to a butter-coloured initial when [avatarUrl] is null,
/// while the image is loading, and on cache error. The initial is taken
/// from [seed] (handle or display name); avatars never show a generic
/// person glyph because the brand voice prefers a warm, hand-cut
/// character.
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

/// Avatar size variants. Dimensions and border weights are tuned per
/// surface so the chunky border reads at every scale.
enum ProfileAvatarSize {
  small(dimension: 36, borderWidth: 1),
  medium(dimension: 48, borderWidth: 1.5),
  large(dimension: 96, borderWidth: 2);

  const ProfileAvatarSize({required this.dimension, required this.borderWidth});

  final double dimension;
  final double borderWidth;

  /// Hard-offset drop shadow scaled to the avatar's surface. All sizes
  /// currently use `dropSm` (3,3) so the avatar's shadow tail matches
  /// the chunky buttons it sits next to in the profile header — kept
  /// as a switch so we can dial up to `drop` (6,6) for hero variants
  /// later without touching call sites.
  List<BoxShadow> shadowsFrom(BrandShadowTheme shadows) {
    return switch (this) {
      ProfileAvatarSize.small => shadows.dropSm,
      ProfileAvatarSize.medium => shadows.dropSm,
      ProfileAvatarSize.large => shadows.dropSm,
    };
  }
}
```

- [ ] **Step 4: Run the avatar tests to confirm they pass**

```bash
cd app && flutter test test/profile/widgets/profile_avatar_test.dart
```

Expected: all tests pass.

- [ ] **Step 5: Run the full app suite to confirm nothing else broke** (`ProfileAvatar` is used by `PostCard`, `ProfileSliverAppBar`, `EditProfileBannerAvatar`, `profile_page_test.dart`, etc.):

```bash
cd app && flutter test
```

Expected: green. If a downstream test fails because it instantiated `ProfileAvatar` outside a `ProviderScope`, fix that test by wrapping in `ProviderScope` (no overrides needed for the null-URL case).

- [ ] **Step 6: Format**

```bash
cd app && dart format lib/profile/widgets/profile_avatar.dart \
                     test/profile/widgets/profile_avatar_test.dart
```

- [ ] **Step 7: Commit**

```bash
git add app/lib/profile/widgets/profile_avatar.dart \
        app/test/profile/widgets/profile_avatar_test.dart
git commit -m "feat(app): cache avatar images via CachedNetworkImage"
```

If Step 5 required adjusting any other test files, include them in the same commit.

---

## Chunk 5: ProfileBanner refactor

Same shape as the avatar refactor but simpler: no fallback widget, just `SizedBox.shrink()` for placeholder/errorWidget so the existing colour swatch (set as the `Container.color`) shows through.

### Task 6: Write the failing banner tests

**Files:**
- Create: `app/test/profile/widgets/profile_banner_test.dart`

- [ ] **Step 1: Create the test file**:

```dart
import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/profile/widgets/profile_banner.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/image_cache_fakes.dart';

Widget _wrap(Widget child, {List<Override> overrides = const []}) {
  return ProviderScope(
    overrides: overrides,
    child: MaterialApp(
      theme: AppTheme.lightThemeData,
      home: Scaffold(body: child),
    ),
  );
}

void main() {
  group('ProfileBanner', () {
    testWidgets('renders no CachedNetworkImage when bannerUrl is null',
        (tester) async {
      await tester.pumpWidget(
        _wrap(const ProfileBanner(color: Color(0xFFCC8866))),
      );

      expect(find.byType(CachedNetworkImage), findsNothing);
    });

    testWidgets(
        'mounts CachedNetworkImage with the profile cache manager '
        'when bannerUrl is set', (tester) async {
      final fake = FakeBaseCacheManager();

      await tester.pumpWidget(
        _wrap(
          const ProfileBanner(
            color: Color(0xFFCC8866),
            bannerUrl: 'https://example.test/banner.jpg',
          ),
          overrides: [
            profileImageCacheManagerProvider.overrideWith((ref) => fake),
          ],
        ),
      );
      await tester.pump();

      final image =
          tester.widget<CachedNetworkImage>(find.byType(CachedNetworkImage));
      expect(image.imageUrl, 'https://example.test/banner.jpg');
      expect(image.cacheManager, same(fake));
      expect(image.fit, BoxFit.cover);
    });

    testWidgets('respects the height parameter', (tester) async {
      await tester.pumpWidget(
        _wrap(const ProfileBanner(color: Color(0xFFCC8866), height: 200)),
      );

      // Find the outermost Container that carries the swatch colour and
      // explicit height — that's the banner.
      final containers = tester.widgetList<Container>(find.byType(Container));
      final banner = containers.firstWhere(
        (c) => c.constraints?.maxHeight == 200,
      );
      expect(banner.color, const Color(0xFFCC8866));
    });
  });
}
```

- [ ] **Step 2: Run the test to confirm it fails**

```bash
cd app && flutter test test/profile/widgets/profile_banner_test.dart
```

Expected: the `CachedNetworkImage` test fails because the current `ProfileBanner` uses `DecorationImage(NetworkImage(...))`.

- [ ] **Step 3: Refactor `app/lib/profile/widgets/profile_banner.dart`**:

```dart
import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Flat coloured banner that sits behind the profile header. Renders a
/// user-supplied banner image on top of the colour swatch when one is
/// set; otherwise the swatch is the banner. The swatch shows through
/// during image load and on cache error.
///
/// Height is fixed so the avatar overlap math in `ProfileHeaderHero`
/// stays predictable.
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

- [ ] **Step 4: Run the banner tests to confirm they pass**

```bash
cd app && flutter test test/profile/widgets/profile_banner_test.dart
```

Expected: all tests pass.

- [ ] **Step 5: Run the full app suite to confirm nothing else broke**

```bash
cd app && flutter test
```

Expected: green. If a downstream test that instantiates `ProfileBanner` fails, wrap that test's widget tree in a `ProviderScope`.

- [ ] **Step 6: Format**

```bash
cd app && dart format lib/profile/widgets/profile_banner.dart \
                     test/profile/widgets/profile_banner_test.dart
```

- [ ] **Step 7: Commit**

```bash
git add app/lib/profile/widgets/profile_banner.dart \
        app/test/profile/widgets/profile_banner_test.dart
git commit -m "feat(app): cache banner images via CachedNetworkImage"
```

---

## Chunk 6: ClearImageCache mutation provider

A standard Riverpod mutation per the project's conventions (`FutureOr<void> build() => null`, `AsyncValue.guard`). Calls `emptyCache()` on both managers in parallel.

### Task 7: Write the failing mutation tests

**Files:**
- Create: `app/test/shared/image/clear_image_cache_provider_test.dart`

- [ ] **Step 1: Create the test file**:

```dart
import 'package:craftsky_app/shared/image/clear_image_cache_provider.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../fakes/image_cache_fakes.dart';

void main() {
  group('ClearImageCache', () {
    test('starts in an idle (data) state', () {
      final container = ProviderContainer.test();
      expect(container.read(clearImageCacheProvider), isA<AsyncData<void>>());
    });

    test('calls emptyCache() on both cache managers', () async {
      final profileFake = FakeBaseCacheManager();
      final feedFake = FakeBaseCacheManager();
      final container = ProviderContainer.test(
        overrides: [
          profileImageCacheManagerProvider.overrideWith((ref) => profileFake),
          feedImageCacheManagerProvider.overrideWith((ref) => feedFake),
        ],
      );

      await container.read(clearImageCacheProvider.notifier).clear();

      expect(profileFake.emptyCacheCalls, 1);
      expect(feedFake.emptyCacheCalls, 1);
      expect(container.read(clearImageCacheProvider), isA<AsyncData<void>>());
    });

    test('reports AsyncError when the profile cache fails', () async {
      final profileFake = FakeBaseCacheManager()
        ..throwOnEmptyCache = StateError('boom');
      final feedFake = FakeBaseCacheManager();
      final container = ProviderContainer.test(
        overrides: [
          profileImageCacheManagerProvider.overrideWith((ref) => profileFake),
          feedImageCacheManagerProvider.overrideWith((ref) => feedFake),
        ],
      );

      await container.read(clearImageCacheProvider.notifier).clear();

      expect(container.read(clearImageCacheProvider), isA<AsyncError<void>>());
    });

    test('reports AsyncError when the feed cache fails', () async {
      final profileFake = FakeBaseCacheManager();
      final feedFake = FakeBaseCacheManager()
        ..throwOnEmptyCache = StateError('boom');
      final container = ProviderContainer.test(
        overrides: [
          profileImageCacheManagerProvider.overrideWith((ref) => profileFake),
          feedImageCacheManagerProvider.overrideWith((ref) => feedFake),
        ],
      );

      await container.read(clearImageCacheProvider.notifier).clear();

      expect(container.read(clearImageCacheProvider), isA<AsyncError<void>>());
    });
  });
}
```

- [ ] **Step 2: Run the test to confirm it fails**

```bash
cd app && flutter test test/shared/image/clear_image_cache_provider_test.dart
```

Expected: compile error — `clear_image_cache_provider.dart` doesn't exist.

- [ ] **Step 3: Create `app/lib/shared/image/clear_image_cache_provider.dart`**:

```dart
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'clear_image_cache_provider.g.dart';

/// Mutation that empties both image caches in parallel. Idle by default;
/// transitions through AsyncLoading on each invocation of [clear].
@riverpod
class ClearImageCache extends _$ClearImageCache {
  @override
  FutureOr<void> build() => null;

  Future<void> clear() async {
    final profileCache = ref.read(profileImageCacheManagerProvider);
    final feedCache = ref.read(feedImageCacheManagerProvider);

    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      await Future.wait(<Future<void>>[
        profileCache.emptyCache(),
        feedCache.emptyCache(),
      ]);
    });
  }
}
```

- [ ] **Step 4: Run codegen**

```bash
cd app && dart run build_runner build --delete-conflicting-outputs
```

- [ ] **Step 5: Run the mutation tests to confirm they pass**

```bash
cd app && flutter test test/shared/image/clear_image_cache_provider_test.dart
```

Expected: all 4 tests pass.

- [ ] **Step 6: Format**

```bash
cd app && dart format lib/shared/image/clear_image_cache_provider.dart \
                     test/shared/image/clear_image_cache_provider_test.dart
```

- [ ] **Step 7: Commit**

```bash
git add app/lib/shared/image/clear_image_cache_provider.dart \
        app/lib/shared/image/clear_image_cache_provider.g.dart \
        app/test/shared/image/clear_image_cache_provider_test.dart
git commit -m "feat(app): add ClearImageCache mutation provider"
```

---

## Chunk 7: ClearImageCacheTile and SettingsPageBody wiring

### Task 8: Write the failing tile tests

**Files:**
- Create: `app/test/settings/clear_image_cache_tile_test.dart`

- [ ] **Step 1: Create the test file**:

```dart
import 'package:craftsky_app/settings/widgets/clear_image_cache_tile.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/image_cache_fakes.dart';

void main() {
  group('ClearImageCacheTile', () {
    testWidgets('tap calls emptyCache on both managers', (tester) async {
      final profileFake = FakeBaseCacheManager();
      final feedFake = FakeBaseCacheManager();

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            profileImageCacheManagerProvider
                .overrideWith((ref) => profileFake),
            feedImageCacheManagerProvider.overrideWith((ref) => feedFake),
          ],
          child: const MaterialApp(
            home: Scaffold(body: ClearImageCacheTile()),
          ),
        ),
      );

      await tester.tap(find.byType(ClearImageCacheTile));
      await tester.pump(); // schedule the future
      await tester.pump(); // resolve

      expect(profileFake.emptyCacheCalls, 1);
      expect(feedFake.emptyCacheCalls, 1);
    });

    testWidgets('shows success snackbar when both caches clear',
        (tester) async {
      final profileFake = FakeBaseCacheManager();
      final feedFake = FakeBaseCacheManager();

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            profileImageCacheManagerProvider
                .overrideWith((ref) => profileFake),
            feedImageCacheManagerProvider.overrideWith((ref) => feedFake),
          ],
          child: const MaterialApp(
            home: Scaffold(body: ClearImageCacheTile()),
          ),
        ),
      );

      await tester.tap(find.byType(ClearImageCacheTile));
      await tester.pump();
      await tester.pump();

      expect(find.text('Image cache cleared'), findsOneWidget);
    });

    testWidgets('shows error snackbar when a cache fails to clear',
        (tester) async {
      final profileFake = FakeBaseCacheManager()
        ..throwOnEmptyCache = StateError('disk full');
      final feedFake = FakeBaseCacheManager();

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            profileImageCacheManagerProvider
                .overrideWith((ref) => profileFake),
            feedImageCacheManagerProvider.overrideWith((ref) => feedFake),
          ],
          child: const MaterialApp(
            home: Scaffold(body: ClearImageCacheTile()),
          ),
        ),
      );

      await tester.tap(find.byType(ClearImageCacheTile));
      await tester.pump();
      await tester.pump();

      expect(
        find.textContaining('Could not clear cache'),
        findsOneWidget,
      );
    });
  });
}
```

- [ ] **Step 2: Run the test to confirm it fails**

```bash
cd app && flutter test test/settings/clear_image_cache_tile_test.dart
```

Expected: compile error — `clear_image_cache_tile.dart` doesn't exist.

- [ ] **Step 3: Create `app/lib/settings/widgets/clear_image_cache_tile.dart`**:

```dart
import 'package:craftsky_app/shared/image/clear_image_cache_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Settings tile that empties both image caches. The action is reversible
/// (images re-download on next view) so there is no confirmation dialog.
class ClearImageCacheTile extends ConsumerWidget {
  const ClearImageCacheTile({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(clearImageCacheProvider);

    ref.listen(clearImageCacheProvider, (prev, next) {
      switch ((prev, next)) {
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

- [ ] **Step 4: Run the tile tests to confirm they pass**

```bash
cd app && flutter test test/settings/clear_image_cache_tile_test.dart
```

Expected: all 3 tests pass.

- [ ] **Step 5: Wire the tile into `SettingsPageBody`** — edit `app/lib/settings/pages/settings_page.dart`:

```dart
import 'package:craftsky_app/settings/widgets/clear_image_cache_tile.dart';
import 'package:craftsky_app/settings/widgets/sign_out_tile.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class SettingsPage extends ConsumerWidget {
  const SettingsPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Scaffold(
      appBar: AppBar(title: const Text('Settings')),
      body: const SettingsPageBody(),
    );
  }
}

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

- [ ] **Step 6: Run the full app suite**

```bash
cd app && flutter test
```

Expected: green. The existing `app/test/settings/settings_page_test.dart` may have assertions that enumerate the tiles in `SettingsPageBody`. If it does and now fails, update that test to expect both `ClearImageCacheTile` and `SignOutTile`. Wrap any new test setup in `ProviderScope` overrides for both cache-manager providers if those tests now exercise the clear-cache codepath.

- [ ] **Step 7: Format**

```bash
cd app && dart format lib/settings/ test/settings/
```

- [ ] **Step 8: Commit**

```bash
git add app/lib/settings/widgets/clear_image_cache_tile.dart \
        app/lib/settings/pages/settings_page.dart \
        app/test/settings/clear_image_cache_tile_test.dart
git commit -m "feat(app): add Clear image cache settings tile"
```

If Step 6 required adjusting `settings_page_test.dart` or any other downstream test, include those changes in the same commit.

---

## Chunk 8: Manual smoke test

The widget tests cover wiring; only a real device will confirm the disk-cache behaviour. This chunk is non-TDD and produces no commits.

### Task 9: Smoke-test on a device

- [ ] **Step 1: Launch the app** on iOS simulator or Android emulator using the Dart MCP `launch_app` tool, or via:

```bash
cd app && flutter run -d <device-id>
```

- [ ] **Step 2: Sign in** with a test account that has an avatar set on their Bluesky profile (any `*.bsky.social` account with `app.bsky.actor.profile.avatar` populated). Open the profile tab. Confirm the avatar loads (you'll see the initial letter briefly during the first download, then the real image).

- [ ] **Step 3: Force-quit the app and relaunch.** Open the profile tab again. The avatar should appear instantly (no initial-letter flash) — proof that the disk cache is doing its job across launches.

- [ ] **Step 4: Toggle airplane mode on the device, force-quit, relaunch.** The cached avatar should still render (proof: no network round-trip required for cache hits). Toggle airplane mode off afterward.

- [ ] **Step 5: Open Settings, tap "Clear image cache".** Confirm the snackbar reads "Image cache cleared".

- [ ] **Step 6: Force-quit and relaunch with airplane mode on.** Open the profile tab. The avatar should now show only the initial-letter fallback (cache was wiped; offline can't refetch). Toggle airplane mode off — the avatar should load on next paint.

- [ ] **Step 7: Note any anomalies** in the PR description. If everything behaved as expected, no further commits are needed.

---

## Self-review checklist (run before opening the PR)

- [ ] All tests pass: `cd app && flutter test`
- [ ] Analyser is clean: `cd app && dart analyze`
- [ ] Format is clean: `cd app && dart format --set-exit-if-changed .` exits 0
- [ ] Generated `.g.dart` files are committed alongside their source `.dart` files (no orphan generated code, no missing `.g.dart` for a `part` directive)
- [ ] No emojis in new code, comments, or commit messages
- [ ] `pubspec.lock` includes `cached_network_image` and `flutter_cache_manager` entries
- [ ] Smoke test passed on at least one real or emulated device

## Out-of-scope reminders

These are explicit non-goals from the spec — do not implement here:

- Feed post-image call sites (no UI work targets `FeedImageCacheManager` in v1).
- Manual cache warming / precaching at app launch.
- Image transformation or resizing.
- Tuning Flutter's in-memory `ImageCache`.
- Web-platform support.
- Per-cache "Clear" controls (single tile clears both).
- Confirmation dialog before clearing the cache.

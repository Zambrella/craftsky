// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'save_profile_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Mutation notifier for the profile-edit page.
///
/// Holds idle state until [save] runs, then transitions
/// `AsyncLoading` → `AsyncData(updatedProfile)` on success or
/// `AsyncError` on failure. Callers wire navigation/snackbars via
/// `ref.listen` rather than try/catch — see `edit_profile_page.dart`.
///
/// Diff semantics live in the caller: pass only the fields that
/// changed. The repository turns missing (`null`) fields into omitted
/// keys, so an unchanged field is never overwritten on the server.
/// Pass an empty string to clear `displayName`/`description`, an empty
/// list to clear `crafts`.
///
/// On success this provider invalidates the entire `userProfileProvider`
/// family so any open profile pages refetch fresh data. The signed-in
/// user's own entry is the only one that's actually changing — but
/// we'd otherwise need the caller to know its own handle/DID, and the
/// over-invalidation cost (a single refetch on screens that happen to
/// be open) is negligible compared to the bug surface of getting the
/// key wrong.

@ProviderFor(SaveProfile)
final saveProfileProvider = SaveProfileProvider._();

/// Mutation notifier for the profile-edit page.
///
/// Holds idle state until [save] runs, then transitions
/// `AsyncLoading` → `AsyncData(updatedProfile)` on success or
/// `AsyncError` on failure. Callers wire navigation/snackbars via
/// `ref.listen` rather than try/catch — see `edit_profile_page.dart`.
///
/// Diff semantics live in the caller: pass only the fields that
/// changed. The repository turns missing (`null`) fields into omitted
/// keys, so an unchanged field is never overwritten on the server.
/// Pass an empty string to clear `displayName`/`description`, an empty
/// list to clear `crafts`.
///
/// On success this provider invalidates the entire `userProfileProvider`
/// family so any open profile pages refetch fresh data. The signed-in
/// user's own entry is the only one that's actually changing — but
/// we'd otherwise need the caller to know its own handle/DID, and the
/// over-invalidation cost (a single refetch on screens that happen to
/// be open) is negligible compared to the bug surface of getting the
/// key wrong.
final class SaveProfileProvider
    extends $AsyncNotifierProvider<SaveProfile, Profile?> {
  /// Mutation notifier for the profile-edit page.
  ///
  /// Holds idle state until [save] runs, then transitions
  /// `AsyncLoading` → `AsyncData(updatedProfile)` on success or
  /// `AsyncError` on failure. Callers wire navigation/snackbars via
  /// `ref.listen` rather than try/catch — see `edit_profile_page.dart`.
  ///
  /// Diff semantics live in the caller: pass only the fields that
  /// changed. The repository turns missing (`null`) fields into omitted
  /// keys, so an unchanged field is never overwritten on the server.
  /// Pass an empty string to clear `displayName`/`description`, an empty
  /// list to clear `crafts`.
  ///
  /// On success this provider invalidates the entire `userProfileProvider`
  /// family so any open profile pages refetch fresh data. The signed-in
  /// user's own entry is the only one that's actually changing — but
  /// we'd otherwise need the caller to know its own handle/DID, and the
  /// over-invalidation cost (a single refetch on screens that happen to
  /// be open) is negligible compared to the bug surface of getting the
  /// key wrong.
  SaveProfileProvider._()
    : super(
        from: null,
        argument: null,
        retry: null,
        name: r'saveProfileProvider',
        isAutoDispose: true,
        dependencies: null,
        $allTransitiveDependencies: null,
      );

  @override
  String debugGetCreateSourceHash() => _$saveProfileHash();

  @$internal
  @override
  SaveProfile create() => SaveProfile();
}

String _$saveProfileHash() => r'e04d605ab0d9bce1b560ea50fbe8e366dfc7eeeb';

/// Mutation notifier for the profile-edit page.
///
/// Holds idle state until [save] runs, then transitions
/// `AsyncLoading` → `AsyncData(updatedProfile)` on success or
/// `AsyncError` on failure. Callers wire navigation/snackbars via
/// `ref.listen` rather than try/catch — see `edit_profile_page.dart`.
///
/// Diff semantics live in the caller: pass only the fields that
/// changed. The repository turns missing (`null`) fields into omitted
/// keys, so an unchanged field is never overwritten on the server.
/// Pass an empty string to clear `displayName`/`description`, an empty
/// list to clear `crafts`.
///
/// On success this provider invalidates the entire `userProfileProvider`
/// family so any open profile pages refetch fresh data. The signed-in
/// user's own entry is the only one that's actually changing — but
/// we'd otherwise need the caller to know its own handle/DID, and the
/// over-invalidation cost (a single refetch on screens that happen to
/// be open) is negligible compared to the bug surface of getting the
/// key wrong.

abstract class _$SaveProfile extends $AsyncNotifier<Profile?> {
  FutureOr<Profile?> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref = this.ref as $Ref<AsyncValue<Profile?>, Profile?>;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<AsyncValue<Profile?>, Profile?>,
              AsyncValue<Profile?>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}

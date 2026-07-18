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
/// Callers should pass the **full** desired field values, not a diff.
/// The PUT path on the AppView ultimately writes a new
/// `app.bsky.actor.profile` record on the user's PDS, and atproto
/// records are atomic — fields absent from the body get cleared on the
/// PDS, regardless of any "leave unchanged" wording the AppView's HTTP
/// layer suggests. Always send the complete current state.
///
/// On success this provider pushes the freshly-saved [Profile] back
/// into the `userProfileProvider` family cache for the entries
/// currently being watched (keyed by handle and by DID). That avoids a
/// refetch round-trip and any read-after-write lag, and keeps the
/// profile page in sync the instant the edit page pops.

@ProviderFor(SaveProfile)
final saveProfileProvider = SaveProfileProvider._();

/// Mutation notifier for the profile-edit page.
///
/// Holds idle state until [save] runs, then transitions
/// `AsyncLoading` → `AsyncData(updatedProfile)` on success or
/// `AsyncError` on failure. Callers wire navigation/snackbars via
/// `ref.listen` rather than try/catch — see `edit_profile_page.dart`.
///
/// Callers should pass the **full** desired field values, not a diff.
/// The PUT path on the AppView ultimately writes a new
/// `app.bsky.actor.profile` record on the user's PDS, and atproto
/// records are atomic — fields absent from the body get cleared on the
/// PDS, regardless of any "leave unchanged" wording the AppView's HTTP
/// layer suggests. Always send the complete current state.
///
/// On success this provider pushes the freshly-saved [Profile] back
/// into the `userProfileProvider` family cache for the entries
/// currently being watched (keyed by handle and by DID). That avoids a
/// refetch round-trip and any read-after-write lag, and keeps the
/// profile page in sync the instant the edit page pops.
final class SaveProfileProvider
    extends $AsyncNotifierProvider<SaveProfile, Profile?> {
  /// Mutation notifier for the profile-edit page.
  ///
  /// Holds idle state until [save] runs, then transitions
  /// `AsyncLoading` → `AsyncData(updatedProfile)` on success or
  /// `AsyncError` on failure. Callers wire navigation/snackbars via
  /// `ref.listen` rather than try/catch — see `edit_profile_page.dart`.
  ///
  /// Callers should pass the **full** desired field values, not a diff.
  /// The PUT path on the AppView ultimately writes a new
  /// `app.bsky.actor.profile` record on the user's PDS, and atproto
  /// records are atomic — fields absent from the body get cleared on the
  /// PDS, regardless of any "leave unchanged" wording the AppView's HTTP
  /// layer suggests. Always send the complete current state.
  ///
  /// On success this provider pushes the freshly-saved [Profile] back
  /// into the `userProfileProvider` family cache for the entries
  /// currently being watched (keyed by handle and by DID). That avoids a
  /// refetch round-trip and any read-after-write lag, and keeps the
  /// profile page in sync the instant the edit page pops.
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

String _$saveProfileHash() => r'62bb1c5682e9c5c6471fe18dc4cb425b6fa2e665';

/// Mutation notifier for the profile-edit page.
///
/// Holds idle state until [save] runs, then transitions
/// `AsyncLoading` → `AsyncData(updatedProfile)` on success or
/// `AsyncError` on failure. Callers wire navigation/snackbars via
/// `ref.listen` rather than try/catch — see `edit_profile_page.dart`.
///
/// Callers should pass the **full** desired field values, not a diff.
/// The PUT path on the AppView ultimately writes a new
/// `app.bsky.actor.profile` record on the user's PDS, and atproto
/// records are atomic — fields absent from the body get cleared on the
/// PDS, regardless of any "leave unchanged" wording the AppView's HTTP
/// layer suggests. Always send the complete current state.
///
/// On success this provider pushes the freshly-saved [Profile] back
/// into the `userProfileProvider` family cache for the entries
/// currently being watched (keyed by handle and by DID). That avoids a
/// refetch round-trip and any read-after-write lag, and keeps the
/// profile page in sync the instant the edit page pops.

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

// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'save_profile_provider.dart';

// **************************************************************************
// RiverpodGenerator
// **************************************************************************

// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint, type=warning
/// Mutation notifier for the profile-edit page.
///
/// Holds idle state until [save] runs, then reports an explicit outcome for
/// each independently versioned profile record.
///
/// Callers should pass the **full** desired field values, not a diff.
/// The PUT path on the AppView ultimately writes a new
/// `app.bsky.actor.profile` record on the user's PDS, and atproto
/// records are atomic — fields absent from the body get cleared on the
/// PDS, regardless of any "leave unchanged" wording the AppView's HTTP
/// layer suggests. Always send the complete current state.
///
/// On success the shared operation controller publishes a bounded overlay and
/// refreshes profile reads until the AppView projection agrees.

@ProviderFor(SaveProfile)
final saveProfileProvider = SaveProfileProvider._();

/// Mutation notifier for the profile-edit page.
///
/// Holds idle state until [save] runs, then reports an explicit outcome for
/// each independently versioned profile record.
///
/// Callers should pass the **full** desired field values, not a diff.
/// The PUT path on the AppView ultimately writes a new
/// `app.bsky.actor.profile` record on the user's PDS, and atproto
/// records are atomic — fields absent from the body get cleared on the
/// PDS, regardless of any "leave unchanged" wording the AppView's HTTP
/// layer suggests. Always send the complete current state.
///
/// On success the shared operation controller publishes a bounded overlay and
/// refreshes profile reads until the AppView projection agrees.
final class SaveProfileProvider
    extends $AsyncNotifierProvider<SaveProfile, CombinedProfileSaveResult?> {
  /// Mutation notifier for the profile-edit page.
  ///
  /// Holds idle state until [save] runs, then reports an explicit outcome for
  /// each independently versioned profile record.
  ///
  /// Callers should pass the **full** desired field values, not a diff.
  /// The PUT path on the AppView ultimately writes a new
  /// `app.bsky.actor.profile` record on the user's PDS, and atproto
  /// records are atomic — fields absent from the body get cleared on the
  /// PDS, regardless of any "leave unchanged" wording the AppView's HTTP
  /// layer suggests. Always send the complete current state.
  ///
  /// On success the shared operation controller publishes a bounded overlay and
  /// refreshes profile reads until the AppView projection agrees.
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

String _$saveProfileHash() => r'f18cf7336c5749730b548bd73d45ba2ff60c91eb';

/// Mutation notifier for the profile-edit page.
///
/// Holds idle state until [save] runs, then reports an explicit outcome for
/// each independently versioned profile record.
///
/// Callers should pass the **full** desired field values, not a diff.
/// The PUT path on the AppView ultimately writes a new
/// `app.bsky.actor.profile` record on the user's PDS, and atproto
/// records are atomic — fields absent from the body get cleared on the
/// PDS, regardless of any "leave unchanged" wording the AppView's HTTP
/// layer suggests. Always send the complete current state.
///
/// On success the shared operation controller publishes a bounded overlay and
/// refreshes profile reads until the AppView projection agrees.

abstract class _$SaveProfile
    extends $AsyncNotifier<CombinedProfileSaveResult?> {
  FutureOr<CombinedProfileSaveResult?> build();
  @$mustCallSuper
  @override
  void runBuild() {
    final ref =
        this.ref
            as $Ref<
              AsyncValue<CombinedProfileSaveResult?>,
              CombinedProfileSaveResult?
            >;
    final element =
        ref.element
            as $ClassProviderElement<
              AnyNotifier<
                AsyncValue<CombinedProfileSaveResult?>,
                CombinedProfileSaveResult?
              >,
              AsyncValue<CombinedProfileSaveResult?>,
              Object?,
              Object?
            >;
    element.handleCreate(ref, build);
  }
}

import 'dart:async';

import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'save_profile_provider.g.dart';

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
@riverpod
class SaveProfile extends _$SaveProfile {
  @override
  FutureOr<Profile?> build() => null;

  Future<void> save({
    String? displayName,
    String? description,
    List<String>? crafts,
  }) async {
    state = const AsyncLoading();
    state = await AsyncValue.guard(() async {
      final repo = ref.read(profileRepositoryProvider);
      final updated = await repo.updateMe(
        displayName: displayName,
        description: description,
        crafts: crafts,
      );
      if (!ref.mounted) return null;

      // Push the authoritative server response into any
      // userProfileProvider entries that are already alive. We guard
      // with `ref.exists` so we don't accidentally instantiate a new
      // family entry, which would race a fresh `repo.fetch` against
      // our setCached and overwrite it.
      for (final id in <String>{updated.handle, updated.did}) {
        if (ref.exists(userProfileProvider(id))) {
          ref.read(userProfileProvider(id).notifier).setCached(updated);
        }
      }

      return updated;
    });
  }

  /// Resets the notifier back to its idle state. Call after consuming a
  /// success/failure transition so a re-entry to the edit page doesn't
  /// see the previous result.
  void reset() => state = const AsyncData(null);
}

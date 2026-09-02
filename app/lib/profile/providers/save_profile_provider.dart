import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/business_projection_overlay_provider.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_save_result.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'save_profile_provider.g.dart';

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
/// On success this provider pushes the freshly-saved [Profile] back
/// into the `userProfileProvider` family cache for the entries
/// currently being watched (keyed by handle and by DID). That avoids a
/// refetch round-trip and any read-after-write lag, and keeps the
/// profile page in sync the instant the edit page pops.
@riverpod
class SaveProfile extends _$SaveProfile {
  @override
  FutureOr<CombinedProfileSaveResult?> build() => null;

  Future<CombinedProfileSaveResult?> save({
    Profile? currentProfile,
    bool ordinaryChanged = true,
    BusinessDeclarationDraft? businessDraft,
    bool businessChanged = false,
    String? displayName,
    String? description,
    List<String>? crafts,
    UploadedBlob? avatar,
    bool clearAvatar = false,
    UploadedBlob? banner,
    bool clearBanner = false,
  }) async {
    final ownership = captureActiveAccountOperation(ref);
    state = const AsyncLoading();
    if (businessChanged && businessDraft == null) {
      throw ArgumentError.notNull('businessDraft');
    }
    final businessLease = currentProfile == null
        ? null
        : ownership?.session ??
              AccountSessionLease(
                account: AccountKey(currentProfile.did.toString()),
                sessionGeneration: 0,
              );
    final businessKey = currentProfile == null || businessLease == null
        ? null
        : BusinessProjectionKey.declaration(
            businessLease.account,
            currentProfile.did,
          );
    final businessGeneration = !businessChanged || businessKey == null
        ? null
        : ref
              .read(businessProjectionOverlayProvider.notifier)
              .beginMutation(businessKey, businessLease!);

    // Start both requested writes before awaiting either. Each future captures
    // its own error, so one record can never cancel or hide the other's result.
    final ordinaryFuture = ordinaryChanged
        ? _capture(() {
            final repo = ref.read(profileRepositoryProvider);
            return repo.updateMe(
              displayName: displayName,
              description: description,
              crafts: crafts,
              avatar: avatar,
              clearAvatar: clearAvatar,
              banner: banner,
              clearBanner: clearBanner,
            );
          })
        : null;
    final businessFuture = businessChanged
        ? _capture(() async {
            final draft = businessDraft!;
            final mutation = await ref
                .read(businessRepositoryProvider)
                .putBusinessProfile(
                  draft.toJson(),
                  expectedCid: draft.expectedCid,
                );
            return draft.toProfile(mutation.cid);
          })
        : null;

    final ordinary = ordinaryFuture == null
        ? const PerRecordSaveOutcome<Profile>.skipped()
        : await ordinaryFuture;
    final business = businessFuture == null
        ? const PerRecordSaveOutcome<BusinessProfile>.skipped()
        : await businessFuture;
    if (!isActiveAccountOperationCurrent(ref, ownership)) return null;

    if (business.value case final accepted?
        when businessKey != null &&
            businessLease != null &&
            businessGeneration != null) {
      final installed = ref
          .read(businessProjectionOverlayProvider.notifier)
          .acceptUpsert(
            key: businessKey,
            lease: businessLease,
            requestGeneration: businessGeneration,
            preWriteCid: businessDraft!.expectedCid,
            acceptedCid: accepted.cid,
            acceptedView: accepted,
          );
      if (!installed) return null;
    }

    final result = CombinedProfileSaveResult(
      ordinary: ordinary,
      business: business,
    );
    _cacheAcceptedProfile(currentProfile, ordinary, business);
    state = AsyncData(result);
    return result;
  }

  Future<PerRecordSaveOutcome<T>> _capture<T>(
    Future<T> Function() operation,
  ) async {
    try {
      return PerRecordSaveOutcome.success(await operation());
    } on Object catch (error, stackTrace) {
      return PerRecordSaveOutcome.failure(error, stackTrace);
    }
  }

  void _cacheAcceptedProfile(
    Profile? currentProfile,
    PerRecordSaveOutcome<Profile> ordinary,
    PerRecordSaveOutcome<BusinessProfile> business,
  ) {
    var accepted = ordinary.value ?? currentProfile;
    if (business.value case final savedBusiness?) {
      if (accepted != null) {
        accepted = accepted.copyWith(business: savedBusiness);
      }
    }
    if (accepted == null || (!ordinary.succeeded && !business.succeeded)) {
      return;
    }

    // Guarding with `ref.exists` avoids instantiating a family entry whose
    // initial fetch could race and overwrite the accepted save result.
    for (final id in <String>{accepted.handle, accepted.did}) {
      if (ref.exists(userProfileProvider(id))) {
        ref.read(userProfileProvider(id).notifier).setCached(accepted);
      }
    }
  }

  /// Resets the notifier back to its idle state. Call after consuming a
  /// success/failure transition so a re-entry to the edit page doesn't
  /// see the previous result.
  void reset() => state = const AsyncData(null);
}

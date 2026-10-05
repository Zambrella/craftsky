import 'dart:async';
import 'dart:convert';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_event.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/business/providers/business_record_overlay.dart';
import 'package:craftsky_app/business/providers/business_repository_provider.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_save_result.dart';
import 'package:craftsky_app/profile/providers/profile_record_overlay.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:craftsky_app/shared/mutations/pds_record_operation_controller.dart';
import 'package:craftsky_app/shared/mutations/pds_record_reconciliation.dart';
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
/// On success the shared operation controller publishes a bounded overlay and
/// refreshes profile reads until the AppView projection agrees.
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
    String? pronouns,
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
    final mutationLease = currentProfile == null
        ? null
        : ownership?.session ??
              AccountSessionLease(
                account: AccountKey(currentProfile.did.toString()),
                sessionGeneration: 0,
              );
    final businessLease = mutationLease;
    final ordinaryBody = ordinaryChanged
        ? _profileMutationBody(
            displayName: displayName,
            pronouns: pronouns,
            description: description,
            crafts: crafts,
            avatar: avatar,
            clearAvatar: clearAvatar,
            banner: banner,
            clearBanner: clearBanner,
          )
        : null;
    const ordinaryEndpoint = '/v1/profiles/me';
    final ordinaryImmutableBody = ordinaryBody == null
        ? null
        : jsonEncode(ordinaryBody);
    final businessBody = businessChanged
        ? Map<String, dynamic>.from(
            jsonDecode(
                  jsonEncode(
                    businessDraft!.toJson(
                      preserveProducts: true,
                      preserveUnknownCatalogValues: true,
                    ),
                  ),
                )
                as Map,
          )
        : null;
    const businessEndpoint = '/v1/profiles/me/business';
    final businessImmutableBody = businessBody == null
        ? null
        : jsonEncode(businessBody);
    final commandController = ref.read(pdsRecordOperationControllerProvider);
    final businessOwner =
        currentProfile?.did ?? Did.parse('did:plc:local-test');
    final ordinaryToken = !ordinaryChanged || mutationLease == null
        ? null
        : commandController.beginOrRetry(
            scope: personalProfileMutationScope(mutationLease, businessOwner),
            endpoint: ordinaryEndpoint,
            immutableBody: ordinaryImmutableBody!,
            newOperationKey: newPdsMutationOperationKey,
          );
    final commandToken = !businessChanged || businessLease == null
        ? null
        : commandController.beginOrRetry(
            scope: businessProfileMutationScope(businessLease, businessOwner),
            endpoint: businessEndpoint,
            immutableBody: businessImmutableBody!,
            newOperationKey: newPdsMutationOperationKey,
          );
    final ordinaryOperationKey =
        ordinaryToken?.operationKey ??
        (ordinaryChanged ? newPdsMutationOperationKey() : null);
    final businessOperationKey =
        commandToken?.operationKey ??
        (businessChanged ? newPdsMutationOperationKey() : null);

    // Start both requested writes before awaiting either. Each future captures
    // its own error, so one record can never cancel or hide the other's result.
    final ordinaryFuture = ordinaryChanged
        ? _capture(
            () => ordinaryToken == null
                ? _updateProfileOnce(
                    operationKey: ordinaryOperationKey!,
                    displayName: displayName,
                    pronouns: pronouns,
                    description: description,
                    crafts: crafts,
                    avatar: avatar,
                    clearAvatar: clearAvatar,
                    banner: banner,
                    clearBanner: clearBanner,
                  )
                : _updateProfileWithRetry(
                    ownership: ownership,
                    operationKey: ordinaryOperationKey!,
                    endpoint: ordinaryEndpoint,
                    immutableBody: ordinaryImmutableBody!,
                    controller: commandController,
                    token: ordinaryToken,
                    displayName: displayName,
                    pronouns: pronouns,
                    description: description,
                    crafts: crafts,
                    avatar: avatar,
                    clearAvatar: clearAvatar,
                    banner: banner,
                    clearBanner: clearBanner,
                  ),
          )
        : null;
    final businessFuture = businessChanged
        ? _capture(() async {
            final draft = businessDraft!;
            final mutation = commandToken == null
                ? await ref
                      .read(businessRepositoryProvider)
                      .putBusinessProfile(
                        businessBody!,
                        operationKey: businessOperationKey!,
                        expectedCid: draft.expectedCid,
                      )
                : await _putBusinessProfileWithRetry(
                    ownership: ownership,
                    body: businessBody!,
                    expectedCid: draft.expectedCid,
                    operationKey: businessOperationKey!,
                    endpoint: businessEndpoint,
                    immutableBody: businessImmutableBody!,
                    controller: commandController,
                    token: commandToken,
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

    if (ordinary.value case final accepted? when mutationLease != null) {
      final projection = personalProfileProjection(accepted);
      final reconciliation = PdsCompoundProfileReconciliation(
        bluesky: PdsFixedKeyReconciliation(
          uri: projection.bluesky!.uri,
          controlledContent: projection.bluesky!.content,
        ),
        craftsky: PdsFixedKeyReconciliation(
          uri: projection.craftsky!.uri,
          controlledContent: projection.craftsky!.content,
        ),
      );
      if (!commandController.markAccepted(
        ordinaryToken!,
        optimisticValue: accepted,
        agrees: (value) => reconciliation.agrees(
          value is PdsCompoundProfileProjection ? value : null,
        ),
        refresh: ownership == null ? null : _invalidatePersonalProfileReads,
      )) {
        return null;
      }
    }

    if (business.value case final accepted? when businessLease != null) {
      final reconciliation = PdsFixedKeyReconciliation(
        uri:
            'at://${currentProfile!.did}/social.craftsky.business.profile/self',
        controlledContent: businessProfileProjection(
          currentProfile.did,
          accepted,
        ).content,
      );
      if (!commandController.markAccepted(
        commandToken!,
        optimisticValue: accepted,
        agrees: (value) => reconciliation.agrees(
          value is PdsRecordProjection ? value : null,
        ),
        refresh: _invalidateBusinessProfileReads,
      )) {
        return null;
      }
    } else if (business.value != null && commandToken != null) {
      if (!commandController.markAcceptedWithoutOverlay(commandToken)) {
        return null;
      }
    }

    final result = CombinedProfileSaveResult(
      ordinary: ordinary,
      business: business,
    );
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

  Future<Profile> _updateProfileWithRetry({
    required ActiveAccountLease? ownership,
    required String operationKey,
    required String endpoint,
    required String immutableBody,
    required PdsRecordOperationController controller,
    required PdsMutationToken token,
    String? displayName,
    String? pronouns,
    String? description,
    List<String>? crafts,
    UploadedBlob? avatar,
    bool clearAvatar = false,
    UploadedBlob? banner,
    bool clearBanner = false,
  }) async {
    final startedAt = ref.read(pdsMutationNowProvider)();
    var retryIndex = 0;
    while (true) {
      try {
        return await _updateProfileOnce(
          operationKey: operationKey,
          displayName: displayName,
          pronouns: pronouns,
          description: description,
          crafts: crafts,
          avatar: avatar,
          clearAvatar: clearAvatar,
          banner: banner,
          clearBanner: clearBanner,
        );
      } on PdsMutationAmbiguousException catch (error) {
        if (!isActiveAccountOperationCurrent(ref, ownership)) {
          throw const _ProfileMutationSuperseded();
        }
        controller.markAmbiguous(
          token,
          retryAfterSeconds: error.retryAfterSeconds,
        );
        final delay = const PdsMutationRetryPolicy().nextDelay(
          retryIndex: retryIndex,
          retryAfterSeconds: error.retryAfterSeconds,
          elapsed: ref.read(pdsMutationNowProvider)().difference(startedAt),
          jitterMillis: ref.read(pdsMutationJitterProvider),
        );
        if (delay == null) rethrow;
        retryIndex++;
        await ref.read(pdsMutationDelayProvider)(delay);
        if (!isActiveAccountOperationCurrent(ref, ownership) ||
            !controller.canRetry(
              token,
              operationKey: operationKey,
              endpoint: endpoint,
              immutableBody: immutableBody,
            )) {
          throw const _ProfileMutationSuperseded();
        }
      } on Object {
        controller.markFailed(token);
        rethrow;
      }
    }
  }

  Future<Profile> _updateProfileOnce({
    required String operationKey,
    String? displayName,
    String? pronouns,
    String? description,
    List<String>? crafts,
    UploadedBlob? avatar,
    bool clearAvatar = false,
    UploadedBlob? banner,
    bool clearBanner = false,
  }) => ref
      .read(profileRepositoryProvider)
      .updateMe(
        operationKey: operationKey,
        displayName: displayName,
        pronouns: pronouns,
        description: description,
        crafts: crafts,
        avatar: avatar,
        clearAvatar: clearAvatar,
        banner: banner,
        clearBanner: clearBanner,
      );

  Future<RecordMutationResult> _putBusinessProfileWithRetry({
    required ActiveAccountLease? ownership,
    required Map<String, dynamic> body,
    required Cid? expectedCid,
    required String operationKey,
    required String endpoint,
    required String immutableBody,
    required PdsRecordOperationController controller,
    required PdsMutationToken token,
  }) async {
    final startedAt = ref.read(pdsMutationNowProvider)();
    var retryIndex = 0;
    while (true) {
      try {
        return await ref
            .read(businessRepositoryProvider)
            .putBusinessProfile(
              body,
              operationKey: operationKey,
              expectedCid: expectedCid,
            );
      } on PdsMutationAmbiguousException catch (error) {
        if (!isActiveAccountOperationCurrent(ref, ownership)) {
          throw const _BusinessProfileMutationSuperseded();
        }
        controller.markAmbiguous(
          token,
          retryAfterSeconds: error.retryAfterSeconds,
        );
        final delay = const PdsMutationRetryPolicy().nextDelay(
          retryIndex: retryIndex,
          retryAfterSeconds: error.retryAfterSeconds,
          elapsed: ref.read(pdsMutationNowProvider)().difference(startedAt),
          jitterMillis: ref.read(pdsMutationJitterProvider),
        );
        if (delay == null) rethrow;
        retryIndex++;
        await ref.read(pdsMutationDelayProvider)(delay);
        if (!isActiveAccountOperationCurrent(ref, ownership) ||
            !controller.canRetry(
              token,
              operationKey: operationKey,
              endpoint: endpoint,
              immutableBody: immutableBody,
            )) {
          throw const _BusinessProfileMutationSuperseded();
        }
      } on Object {
        controller.markFailed(token);
        rethrow;
      }
    }
  }

  void _invalidateBusinessProfileReads() {
    if (!ref.mounted) return;
    ref.invalidate(userProfileProvider);
  }

  void _invalidatePersonalProfileReads() {
    if (!ref.mounted) return;
    ref.invalidate(userProfileProvider);
  }

  /// Resets the notifier back to its idle state. Call after consuming a
  /// success/failure transition so a re-entry to the edit page doesn't
  /// see the previous result.
  void reset() => state = const AsyncData(null);
}

Map<String, dynamic> _profileMutationBody({
  String? displayName,
  String? pronouns,
  String? description,
  List<String>? crafts,
  UploadedBlob? avatar,
  bool clearAvatar = false,
  UploadedBlob? banner,
  bool clearBanner = false,
}) {
  final body = <String, dynamic>{
    'displayName': ?displayName,
    'pronouns': ?pronouns,
    'description': ?description,
    'crafts': ?crafts,
  };
  if (clearAvatar) {
    body['avatar'] = null;
  } else if (avatar != null) {
    body['avatar'] = _uploadedBlobMap(avatar);
  }
  if (clearBanner) {
    body['banner'] = null;
  } else if (banner != null) {
    body['banner'] = _uploadedBlobMap(banner);
  }
  return body;
}

Map<String, dynamic> _uploadedBlobMap(UploadedBlob blob) => {
  r'$type': blob.type,
  'ref': {r'$link': blob.ref.link},
  'mimeType': blob.mimeType,
  'size': blob.size,
};

final class _BusinessProfileMutationSuperseded implements Exception {
  const _BusinessProfileMutationSuperseded();
}

final class _ProfileMutationSuperseded implements Exception {
  const _ProfileMutationSuperseded();
}

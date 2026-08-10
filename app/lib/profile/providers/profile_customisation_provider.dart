import 'dart:async';

import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/providers/profile_identity_cache_invalidator.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'profile_customisation_provider.g.dart';

/// Loaded server value and the user's editable local draft.
final class ProfileCustomisationEditorState {
  const ProfileCustomisationEditorState({
    required this.confirmed,
    required this.draft,
  });

  final ProfileCustomisation confirmed;
  final ProfileCustomisation draft;

  bool get isDirty => confirmed != draft;

  ProfileCustomisationEditorState copyWith({
    ProfileCustomisation? confirmed,
    ProfileCustomisation? draft,
  }) => ProfileCustomisationEditorState(
    confirmed: confirmed ?? this.confirmed,
    draft: draft ?? this.draft,
  );
}

@riverpod
class ProfileCustomisationEditor extends _$ProfileCustomisationEditor {
  String? _profileDid;
  String? _profileHandle;

  @override
  Future<ProfileCustomisationEditorState> build() async {
    final profile = await ref.watch(profileRepositoryProvider).fetchMe();
    _profileDid = profile.did;
    _profileHandle = profile.handle;
    return ProfileCustomisationEditorState(
      confirmed: profile.customisation,
      draft: profile.customisation,
    );
  }

  void selectColour(String colour) {
    if (!profileColourCatalogue.contains(colour)) return;
    _updateDraft((value) => value.copyWith(colour: colour));
  }

  void selectBorder(String border) {
    if (!profileBorderCatalogue.contains(border)) return;
    _updateDraft((value) => value.copyWith(border: border));
  }

  void selectBackground(String background) {
    if (!profileBackgroundCatalogue.contains(background)) return;
    _updateDraft((value) => value.copyWith(background: background));
  }

  void discard() {
    final current = state.value;
    if (current == null) return;
    state = AsyncData(current.copyWith(draft: current.confirmed));
  }

  Future<void> save() async {
    final current = state.value;
    if (current == null || !current.isDirty || state.isLoading) return;

    final ownership = captureActiveAccountOperation(ref);
    state = const AsyncLoading();
    try {
      final saved = await ref
          .read(profileRepositoryProvider)
          .updateCustomisation(current.draft);
      if (!ref.mounted) return;
      if (ownership != null) {
        await ref
            .read(sessionRegistryProvider.notifier)
            .updateCachedCustomisation(ownership.session, saved);
      }
      if (!isActiveAccountOperationCurrent(ref, ownership)) return;

      state = AsyncData(
        ProfileCustomisationEditorState(confirmed: saved, draft: saved),
      );
      for (final id in <String?>{_profileDid, _profileHandle}) {
        if (id == null) continue;
        final profileProvider = userProfileProvider(id);
        if (ref.exists(profileProvider)) {
          final profile = ref.read(profileProvider).value;
          if (profile != null) {
            ref
                .read(profileProvider.notifier)
                .setCached(profile.copyWith(customisation: saved));
          }
        }
      }
      ref.read(profileIdentityCacheInvalidatorProvider)();
    } on Object catch (error, stackTrace) {
      if (!isActiveAccountOperationCurrent(ref, ownership)) return;
      state = AsyncError(error, stackTrace);
    }
  }

  void _updateDraft(
    ProfileCustomisation Function(ProfileCustomisation current) update,
  ) {
    final current = state.value;
    if (current == null || state.isLoading) return;
    state = AsyncData(current.copyWith(draft: update(current.draft)));
  }
}

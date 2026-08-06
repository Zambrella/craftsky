import 'dart:collection';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/feed/models/profile_pin_state.dart';
import 'package:craftsky_app/feed/providers/post_repository_provider.dart';
import 'package:craftsky_app/feed/providers/user_posts_provider.dart';
import 'package:craftsky_app/projects/providers/user_projects_provider.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'profile_pins_provider.g.dart';

final class ProfilePinsPresentation {
  ProfilePinsPresentation({
    required this.confirmed,
    Set<ProfilePinSlot> pendingSlots = const {},
  }) : pendingSlots = UnmodifiableSetView(Set.of(pendingSlots));

  final ProfilePinState confirmed;
  final Set<ProfilePinSlot> pendingSlots;

  bool isPending(ProfilePinSlot slot) => pendingSlots.contains(slot);

  ProfilePinsPresentation copyWith({
    ProfilePinState? confirmed,
    Set<ProfilePinSlot>? pendingSlots,
  }) => ProfilePinsPresentation(
    confirmed: confirmed ?? this.confirmed,
    pendingSlots: pendingSlots ?? this.pendingSlots,
  );
}

enum ProfilePinMutationOutcome {
  pinned,
  unpinned,
  pinFailed,
  unpinFailed,
  staleCompletion;

  String? get message => switch (this) {
    pinned => 'Post pinned',
    unpinned => 'Post unpinned',
    pinFailed => 'Couldn’t pin post. Try again.',
    unpinFailed => 'Couldn’t unpin post. Try again.',
    staleCompletion => null,
  };
}

@riverpod
class ProfilePins extends _$ProfilePins {
  @override
  Future<ProfilePinsPresentation> build(ActiveAccountLease accountLease) async {
    final confirmed = await ref.watch(postRepositoryProvider).profilePins();
    return ProfilePinsPresentation(confirmed: confirmed);
  }

  Future<ProfilePinMutationOutcome?> pin({
    required Did did,
    required RecordKey rkey,
    required ProfilePinSlot slot,
    Iterable<String> authorCacheIds = const [],
  }) => _mutate(
    did: did,
    rkey: rkey,
    slot: slot,
    isPin: true,
    authorCacheIds: authorCacheIds,
  );

  Future<ProfilePinMutationOutcome?> unpin({
    required Did did,
    required RecordKey rkey,
    required ProfilePinSlot slot,
    Iterable<String> authorCacheIds = const [],
  }) => _mutate(
    did: did,
    rkey: rkey,
    slot: slot,
    isPin: false,
    authorCacheIds: authorCacheIds,
  );

  Future<ProfilePinMutationOutcome?> _mutate({
    required Did did,
    required RecordKey rkey,
    required ProfilePinSlot slot,
    required bool isPin,
    required Iterable<String> authorCacheIds,
  }) async {
    final current = state.value;
    if (current == null || current.isPending(slot)) return null;
    final ownership = captureActiveAccountOperation(ref);
    if (ownership != null && ownership != accountLease) {
      return ProfilePinMutationOutcome.staleCompletion;
    }

    state = AsyncData(
      current.copyWith(pendingSlots: {...current.pendingSlots, slot}),
    );
    try {
      final repository = ref.read(postRepositoryProvider);
      final confirmed = isPin
          ? await repository.pin(did, rkey)
          : await repository.unpin(did, rkey);
      if (!isActiveAccountOperationCurrent(ref, ownership)) {
        return ProfilePinMutationOutcome.staleCompletion;
      }
      final latest = state.value ?? current;
      state = AsyncData(
        latest.copyWith(
          confirmed: confirmed,
          pendingSlots: {...latest.pendingSlots}..remove(slot),
        ),
      );
      _refreshAffectedProfileLists(
        slot: slot,
        did: did,
        authorCacheIds: authorCacheIds,
      );
      return isPin
          ? ProfilePinMutationOutcome.pinned
          : ProfilePinMutationOutcome.unpinned;
    } on Object {
      if (!isActiveAccountOperationCurrent(ref, ownership)) {
        return ProfilePinMutationOutcome.staleCompletion;
      }
      final latest = state.value ?? current;
      state = AsyncData(
        latest.copyWith(
          pendingSlots: {...latest.pendingSlots}..remove(slot),
        ),
      );
      return isPin
          ? ProfilePinMutationOutcome.pinFailed
          : ProfilePinMutationOutcome.unpinFailed;
    }
  }

  void _refreshAffectedProfileLists({
    required ProfilePinSlot slot,
    required Did did,
    required Iterable<String> authorCacheIds,
  }) {
    for (final id in <String>{did.value, ...authorCacheIds}) {
      switch (slot) {
        case ProfilePinSlot.standard:
          final provider = userPostsProvider(id);
          if (ref.exists(provider)) ref.invalidate(provider);
        case ProfilePinSlot.project:
          final provider = userProjectsProvider(id);
          if (ref.exists(provider)) ref.invalidate(provider);
      }
    }
  }
}

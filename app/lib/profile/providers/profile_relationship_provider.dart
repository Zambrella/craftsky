import 'dart:async';
import 'dart:developer' as developer;

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/feed/providers/timeline_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_new_count_provider.dart';
import 'package:craftsky_app/notifications/providers/notifications_provider.dart';
import 'package:craftsky_app/profile/data/profile_repository.dart';
import 'package:craftsky_app/profile/models/profile_relationship.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/search/providers/blank_search_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'profile_relationship_provider.g.dart';

typedef RelationshipReconciliationScheduler =
    void Function() Function(Duration delay, void Function() callback);

const relationshipReconciliationDelay = Duration(seconds: 2);

final relationshipReconciliationSchedulerProvider =
    Provider<RelationshipReconciliationScheduler>(
      (ref) => (delay, callback) {
        final timer = Timer(delay, callback);
        return timer.cancel;
      },
    );

final relationshipReconciliationDiagnosticProvider = Provider<void Function()>(
  (ref) =>
      () => developer.log(
        'confirmed relationship overlay awaiting index reconciliation',
        name: 'craftsky.relationships',
      ),
);

/// Account-owned relationship overlay for one target.
///
/// Successful mutations remain authoritative over stale AppView reads until
/// Tap exposes the same policy state. A different account gets a different
/// provider instance and therefore cannot see or complete this overlay.
@Riverpod(keepAlive: true)
class ProfileRelationshipController extends _$ProfileRelationshipController {
  void Function()? _cancelReconciliation;

  @override
  ProfileRelationship build(AccountKey account, String handleOrDid) {
    ref.onDispose(() => _cancelReconciliation?.call());
    return const ProfileRelationship();
  }

  void seed(ProfileRelationship server) {
    if (state.pendingAction != null) return;
    if (!state.confirmedOverlay) {
      state = server;
      return;
    }
    if (state.samePolicy(server)) {
      _cancelReconciliation?.call();
      _cancelReconciliation = null;
      state = server;
    }
  }

  Future<void> mutate(ProfileRelationshipAction action) async {
    if (state.pendingAction != null) return;
    final previous = state;
    final lease = _captureLease();
    state = _optimistic(previous, action).copyWith(
      pendingAction: action,
      lastError: null,
    );
    _suppressLoadedSurfaces(action);

    try {
      final repository = await ref.read(
        accountRelationshipRepositoryProvider(account).future,
      );
      final result = await _apply(repository, action);
      if (!_isCurrent(lease)) return;
      state = result.copyWith(
        pendingAction: null,
        lastError: null,
        confirmedOverlay: true,
        initialized: true,
      );
      _invalidateAffectedSurfaces();
      _scheduleReconciliation(lease);
    } on Object catch (error) {
      if (!_isCurrent(lease)) return;
      state = previous.copyWith(pendingAction: null, lastError: error);
      _invalidateAffectedSurfaces();
    }
  }

  AccountSessionLease? _captureLease() =>
      ref.read(sessionRegistryProvider).value?.leaseFor(account);

  bool _isCurrent(AccountSessionLease? captured) {
    if (!ref.mounted) return false;
    if (captured == null) return true;
    return ref.read(sessionRegistryProvider).value?.leaseFor(account) ==
        captured;
  }

  ProfileRelationship _optimistic(
    ProfileRelationship current,
    ProfileRelationshipAction action,
  ) => switch (action) {
    ProfileRelationshipAction.mute => current.copyWith(muted: true),
    ProfileRelationshipAction.unmute => current.copyWith(muted: false),
    ProfileRelationshipAction.block => current.copyWith(blocking: true),
    ProfileRelationshipAction.unblock => current.copyWith(blocking: false),
  };

  Future<ProfileRelationship> _apply(
    ProfileRepository repository,
    ProfileRelationshipAction action,
  ) => switch (action) {
    ProfileRelationshipAction.mute => repository.mute(handleOrDid),
    ProfileRelationshipAction.unmute => repository.unmute(handleOrDid),
    ProfileRelationshipAction.block => repository.block(handleOrDid),
    ProfileRelationshipAction.unblock => repository.unblock(handleOrDid),
  };

  void _invalidateAffectedSurfaces() {
    ref
      ..invalidate(userProfileProvider(handleOrDid))
      ..invalidate(timelineProvider)
      ..invalidate(blankSearchProvider)
      ..invalidate(accountNotificationsProvider(account))
      ..invalidate(notificationsProvider)
      ..invalidate(accountNotificationNewCountProvider(account))
      ..invalidate(notificationNewCountProvider);
  }

  void _suppressLoadedSurfaces(ProfileRelationshipAction action) {
    if (action != ProfileRelationshipAction.mute &&
        action != ProfileRelationshipAction.block) {
      return;
    }
    if (ref.exists(timelineProvider)) {
      ref.read(timelineProvider.notifier).suppressActor(handleOrDid);
    }
    var removed = 0;
    final accountNotifications = accountNotificationsProvider(account);
    if (ref.exists(accountNotifications)) {
      removed = ref
          .read(accountNotifications.notifier)
          .suppressActor(handleOrDid);
    }
    if (ref.exists(notificationsProvider)) {
      final legacyRemoved = ref
          .read(notificationsProvider.notifier)
          .suppressActor(handleOrDid);
      if (legacyRemoved > removed) removed = legacyRemoved;
    }
    final accountCount = accountNotificationNewCountProvider(account);
    if (ref.exists(accountCount)) {
      ref.read(accountCount.notifier).suppress(removed);
    }
    if (ref.exists(notificationNewCountProvider)) {
      ref.read(notificationNewCountProvider.notifier).suppress(removed);
    }
  }

  void _scheduleReconciliation(AccountSessionLease? lease) {
    _cancelReconciliation?.call();
    _cancelReconciliation =
        ref.read(relationshipReconciliationSchedulerProvider)(
          relationshipReconciliationDelay,
          () {
            _cancelReconciliation = null;
            if (!_isCurrent(lease) || !state.confirmedOverlay) return;
            ref.read(relationshipReconciliationDiagnosticProvider)();
            _invalidateAffectedSurfaces();
          },
        );
  }
}

final ProfileRelationshipControllerFamily profileRelationshipProvider =
    profileRelationshipControllerProvider;

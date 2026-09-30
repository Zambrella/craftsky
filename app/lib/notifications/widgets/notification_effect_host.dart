import 'dart:async';

import 'package:craftsky_app/auth/providers/active_account_initialization_provider.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/models/notification_effect.dart';
import 'package:craftsky_app/notifications/providers/notification_new_count_provider.dart';
import 'package:craftsky_app/notifications/providers/notifications_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_permission_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_runtime_provider.dart';
import 'package:craftsky_app/notifications/services/notification_navigation.dart';
import 'package:craftsky_app/profile/models/profile_handle.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/messaging/message_action.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class NotificationEffectHost extends ConsumerStatefulWidget {
  const NotificationEffectHost({required this.child, super.key});
  final Widget child;

  @override
  ConsumerState<NotificationEffectHost> createState() =>
      _NotificationEffectHostState();
}

class _NotificationEffectHostState extends ConsumerState<NotificationEffectHost>
    with WidgetsBindingObserver {
  StreamSubscription<NotificationEffect>? _subscription;
  Did? _did;
  bool _onboarded = false;
  Did? _fetchedForAccount;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _subscription = ref.read(notificationEffectStreamProvider).listen(_handle);
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state != AppLifecycleState.resumed) return;
    ref.invalidate(notificationPermissionProvider);
    if (_did == null || !_onboarded) return;
    _refreshNotifications();
  }

  void _refreshNotifications() {
    final active = ref
        .read(sessionRegistryProvider)
        .value
        ?.activeLease
        ?.session;
    if (active == null) {
      unawaited(_fetchNotifications());
      unawaited(ref.read(notificationNewCountProvider.notifier).refresh());
      return;
    }
    unawaited(_fetchNotifications());
    unawaited(
      ref
          .read(accountNotificationNewCountProvider(active.account).notifier)
          .refresh(),
    );
  }

  @override
  Widget build(BuildContext context) {
    final initialization = ref.watch(activeAccountInitializationProvider).value;
    final did = initialization?.lease.session.account.did;
    final onboarded = initialization?.onboardingComplete ?? false;
    _did = did;
    _onboarded = onboarded;
    if (did == null || !onboarded) {
      _fetchedForAccount = null;
    } else if (_fetchedForAccount != did) {
      _fetchedForAccount = did;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted && _did == did && _onboarded) _refreshNotifications();
      });
    }
    unawaited(
      ref
          .read(notificationRuntimeProvider)
          .updateReadiness(did: did, onboarded: onboarded),
    );
    return widget.child;
  }

  void _handle(NotificationEffect effect) {
    if (!mounted) return;
    final l10n = AppLocalizations.of(context);
    switch (effect) {
      case NotificationBannerEffect(
        :final event,
        :final resolution,
        :final recipient,
      ):
        final recipientHandle = recipient == null
            ? null
            : ProfileHandle(
                recipient.handle,
              ).currentLabel(unavailableLabel: l10n.handleUnavailable);
        final recipientLabel = recipientHandle == null
            ? ''
            : '\nFor $recipientHandle';
        context.showInfo(
          '${event.title}\n${event.body}$recipientLabel',
          action: MessageAction(
            label: l10n.notificationBannerOpen,
            onPressed: () => unawaited(
              ref
                  .read(notificationRuntimeProvider)
                  .receiveResolvedOpen(event.openAttempt, resolution),
            ),
          ),
        );
      case NotificationUnavailableEffect():
        context.showWarning(l10n.notificationUnavailableRow);
      case NotificationRemovedAccountEffect():
        context.showWarning(
          'This notification belongs to an account that is no longer signed in',
        );
      case NotificationNavigationEffect(:final outcome):
        unawaited(_refreshOpenedNotificationCount());
        _invalidateOpenedNotificationList();
        navigateToNotificationOutcome(
          context,
          ref.read(goRouterProvider),
          outcome,
        );
    }
  }

  void _suppressOpenedNotificationCount() {
    final activeAccount = ref
        .read(sessionRegistryProvider)
        .value
        ?.activeLease
        ?.session
        .account;
    if (activeAccount != null) {
      final countProvider = accountNotificationNewCountProvider(activeAccount);
      if (ref.exists(countProvider)) {
        ref.read(countProvider.notifier).suppress(1);
      }
      return;
    }
    if (ref.exists(notificationNewCountProvider)) {
      ref.read(notificationNewCountProvider.notifier).suppress(1);
    }
  }

  Future<void> _refreshOpenedNotificationCount() async {
    final account = ref
        .read(sessionRegistryProvider)
        .value
        ?.activeLease
        ?.session
        .account;
    if (account != null) {
      await ref
          .read(accountNotificationNewCountProvider(account).notifier)
          .refresh();
    } else {
      await ref.read(notificationNewCountProvider.notifier).refresh();
    }
    if (mounted) _suppressOpenedNotificationCount();
  }

  void _invalidateOpenedNotificationList() {
    unawaited(_fetchNotifications());
  }

  Future<void> _fetchNotifications() async {
    final account = ref
        .read(sessionRegistryProvider)
        .value
        ?.activeLease
        ?.session
        .account;
    if (account != null) {
      await AsyncValue.guard(
        () => ref.refresh(accountNotificationsProvider(account).future),
      );
    } else {
      await AsyncValue.guard(() => ref.refresh(notificationsProvider.future));
    }
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    unawaited(_subscription?.cancel());
    super.dispose();
  }
}

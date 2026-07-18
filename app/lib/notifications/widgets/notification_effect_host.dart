import 'dart:async';

import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/models/notification_effect.dart';
import 'package:craftsky_app/notifications/providers/notification_new_count_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_permission_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_runtime_provider.dart';
import 'package:craftsky_app/notifications/providers/notification_sign_out_recovery_provider.dart';
import 'package:craftsky_app/notifications/services/notification_navigation.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_status_provider.dart';
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

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _subscription = ref.read(notificationEffectStreamProvider).listen(_handle);
    unawaited(ref.read(notificationSignOutRecoveryProvider).retry());
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state != AppLifecycleState.resumed) return;
    ref.invalidate(notificationPermissionProvider);
    unawaited(ref.read(notificationSignOutRecoveryProvider).retry());
    if (_did == null || !_onboarded) return;
    final active = ref
        .read(sessionRegistryProvider)
        .value
        ?.activeLease
        ?.session;
    if (active == null) {
      unawaited(ref.read(notificationNewCountProvider.notifier).refresh());
      return;
    }
    unawaited(
      ref
          .read(accountNotificationNewCountProvider(active.account).notifier)
          .refresh(),
    );
  }

  @override
  Widget build(BuildContext context) {
    final auth = ref.watch(authSessionProvider).value;
    final did = auth is SignedIn ? auth.did : null;
    final onboarded = did != null && ref.watch(onboardingStatusProvider(did));
    _did = did;
    _onboarded = onboarded;
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
        context.showInfo(
          '${event.title}\n${event.body}'
          '${recipient == null ? '' : '\nFor @${recipient.handle}'}',
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
        navigateToNotificationOutcome(
          context,
          ref.read(goRouterProvider),
          outcome,
        );
    }
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    unawaited(_subscription?.cancel());
    super.dispose();
  }
}

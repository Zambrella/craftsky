import 'dart:async';

import 'package:craftsky_app/feed/models/post_uri.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/services/notification_resolution_policy.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:flutter/widgets.dart';

void navigateToNotificationOutcome(
  BuildContext context,
  NotificationResolutionOutcome outcome,
) {
  if (outcome.feedback != null) {
    context.showWarning(
      AppLocalizations.of(context).notificationUnavailableRow,
    );
  }
  switch (outcome.destination) {
    case NotificationsDestination():
      const NotificationsRoute().go(context);
    case ProfileDestination(:final did):
      unawaited(UserProfileRoute(handle: did.toString()).push<void>(context));
    case PostDestination(:final uri):
      final parts = parseCraftskyPostUri(uri);
      if (parts == null) {
        const NotificationsRoute().go(context);
        return;
      }
      unawaited(
        PostThreadRoute(
          did: parts.did.toString(),
          rkey: parts.rkey.toString(),
        ).push<void>(context),
      );
  }
}

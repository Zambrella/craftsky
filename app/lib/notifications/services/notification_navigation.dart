import 'dart:async';

import 'package:craftsky_app/feed/models/post_uri.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/models/notification_destination.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:flutter/widgets.dart';

void navigateToNotificationOutcome(
  BuildContext context,
  NotificationOpenOutcome outcome,
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
    case final PostDestination destination:
      final route = postThreadRouteForNotification(destination);
      if (route == null) {
        const NotificationsRoute().go(context);
        return;
      }
      unawaited(route.push<void>(context));
  }
}

PostThreadRoute? postThreadRouteForNotification(
  PostDestination destination,
) {
  final parts = parseCraftskyPostUri(destination.subjectUri);
  if (parts == null) return null;
  return PostThreadRoute(
    did: parts.did.toString(),
    rkey: parts.rkey.toString(),
    focus: destination.focusUri?.toString(),
  );
}

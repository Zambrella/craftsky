import 'dart:async';

import 'package:craftsky_app/feed/models/post_uri.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/models/notification_destination.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:flutter/widgets.dart';
import 'package:go_router/go_router.dart';

void navigateToNotificationOutcome(
  BuildContext context,
  GoRouter router,
  NotificationOpenOutcome outcome,
) {
  if (outcome.feedback != null) {
    context.showWarning(
      AppLocalizations.of(context).notificationUnavailableRow,
    );
  }
  switch (outcome.destination) {
    case InstagramMigrationDestination():
      unawaited(
        router.push<void>(const InstagramMigrationRoute().location),
      );
    case NotificationsDestination():
      router.go(const NotificationsRoute().location);
    case ProfileDestination(:final did):
      unawaited(
        router.push<void>(UserProfileRoute(handle: did.toString()).location),
      );
    case final PostDestination destination:
      final route = postThreadRouteForNotification(destination);
      if (route == null) {
        router.go(const NotificationsRoute().location);
        return;
      }
      unawaited(router.push<void>(route.location));
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

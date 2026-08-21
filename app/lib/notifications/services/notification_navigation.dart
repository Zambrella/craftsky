import 'dart:async';

import 'package:craftsky_app/feed/models/post_uri.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/models/notification_destination.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:flutter/widgets.dart';
import 'package:go_router/go_router.dart';

/// Uses the root navigator context because [context] belongs to
/// `MaterialApp.router.builder`, above the Router inherited widget.
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
  final navigationContext = router.routerDelegate.navigatorKey.currentContext;
  if (navigationContext == null) return;
  switch (outcome.destination) {
    case InstagramMigrationDestination():
      unawaited(
        const InstagramMigrationRoute().push<void>(navigationContext),
      );
    case NotificationsDestination():
      unawaited(const NotificationsRoute().push<void>(navigationContext));
    case ProfileDestination(:final did):
      unawaited(
        UserProfileRoute(
          handle: did.toString(),
        ).push<void>(navigationContext),
      );
    case final PostDestination destination:
      final route = postThreadRouteForNotification(destination);
      if (route == null) {
        unawaited(const NotificationsRoute().push<void>(navigationContext));
        return;
      }
      unawaited(route.push<void>(navigationContext));
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

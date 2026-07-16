import 'dart:async';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/notifications/services/notification_navigation.dart';
import 'package:craftsky_app/notifications/services/notification_resolution_policy.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class NotificationRow extends ConsumerWidget {
  const NotificationRow({required this.notification, super.key});

  final CraftskyNotification notification;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final actor = notification.actor.displayLabel;
    final (title, subtitle) = switch (notification) {
      FollowNotification() => (l10n.notificationFollowRow(actor), null),
      LikeNotification(:final subjectPost) => (
        l10n.notificationLikeRow(actor),
        subjectPost.text,
      ),
      RepostNotification(:final subjectPost) => (
        l10n.notificationRepostRow(actor),
        subjectPost.text,
      ),
      ReplyNotification(:final subjectPost) => (
        l10n.notificationReplyRow(actor),
        subjectPost.text,
      ),
      MentionNotification(:final subjectPost) => (
        l10n.notificationMentionRow(actor),
        subjectPost.text,
      ),
      QuoteNotification(:final subjectPost) => (
        l10n.notificationQuoteRow(actor),
        subjectPost.text,
      ),
      GenericNotification() => (l10n.notificationGenericRow, null),
      UnavailableNotification() => (l10n.notificationUnavailableRow, null),
    };
    return ListTile(
      title: Text(title),
      subtitle: subtitle == null ? null : Text(subtitle),
      onTap: () => _open(context, ref),
    );
  }

  void _open(BuildContext context, WidgetRef ref) {
    switch (notification) {
      case FollowNotification(:final actor):
        unawaited(
          UserProfileRoute(handle: actor.handle.toString()).push<void>(context),
        );
      case LikeNotification(:final subjectPost):
      case RepostNotification(:final subjectPost):
      case MentionNotification(:final subjectPost):
      case QuoteNotification(:final subjectPost):
        unawaited(
          PostThreadRoute(
            did: subjectPost.author.did.toString(),
            rkey: subjectPost.rkey.toString(),
            $extra: subjectPost,
          ).push<void>(context),
        );
      case GenericNotification():
        unawaited(_resolveGeneric(context, ref));
      case UnavailableNotification():
        context.showWarning(
          AppLocalizations.of(context).notificationUnavailableRow,
        );
      case ReplyNotification(:final subjectPost, :final reply):
        unawaited(
          PostThreadRoute(
            did: subjectPost.author.did.toString(),
            rkey: subjectPost.rkey.toString(),
            focus: reply?.uri.toString(),
            $extra: subjectPost,
          ).push<void>(context),
        );
    }
  }

  Future<void> _resolveGeneric(BuildContext context, WidgetRef ref) async {
    final outcome = await () async {
      try {
        final resolution = await ref
            .read(notificationResolutionRepositoryProvider)
            .resolve(NotificationId.parse(notification.id));
        return NotificationResolutionPolicy.forResolution(resolution);
      } on Object catch (error) {
        return NotificationResolutionPolicy.forException(error);
      }
    }();
    if (!context.mounted) return;
    navigateToNotificationOutcome(context, outcome);
  }
}

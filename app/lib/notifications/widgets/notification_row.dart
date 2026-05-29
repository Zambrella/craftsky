import 'dart:async';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:flutter/material.dart';

class NotificationRow extends StatelessWidget {
  const NotificationRow({required this.notification, super.key});

  final CraftskyNotification notification;

  @override
  Widget build(BuildContext context) {
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
    };
    return ListTile(
      title: Text(title),
      subtitle: subtitle == null ? null : Text(subtitle),
      onTap: () => _open(context),
    );
  }

  void _open(BuildContext context) {
    switch (notification) {
      case FollowNotification(:final actor):
        unawaited(
          UserProfileRoute(handle: actor.handle.toString()).push<void>(context),
        );
      case LikeNotification(:final subjectPost):
      case RepostNotification(:final subjectPost):
        unawaited(
          PostThreadRoute(
            did: subjectPost.author.did.toString(),
            rkey: subjectPost.rkey.toString(),
            $extra: subjectPost,
          ).push<void>(context),
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
}

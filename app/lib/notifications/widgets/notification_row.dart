import 'dart:async';

import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:flutter/material.dart';

class NotificationRow extends StatelessWidget {
  const NotificationRow({required this.notification, super.key});

  final CraftskyNotification notification;

  @override
  Widget build(BuildContext context) {
    final actor = notification.actor.displayLabel;
    final (title, subtitle) = switch (notification) {
      FollowNotification() => ('$actor followed you', null),
      LikeNotification(:final subjectPost) => (
        '$actor liked your post',
        subjectPost.text,
      ),
      RepostNotification(:final subjectPost) => (
        '$actor reposted your post',
        subjectPost.text,
      ),
      ReplyNotification(:final subjectPost) => (
        '$actor replied to your post',
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

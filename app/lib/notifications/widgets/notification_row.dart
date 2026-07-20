import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart' as auth_model;
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_uri.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/widgets/notification_category_icon.dart';
import 'package:craftsky_app/profile/models/profile_relationship.dart';
import 'package:craftsky_app/profile/providers/profile_relationship_provider.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/time/relative_time_text.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

bool canOpenNotificationRow(
  AccountSessionLease owner,
  auth_model.SessionRegistry registry,
) => registry.activeLease?.session == owner;

class NotificationRow extends ConsumerWidget {
  const NotificationRow({required this.notification, this.owner, super.key});

  final CraftskyNotification notification;
  final AccountSessionLease? owner;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final registry = ref.watch(sessionRegistryProvider).value;
    final account = owner?.account ?? registry?.activeLease?.session.account;
    final actorRelationshipProvider =
        account == null ||
            !notification.actor.available ||
            account.did == notification.actor.did
        ? null
        : profileRelationshipProvider(
            account,
            notification.actor.did.toString(),
          );
    final cachedRelationship = actorRelationshipProvider == null
        ? null
        : ref.watch(actorRelationshipProvider);
    final serverRelationship = notification.actor.hasViewerState
        ? ProfileRelationship.fromProfileFlags(
            muted: notification.actor.muted ?? false,
            blocking: notification.actor.blocking ?? false,
            blockedBy: notification.actor.blockedBy ?? false,
          )
        : const ProfileRelationship(initialized: true);
    if (actorRelationshipProvider != null &&
        !(cachedRelationship?.initialized ?? false)) {
      unawaited(
        Future<void>.microtask(
          () => ref
              .read(actorRelationshipProvider.notifier)
              .seed(serverRelationship),
        ),
      );
    }
    final relationship = cachedRelationship?.initialized ?? false
        ? cachedRelationship
        : notification.actor.hasViewerState
        ? serverRelationship
        : null;
    if ((relationship?.muted ?? false) || (relationship?.hasBlock ?? false)) {
      return const SizedBox.shrink();
    }
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final actor = notification.actor.displayLabel;
    final actionColor = _actionColor(notification, theme.colorScheme);
    final (title, subtitle) = switch (notification) {
      FollowNotification() => (l10n.notificationFollowRow(actor), null),
      LikeNotification(:final subjectPost) => (
        switch (_roleOf(subjectPost)) {
          _NotificationContentRole.post => l10n.notificationLikeRow(actor),
          _NotificationContentRole.comment => l10n.notificationLikeCommentRow(
            actor,
          ),
          _NotificationContentRole.reply => l10n.notificationLikeReplyRow(
            actor,
          ),
        },
        subjectPost.text,
      ),
      RepostNotification(:final subjectPost) => (
        switch (_roleOf(subjectPost)) {
          _NotificationContentRole.post => l10n.notificationRepostRow(actor),
          _NotificationContentRole.comment => l10n.notificationRepostCommentRow(
            actor,
          ),
          _NotificationContentRole.reply => l10n.notificationRepostReplyRow(
            actor,
          ),
        },
        subjectPost.text,
      ),
      ReplyNotification(:final subjectPost) => (
        switch (_roleOf(subjectPost)) {
          _NotificationContentRole.post => l10n.notificationReplyRow(actor),
          _NotificationContentRole.comment =>
            l10n.notificationReplyToCommentRow(actor),
          _NotificationContentRole.reply => l10n.notificationReplyToReplyRow(
            actor,
          ),
        },
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
    final onTap = notification is GenericNotification
        ? null
        : () => _open(context, ref);
    return Material(
      type: MaterialType.transparency,
      child: InkWell(
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Padding(
                padding: const EdgeInsets.only(top: 6),
                child: ExcludeSemantics(
                  child: Icon(
                    notificationCategoryIcon(notification.type),
                    color: actionColor,
                    size: 24,
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    ProfileAvatar(
                      seed: actor,
                      avatarUrl: notification.actor.displayAvatarUrl,
                      size: ProfileAvatarSize.small,
                    ),
                    const SizedBox(height: 8),
                    Wrap(
                      spacing: 6,
                      runSpacing: 2,
                      crossAxisAlignment: WrapCrossAlignment.center,
                      children: [
                        Text.rich(
                          _titleSpan(title: title, actor: actor),
                          style: theme.textTheme.bodyLarge,
                        ),
                        Text(
                          '·',
                          style: theme.textTheme.bodySmall?.copyWith(
                            color: theme.colorScheme.outline,
                          ),
                        ),
                        RelativeTimeText(timestamp: notification.createdAt),
                      ],
                    ),
                    if (subtitle != null) ...[
                      const SizedBox(height: 4),
                      Text(
                        subtitle,
                        style: theme.textTheme.bodyMedium?.copyWith(
                          color: theme.colorScheme.onSurfaceVariant,
                        ),
                      ),
                    ],
                    if (notification is FollowNotification &&
                        notification.actor.available) ...[
                      const SizedBox(height: 8),
                      _NotificationFollowButton(actor: notification.actor),
                    ],
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  void _open(BuildContext context, WidgetRef ref) {
    final rowOwner = owner;
    if (rowOwner != null &&
        !canOpenNotificationRow(
          rowOwner,
          ref.read(sessionRegistryProvider).requireValue,
        )) {
      return;
    }
    switch (notification) {
      case FollowNotification(:final actor):
        unawaited(
          UserProfileRoute(handle: actor.handle.toString()).push<void>(context),
        );
      case LikeNotification(:final subjectPost):
      case RepostNotification(:final subjectPost):
      case MentionNotification(:final subjectPost):
      case QuoteNotification(:final subjectPost):
        _openPost(context, subjectPost);
      case GenericNotification():
        break;
      case UnavailableNotification():
        context.showWarning(
          AppLocalizations.of(context).notificationUnavailableRow,
        );
      case ReplyNotification(:final subjectPost, :final reply):
        _openPost(context, subjectPost, focus: reply?.uri);
    }
  }

  void _openPost(BuildContext context, Post post, {AtUri? focus}) {
    final root = post.reply?.root.uri;
    final rootParts = root == null ? null : parseCraftskyPostUri(root);
    unawaited(
      PostThreadRoute(
        did: (rootParts?.did ?? post.author.did).toString(),
        rkey: (rootParts?.rkey ?? post.rkey).toString(),
        focus: (focus ?? (root == null ? null : post.uri))?.toString(),
        $extra: post,
      ).push<void>(context),
    );
  }
}

class _NotificationFollowButton extends ConsumerStatefulWidget {
  const _NotificationFollowButton({required this.actor});

  final NotificationActor actor;

  @override
  ConsumerState<_NotificationFollowButton> createState() =>
      _NotificationFollowButtonState();
}

class _NotificationFollowButtonState
    extends ConsumerState<_NotificationFollowButton> {
  late bool _isFollowing;
  bool _isBusy = false;

  @override
  void initState() {
    super.initState();
    _isFollowing = widget.actor.viewerIsFollowing;
  }

  @override
  void didUpdateWidget(covariant _NotificationFollowButton oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.actor.did != widget.actor.did ||
        oldWidget.actor.viewerIsFollowing != widget.actor.viewerIsFollowing) {
      _isFollowing = widget.actor.viewerIsFollowing;
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final l10n = AppLocalizations.of(context);
    return ChunkyButton(
      onPressed: _isBusy ? null : _toggle,
      backgroundColor: _isFollowing ? swatches.paper3 : null,
      foregroundColor: _isFollowing ? theme.colorScheme.onSurface : null,
      style: const ButtonStyle(
        minimumSize: WidgetStatePropertyAll(Size(64, 36)),
        padding: WidgetStatePropertyAll(
          EdgeInsets.symmetric(horizontal: 18, vertical: 8),
        ),
      ),
      child: Text(
        _isFollowing ? l10n.profileFollowingAction : l10n.profileFollowAction,
      ),
    );
  }

  Future<void> _toggle() async {
    if (_isBusy) return;
    final previous = _isFollowing;
    setState(() {
      _isFollowing = !previous;
      _isBusy = true;
    });
    try {
      final repository = ref.read(profileRepositoryProvider);
      final updated = previous
          ? await repository.unfollow(widget.actor.did.toString())
          : await repository.follow(widget.actor.did.toString());
      if (!mounted) return;
      setState(() => _isFollowing = updated.viewerIsFollowing);
      ref
        ..invalidate(userProfileProvider(widget.actor.did.toString()))
        ..invalidate(userProfileProvider(widget.actor.handle.toString()));
    } on Object {
      if (!mounted) return;
      setState(() => _isFollowing = previous);
      context.showError(
        AppLocalizations.of(context).profileFollowToggleError,
      );
    } finally {
      if (mounted) setState(() => _isBusy = false);
    }
  }
}

Color _actionColor(
  CraftskyNotification notification,
  ColorScheme colors,
) => switch (notification) {
  FollowNotification() => colors.primary,
  LikeNotification() => colors.error,
  RepostNotification() => colors.tertiary,
  ReplyNotification() => colors.primary,
  MentionNotification() || QuoteNotification() => colors.secondary,
  GenericNotification() => colors.outline,
  UnavailableNotification() => colors.error,
};

TextSpan _titleSpan({required String title, required String actor}) {
  final actorIndex = title.indexOf(actor);
  if (actorIndex < 0) {
    return TextSpan(
      children: [
        TextSpan(
          text: actor,
          style: const TextStyle(fontWeight: FontWeight.bold),
        ),
        TextSpan(text: ' · $title'),
      ],
    );
  }
  return TextSpan(
    children: [
      if (actorIndex > 0) TextSpan(text: title.substring(0, actorIndex)),
      TextSpan(
        text: actor,
        style: const TextStyle(fontWeight: FontWeight.bold),
      ),
      if (actorIndex + actor.length < title.length)
        TextSpan(text: title.substring(actorIndex + actor.length)),
    ],
  );
}

enum _NotificationContentRole { post, comment, reply }

_NotificationContentRole _roleOf(Post post) {
  final reply = post.reply;
  if (reply == null) return _NotificationContentRole.post;
  return reply.parent.uri == reply.root.uri
      ? _NotificationContentRole.comment
      : _NotificationContentRole.reply;
}

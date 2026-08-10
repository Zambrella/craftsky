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
import 'package:craftsky_app/profile/widgets/profile_card_modal.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/time/relative_time_text.dart';
import 'package:craftsky_app/shared/widgets/post_summary.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/gestures.dart';
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
    if (notification case final SystemNotification system) {
      return _buildSystem(context, ref, system);
    }
    final actorNotification = notification as ActorNotification;
    final registry = ref.watch(sessionRegistryProvider).value;
    final account = owner?.account ?? registry?.activeLease?.session.account;
    final actorRelationshipProvider =
        account == null ||
            !actorNotification.actor.available ||
            account.did == actorNotification.actor.did
        ? null
        : profileRelationshipProvider(
            account,
            actorNotification.actor.did.toString(),
          );
    final cachedRelationship = actorRelationshipProvider == null
        ? null
        : ref.watch(actorRelationshipProvider);
    final serverRelationship = actorNotification.actor.hasViewerState
        ? ProfileRelationship.fromProfileFlags(
            muted: actorNotification.actor.muted ?? false,
            blocking: actorNotification.actor.blocking ?? false,
            blockedBy: actorNotification.actor.blockedBy ?? false,
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
        : actorNotification.actor.hasViewerState
        ? serverRelationship
        : null;
    if ((relationship?.muted ?? false) || (relationship?.hasBlock ?? false)) {
      return const SizedBox.shrink();
    }
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final actor = actorNotification.actor.displayLabel;
    final actionColor = _actionColor(actorNotification, theme.colorScheme);
    final (title, subjectPost) = switch (actorNotification) {
      FollowNotification() => (l10n.notificationFollowRow(actor), null),
      InstagramMatchNotification() => (
        l10n.notificationInstagramMatchActorRow(actor),
        null,
      ),
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
        subjectPost,
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
        subjectPost,
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
        subjectPost,
      ),
      MentionNotification(:final subjectPost) => (
        l10n.notificationMentionRow(actor),
        subjectPost,
      ),
      QuoteNotification(:final subjectPost) => (
        l10n.notificationQuoteRow(actor),
        subjectPost,
      ),
      GenericNotification() => (l10n.notificationGenericRow, null),
      UnavailableNotification() => (l10n.notificationUnavailableRow, null),
    };
    final onTap = actorNotification is GenericNotification
        ? null
        : () => _open(context, ref);
    final onActorTap = actorNotification.actor.available
        ? () => unawaited(
            showUserProfileCard(
              context,
              handleOrDid: actorNotification.actor.handle.toString(),
            ),
          )
        : null;
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
                    GestureDetector(
                      behavior: HitTestBehavior.opaque,
                      onTap: onActorTap,
                      child: ProfileAvatar(
                        seed: actor,
                        avatarUrl: actorNotification.actor.displayAvatarUrl,
                        size: ProfileAvatarSize.small,
                        customisation:
                            actorNotification.actor.customisation,
                      ),
                    ),
                    const SizedBox(height: 8),
                    Wrap(
                      spacing: 6,
                      runSpacing: 2,
                      crossAxisAlignment: WrapCrossAlignment.center,
                      children: [
                        _NotificationTitleText(
                          title: title,
                          actor: actor,
                          onActorTap: onActorTap,
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
                    if (subjectPost != null) ...[
                      const SizedBox(height: 4),
                      PostSummary(
                        data: PostSummaryData.notificationSubject(subjectPost),
                        padding: EdgeInsets.zero,
                      ),
                    ],
                    if ((actorNotification is FollowNotification ||
                            actorNotification is InstagramMatchNotification) &&
                        actorNotification.actor.available) ...[
                      const SizedBox(height: 8),
                      _NotificationFollowButton(
                        actor: actorNotification.actor,
                        owner: owner,
                      ),
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

  Widget _buildSystem(
    BuildContext context,
    WidgetRef ref,
    SystemNotification system,
  ) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final title = l10n.notificationGenericRow;
    return Material(
      type: MaterialType.transparency,
      child: InkWell(
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          child: Row(
            children: [
              ExcludeSemantics(
                child: Icon(
                  notificationCategoryIcon(system.type),
                  color: _actionColor(system, theme.colorScheme),
                  size: 24,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Wrap(
                  spacing: 6,
                  runSpacing: 2,
                  crossAxisAlignment: WrapCrossAlignment.center,
                  children: [
                    Text(title, style: theme.textTheme.bodyLarge),
                    Text(
                      '·',
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.outline,
                      ),
                    ),
                    RelativeTimeText(timestamp: system.createdAt),
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
      case GenericSystemNotification():
        break;
      case InstagramMatchNotification(:final actor):
        unawaited(
          UserProfileRoute(handle: actor.handle.toString()).push<void>(context),
        );
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
  const _NotificationFollowButton({required this.actor, required this.owner});

  final NotificationActor actor;
  final AccountSessionLease? owner;

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
    if (_isBusy || !_isOwnerCurrent()) return;
    final previous = _isFollowing;
    setState(() {
      _isFollowing = !previous;
      _isBusy = true;
    });
    try {
      final owner = widget.owner;
      final repository = owner == null
          ? ref.read(profileRepositoryProvider)
          : await ref.read(
              accountRelationshipRepositoryProvider(owner.account).future,
            );
      if (!_isOwnerCurrent()) return;
      final updated = previous
          ? await repository.unfollow(widget.actor.did.toString())
          : await repository.follow(widget.actor.did.toString());
      if (!mounted || !_isOwnerCurrent()) return;
      setState(() => _isFollowing = updated.viewerIsFollowing);
      ref
        ..invalidate(userProfileProvider(widget.actor.did.toString()))
        ..invalidate(userProfileProvider(widget.actor.handle.toString()));
    } on Object {
      if (!mounted || !_isOwnerCurrent()) return;
      setState(() => _isFollowing = previous);
      context.showError(
        AppLocalizations.of(context).profileFollowToggleError,
      );
    } finally {
      if (mounted && _isOwnerCurrent()) {
        setState(() => _isBusy = false);
      }
    }
  }

  bool _isOwnerCurrent() {
    final owner = widget.owner;
    if (owner == null) return true;
    return ref.read(sessionRegistryProvider).value?.activeLease?.session ==
        owner;
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
  InstagramMatchNotification() => colors.primary,
  GenericNotification() => colors.outline,
  GenericSystemNotification() => colors.outline,
  UnavailableNotification() => colors.error,
};

class _NotificationTitleText extends StatefulWidget {
  const _NotificationTitleText({
    required this.title,
    required this.actor,
    required this.onActorTap,
    required this.style,
  });

  final String title;
  final String actor;
  final VoidCallback? onActorTap;
  final TextStyle? style;

  @override
  State<_NotificationTitleText> createState() => _NotificationTitleTextState();
}

class _NotificationTitleTextState extends State<_NotificationTitleText> {
  late final TapGestureRecognizer _actorTapRecognizer;

  @override
  void initState() {
    super.initState();
    _actorTapRecognizer = TapGestureRecognizer()..onTap = widget.onActorTap;
  }

  @override
  void didUpdateWidget(covariant _NotificationTitleText oldWidget) {
    super.didUpdateWidget(oldWidget);
    _actorTapRecognizer.onTap = widget.onActorTap;
  }

  @override
  void dispose() {
    _actorTapRecognizer.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Text.rich(
      _titleSpan(
        title: widget.title,
        actor: widget.actor,
        actorRecognizer: _actorTapRecognizer,
      ),
      style: widget.style,
    );
  }
}

TextSpan _titleSpan({
  required String title,
  required String actor,
  GestureRecognizer? actorRecognizer,
}) {
  final actorIndex = title.indexOf(actor);
  if (actorIndex < 0) {
    return TextSpan(
      children: [
        TextSpan(
          text: actor,
          style: const TextStyle(fontWeight: FontWeight.bold),
          recognizer: actorRecognizer,
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
        recognizer: actorRecognizer,
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

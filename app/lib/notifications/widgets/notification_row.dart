import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart' as auth_model;
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_uri.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/notifications/widgets/notification_category_icon.dart';
import 'package:craftsky_app/profile/models/profile_relationship.dart';
import 'package:craftsky_app/profile/providers/block_profile_overlay.dart';
import 'package:craftsky_app/profile/providers/profile_relationship_provider.dart';
import 'package:craftsky_app/profile/providers/toggle_block_profile_provider.dart';
import 'package:craftsky_app/profile/providers/toggle_follow_profile_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/profile/widgets/follow_reaction.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar_hero.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/time/relative_time_text.dart';
import 'package:craftsky_app/shared/widgets/action_progress.dart';
import 'package:craftsky_app/shared/widgets/post_summary.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

bool canOpenNotificationRow(
  AccountSessionLease owner,
  auth_model.SessionRegistry registry,
) => registry.activeLease?.session == owner;

class NotificationRow extends ConsumerWidget {
  const new({
    required this.notification,
    this.owner,
    this.showNewness = true,
    super.key,
  });

  final CraftskyNotification notification;
  final bool showNewness;
  final AccountSessionLease? owner;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    if (notification is! FollowNotification &&
        notification is! InstagramMatchNotification) {
      return _build(context, ref);
    }
    return FollowReactionScope(
      identity: notification.id,
      builder: (context) =>
          Consumer(builder: (context, ref, _) => _build(context, ref)),
    );
  }

  Widget _build(BuildContext context, WidgetRef ref) {
    if (notification case final SystemNotification system) {
      return _buildSystem(context, ref, system);
    }
    final actorNotification = notification as ActorNotification;
    ref.watch(toggleBlockProfileProvider);
    final registry = ref.watch(sessionRegistryProvider).value;
    final account = owner?.account ?? registry?.activeLease?.session.account;
    final actorRelationshipProvider =
        account == null ||
            !actorNotification.actor.available ||
            account.did == actorNotification.actor.did
        ? null
        : profileRelationshipProvider(account, actorNotification.actor.did);
    final cachedRelationship = actorRelationshipProvider == null
        ? null
        : ref.watch(actorRelationshipProvider);
    final serverRelationship = applyBlockRelationshipOverlay(
      ref.read,
      actorNotification.actor.hasViewerState
          ? ProfileRelationship.fromProfileFlags(
              muted: actorNotification.actor.muted ?? false,
              blocking: actorNotification.actor.blocking ?? false,
              blockedBy: actorNotification.actor.blockedBy ?? false,
            )
          : const ProfileRelationship(initialized: true),
      actorNotification.actor.did,
    );
    final relationship = applyBlockRelationshipOverlay(
      ref.read,
      cachedRelationship?.initialized ?? false
          ? cachedRelationship!
          : actorNotification.actor.hasViewerState
          ? serverRelationship
          : const ProfileRelationship(initialized: true),
      actorNotification.actor.did,
    );
    if (relationship.muted || relationship.hasBlock) {
      return const SizedBox.shrink();
    }
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final actor = actorNotification.actor.displayLabel(l10n.handleUnavailable);
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
        switch (_roleOf(subjectPost)) {
          _NotificationContentRole.post => l10n.notificationMentionPostRow(
            actor,
          ),
          _NotificationContentRole.comment =>
            l10n.notificationMentionCommentRow(actor),
          _NotificationContentRole.reply => l10n.notificationMentionReplyRow(
            actor,
          ),
        },
        subjectPost,
      ),
      QuoteNotification(:final subjectPost) => (
        switch (actorNotification.quotedPreview?.role) {
          'comment' => l10n.notificationQuoteCommentRow(actor),
          'reply' => l10n.notificationQuoteReplyRow(actor),
          _ => l10n.notificationQuoteRow(actor),
        },
        subjectPost,
      ),
      GenericNotification() => (l10n.notificationGenericRow, null),
      UnavailableNotification() => (l10n.notificationUnavailableRow, null),
    };
    final replyContext = actorNotification is ReplyNotification
        ? _replyContext(actorNotification.subjectPost, l10n)
        : null;
    return ProfileAvatarHeroSource(
      did: actorNotification.actor.did,
      handle: actorNotification.actor.handle.toString(),
      displayName: actorNotification.actor.displayName,
      crafts: actorNotification.actor.crafts,
      avatar: ProfileAvatar(
        seed: actor,
        avatarUrl: actorNotification.actor.displayAvatarUrl,
        showShadow: false,
        customisation: actorNotification.actor.customisation,
      ),
      builder: (context, avatar, openProfile) {
        void openActorProfile() {
          final rowOwner = owner;
          if (rowOwner != null &&
              !canOpenNotificationRow(
                rowOwner,
                ref.read(sessionRegistryProvider).requireValue,
              )) {
            return;
          }
          openProfile();
        }

        return _NotificationSurface(
          isNew: notification.isNew && showNewness,
          child: InkWell(
            onTap: actorNotification is GenericNotification
                ? null
                : (actorNotification is FollowNotification ||
                      actorNotification is InstagramMatchNotification)
                ? openActorProfile
                : () => _open(context, ref),
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Semantics(
                    button: actorNotification.actor.available,
                    label: actor,
                    child: GestureDetector(
                      behavior: HitTestBehavior.opaque,
                      onTap: actorNotification.actor.available
                          ? openActorProfile
                          : null,
                      child: Stack(
                        children: [
                          Padding(
                            padding: const EdgeInsetsDirectional.only(
                              end: 4,
                              bottom: 4,
                            ),
                            child: FollowReactionAvatar(
                              compact: true,
                              child: avatar,
                            ),
                          ),
                          PositionedDirectional(
                            end: 0,
                            bottom: 0,
                            child: ExcludeSemantics(
                              child: Container(
                                width: 26,
                                height: 26,
                                decoration: BoxDecoration(
                                  color: actionColor,
                                  shape: BoxShape.circle,
                                  border: Border.all(
                                    color: theme.scaffoldBackgroundColor,
                                    width: 2,
                                  ),
                                ),
                                child: Icon(
                                  notificationCategoryIcon(
                                    notification.type,
                                    filled: true,
                                  ),
                                  color:
                                      ThemeData.estimateBrightnessForColor(
                                            actionColor,
                                          ) ==
                                          Brightness.dark
                                      ? Colors.white
                                      : Colors.black,
                                  size: 14,
                                ),
                              ),
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        _NotificationTitleText(
                          title: title,
                          actor: actor,
                          onActorTap: actorNotification.actor.available
                              ? openActorProfile
                              : null,
                          style: theme.textTheme.bodyLarge,
                        ),
                        if (replyContext != null) ...[
                          const SizedBox(height: 4),
                          Text(
                            replyContext,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: theme.textTheme.bodySmall?.copyWith(
                              color: theme.colorScheme.outline,
                            ),
                          ),
                        ],
                        if (subjectPost != null) ...[
                          const SizedBox(height: 4),
                          PostSummary(
                            maxTextLines: 3,
                            data: _previewData(
                              actorNotification as SubjectPostNotification,
                            ),
                            padding: EdgeInsets.zero,
                          ),
                        ],
                        if (actorNotification case QuoteNotification(
                          :final quotedPreview?,
                        )) ...[
                          const SizedBox(height: 8),
                          Container(
                            padding: const EdgeInsets.only(left: 12),
                            decoration: BoxDecoration(
                              border: Border(
                                left: BorderSide(
                                  color: theme.colorScheme.outlineVariant,
                                  width: 2,
                                ),
                              ),
                            ),
                            child: PostSummary(
                              maxTextLines: 3,
                              data: PostSummaryData(
                                state: PostSummaryState.visible,
                                text: quotedPreview.text,
                                projectTitle: quotedPreview.projectTitle,
                              ),
                              padding: EdgeInsets.zero,
                            ),
                          ),
                        ],
                        const SizedBox(height: 4),
                        RelativeTimeText(timestamp: notification.createdAt),
                        if ((actorNotification is FollowNotification ||
                                actorNotification
                                    is InstagramMatchNotification) &&
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
      },
    );
  }

  String? _replyContext(Post parent, AppLocalizations l10n) {
    if (parent.availability != null && parent.availability != 'visible') {
      return null;
    }
    final isPost = parent.reply == null;
    final projectTitle = isPost ? parent.project?.common.title?.trim() : null;
    final text =
        (projectTitle?.isNotEmpty == true ? projectTitle! : parent.text)
            .replaceAll(RegExp(r'\s+'), ' ')
            .trim();
    if (text.isEmpty) return null;
    return isPost
        ? l10n.notificationPostContext(text)
        : l10n.notificationReplyContext(text);
  }

  PostSummaryData _previewData(SubjectPostNotification notification) {
    final preview = notification.preview;
    if (preview == null) {
      // A reply's parent is context, never a substitute for the new reply.
      if (notification is ReplyNotification) {
        return const PostSummaryData(state: PostSummaryState.visible);
      }
      return PostSummaryData.notificationSubject(notification.subjectPost);
    }
    return PostSummaryData(
      state: PostSummaryState.visible,
      text: preview.text,
      projectTitle: preview.projectTitle,
    );
  }

  Widget _buildSystem(
    BuildContext context,
    WidgetRef ref,
    SystemNotification system,
  ) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final title = switch (system) {
      ModerationNotification() => l10n.notificationModerationRow,
      GenericSystemNotification() => l10n.notificationGenericRow,
    };
    return _NotificationSurface(
      isNew: notification.isNew && showNewness,
      child: InkWell(
        onTap: system is ModerationNotification
            ? () => _open(context, ref)
            : null,
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
      case InstagramMatchNotification(:final actor):
        unawaited(UserProfileRoute(did: actor.did).push<void>(context));
      case LikeNotification(:final subjectPost):
      case RepostNotification(:final subjectPost):
      case MentionNotification(:final subjectPost):
      case QuoteNotification(:final subjectPost):
        _openPost(context, subjectPost);
      case GenericNotification():
        break;
      case GenericSystemNotification():
        break;
      case ModerationNotification():
        unawaited(const AccountStandingRoute().push<void>(context));
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
      ).push<void>(context),
    );
  }
}

class _NotificationSurface extends StatelessWidget {
  const new({required this.isNew, required this.child});

  final bool isNew;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final row = Material(type: MaterialType.transparency, child: child);
    if (!isNew) return row;
    final color = Theme.of(context).colorScheme.primary;
    return Semantics(
      label: AppLocalizations.of(context).notificationNew,
      child: ColoredBox(
        key: const ValueKey('new-notification-highlight'),
        color: color.withValues(alpha: .07),
        child: Stack(
          children: [
            row,
            PositionedDirectional(
              top: 8,
              end: 8,
              child: ExcludeSemantics(
                child: Container(
                  width: 6,
                  height: 6,
                  decoration: BoxDecoration(
                    color: color,
                    shape: BoxShape.circle,
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _NotificationFollowButton extends ConsumerStatefulWidget {
  const new({required this.actor, required this.owner});

  final NotificationActor actor;
  final AccountSessionLease? owner;

  @override
  ConsumerState<_NotificationFollowButton> createState() =>
      _NotificationFollowButtonState();
}

class _NotificationFollowButtonState
    extends ConsumerState<_NotificationFollowButton> {
  bool _isBusy = false;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final l10n = AppLocalizations.of(context);
    // Share the profile page's reconciled follow state. The notification's
    // snapshot is only a fallback while the profile is unavailable.
    final isFollowing =
        ref.watch(
          userProfileProvider(widget.actor.did).select(
            (profile) => profile.value?.viewerIsFollowing,
          ),
        ) ??
        widget.actor.viewerIsFollowing;
    return ChunkyButton(
      onPressed: _isBusy ? null : _toggle,
      backgroundColor: isFollowing ? swatches.paper3 : null,
      foregroundColor: isFollowing ? theme.colorScheme.onSurface : null,
      style: const ButtonStyle(
        minimumSize: WidgetStatePropertyAll(Size(64, 36)),
        padding: WidgetStatePropertyAll(
          EdgeInsets.symmetric(horizontal: 18, vertical: 8),
        ),
      ),
      child: ActionProgress(
        isLoading: _isBusy,
        child: Text(
          isFollowing ? l10n.profileFollowingAction : l10n.profileFollowAction,
        ),
      ),
    );
  }

  Future<void> _toggle() => FollowReactionScope.run(context, _toggleAndConfirm);

  Future<bool> _toggleAndConfirm() async {
    if (_isBusy || !_isOwnerCurrent()) return false;
    setState(() => _isBusy = true);
    // Keep auto-disposed providers alive across the profile read and mutation.
    final profileSubscription = ref.listenManual(
      userProfileProvider(widget.actor.did),
      (_, _) {},
    );
    final followSubscription = ref.listenManual(
      toggleFollowProfileProvider,
      (_, _) {},
    );
    try {
      final profile = await ref.read(
        userProfileProvider(widget.actor.did).future,
      );
      if (!mounted || !_isOwnerCurrent()) return false;
      await ref
          .read(toggleFollowProfileProvider.notifier)
          .toggle(cacheKey: widget.actor.did, profile: profile);
      if (!mounted || !_isOwnerCurrent()) return false;
      final result = ref.read(toggleFollowProfileProvider);
      if (result.hasError) throw Exception('Follow mutation failed');
      final updated = result.value;
      if (updated != null) {
        return !profile.viewerIsFollowing && updated.viewerIsFollowing;
      }
      return false;
    } on Object {
      if (!mounted || !_isOwnerCurrent()) return false;
      context.showError(AppLocalizations.of(context).profileFollowToggleError);
      return false;
    } finally {
      profileSubscription.close();
      followSubscription.close();
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

Color _actionColor(CraftskyNotification notification, ColorScheme colors) =>
    switch (notification) {
      FollowNotification() => colors.primary,
      InstagramMatchNotification() => colors.primary,
      LikeNotification() => colors.error,
      RepostNotification() => colors.tertiary,
      ReplyNotification() => colors.primary,
      MentionNotification() || QuoteNotification() => colors.secondary,
      GenericNotification() => colors.outline,
      GenericSystemNotification() => colors.outline,
      ModerationNotification() => colors.primary,
      UnavailableNotification() => colors.error,
    };

class _NotificationTitleText extends StatelessWidget {
  const new({
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
  Widget build(BuildContext context) {
    final actorIndex = title.indexOf(actor);
    return LayoutBuilder(
      builder: (context, constraints) => Text.rich(
        TextSpan(
          children: [
            if (actorIndex > 0) TextSpan(text: title.substring(0, actorIndex)),
            WidgetSpan(
              alignment: PlaceholderAlignment.baseline,
              baseline: TextBaseline.alphabetic,
              child: Semantics(
                label: actor,
                button: onActorTap != null,
                onTap: onActorTap,
                excludeSemantics: true,
                child: GestureDetector(
                  onTap: onActorTap,
                  child: ConstrainedBox(
                    constraints: BoxConstraints(maxWidth: constraints.maxWidth),
                    child: ProfileIdentityTextHero.source(
                      context,
                      part: ProfileIdentityHeroPart.name,
                      preservePlaceholderBaseline: true,
                      child: Text(
                        actor,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: style?.copyWith(fontWeight: FontWeight.w600),
                      ),
                    ),
                  ),
                ),
              ),
            ),
            TextSpan(
              text: actorIndex < 0
                  ? ' · $title'
                  : title.substring(actorIndex + actor.length),
            ),
          ],
        ),
        style: style,
      ),
    );
  }
}

enum _NotificationContentRole { post, comment, reply }

_NotificationContentRole _roleOf(Post post) {
  final reply = post.reply;
  if (reply == null) return _NotificationContentRole.post;
  return reply.parent.uri == reply.root.uri
      ? _NotificationContentRole.comment
      : _NotificationContentRole.reply;
}

import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/shared/image/image_cache_providers.dart';
import 'package:craftsky_app/shared/time/relative_time_text.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:dart_mappable/dart_mappable.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

part 'post_summary.mapper.dart';

enum PostSummaryState { visible, muted, hidden, blocked, unavailable }

@immutable
@MappableClass(generateMethods: GenerateMethods.copy | GenerateMethods.equals)
final class PostSummaryData with PostSummaryDataMappable {
  const PostSummaryData({
    required this.state,
    this.author,
    this.text,
    this.createdAt,
    this.projectTitle,
    this.image,
    this.revealable = false,
  });

  factory PostSummaryData.fromPost(Post post) {
    if (post.availability == 'muted') {
      return PostSummaryData(
        state: PostSummaryState.muted,
        revealable: post.relationship?.revealable ?? false,
      );
    }
    if (post.availability == 'blocked') {
      return const PostSummaryData(state: PostSummaryState.blocked);
    }
    return PostSummaryData._visible(
      author: post.author,
      text: post.text,
      createdAt: post.createdAt,
      projectTitle: post.project?.common.title,
      images: post.images,
    );
  }

  factory PostSummaryData.fromQuoteView(QuoteView quote) => switch ((
    quote.state,
    quote.post,
  )) {
    ('visible', final post?) => PostSummaryData._visible(
      author: post.author,
      text: post.text,
      createdAt: post.createdAt,
      projectTitle: post.project?.common.title,
      images: post.images,
    ),
    ('muted', _) => PostSummaryData(
      state: PostSummaryState.muted,
      revealable: quote.revealable ?? false,
    ),
    ('hidden', _) => const PostSummaryData(state: PostSummaryState.hidden),
    ('blocked', _) => const PostSummaryData(state: PostSummaryState.blocked),
    _ => const PostSummaryData(state: PostSummaryState.unavailable),
  };

  factory PostSummaryData.notificationSubject(Post post) => PostSummaryData(
    state: PostSummaryState.visible,
    text: post.text,
  );

  factory PostSummaryData.savedPost(Post post) {
    if (post.availability == 'muted') {
      return PostSummaryData(
        state: PostSummaryState.muted,
        revealable: post.relationship?.revealable ?? false,
      );
    }
    if (post.availability == 'blocked') {
      return const PostSummaryData(state: PostSummaryState.blocked);
    }
    return PostSummaryData._visible(
      author: post.author,
      text: post.text,
      projectTitle: post.project?.common.title,
      images: post.images,
    );
  }

  factory PostSummaryData._visible({
    required PostAuthor author,
    required String text,
    required String? projectTitle,
    required List<PostImage>? images,
    DateTime? createdAt,
  }) => PostSummaryData(
    state: PostSummaryState.visible,
    author: author,
    text: text,
    createdAt: createdAt,
    projectTitle: switch (projectTitle?.trim()) {
      final title? when title.isNotEmpty => title,
      _ => null,
    },
    image: images?.firstOrNull,
  );

  final PostSummaryState state;
  final PostAuthor? author;
  final String? text;
  final DateTime? createdAt;
  final String? projectTitle;
  final PostImage? image;
  final bool revealable;

  @override
  String toString() => 'PostSummaryData(<redacted>)';
}

class PostSummary extends StatelessWidget {
  const PostSummary({
    required this.data,
    this.onTap,
    this.onAuthorTap,
    this.onReveal,
    this.padding = const EdgeInsets.all(12),
    super.key,
  });

  final PostSummaryData data;
  final VoidCallback? onTap;
  final VoidCallback? onAuthorTap;
  final VoidCallback? onReveal;
  final EdgeInsetsGeometry padding;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return switch (data.state) {
      PostSummaryState.visible => InkWell(
        onTap: onTap,
        child: Padding(
          padding: padding,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              if (data.author case final author?)
                GestureDetector(
                  behavior: HitTestBehavior.opaque,
                  onTap: onAuthorTap,
                  child: _PostSummaryAuthor(author: author),
                ),
              if (data.image case final image?) ...[
                const SizedBox(height: 8),
                _PostSummaryImage(image: image),
              ],
              if (data.projectTitle case final title?) ...[
                const SizedBox(height: 8),
                Text(title, style: Theme.of(context).textTheme.titleSmall),
              ],
              if (data.text case final text?) ...[
                if (data.author != null ||
                    data.image != null ||
                    data.projectTitle != null)
                  const SizedBox(height: 8),
                Text(text, maxLines: 4, overflow: TextOverflow.ellipsis),
              ],
              if (data.createdAt case final createdAt?) ...[
                const SizedBox(height: 4),
                RelativeTimeText(timestamp: createdAt),
              ],
            ],
          ),
        ),
      ),
      PostSummaryState.muted => Padding(
        padding: padding,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(l10n.postMutedPlaceholder),
            if (data.revealable && onReveal != null)
              TextButton(
                onPressed: onReveal,
                child: Text(l10n.postRevealAction),
              ),
          ],
        ),
      ),
      PostSummaryState.hidden => _PostSummaryPlaceholder(
        l10n.postQuoteHidden,
        padding: padding,
      ),
      PostSummaryState.blocked => _PostSummaryPlaceholder(
        l10n.postUnavailablePlaceholder,
        padding: padding,
      ),
      PostSummaryState.unavailable => _PostSummaryPlaceholder(
        l10n.postQuoteUnavailable,
        padding: padding,
      ),
    };
  }
}

class _PostSummaryAuthor extends StatelessWidget {
  const _PostSummaryAuthor({required this.author});

  final PostAuthor author;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final displayName = author.displayName;
    return Row(
      children: [
        ProfileAvatar(
          seed: displayName ?? author.handle,
          avatarUrl: author.avatar,
          size: ProfileAvatarSize.small,
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              if (displayName != null && displayName.trim().isNotEmpty)
                Text(
                  displayName,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: theme.textTheme.titleSmall,
                ),
              Text(
                '@${author.handle}',
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.outline,
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class _PostSummaryImage extends ConsumerWidget {
  const _PostSummaryImage({required this.image});

  final PostImage image;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final radii = Theme.of(context).extension<RadiusTheme>()!;
    final imageUrl = image.thumb ?? image.fullsize;
    return ClipRRect(
      key: const Key('post-summary-image'),
      borderRadius: BorderRadius.circular(radii.r1),
      child: SizedBox(
        width: double.infinity,
        height: 160,
        child: Semantics(
          label: image.alt,
          image: true,
          child: imageUrl == null
              ? const DecoratedBox(
                  decoration: BoxDecoration(color: Color(0xFFEAEAEA)),
                )
              : CachedNetworkImage(
                  imageUrl: imageUrl,
                  cacheManager: ref.watch(feedImageCacheManagerProvider),
                  fit: BoxFit.cover,
                ),
        ),
      ),
    );
  }
}

class _PostSummaryPlaceholder extends StatelessWidget {
  const _PostSummaryPlaceholder(this.text, {required this.padding});

  final String text;
  final EdgeInsetsGeometry padding;

  @override
  Widget build(BuildContext context) => Padding(
    padding: padding,
    child: Text(
      text,
      style: Theme.of(context).textTheme.bodyMedium?.copyWith(
        color: Theme.of(context).colorScheme.outline,
      ),
    ),
  );
}

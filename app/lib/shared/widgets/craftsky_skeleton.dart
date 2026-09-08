import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:skeletonizer/skeletonizer.dart';

const _shimmerDuration = Duration(milliseconds: 1400);

/// Applies CraftSky's loading treatment to a box widget subtree.
class CraftskySkeleton extends StatelessWidget {
  const CraftskySkeleton({required this.child, super.key});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Semantics(
      container: true,
      liveRegion: true,
      label: AppLocalizations.of(context).loading,
      excludeSemantics: true,
      child: Skeletonizer.zone(
        effect: craftskySkeletonEffect(context),
        child: child,
      ),
    );
  }
}

/// A finite, non-interactive skeleton list for box-based page bodies.
class CraftskySkeletonList extends StatelessWidget {
  const CraftskySkeletonList({
    required this.itemBuilder,
    super.key,
    this.itemCount = 6,
    this.padding,
  });

  final IndexedWidgetBuilder itemBuilder;
  final int itemCount;
  final EdgeInsetsGeometry? padding;

  @override
  Widget build(BuildContext context) {
    return CraftskySkeleton(
      child: ListView.builder(
        padding: padding,
        physics: const NeverScrollableScrollPhysics(),
        itemCount: itemCount,
        itemBuilder: itemBuilder,
      ),
    );
  }
}

/// A finite skeleton list that can be inserted directly into a scroll view.
class CraftskySkeletonSliverList extends StatelessWidget {
  const CraftskySkeletonSliverList({
    required this.itemBuilder,
    super.key,
    this.itemCount = 6,
  });

  final IndexedWidgetBuilder itemBuilder;
  final int itemCount;

  @override
  Widget build(BuildContext context) {
    return SliverMainAxisGroup(
      slivers: [
        SliverToBoxAdapter(
          child: Semantics(
            container: true,
            liveRegion: true,
            label: AppLocalizations.of(context).loading,
            child: const SizedBox(height: 1),
          ),
        ),
        SliverSkeletonizer.zone(
          effect: craftskySkeletonEffect(context),
          child: SliverList.builder(
            itemCount: itemCount,
            itemBuilder: (context, index) => ExcludeSemantics(
              child: itemBuilder(context, index),
            ),
          ),
        ),
      ],
    );
  }
}

@visibleForTesting
PaintingEffect craftskySkeletonEffect(BuildContext context) {
  final colors = Theme.of(context).colorScheme;
  final base = Color.alphaBlend(
    colors.onSurface.withValues(alpha: 0.10),
    colors.surface,
  );
  if (MediaQuery.disableAnimationsOf(context)) {
    return SolidColorEffect(color: base);
  }
  return ShimmerEffect(
    baseColor: base,
    highlightColor: Color.alphaBlend(
      colors.onSurface.withValues(alpha: 0.03),
      colors.surface,
    ),
    duration: _shimmerDuration,
  );
}

/// Placeholder for post and project cards used by feeds and search results.
class PostCardSkeleton extends StatelessWidget {
  const PostCardSkeleton({super.key, this.showMedia = false});

  final bool showMedia;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    return Padding(
      padding: EdgeInsets.symmetric(
        horizontal: spacing.sp4,
        vertical: spacing.sp3,
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Bone.circle(size: 40),
              SizedBox(width: spacing.sp3),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Bone.text(width: 132, style: theme.textTheme.titleSmall),
                    SizedBox(height: spacing.sp1),
                    Bone.text(width: 88, style: theme.textTheme.bodySmall),
                  ],
                ),
              ),
              const Bone.iconButton(size: 32),
            ],
          ),
          SizedBox(height: spacing.sp3),
          Bone.multiText(lines: 3, style: theme.textTheme.bodyMedium),
          if (showMedia) ...[
            SizedBox(height: spacing.sp3),
            AspectRatio(
              aspectRatio: 16 / 9,
              child: Bone(uniRadius: radii.r2),
            ),
          ],
          SizedBox(height: spacing.sp3),
          const Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Bone(width: 54, height: 22, uniRadius: 11),
              Bone(width: 54, height: 22, uniRadius: 11),
              Bone(width: 54, height: 22, uniRadius: 11),
              Bone(width: 32, height: 22, uniRadius: 11),
            ],
          ),
          SizedBox(height: spacing.sp3),
        ],
      ),
    );
  }
}

/// Placeholder for profile, hashtag, and relationship rows.
class AccountRowSkeleton extends StatelessWidget {
  const AccountRowSkeleton({super.key, this.showTrailing = true});

  final bool showTrailing;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    return Padding(
      padding: EdgeInsets.symmetric(
        horizontal: spacing.sp4,
        vertical: spacing.sp3,
      ),
      child: Row(
        children: [
          const Bone.circle(size: 48),
          SizedBox(width: spacing.sp3),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Bone.text(width: 128, style: theme.textTheme.titleSmall),
                SizedBox(height: spacing.sp1),
                Bone.text(width: 92, style: theme.textTheme.bodySmall),
              ],
            ),
          ),
          if (showTrailing) const Bone(width: 72, height: 36, uniRadius: 18),
        ],
      ),
    );
  }
}

/// Placeholder for notifications and other activity rows.
class ActivityRowSkeleton extends StatelessWidget {
  const ActivityRowSkeleton({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    return Padding(
      padding: EdgeInsets.symmetric(
        horizontal: spacing.sp4,
        vertical: spacing.sp3,
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Bone.circle(size: 24),
          SizedBox(width: spacing.sp3),
          const Bone.circle(size: 40),
          SizedBox(width: spacing.sp3),
          Expanded(
            child: Bone.multiText(style: theme.textTheme.bodyMedium),
          ),
        ],
      ),
    );
  }
}

/// Placeholder for comment and reply rows.
class CommentRowSkeleton extends StatelessWidget {
  const CommentRowSkeleton({super.key, this.indent = 0});

  final double indent;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    return Padding(
      padding: EdgeInsetsDirectional.fromSTEB(
        spacing.sp4 + indent,
        spacing.sp3,
        spacing.sp4,
        spacing.sp3,
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Bone.circle(size: 36),
          SizedBox(width: spacing.sp3),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Bone.text(width: 116, style: theme.textTheme.titleSmall),
                SizedBox(height: spacing.sp2),
                Bone.multiText(style: theme.textTheme.bodyMedium),
                SizedBox(height: spacing.sp2),
                const Bone(width: 110, height: 18, uniRadius: 9),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

/// Placeholder for drafts, saved posts, products, and events.
class ManagementRowSkeleton extends StatelessWidget {
  const ManagementRowSkeleton({super.key, this.imageSize = 72});

  final double imageSize;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    return Padding(
      padding: EdgeInsets.symmetric(
        horizontal: spacing.sp4,
        vertical: spacing.sp3,
      ),
      child: Row(
        children: [
          Bone.square(size: imageSize, uniRadius: radii.r2),
          SizedBox(width: spacing.sp3),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Bone.text(width: 156, style: theme.textTheme.titleSmall),
                SizedBox(height: spacing.sp2),
                Bone.text(width: 112, style: theme.textTheme.bodySmall),
                SizedBox(height: spacing.sp1),
                Bone.text(width: 84, style: theme.textTheme.bodySmall),
              ],
            ),
          ),
          const Bone.iconButton(size: 36),
        ],
      ),
    );
  }
}

/// Placeholder for event cards with optional hero media and metadata rows.
class EventCardSkeleton extends StatelessWidget {
  const EventCardSkeleton({super.key, this.showMedia = false});

  final bool showMedia;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    return CraftskyCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (showMedia) const AspectRatio(aspectRatio: 16 / 9, child: Bone()),
          Padding(
            padding: EdgeInsets.all(spacing.sp3),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Expanded(
                      child: Bone.text(
                        width: 180,
                        style: theme.textTheme.titleLarge,
                      ),
                    ),
                    const Bone.iconButton(size: 32),
                  ],
                ),
                SizedBox(height: spacing.sp2),
                for (final width in const [156.0, 116.0, 196.0]) ...[
                  Row(
                    children: [
                      const Bone.circle(size: 18),
                      SizedBox(width: spacing.sp2),
                      Bone(width: width, height: 14, uniRadius: 7),
                    ],
                  ),
                  SizedBox(height: spacing.sp1),
                ],
              ],
            ),
          ),
        ],
      ),
    );
  }
}

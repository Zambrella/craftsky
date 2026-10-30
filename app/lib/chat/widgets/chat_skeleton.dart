import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/shared/widgets/craftsky_skeleton.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:material_ui/material_ui.dart';
import 'package:skeletonizer/skeletonizer.dart';

/// Uses the same account rows and shimmer as feed-related lists.
class ChatListSkeleton extends StatelessWidget {
  const new({super.key});

  @override
  Widget build(BuildContext context) {
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    final minimumRowHeight = 48 + spacing.sp3 * 2;
    return LayoutBuilder(
      builder: (context, constraints) => CraftskySkeletonList(
        itemCount: (constraints.maxHeight / minimumRowHeight).ceil() + 1,
        itemBuilder: (context, index) =>
            const AccountRowSkeleton(showTrailing: false),
      ),
    );
  }
}

/// Message-shaped placeholders without fabricated identities or content.
class ChatConversationSkeleton extends StatelessWidget {
  const new({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final radii = theme.extension<RadiusTheme>()!;
    return LayoutBuilder(
      builder: (context, constraints) => CraftskySkeletonList(
        itemCount: (constraints.maxHeight / (48 + spacing.sp3)).ceil() + 1,
        padding: EdgeInsets.all(spacing.sp4),
        itemBuilder: (context, index) => Padding(
          padding: EdgeInsets.only(bottom: spacing.sp3),
          child: Align(
            alignment: index.isOdd
                ? AlignmentDirectional.centerEnd
                : AlignmentDirectional.centerStart,
            child: FractionallySizedBox(
              widthFactor: index % 3 == 0 ? 0.6 : 0.8,
              child: Bone(
                height: index.isEven ? 76 : 48,
                uniRadius: radii.r2,
              ),
            ),
          ),
        ),
      ),
    );
  }
}

/// Reserves the same avatar and two-line identity space as a loaded header.
class ChatProfileHeaderSkeleton extends StatelessWidget {
  const new({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    return CraftskySkeleton(
      child: ConstrainedBox(
        constraints: const BoxConstraints(minHeight: 48),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Bone.circle(size: ProfileAvatarSize.small.dimension),
            SizedBox(width: spacing.sp3),
            Flexible(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Bone.text(width: 132, style: theme.textTheme.titleMedium),
                  Bone.text(width: 92, style: theme.textTheme.bodySmall),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

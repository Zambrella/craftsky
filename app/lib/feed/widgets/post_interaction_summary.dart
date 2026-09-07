import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

class PostInteractionSummary extends StatelessWidget {
  const PostInteractionSummary({
    required this.post,
    required this.onLikes,
    required this.onReposts,
    required this.onQuotes,
    super.key,
  });

  final Post post;
  final VoidCallback onLikes;
  final VoidCallback onReposts;
  final VoidCallback onQuotes;

  @override
  Widget build(BuildContext context) {
    if (post.likeCount == 0 && post.repostCount == 0 && post.quoteCount == 0) {
      return const SizedBox.shrink();
    }

    final l10n = AppLocalizations.of(context);
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    final entries = <({String label, String hint, VoidCallback onPressed})>[
      if (post.likeCount > 0)
        (
          label: l10n.postInteractionLikesSummary(post.likeCount),
          hint: l10n.postInteractionLikesSummaryHint,
          onPressed: onLikes,
        ),
      if (post.repostCount > 0)
        (
          label: l10n.postInteractionRepostsSummary(post.repostCount),
          hint: l10n.postInteractionRepostsSummaryHint,
          onPressed: onReposts,
        ),
      if (post.quoteCount > 0)
        (
          label: l10n.postInteractionQuotesSummary(post.quoteCount),
          hint: l10n.postInteractionQuotesSummaryHint,
          onPressed: onQuotes,
        ),
    ];

    return Wrap(
      spacing: spacing.sp1,
      runSpacing: spacing.sp1,
      children: [
        for (final entry in entries)
          Semantics(
            button: true,
            label: entry.label,
            hint: entry.hint,
            onTap: entry.onPressed,
            excludeSemantics: true,
            child: TextButton(
              style: TextButton.styleFrom(
                minimumSize: const Size(48, 48),
                padding: EdgeInsets.symmetric(horizontal: spacing.sp2),
              ),
              onPressed: entry.onPressed,
              child: Text(entry.label),
            ),
          ),
      ],
    );
  }
}

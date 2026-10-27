import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/models/post_uri.dart';
import 'package:craftsky_app/feed/providers/post_provider.dart';
import 'package:craftsky_app/shared/widgets/post_summary.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

/// Fixed conversation context. These previews never supply replacement refs
/// to the edit payload, including when their current content has a newer CID.
final class PublishedPostEditContext extends ConsumerWidget {
  const new({required this.post, super.key});
  final Post post;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final parent = post.reply?.parent;
    final parts = parent == null ? null : parseCraftskyPostUri(parent.uri);
    final parentState = parts == null
        ? null
        : ref.watch(postProvider(parts.did, parts.rkey));
    const unavailable = PostSummaryData(state: PostSummaryState.unavailable);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (parent != null)
          switch (parentState) {
            AsyncData(:final value) => PostSummary(
              data:
                  value.isProtected &&
                      value.availability != 'muted' &&
                      value.availability != 'blocked'
                  ? unavailable
                  : PostSummaryData.fromPost(value),
            ),
            AsyncLoading() => const Center(
              child: StitchProgressIndicator(size: 18),
            ),
            _ => const PostSummary(data: unavailable),
          },
        if (post.quote != null)
          PostSummary(
            data: post.quoteView == null
                ? unavailable
                : PostSummaryData.fromQuoteView(post.quoteView!),
          ),
      ],
    );
  }
}

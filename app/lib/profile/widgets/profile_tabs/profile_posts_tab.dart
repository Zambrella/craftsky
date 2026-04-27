import 'package:craftsky_app/feed/models/placeholder_post.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:flutter/material.dart';

/// Posts tab body. Returns a [SliverList] so it slots into the page's
/// outer [CustomScrollView] without nesting another scrollable. Backed
/// by a hard-coded list of [PlaceholderPost] until feed wiring lands.
class ProfilePostsTab extends StatelessWidget {
  const ProfilePostsTab({required this.handle, super.key});

  final String handle;

  @override
  Widget build(BuildContext context) {
    final posts = _placeholderPosts(handle);
    return SliverList.builder(
      itemCount: posts.length,
      itemBuilder: (context, index) => PostCard(post: posts[index]),
    );
  }

  List<PlaceholderPost> _placeholderPosts(String handle) {
    final now = DateTime.now();
    return [
      PlaceholderPost(
        id: '1',
        authorHandle: handle,
        authorDisplayName: 'Maya Chen',
        body:
            'the binding tape on this jacket is fighting me. four attempts at '
            'the back collar and counting. taking a tea break.',
        postedAt: now.subtract(const Duration(hours: 2)),
        replyCount: 8,
        repostCount: 3,
        likeCount: 42,
      ),
      PlaceholderPost(
        id: '2',
        authorHandle: handle,
        authorDisplayName: 'Maya Chen',
        body:
            "today's haul from textile garden. the indigo cotton voile is "
            'sheer in the most beautiful way — already plotting a summer '
            'blouse.',
        postedAt: now.subtract(const Duration(days: 1)),
        replyCount: 4,
        repostCount: 1,
        likeCount: 28,
      ),
    ];
  }
}

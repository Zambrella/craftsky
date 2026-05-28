import 'dart:async';

import 'package:craftsky_app/profile/models/profile_account_page.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

enum FollowListKind { followers, following }

class FollowListPage extends ConsumerStatefulWidget {
  const FollowListPage({required this.kind, super.key});

  final FollowListKind kind;

  @override
  ConsumerState<FollowListPage> createState() => _FollowListPageState();
}

class _FollowListPageState extends ConsumerState<FollowListPage> {
  final _items = <ProfileAccountSummary>[];
  String? _cursor;
  int _totalCount = 0;
  var _isInitialLoading = true;
  var _isLoadingMore = false;

  @override
  void initState() {
    super.initState();
    unawaited(_loadFirstPage());
  }

  Future<void> _loadFirstPage() async {
    final page = await _fetchPage();
    if (!mounted) return;
    setState(() {
      _items
        ..clear()
        ..addAll(page.items);
      _cursor = page.cursor;
      _totalCount = page.totalCount;
      _isInitialLoading = false;
    });
  }

  Future<void> _loadMore() async {
    final cursor = _cursor;
    if (cursor == null || _isLoadingMore) return;
    setState(() => _isLoadingMore = true);
    final page = await _fetchPage(cursor: cursor);
    if (!mounted) return;
    setState(() {
      _items.addAll(page.items);
      _cursor = page.cursor;
      _totalCount = page.totalCount;
      _isLoadingMore = false;
    });
  }

  Future<ProfileAccountPage> _fetchPage({String? cursor}) {
    final repo = ref.read(profileRepositoryProvider);
    return switch (widget.kind) {
      FollowListKind.followers => repo.listFollowersMe(cursor: cursor),
      FollowListKind.following => repo.listFollowingMe(cursor: cursor),
    };
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text('$_title ($_totalCount)')),
      body: _isInitialLoading
          ? const Center(child: StitchProgressIndicator())
          : _FollowListBody(
              kind: widget.kind,
              items: _items,
              hasMore: _cursor != null,
              isLoadingMore: _isLoadingMore,
              onLoadMore: _loadMore,
            ),
    );
  }

  String get _title => switch (widget.kind) {
    FollowListKind.followers => 'Followers',
    FollowListKind.following => 'Following',
  };
}

class _FollowListBody extends StatelessWidget {
  const _FollowListBody({
    required this.kind,
    required this.items,
    required this.hasMore,
    required this.isLoadingMore,
    required this.onLoadMore,
  });

  final FollowListKind kind;
  final List<ProfileAccountSummary> items;
  final bool hasMore;
  final bool isLoadingMore;
  final VoidCallback onLoadMore;

  @override
  Widget build(BuildContext context) {
    if (items.isEmpty) {
      return Center(
        child: Text(
          switch (kind) {
            FollowListKind.followers => 'No one follows you yet',
            FollowListKind.following => 'You are not following anyone',
          },
        ),
      );
    }
    return ListView.builder(
      itemCount: items.length + (hasMore ? 1 : 0),
      itemBuilder: (context, index) {
        if (index == items.length) {
          return Padding(
            padding: const EdgeInsets.symmetric(vertical: 16),
            child: Center(
              child: isLoadingMore
                  ? const StitchProgressIndicator()
                  : TextButton(
                      onPressed: onLoadMore,
                      child: const Text('Load more'),
                    ),
            ),
          );
        }
        final account = items[index];
        final title = account.displayName?.isNotEmpty ?? false
            ? account.displayName!
            : account.handle.toString();
        return ListTile(
          title: Text(title),
          subtitle: Text('@${account.handle}'),
          trailing: const Icon(Icons.chevron_right),
          onTap: () => unawaited(
            UserProfileRoute(handle: account.handle.toString()).push<void>(
              context,
            ),
          ),
        );
      },
    );
  }
}

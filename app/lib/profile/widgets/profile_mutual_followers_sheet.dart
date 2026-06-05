import 'dart:async';

import 'package:craftsky_app/profile/models/profile_account_page.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class ProfileMutualFollowersSheet extends ConsumerStatefulWidget {
  const ProfileMutualFollowersSheet({
    required this.targetHandleOrDid,
    super.key,
  });

  final String targetHandleOrDid;

  @override
  ConsumerState<ProfileMutualFollowersSheet> createState() =>
      _ProfileMutualFollowersSheetState();
}

class _ProfileMutualFollowersSheetState
    extends ConsumerState<ProfileMutualFollowersSheet> {
  final _items = <ProfileAccountSummary>[];
  String? _cursor;
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
      _isLoadingMore = false;
    });
  }

  Future<ProfileAccountPage> _fetchPage({String? cursor}) {
    return ref
        .read(profileRepositoryProvider)
        .listMutualFollowers(widget.targetHandleOrDid, cursor: cursor);
  }

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      child: Column(
        children: [
          AppBar(
            title: const Text('Mutual followers'),
            automaticallyImplyLeading: false,
          ),
          Expanded(
            child: _isInitialLoading
                ? const Center(child: StitchProgressIndicator())
                : _ProfileMutualFollowersBody(
                    items: _items,
                    hasMore: _cursor != null,
                    isLoadingMore: _isLoadingMore,
                    onLoadMore: _loadMore,
                  ),
          ),
        ],
      ),
    );
  }
}

class _ProfileMutualFollowersBody extends StatelessWidget {
  const _ProfileMutualFollowersBody({
    required this.items,
    required this.hasMore,
    required this.isLoadingMore,
    required this.onLoadMore,
  });

  final List<ProfileAccountSummary> items;
  final bool hasMore;
  final bool isLoadingMore;
  final VoidCallback onLoadMore;

  @override
  Widget build(BuildContext context) {
    if (items.isEmpty) {
      return const SizedBox.shrink();
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
        );
      },
    );
  }
}

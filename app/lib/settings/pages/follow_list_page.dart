import 'dart:async';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile_account_page.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/profile/widgets/profile_account_list_tile.dart';
import 'package:craftsky_app/shared/widgets/craftsky_empty_state.dart';
import 'package:craftsky_app/shared/widgets/craftsky_skeleton.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
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
  var _isRefreshing = false;
  Object? _initialError;
  var _loadGeneration = 0;

  @override
  void initState() {
    super.initState();
    unawaited(_loadFirstPage());
  }

  Future<void> _loadFirstPage() async {
    final generation = ++_loadGeneration;
    final isInitialRequest = _isInitialLoading || _initialError != null;
    if (mounted) {
      setState(() {
        _isRefreshing = true;
        _isLoadingMore = false;
        if (isInitialRequest) {
          _isInitialLoading = true;
          _initialError = null;
        }
      });
    }
    late final ProfileAccountPage page;
    try {
      page = await _fetchPage();
    } on Object catch (error) {
      if (!mounted || generation != _loadGeneration) return;
      setState(() {
        _isInitialLoading = false;
        _isRefreshing = false;
        if (isInitialRequest) _initialError = error;
      });
      if (!isInitialRequest) _showLoadError();
      return;
    }
    if (!mounted || generation != _loadGeneration) return;
    setState(() {
      _items
        ..clear()
        ..addAll(page.items);
      _cursor = page.cursor;
      _totalCount = page.totalCount;
      _isInitialLoading = false;
      _isRefreshing = false;
      _initialError = null;
    });
  }

  Future<void> _loadMore() async {
    final cursor = _cursor;
    if (cursor == null || _isLoadingMore || _isRefreshing) return;
    final generation = _loadGeneration;
    setState(() => _isLoadingMore = true);
    late final ProfileAccountPage page;
    try {
      page = await _fetchPage(cursor: cursor);
    } on Object {
      if (!mounted || generation != _loadGeneration) return;
      setState(() => _isLoadingMore = false);
      _showLoadError();
      return;
    }
    if (!mounted || generation != _loadGeneration) return;
    setState(() {
      _items.addAll(page.items);
      _cursor = page.cursor;
      _totalCount = page.totalCount;
      _isLoadingMore = false;
    });
  }

  void _showLoadError() {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(AppLocalizations.of(context).errorBackgroundLoadFailed),
      ),
    );
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
    final l10n = AppLocalizations.of(context);
    final title = widget.kind == FollowListKind.followers
        ? l10n.settingsFollowers
        : l10n.settingsFollowing;
    return Scaffold(
      appBar: AppBar(title: Text('$title ($_totalCount)')),
      body: _isInitialLoading
          ? const CraftskySkeletonList(itemBuilder: _buildAccountSkeleton)
          : _initialError != null
          ? _FollowListError(onRetry: _loadFirstPage)
          : _FollowListBody(
              kind: widget.kind,
              items: _items,
              hasMore: _cursor != null,
              isLoadingMore: _isLoadingMore,
              onLoadMore: _loadMore,
              onRefresh: _loadFirstPage,
            ),
    );
  }
}

Widget _buildAccountSkeleton(BuildContext context, int index) =>
    const AccountRowSkeleton();

class _FollowListError extends StatelessWidget {
  const _FollowListError({required this.onRetry});

  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(l10n.errorBackgroundLoadFailed),
          TextButton(onPressed: onRetry, child: Text(l10n.retryButton)),
        ],
      ),
    );
  }
}

class _FollowListBody extends StatelessWidget {
  const _FollowListBody({
    required this.kind,
    required this.items,
    required this.hasMore,
    required this.isLoadingMore,
    required this.onLoadMore,
    required this.onRefresh,
  });

  final FollowListKind kind;
  final List<ProfileAccountSummary> items;
  final bool hasMore;
  final bool isLoadingMore;
  final VoidCallback onLoadMore;
  final RefreshCallback onRefresh;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return RefreshIndicator(
      onRefresh: onRefresh,
      child: CustomScrollView(
        physics: const AlwaysScrollableScrollPhysics(),
        slivers: [
          if (items.isEmpty)
            SliverFillRemaining(
              hasScrollBody: false,
              child: CraftskyEmptyState(
                icon: CraftskyIcons.people,
                title: switch (kind) {
                  FollowListKind.followers => l10n.settingsFollowers,
                  FollowListKind.following => l10n.settingsFollowing,
                },
                subtitle: switch (kind) {
                  FollowListKind.followers => 'No one follows you yet',
                  FollowListKind.following => 'You are not following anyone',
                },
              ),
            )
          else
            SliverList.builder(
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
                              child: Text(l10n.relationshipListLoadMore),
                            ),
                    ),
                  );
                }
                return ProfileAccountListTile(account: items[index]);
              },
            ),
        ],
      ),
    );
  }
}

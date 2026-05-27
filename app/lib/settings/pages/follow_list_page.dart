import 'package:craftsky_app/profile/models/profile_account_page.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
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
  late final Future<ProfileAccountPage> _pageFuture;

  @override
  void initState() {
    super.initState();
    final repo = ref.read(profileRepositoryProvider);
    _pageFuture = switch (widget.kind) {
      FollowListKind.followers => repo.listFollowersMe(),
      FollowListKind.following => repo.listFollowingMe(),
    };
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<ProfileAccountPage>(
      future: _pageFuture,
      builder: (context, snapshot) {
        final page = snapshot.data;
        final count = page?.totalCount ?? 0;
        return Scaffold(
          appBar: AppBar(title: Text('${_title} ($count)')),
          body: switch (snapshot.connectionState) {
            ConnectionState.done => _FollowListBody(
              kind: widget.kind,
              page: page,
            ),
            _ => const Center(child: StitchProgressIndicator()),
          },
        );
      },
    );
  }

  String get _title => switch (widget.kind) {
    FollowListKind.followers => 'Followers',
    FollowListKind.following => 'Following',
  };
}

class _FollowListBody extends StatelessWidget {
  const _FollowListBody({required this.kind, required this.page});

  final FollowListKind kind;
  final ProfileAccountPage? page;

  @override
  Widget build(BuildContext context) {
    final items = page?.items ?? [];
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
      itemCount: items.length,
      itemBuilder: (context, index) {
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

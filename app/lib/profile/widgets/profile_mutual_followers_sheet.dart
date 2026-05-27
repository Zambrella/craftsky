import 'package:craftsky_app/profile/models/profile_account_page.dart';
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
  late final Future<ProfileAccountPage> _pageFuture;

  @override
  void initState() {
    super.initState();
    _pageFuture = ref
        .read(profileRepositoryProvider)
        .listMutualFollowers(widget.targetHandleOrDid);
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
            child: FutureBuilder<ProfileAccountPage>(
              future: _pageFuture,
              builder: (context, snapshot) {
                if (snapshot.connectionState != ConnectionState.done) {
                  return const Center(child: StitchProgressIndicator());
                }
                final page = snapshot.data;
                if (page == null || page.items.isEmpty) {
                  return const SizedBox.shrink();
                }
                return ListView.builder(
                  itemCount: page.items.length,
                  itemBuilder: (context, index) {
                    final account = page.items[index];
                    final title = account.displayName?.isNotEmpty ?? false
                        ? account.displayName!
                        : account.handle.toString();
                    return ListTile(
                      title: Text(title),
                      subtitle: Text('@${account.handle}'),
                    );
                  },
                );
              },
            ),
          ),
        ],
      ),
    );
  }
}

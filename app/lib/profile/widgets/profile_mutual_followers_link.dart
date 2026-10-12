import 'dart:async';

import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:material_ui/material_ui.dart';

class ProfileMutualFollowersLink extends StatelessWidget {
  const new({
    required this.count,
    required this.targetDid,
    super.key,
  });

  final int count;
  final Did targetDid;

  @override
  Widget build(BuildContext context) {
    if (count <= 0) {
      return const SizedBox.shrink();
    }
    return TextButton(
      onPressed: () {
        unawaited(
          ProfileMutualFollowersRoute(did: targetDid).push<void>(context),
        );
      },
      child: Text('$count mutual followers'),
    );
  }
}

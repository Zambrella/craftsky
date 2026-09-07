import 'dart:async';

import 'package:craftsky_app/profile/widgets/profile_mutual_followers_sheet.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter/material.dart';

class ProfileMutualFollowersLink extends StatelessWidget {
  const ProfileMutualFollowersLink({
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
          showModalBottomSheet<void>(
            context: context,
            isScrollControlled: true,
            builder: (context) => FractionallySizedBox(
              heightFactor: 0.9,
              child: ProfileMutualFollowersSheet(
                targetDid: targetDid,
              ),
            ),
          ),
        );
      },
      child: Text('$count mutual followers'),
    );
  }
}

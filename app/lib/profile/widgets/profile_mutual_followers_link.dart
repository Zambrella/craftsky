import 'package:craftsky_app/profile/widgets/profile_mutual_followers_sheet.dart';
import 'package:flutter/material.dart';

class ProfileMutualFollowersLink extends StatelessWidget {
  const ProfileMutualFollowersLink({
    required this.count,
    required this.targetHandleOrDid,
    super.key,
  });

  final int count;
  final String targetHandleOrDid;

  @override
  Widget build(BuildContext context) {
    if (count <= 0) {
      return const SizedBox.shrink();
    }
    return TextButton(
      onPressed: () {
        showModalBottomSheet<void>(
          context: context,
          isScrollControlled: true,
          builder: (context) => FractionallySizedBox(
            heightFactor: 0.9,
            child: ProfileMutualFollowersSheet(
              targetHandleOrDid: targetHandleOrDid,
            ),
          ),
        );
      },
      child: Text('$count mutual followers'),
    );
  }
}

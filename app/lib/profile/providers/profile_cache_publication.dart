import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

void publishProfileCache(Ref ref, Profile profile) {
  for (final id in <String>{profile.handle, profile.did}) {
    if (ref.exists(userProfileProvider(id))) {
      ref.read(userProfileProvider(id).notifier).setCached(profile);
    }
  }
}

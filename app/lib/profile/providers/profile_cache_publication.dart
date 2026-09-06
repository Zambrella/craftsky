import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

void publishProfileCache(Ref ref, Profile profile) {
  final provider = userProfileProvider(profile.did);
  if (ref.exists(provider)) {
    ref.read(provider.notifier).setCached(profile);
  }
}

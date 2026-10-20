import 'package:patrol/patrol.dart';

import 'auth.dart';
import 'onboarding.dart';
import 'posts.dart';

final class Modules {
  new(PatrolIntegrationTester $)
    : auth = AuthModule($),
      onboarding = OnboardingModule($),
      posts = PostsModule($);

  final AuthModule auth;
  final OnboardingModule onboarding;
  final PostsModule posts;
}

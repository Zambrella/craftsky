import 'package:craftsky_app/onboarding/models/onboarding_completion.dart';

abstract interface class OnboardingRepository {
  Future<OnboardingCompletion> readStatus();
  Future<OnboardingCompletion> complete();
}

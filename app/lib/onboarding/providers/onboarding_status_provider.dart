import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'onboarding_status_provider.g.dart';

/// Stubbed onboarding completion status. Real implementation will be backed
/// by the user's profile record once onboarding actually persists data.
@riverpod
class OnboardingStatus extends _$OnboardingStatus {
  @override
  bool build() => false;

  void finish() => state = true;
}

import 'package:craftsky_app/onboarding/models/onboarding_flow_state.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('steps expose stable one-based progress', () {
    expect(OnboardingStep.profile.number, 1);
    expect(OnboardingStep.profile.progress, closeTo(1 / 3, 0.001));
    expect(OnboardingStep.crafts.number, 2);
    expect(OnboardingStep.crafts.progress, closeTo(2 / 3, 0.001));
    expect(OnboardingStep.instagram.number, 3);
    expect(OnboardingStep.instagram.progress, 1);
  });
}

import 'package:craftsky_app/onboarding/models/onboarding_action_state.dart';
import 'package:craftsky_app/onboarding/models/onboarding_flow_state.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('derives Next, Save & next, and Finish with navigation gates', () {
    expect(
      deriveOnboardingActionState(
        step: OnboardingStep.profile,
        dirty: false,
        valid: true,
      ).kind,
      OnboardingActionKind.next,
    );
    expect(
      deriveOnboardingActionState(
        step: OnboardingStep.crafts,
        dirty: true,
        valid: true,
      ).kind,
      OnboardingActionKind.saveAndNext,
    );
    final invalid = deriveOnboardingActionState(
      step: OnboardingStep.profile,
      dirty: true,
      valid: false,
    );
    expect(invalid.canSubmit, isFalse);
    final saving = deriveOnboardingActionState(
      step: OnboardingStep.crafts,
      dirty: true,
      valid: true,
      saving: true,
    );
    expect(saving.canSubmit, isFalse);
    expect(saving.canSkip, isFalse);
    expect(saving.canGoBack, isFalse);
    expect(
      deriveOnboardingActionState(
        step: OnboardingStep.instagram,
        dirty: false,
        valid: true,
      ).kind,
      OnboardingActionKind.finish,
    );
  });
}

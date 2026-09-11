import 'package:craftsky_app/onboarding/models/onboarding_flow_state.dart';

enum OnboardingActionKind { next, saveAndNext, finish }

final class OnboardingActionState {
  const OnboardingActionState({
    required this.kind,
    required this.canSubmit,
    required this.canGoBack,
    required this.busy,
  });

  final OnboardingActionKind kind;
  final bool canSubmit;
  final bool canGoBack;
  final bool busy;
}

OnboardingActionState deriveOnboardingActionState({
  required OnboardingStep step,
  required bool dirty,
  required bool valid,
  bool saving = false,
}) {
  final kind = switch (step) {
    OnboardingStep.guidelines => OnboardingActionKind.finish,
    _ when dirty => OnboardingActionKind.saveAndNext,
    _ => OnboardingActionKind.next,
  };
  return OnboardingActionState(
    kind: kind,
    canSubmit: !saving && (kind != OnboardingActionKind.saveAndNext || valid),
    canGoBack: step != OnboardingStep.profile && !saving,
    busy: saving,
  );
}

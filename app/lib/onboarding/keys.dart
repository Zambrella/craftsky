import 'package:craftsky_app/onboarding/models/onboarding_action_state.dart';
import 'package:flutter/widgets.dart';

class OnboardingKeys {
  final bio = const ValueKey<String>('onboarding-bio');
  final minimumAge = const ValueKey<String>(
    'onboarding-minimum-age-declaration',
  );
  ValueKey<String> next(OnboardingActionKind kind) =>
      ValueKey('onboarding-next-${kind.name}');
}

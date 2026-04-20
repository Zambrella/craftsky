import 'package:flutter/foundation.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'onboarding_status_provider.g.dart';

/// Stubbed onboarding completion status. Real implementation will be backed
/// by the user's profile record once onboarding actually persists data.
@riverpod
class OnboardingStatus extends _$OnboardingStatus {
  // Flip the first operand to `true` locally to skip onboarding during manual
  // dev runs. kReleaseMode always defaults to `false`.
  @override
  bool build() =>
      // Intentional same-literal ternary — lets the dev flip the first
      // operand without touching the second. Disabled lint would hide the
      // toggle surface.
      // ignore: avoid_bool_literals_in_conditional_expressions
      kDebugMode ? false : false;

  void finish() => state = true;
}

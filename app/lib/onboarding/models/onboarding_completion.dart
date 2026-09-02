import 'package:flutter/foundation.dart';

@immutable
final class OnboardingCompletion {
  const OnboardingCompletion({required this.completed, this.completedAt});

  factory OnboardingCompletion.fromJson(Map<String, dynamic> json) {
    final completed = json['completed'];
    if (completed is! bool) {
      throw const FormatException('Invalid onboarding completion');
    }
    final completedAt = json['completedAt'];
    return OnboardingCompletion(
      completed: completed,
      completedAt: completedAt is String ? DateTime.parse(completedAt) : null,
    );
  }

  final bool completed;
  final DateTime? completedAt;

  @override
  bool operator ==(Object other) =>
      other is OnboardingCompletion &&
      other.completed == completed &&
      other.completedAt == completedAt;

  @override
  int get hashCode => Object.hash(completed, completedAt);
}

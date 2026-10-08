import 'package:flutter/foundation.dart';

@immutable
final class OnboardingCompletion {
  const OnboardingCompletion({
    required this.completed,
    this.requiredPolicyVersion = '1',
    this.completedAt,
    this.acceptedPolicyVersion,
    this.acceptedAt,
  });

  factory OnboardingCompletion.fromJson(Map<String, dynamic> json) {
    final completed = json['completed'];
    if (completed is! bool) {
      throw const FormatException('Invalid onboarding completion');
    }
    final completedAt = json['completedAt'];
    final requiredPolicyVersion = json['requiredPolicyVersion'];
    if (requiredPolicyVersion is! String || requiredPolicyVersion.isEmpty) {
      throw const FormatException('Invalid onboarding policy version');
    }
    final acceptedAt = json['acceptedAt'];
    return OnboardingCompletion(
      completed: completed,
      requiredPolicyVersion: requiredPolicyVersion,
      completedAt: completedAt is String ? DateTime.parse(completedAt) : null,
      acceptedPolicyVersion: json['acceptedPolicyVersion'] as String?,
      acceptedAt: acceptedAt is String ? DateTime.parse(acceptedAt) : null,
    );
  }

  final bool completed;
  final DateTime? completedAt;
  final String requiredPolicyVersion;
  final String? acceptedPolicyVersion;
  final DateTime? acceptedAt;

  @override
  bool operator ==(Object other) =>
      other is OnboardingCompletion &&
      other.completed == completed &&
      other.completedAt == completedAt &&
      other.requiredPolicyVersion == requiredPolicyVersion &&
      other.acceptedPolicyVersion == acceptedPolicyVersion &&
      other.acceptedAt == acceptedAt;

  @override
  int get hashCode => Object.hash(
    completed,
    completedAt,
    requiredPolicyVersion,
    acceptedPolicyVersion,
    acceptedAt,
  );
}

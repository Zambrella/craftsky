enum AccountEligibilityState { eligible, restricted }

final class AccountEligibilityStatus {
  const AccountEligibilityStatus({
    required this.state,
    required this.appealable,
    this.appealGuidance,
  });

  factory AccountEligibilityStatus.fromJson(Map<String, dynamic> json) {
    final state = switch (json['state']) {
      'eligible' => AccountEligibilityState.eligible,
      'restricted' => AccountEligibilityState.restricted,
      _ => throw const FormatException('Invalid account eligibility state'),
    };
    final appealable = json['appealable'];
    if (appealable is! bool) {
      throw const FormatException('Invalid account eligibility status');
    }
    return AccountEligibilityStatus(
      state: state,
      appealable: appealable,
      appealGuidance: json['appealGuidance'] as String?,
    );
  }

  final AccountEligibilityState state;
  final bool appealable;
  final String? appealGuidance;

  bool get restricted => state == AccountEligibilityState.restricted;
}

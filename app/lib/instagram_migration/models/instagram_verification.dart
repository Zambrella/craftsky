enum InstagramVerificationState {
  pendingDm,
  processing,
  pendingConfirmation,
  confirmed,
  expired,
  cancelled,
  superseded,
  rejected,
  conflicted,
  unknown;

  static InstagramVerificationState fromWire(String value) => switch (value) {
    'pendingDm' => pendingDm,
    'processing' => processing,
    'pendingConfirmation' => pendingConfirmation,
    'confirmed' => confirmed,
    'expired' => expired,
    'cancelled' => cancelled,
    'superseded' => superseded,
    'rejected' => rejected,
    'conflicted' => conflicted,
    _ => unknown,
  };
}

enum InstagramVerificationRetryCode {
  profileLookupUnavailable,
  invalidProfileResponse,
  membershipInactive,
  unknown;

  static InstagramVerificationRetryCode fromWire(String value) =>
      switch (value) {
        'profileLookupUnavailable' => profileLookupUnavailable,
        'invalidProfileResponse' => invalidProfileResponse,
        'membershipInactive' => membershipInactive,
        _ => unknown,
      };
}

final class InstagramVerificationAttempt {
  const InstagramVerificationAttempt({
    required this.verificationId,
    required this.state,
    required this.expiresAt,
    this.challenge,
    this.dmUrl,
    this.candidateUsername,
    this.retryCode,
  });

  factory InstagramVerificationAttempt.fromMap(Map<String, dynamic> map) {
    final attempt = InstagramVerificationAttempt._decode(map);
    final candidateRequired =
        attempt.state == InstagramVerificationState.pendingConfirmation;
    final retryRequired = attempt.state == InstagramVerificationState.rejected;
    if (attempt.challenge != null ||
        attempt.dmUrl != null ||
        (attempt.candidateUsername != null) != candidateRequired ||
        (attempt.retryCode != null) != retryRequired) {
      throw const FormatException('invalid_instagram_verification_status');
    }
    return attempt;
  }

  factory InstagramVerificationAttempt.fromCreationMap(
    Map<String, dynamic> map,
  ) {
    final attempt = InstagramVerificationAttempt._decode(map);
    if (attempt.state != InstagramVerificationState.pendingDm ||
        attempt.challenge == null ||
        attempt.dmUrl == null ||
        attempt.dmUrl!.scheme != 'https' ||
        !attempt.dmUrl!.hasAuthority ||
        attempt.candidateUsername != null ||
        attempt.retryCode != null) {
      throw const FormatException('invalid_instagram_verification_creation');
    }
    return attempt;
  }

  factory InstagramVerificationAttempt._decode(Map<String, dynamic> map) {
    final verificationId = map['verificationId'];
    final state = map['state'];
    final expiresAt = map['expiresAt'];
    final challenge = map['challenge'];
    final dmUrl = map['dmUrl'];
    final candidateUsername = map['candidateUsername'];
    final retryCode = map['retryCode'];
    if (verificationId is! String ||
        state is! String ||
        expiresAt is! String ||
        challenge is! String? ||
        dmUrl is! String? ||
        candidateUsername is! String? ||
        retryCode is! String?) {
      throw const FormatException('invalid_instagram_verification');
    }
    final DateTime parsedExpiresAt;
    final Uri? parsedDmUrl;
    try {
      parsedExpiresAt = DateTime.parse(expiresAt).toUtc();
      parsedDmUrl = dmUrl == null ? null : Uri.parse(dmUrl);
    } on FormatException {
      throw const FormatException('invalid_instagram_verification');
    }
    return InstagramVerificationAttempt(
      verificationId: verificationId,
      state: InstagramVerificationState.fromWire(state),
      expiresAt: parsedExpiresAt,
      challenge: challenge,
      dmUrl: parsedDmUrl,
      candidateUsername: candidateUsername,
      retryCode: retryCode == null
          ? null
          : InstagramVerificationRetryCode.fromWire(retryCode),
    );
  }

  final String verificationId;
  final InstagramVerificationState state;
  final DateTime expiresAt;
  final String? challenge;
  final Uri? dmUrl;
  final String? candidateUsername;
  final InstagramVerificationRetryCode? retryCode;

  @override
  String toString() => 'InstagramVerificationAttempt([REDACTED])';
}

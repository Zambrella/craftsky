import 'package:craftsky_app/instagram_migration/models/instagram_verification.dart';

enum InstagramAccountLinkState {
  active,
  membershipInactive,
  revoked,
  superseded,
  disputed,
  unknown;

  static InstagramAccountLinkState fromWire(String value) => switch (value) {
    'active' => active,
    'membershipInactive' => membershipInactive,
    'revoked' => revoked,
    'superseded' => superseded,
    'disputed' => disputed,
    _ => unknown,
  };
}

final class InstagramAccountLink {
  const InstagramAccountLink({
    required this.state,
    required this.username,
    required this.discoverable,
    required this.conflictPending,
    required this.reactivationRequired,
    required this.verifiedAt,
  });

  factory InstagramAccountLink.fromMap(Map<String, dynamic> map) {
    final state = map['state'];
    final username = map['username'];
    final discoverable = map['discoverable'];
    final conflictPending = map['conflictPending'];
    final reactivationRequired = map['reactivationRequired'];
    final verifiedAt = map['verifiedAt'];
    if (state is! String ||
        username is! String ||
        discoverable is! bool ||
        conflictPending is! bool ||
        reactivationRequired is! bool ||
        verifiedAt is! String) {
      throw const FormatException('invalid_instagram_account');
    }
    return InstagramAccountLink(
      state: InstagramAccountLinkState.fromWire(state),
      username: username,
      discoverable: discoverable,
      conflictPending: conflictPending,
      reactivationRequired: reactivationRequired,
      verifiedAt: DateTime.parse(verifiedAt).toUtc(),
    );
  }

  final InstagramAccountLinkState state;
  final String username;
  final bool discoverable;
  final bool conflictPending;
  final bool reactivationRequired;
  final DateTime verifiedAt;

  @override
  String toString() => 'InstagramAccountLink([REDACTED])';
}

final class InstagramAccountStatus {
  const InstagramAccountStatus({
    required this.integrationAvailable,
    required this.account,
  });

  factory InstagramAccountStatus.fromMap(Map<String, dynamic> map) {
    final integrationAvailable = map['integrationAvailable'];
    final account = map['account'];
    if (integrationAvailable is! bool || account is! Map<String, dynamic>?) {
      throw const FormatException('invalid_instagram_account_status');
    }
    return InstagramAccountStatus(
      integrationAvailable: integrationAvailable,
      account: account == null ? null : InstagramAccountLink.fromMap(account),
    );
  }

  final bool integrationAvailable;
  final InstagramAccountLink? account;

  @override
  String toString() => 'InstagramAccountStatus([REDACTED])';
}

final class InstagramVerificationConfirmation {
  const InstagramVerificationConfirmation({
    required this.state,
    required this.account,
  });

  factory InstagramVerificationConfirmation.fromMap(
    Map<String, dynamic> map,
  ) {
    final state = map['state'];
    final account = map['account'];
    if (state is! String || account is! Map<String, dynamic>) {
      throw const FormatException('invalid_instagram_confirmation');
    }
    return InstagramVerificationConfirmation(
      state: InstagramVerificationState.fromWire(state),
      account: InstagramAccountLink.fromMap(account),
    );
  }

  final InstagramVerificationState state;
  final InstagramAccountLink account;

  @override
  String toString() => 'InstagramVerificationConfirmation([REDACTED])';
}

final class InstagramAccountSettingsPatch {
  const InstagramAccountSettingsPatch({this.discoverable, this.reactivate})
    : assert(
        discoverable != null || reactivate != null,
        'At least one account setting must be supplied.',
      );

  final bool? discoverable;
  final bool? reactivate;

  Map<String, Object?> toMap() => {
    if (discoverable != null) 'discoverable': discoverable,
    if (reactivate != null) 'reactivate': reactivate,
  };

  @override
  String toString() => 'InstagramAccountSettingsPatch([REDACTED])';
}

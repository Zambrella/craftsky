import 'package:craftsky_app/instagram_migration/models/instagram_verification.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'instagram_account.mapper.dart';

const int _instagramAccountValueMethods =
    GenerateMethods.copy | GenerateMethods.equals;
const int _instagramAccountDecodeMethods =
    GenerateMethods.decode | _instagramAccountValueMethods;

@MappableEnum(defaultValue: InstagramAccountLinkState.unknown)
enum InstagramAccountLinkState {
  active,
  membershipInactive,
  revoked,
  superseded,
  disputed,
  unknown;

  static InstagramAccountLinkState fromWire(String value) =>
      InstagramAccountLinkStateMapper.fromValue(value);
}

@MappableClass(generateMethods: _instagramAccountDecodeMethods)
final class InstagramAccountLink with InstagramAccountLinkMappable {
  const InstagramAccountLink({
    required this.state,
    required this.username,
    required this.discoverable,
    required this.conflictPending,
    required this.reactivationRequired,
    required this.verifiedAt,
  });

  factory InstagramAccountLink.fromMap(Map<String, dynamic> map) =>
      InstagramAccountLinkMapper.fromMap(map);

  final InstagramAccountLinkState state;
  final String username;
  final bool discoverable;
  final bool conflictPending;
  final bool reactivationRequired;
  final DateTime verifiedAt;

  @override
  String toString() => 'InstagramAccountLink([REDACTED])';
}

@MappableClass(generateMethods: _instagramAccountDecodeMethods)
final class InstagramAccountStatus with InstagramAccountStatusMappable {
  const InstagramAccountStatus({
    required this.integrationAvailable,
    required this.account,
  });

  factory InstagramAccountStatus.fromMap(Map<String, dynamic> map) =>
      InstagramAccountStatusMapper.fromMap(map);

  final bool integrationAvailable;
  final InstagramAccountLink? account;

  @override
  String toString() => 'InstagramAccountStatus([REDACTED])';
}

@MappableClass(generateMethods: _instagramAccountDecodeMethods)
final class InstagramVerificationConfirmation
    with InstagramVerificationConfirmationMappable {
  const InstagramVerificationConfirmation({
    required this.state,
    required this.account,
  });

  factory InstagramVerificationConfirmation.fromMap(
    Map<String, dynamic> map,
  ) => InstagramVerificationConfirmationMapper.fromMap(map);

  final InstagramVerificationState state;
  final InstagramAccountLink account;

  @override
  String toString() => 'InstagramVerificationConfirmation([REDACTED])';
}

@MappableClass(
  ignoreNull: true,
  generateMethods:
      GenerateMethods.encode | GenerateMethods.copy | GenerateMethods.equals,
)
final class InstagramAccountSettingsPatch
    with InstagramAccountSettingsPatchMappable {
  const InstagramAccountSettingsPatch({this.discoverable, this.reactivate})
    : assert(
        discoverable != null || reactivate != null,
        'At least one account setting must be supplied.',
      );

  final bool? discoverable;
  final bool? reactivate;

  @override
  String toString() => 'InstagramAccountSettingsPatch([REDACTED])';
}

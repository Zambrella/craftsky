import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dart_mappable/dart_mappable.dart';

part 'pending_auth.mapper.dart';

@MappableEnum()
enum PendingAuthPurpose { signIn, registration }

/// Records that an authentication flow is in progress.
///
/// `startedAt` is a UI/diagnostic hint only. The server clock is authoritative
/// for exchange-code expiry.
@MappableClass(includeCustomMappers: [HandleMapper()])
class PendingAuth with PendingAuthMappable {
  PendingAuth({
    required this.purpose,
    required String? handle,
    required this.startedAt,
  }) : handle = _parseHandle(purpose, handle);

  factory PendingAuth.signIn({
    required String handle,
    required DateTime startedAt,
  }) => PendingAuth(
    purpose: PendingAuthPurpose.signIn,
    handle: handle,
    startedAt: startedAt,
  );

  factory PendingAuth.registration({required DateTime startedAt}) =>
      PendingAuth(
        purpose: PendingAuthPurpose.registration,
        handle: null,
        startedAt: startedAt,
      );

  final PendingAuthPurpose purpose;
  final Handle? handle;
  final DateTime startedAt;

  static Handle? _parseHandle(PendingAuthPurpose purpose, String? handle) =>
      switch ((purpose, handle)) {
        (PendingAuthPurpose.signIn, final String value) => Handle.parse(value),
        (PendingAuthPurpose.registration, null) => null,
        (PendingAuthPurpose.signIn, null) => throw ArgumentError.notNull(
          'handle',
        ),
        (PendingAuthPurpose.registration, String()) =>
          throw ArgumentError.value(
            handle,
            'handle',
            'Registration must not have a handle',
          ),
      };
}

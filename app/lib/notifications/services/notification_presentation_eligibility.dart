import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/notifications/models/account_subscription_id.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

// A named capability keeps presenters independent of secure-storage details.
// ignore: one_member_abstracts
abstract interface class NotificationPresentationEligibility {
  Future<bool> allows(AccountSubscriptionId accountSubscriptionId);
}

/// Revalidates provider routing against the latest secure account snapshot.
/// A delayed notification for a signed-out/removed account therefore cannot
/// display or navigate merely because it still has a syntactically valid
/// provider payload.
final class SessionRegistryNotificationPresentationEligibility
    implements NotificationPresentationEligibility {
  const SessionRegistryNotificationPresentationEligibility(this._storage);

  final SessionRegistryStorage _storage;

  @override
  Future<bool> allows(AccountSubscriptionId accountSubscriptionId) async {
    try {
      final registry = await _storage.read();
      return registry.routingBindings.entries.any(
        (entry) =>
            entry.value == accountSubscriptionId.wireValue &&
            registry.sessions.containsKey(entry.key),
      );
    } on Object {
      return false;
    }
  }
}

NotificationPresentationEligibility
createDefaultNotificationPresentationEligibility() =>
    SessionRegistryNotificationPresentationEligibility(
      SecureSessionRegistryStorage(const FlutterSecureStorage()),
    );

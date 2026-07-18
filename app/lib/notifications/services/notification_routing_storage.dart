import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/auth/models/stored_session.dart';
import 'package:craftsky_app/notifications/models/account_subscription_id.dart';

sealed class NotificationRecipientResolution {
  const NotificationRecipientResolution();

  @override
  String toString() => switch (this) {
    ExactNotificationRecipient() => 'ExactNotificationRecipient(<redacted>)',
    InvalidNotificationRecipient() => 'InvalidNotificationRecipient()',
    RemovedNotificationRecipient() => 'RemovedNotificationRecipient()',
  };
}

final class ExactNotificationRecipient extends NotificationRecipientResolution {
  const ExactNotificationRecipient(this.lease);

  final AccountSessionLease lease;
}

final class InvalidNotificationRecipient
    extends NotificationRecipientResolution {
  const InvalidNotificationRecipient();
}

final class RemovedNotificationRecipient
    extends NotificationRecipientResolution {
  const RemovedNotificationRecipient();
}

/// Registry-backed adapter for the secure DID-to-routing-ID binding map.
///
/// Resolution returns a session-generation lease rather than a DID so an open
/// cannot cross account removal or reauthentication while it is in flight.
final class NotificationRoutingStorage {
  const NotificationRoutingStorage(this._readRegistry);

  final SessionRegistry Function() _readRegistry;

  bool isCurrentLease(AccountSessionLease lease) =>
      _readRegistry().leaseFor(lease.account) == lease;

  bool isActiveLease(AccountSessionLease lease) =>
      _readRegistry().activeLease?.session == lease;

  StoredSession? sessionFor(AccountSessionLease lease) {
    final registry = _readRegistry();
    return registry.leaseFor(lease.account) == lease
        ? registry.sessions[lease.account.did]
        : null;
  }

  AccountSubscriptionId? read(AccountSessionLease lease) {
    final registry = _readRegistry();
    if (registry.leaseFor(lease.account) != lease) return null;
    final value = registry.routingBindings[lease.account.did];
    if (value == null) return null;
    try {
      return AccountSubscriptionId.parse(value);
    } on FormatException {
      return null;
    }
  }

  NotificationRecipientResolution resolve(AccountSubscriptionId? binding) {
    if (binding == null) return const InvalidNotificationRecipient();
    final registry = _readRegistry();
    final matchingDids = <String>[];
    var hasMalformedBinding = false;

    for (final entry in registry.routingBindings.entries) {
      try {
        final candidate = AccountSubscriptionId.parse(entry.value);
        if (candidate == binding) matchingDids.add(entry.key.value);
      } on FormatException {
        hasMalformedBinding = true;
      }
    }

    if (hasMalformedBinding || matchingDids.length > 1) {
      return const InvalidNotificationRecipient();
    }
    if (matchingDids.isEmpty) return const RemovedNotificationRecipient();

    final lease = registry.leaseFor(AccountKey(matchingDids.single));
    return lease == null
        ? const RemovedNotificationRecipient()
        : ExactNotificationRecipient(lease);
  }
}

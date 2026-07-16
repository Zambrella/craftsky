import 'dart:convert';

import 'package:craftsky_app/notifications/models/notification_open_event.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

abstract interface class NotificationRoutingStorageBackend {
  Future<String?> read();
  Future<void> write(String value);
  Future<void> delete();
}

final class FlutterSecureNotificationRoutingStorageBackend
    implements NotificationRoutingStorageBackend {
  const FlutterSecureNotificationRoutingStorageBackend(this._storage);

  static const _key = 'craftsky_notification_routing_bindings';

  final FlutterSecureStorage _storage;

  @override
  Future<void> delete() => _storage.delete(key: _key);

  @override
  Future<String?> read() => _storage.read(key: _key);

  @override
  Future<void> write(String value) => _storage.write(key: _key, value: value);
}

final class NotificationRoutingStorage {
  const NotificationRoutingStorage(this._backend);

  final NotificationRoutingStorageBackend _backend;

  Future<AccountSubscriptionId?> read(Did did) async {
    final bindings = await _readBindings();
    final value = bindings[did];
    if (value == null) return null;
    try {
      return AccountSubscriptionId.parse(value);
    } on FormatException {
      await _backend.delete();
      return null;
    }
  }

  Future<void> replace(Did did, AccountSubscriptionId binding) async {
    final bindings = await _readBindings();
    bindings[did] = binding.wireValue;
    await _backend.write(jsonEncode(bindings));
  }

  Future<void> remove(Did did) async {
    final bindings = await _readBindings();
    if (bindings.remove(did) == null) return;
    if (bindings.isEmpty) {
      await _backend.delete();
      return;
    }
    await _backend.write(jsonEncode(bindings));
  }

  Future<Map<String, String>> _readBindings() async {
    final raw = await _backend.read();
    if (raw == null) return <String, String>{};
    try {
      final decoded = jsonDecode(raw);
      if (decoded is! Map<String, dynamic>) {
        throw const FormatException('Invalid routing bindings');
      }
      return decoded.map((key, value) {
        if (value is! String) {
          throw const FormatException('Invalid routing binding');
        }
        return MapEntry(key, value);
      });
    } on Object {
      await _backend.delete();
      return <String, String>{};
    }
  }
}

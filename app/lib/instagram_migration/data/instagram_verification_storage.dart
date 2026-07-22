import 'dart:convert';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

@immutable
final class InstagramVerificationSnapshot {
  const InstagramVerificationSnapshot({
    required this.verificationId,
    required this.challenge,
    required this.dmUrl,
    required this.expiresAt,
  });

  factory InstagramVerificationSnapshot.fromMap(Map<String, dynamic> map) {
    final verificationId = map['verificationId'];
    final challenge = map['challenge'];
    final rawDmUrl = map['dmUrl'];
    final rawExpiresAt = map['expiresAt'];
    if (verificationId is! String ||
        verificationId.isEmpty ||
        challenge is! String ||
        challenge.isEmpty ||
        rawDmUrl is! String ||
        rawExpiresAt is! String) {
      throw const FormatException('invalid_instagram_verification_snapshot');
    }
    final dmUrl = Uri.parse(rawDmUrl);
    final expiresAt = DateTime.parse(rawExpiresAt).toUtc();
    if (dmUrl.scheme != 'https' || !dmUrl.hasAuthority) {
      throw const FormatException('invalid_instagram_verification_snapshot');
    }
    return InstagramVerificationSnapshot(
      verificationId: verificationId,
      challenge: challenge,
      dmUrl: dmUrl,
      expiresAt: expiresAt,
    );
  }

  final String verificationId;
  final String challenge;
  final Uri dmUrl;
  final DateTime expiresAt;

  Map<String, Object> toMap() => {
    'verificationId': verificationId,
    'challenge': challenge,
    'dmUrl': dmUrl.toString(),
    'expiresAt': expiresAt.toUtc().toIso8601String(),
  };

  @override
  String toString() => 'InstagramVerificationSnapshot([REDACTED])';
}

abstract interface class InstagramVerificationStorageBackend {
  Future<String?> read(String key);

  Future<void> write(String key, String value);

  Future<void> delete(String key);
}

abstract interface class InstagramVerificationStorage {
  Future<InstagramVerificationSnapshot?> read(AccountKey account);

  Future<void> write(
    AccountKey account,
    InstagramVerificationSnapshot snapshot,
  );

  Future<void> delete(AccountKey account, {String? verificationId});
}

final class _FlutterInstagramVerificationStorageBackend
    implements InstagramVerificationStorageBackend {
  const _FlutterInstagramVerificationStorageBackend(this._storage);

  final FlutterSecureStorage _storage;

  @override
  Future<void> delete(String key) => _storage.delete(key: key);

  @override
  Future<String?> read(String key) => _storage.read(key: key);

  @override
  Future<void> write(String key, String value) =>
      _storage.write(key: key, value: value);
}

final class SecureInstagramVerificationStorage
    implements InstagramVerificationStorage {
  SecureInstagramVerificationStorage(FlutterSecureStorage storage)
    : _backend = _FlutterInstagramVerificationStorageBackend(storage);

  SecureInstagramVerificationStorage.withBackend(this._backend);

  static const _keyPrefix = 'craftsky_instagram_verification_v1_';

  final InstagramVerificationStorageBackend _backend;

  @override
  Future<InstagramVerificationSnapshot?> read(AccountKey account) async {
    final key = _key(account);
    final String? source;
    try {
      source = await _backend.read(key);
    } on Object {
      return null;
    }
    if (source == null) return null;
    try {
      final decoded = jsonDecode(source);
      if (decoded is! Map<String, dynamic>) {
        throw const FormatException('invalid_instagram_verification_snapshot');
      }
      return InstagramVerificationSnapshot.fromMap(decoded);
    } on Object {
      try {
        await _backend.delete(key);
      } on Object {
        // A malformed snapshot still fails closed if secure storage is down.
      }
      return null;
    }
  }

  @override
  Future<void> write(
    AccountKey account,
    InstagramVerificationSnapshot snapshot,
  ) => _backend.write(_key(account), jsonEncode(snapshot.toMap()));

  @override
  Future<void> delete(AccountKey account, {String? verificationId}) async {
    final key = _key(account);
    if (verificationId == null) {
      await _backend.delete(key);
      return;
    }
    final current = await read(account);
    if (current?.verificationId == verificationId) {
      await _backend.delete(key);
    }
  }

  String _key(AccountKey account) {
    final encoded = base64Url
        .encode(utf8.encode(account.did.value))
        .replaceAll('=', '');
    return '$_keyPrefix$encoded';
  }
}

final instagramVerificationStorageProvider =
    Provider<InstagramVerificationStorage>(
      (_) => SecureInstagramVerificationStorage(const FlutterSecureStorage()),
    );

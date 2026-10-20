import 'dart:async';
import 'dart:convert';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/chat/models/chat_draft.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:crypto/crypto.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

final class ChatDraftLeaseExpired implements Exception {
  const new();
}

final class ChatDraftStorageException
    implements Exception, DiagnosticFailureCause {
  const new(this.diagnosticCause, this.diagnosticStack);
  @override
  final Object diagnosticCause;
  @override
  final StackTrace diagnosticStack;
  @override
  String toString() => 'Local messaging draft storage unavailable';
}

final class ChatDraftLease {
  const new _(this._store, this._owner, this._generation);
  final Object _store;
  final String _owner;
  final int _generation;
  @override
  String toString() => 'ChatDraftLease(<redacted>)';
}

abstract interface class ChatDraftStorageBackend {
  Future<String?> read(String key);
  Future<void> write(String key, String value);
  Future<void> delete(String key);
}

final class _SecureDraftBackend implements ChatDraftStorageBackend {
  const new(this.storage);
  final FlutterSecureStorage storage;
  @override
  Future<String?> read(String key) => storage.read(key: key);
  @override
  Future<void> write(String key, String value) =>
      storage.write(key: key, value: value);
  @override
  Future<void> delete(String key) => storage.delete(key: key);
}

/// Serialized local storage. No message transport or server draft dependency.
final class ChatDraftStore {
  new(FlutterSecureStorage storage) : _backend = _SecureDraftBackend(storage);
  new withBackend(this._backend);

  static const localStorage = FlutterSecureStorage(
    iOptions: IOSOptions(
      // Keep the privacy boundary explicit when plugin defaults change.
      // ignore: avoid_redundant_argument_values
      synchronizable: false,
      accessibility: KeychainAccessibility.unlocked_this_device,
    ),
    aOptions: AndroidOptions(
      resetOnError: false,
      storageNamespace: 'craftsky_chat_drafts',
    ),
    webOptions: WebOptions(dbName: 'CraftSkyChatDrafts'),
  );
  static const _indexKey = 'craftsky.chat.drafts.v1.index';
  final ChatDraftStorageBackend _backend;
  final Object _identity = Object();
  final Map<String, int> _generations = {};
  final Set<String> _retained = {};
  Map<String, dynamic> _index = {};
  Future<void> _tail = Future.value();
  bool _ready = false;
  int _registryVersion = 0;

  String _hash(String value) => sha256.convert(utf8.encode(value)).toString();
  String _key(String owner, String conversation) =>
      'craftsky.chat.drafts.v1.$owner.$conversation';

  Future<T> _serialized<T>(Future<T> Function() action) {
    final result = _tail.then((_) async {
      try {
        return await action();
      } on ChatDraftLeaseExpired {
        rethrow;
      } on Object catch (error, stack) {
        throw ChatDraftStorageException(error, stack);
      }
    });
    _tail = result.then<void>((_) {}, onError: (Object _, StackTrace _) {});
    return result;
  }

  Future<void> _writeIndex() => _backend.write(_indexKey, jsonEncode(_index));

  Future<void> initialize(Set<AccountKey> accounts) {
    final version = ++_registryVersion;
    final retained = accounts
        .map((account) => _hash(account.did.value))
        .toSet();
    for (final removed in _retained.difference(retained)) {
      _retained.remove(removed);
      _generations[removed] = (_generations[removed] ?? 0) + 1;
    }
    return _serialized(() async {
      if (version != _registryVersion) return;
      final raw = _ready ? null : await _backend.read(_indexKey);
      if (version != _registryVersion) return;
      if (raw != null) {
        final decoded = jsonDecode(raw);
        if (decoded is! Map<String, dynamic>) {
          throw const FormatException('Invalid local draft index');
        }
        _index = decoded;
      }
      for (final owner in _index.keys.toList()) {
        final entry = _index[owner];
        if (!RegExp(r'^[a-f0-9]{64}$').hasMatch(owner) ||
            entry is! Map<String, dynamic> ||
            entry['generation'] is! int ||
            entry['conversations'] is! List ||
            entry['purging'] is! bool) {
          throw const FormatException('Invalid local draft index');
        }
        if (!retained.contains(owner) || entry['purging'] == true) {
          await _purge(owner);
        }
      }
      if (version != _registryVersion) return;
      _retained.addAll(retained);
      for (final owner in retained) {
        final entry = _index.putIfAbsent(
          owner,
          () => {
            'generation': _generations[owner] ?? 0,
            'purging': false,
            'conversations': <String>[],
          },
        ) as Map<String, dynamic>;
        _generations[owner] = entry['generation'] as int;
      }
      await _writeIndex();
      _ready = true;
    });
  }

  ChatDraftLease lease(AccountKey account) {
    final owner = _hash(account.did.value);
    if (!_ready || !_retained.contains(owner)) {
      throw const ChatDraftLeaseExpired();
    }
    return ChatDraftLease._(_identity, owner, _generations[owner]!);
  }

  void _check(ChatDraftLease lease) {
    if (!_ready ||
        !identical(lease._store, _identity) ||
        !_retained.contains(lease._owner) ||
        _generations[lease._owner] != lease._generation) {
      throw const ChatDraftLeaseExpired();
    }
  }

  Future<ChatDraft> load(
    ChatDraftLease lease,
    ChatConversationId conversation,
  ) => _serialized(() async {
    _check(lease);
    final raw = await _backend.read(
      _key(lease._owner, _hash(conversation.value)),
    );
    _check(lease);
    if (raw == null) return const ChatDraft.empty();
    final decoded = jsonDecode(raw);
    if (decoded is! Map<String, dynamic>) {
      throw const FormatException('Invalid local draft');
    }
    return ChatDraft.fromStorage(decoded);
  });

  Future<void> save(
    ChatDraftLease lease,
    ChatConversationId conversation,
    ChatDraft draft,
  ) => _serialized(() async {
    _check(lease);
    final hash = _hash(conversation.value);
    final entry = _index[lease._owner] as Map<String, dynamic>;
    final keys = List<String>.from(entry['conversations'] as List);
    final key = _key(lease._owner, hash);
    if (draft.text.isEmpty && draft.pending == null) {
      await _backend.delete(key);
      keys.remove(hash);
      entry['conversations'] = keys;
      await _writeIndex();
    } else {
      if (!keys.contains(hash)) keys.add(hash);
      entry['conversations'] = keys;
      // Index first: interrupted saves remain discoverable by startup purge.
      await _writeIndex();
      _check(lease);
      await _backend.write(key, jsonEncode(draft.toStorage()));
    }
    _check(lease);
  });

  Future<void> removeAccount(AccountKey account) {
    ++_registryVersion;
    final owner = _hash(account.did.value);
    // Invalidate immediately, including work already queued behind storage I/O.
    _retained.remove(owner);
    _generations[owner] = (_generations[owner] ?? 0) + 1;
    return _serialized(() async {
      if (!_ready) {
        final raw = await _backend.read(_indexKey);
        if (raw != null) {
          final decoded = jsonDecode(raw);
          if (decoded is! Map<String, dynamic>) {
            throw const FormatException('Invalid local draft index');
          }
          _index = decoded;
        }
      }
      await _purge(owner);
    });
  }

  Future<void> _purge(String owner) async {
    final entry = _index[owner];
    if (entry is! Map<String, dynamic>) return;
    entry['purging'] = true;
    await _writeIndex();
    for (final hash in List<String>.from(entry['conversations'] as List)) {
      if (!RegExp(r'^[a-f0-9]{64}$').hasMatch(hash)) {
        throw const FormatException('Invalid local draft key');
      }
      await _backend.delete(_key(owner, hash));
    }
    _index.remove(owner);
    await _writeIndex();
  }
}

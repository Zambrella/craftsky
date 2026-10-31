import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/chat/data/chat_draft_store.dart';
import 'package:craftsky_app/chat/models/chat_draft.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';

// IT-017 / FR-019 / AC-028. Real storage adapter, mocked platform channel only.
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  test('IT-017 a queued initialization cannot restore '
      'removed-owner draft authority', () async {
    final backend = _DelayedBackend();
    final store = ChatDraftStore.withBackend(backend);
    final a = AccountKey('did:plc:owner');
    final id = ChatConversationId.parse('conversation');
    await store.initialize({a});
    final old = store.lease(a);
    backend.holdDraftWrite = true;
    final save = store.save(
      old,
      id,
      const ChatDraft.empty().edit('private-old'),
    );
    final saved = expectLater(save, throwsA(isA<ChatDraftLeaseExpired>()));
    await backend.entered.future;
    final staleInitialize = store.initialize({a});
    final removal = store.removeAccount(a);
    backend.release.complete();
    await saved;
    await staleInitialize;
    await removal;
    expect(() => store.lease(a), throwsA(isA<ChatDraftLeaseExpired>()));
    await store.initialize({a});
    await store.save(store.lease(a), id, const ChatDraft.empty().edit('fresh'));
    await expectLater(
      store.save(old, id, const ChatDraft.empty().edit('late')),
      throwsA(isA<ChatDraftLeaseExpired>()),
    );
    expect((await store.load(store.lease(a), id)).text, 'fresh');
  });

  test('registry changes fence queued saves'
      ' and a re-added account gets fresh authority', () async {
    final backend = _DelayedBackend();
    final store = ChatDraftStore.withBackend(backend);
    final a = AccountKey('did:plc:draft-owner-a');
    final b = AccountKey('did:plc:draft-owner-b');
    final id = ChatConversationId.parse('private-conversation');
    await store.initialize({a});
    final old = store.lease(a);
    backend.holdDraftWrite = true;
    final save = store.save(
      old,
      id,
      const ChatDraft.empty().edit('private-delayed'),
    );
    final observed = expectLater(save, throwsA(isA<ChatDraftLeaseExpired>()));
    await backend.entered.future;
    final purge = store.removeAccount(a);
    final stale = store.save(
      old,
      id,
      const ChatDraft.empty().edit('private-stale'),
    );
    final staleObserved = expectLater(
      stale,
      throwsA(isA<ChatDraftLeaseExpired>()),
    );
    backend.release.complete();
    await observed;
    await purge;
    await staleObserved;
    expect(backend.values.values.join(), isNot(contains('private-delayed')));
    await store.initialize({a, b});
    await store.save(
      store.lease(b),
      id,
      const ChatDraft.empty().edit('private-b'),
    );
    expect((await store.load(store.lease(a), id)).text, isEmpty);
    await expectLater(
      store.save(old, id, const ChatDraft.empty().edit('private-stale')),
      throwsA(isA<ChatDraftLeaseExpired>()),
    );
    await store.initialize({b});
    expect(() => store.lease(a), throwsA(isA<ChatDraftLeaseExpired>()));
    expect((await store.load(store.lease(b), id)).text, 'private-b');
  });
  test(
    'local drafts restore isolated owner state and purge only removed owner',
    () async {
      FlutterSecureStorage.setMockInitialValues({});
      final a = AccountKey('did:plc:draft-owner-a');
      final b = AccountKey('did:plc:draft-owner-b');
      final id = ChatConversationId.parse('private-conversation');
      final operation = ChatOperationId.parse(
        '00000000-0000-4000-8000-000000000123',
      );
      final store = ChatDraftStore(const FlutterSecureStorage());
      await store.initialize({a, b});
      final leaseA = store.lease(a);
      await store.save(
        leaseA,
        id,
        const ChatDraft.empty().edit('private-a').beginSend(operation),
      );
      await store.save(
        store.lease(b),
        id,
        const ChatDraft.empty().edit('private-b'),
      );
      final restarted = ChatDraftStore(const FlutterSecureStorage());
      await restarted.initialize({a, b});
      final restored = await restarted.load(restarted.lease(a), id);
      expect(restored.text, 'private-a');
      expect(restored.pending!.state, ChatDraftSendState.uncertain);
      expect(restored.pending!.operationId, operation);
      expect((await restarted.load(restarted.lease(b), id)).text, 'private-b');
      // Logout may precede opening any chat screen in this process.
      await ChatDraftStore(const FlutterSecureStorage()).removeAccount(a);
      final afterColdLogout = ChatDraftStore(const FlutterSecureStorage());
      await afterColdLogout.initialize({a, b});
      expect(
        (await afterColdLogout.load(afterColdLogout.lease(a), id)).text,
        isEmpty,
      );
      await restarted.removeAccount(a);
      await expectLater(
        restarted.save(
          leaseA,
          id,
          const ChatDraft.empty().edit('stale-private-a'),
        ),
        throwsA(isA<ChatDraftLeaseExpired>()),
      );
      final next = ChatDraftStore(const FlutterSecureStorage());
      await next.initialize({b});
      expect(() => next.lease(a), throwsA(isA<ChatDraftLeaseExpired>()));
      expect((await next.load(next.lease(b), id)).text, 'private-b');
      final values = await const FlutterSecureStorage().readAll();
      expect(
        values.keys.any(
          (key) =>
              key.contains('draft-owner') ||
              key.contains('private-conversation'),
        ),
        isFalse,
      );
      expect(values.values.join(), isNot(contains('private-a')));
      await next.save(next.lease(b), id, const ChatDraft.empty());
      expect((await next.load(next.lease(b), id)).text, isEmpty);
    },
  );
}

class _DelayedBackend implements ChatDraftStorageBackend {
  final values = <String, String>{};
  bool holdDraftWrite = false;
  final entered = Completer<void>();
  final release = Completer<void>();

  @override
  Future<String?> read(String key) async => values[key];
  @override
  Future<void> delete(String key) async {
    values.remove(key);
  }

  @override
  Future<void> write(String key, String value) async {
    if (holdDraftWrite && !key.endsWith('.index')) {
      holdDraftWrite = false;
      entered.complete();
      await release.future;
    }
    values[key] = value;
  }
}

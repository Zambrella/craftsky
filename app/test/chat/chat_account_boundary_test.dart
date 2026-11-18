import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_boundary_provider.dart';
import 'package:craftsky_app/chat/data/chat_draft_store.dart';
import 'package:craftsky_app/chat/models/chat_draft.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/providers/chat_draft_store_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';

// IT-017 / FR-016: the actual account-removal cleanup covers delivery recovery.
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  test('account cleanup awaits owner-only delivery recovery purge', () async {
    FlutterSecureStorage.setMockInitialValues({});
    final store = ChatDraftStore(const FlutterSecureStorage());
    final a = AccountKey('did:plc:draft-owner-a');
    final b = AccountKey('did:plc:draft-owner-b');
    final id = ChatConversationId.parse('private-conversation');
    await store.initialize({a, b});
    final stale = store.lease(a);
    final operation = ChatOperationId.parse(
      '00000000-0000-4000-8000-000000000123',
    );
    await store.save(
      stale,
      id,
      const ChatDraft.empty().edit('private-a').beginSend(operation),
    );
    await store.save(
      store.lease(b),
      id,
      const ChatDraft.empty().edit('private-b').beginSend(operation),
    );
    final container = ProviderContainer(
      overrides: [chatDraftStoreProvider.overrideWithValue(store)],
    );
    addTearDown(container.dispose);
    await container.read(accountSessionPrivateStateCleanerProvider)(
      AccountSessionLease(account: a, sessionGeneration: 1),
    );
    expect(() => store.lease(a), throwsA(isA<ChatDraftLeaseExpired>()));
    expect((await store.load(store.lease(b), id)).text, 'private-b');
  });
}

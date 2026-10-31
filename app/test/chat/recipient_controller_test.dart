import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/services/chat_recipient_controller.dart';
import 'package:craftsky_app/search/data/search_repository.dart';
import 'package:craftsky_app/search/models/profile_search_page.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test(
    'IT-003 selection checks availability but only explicit start creates',
    () async {
      final chat = _Chat();
      final search = _Search();
      var current = true;
      final controller = ChatRecipientController(
        account: AccountKey('did:plc:owner'),
        repository: chat,
        searchRepository: search,
        operation: ActiveAccountOperation.withGuard(
          ownership: null,
          isCurrent: () => current,
        ),
      );
      addTearDown(controller.dispose);
      await controller.search('craft');
      expect(search.query, 'craft');
      expect(controller.items.map((item) => item.did.value), ['did:plc:peer']);
      await controller.select(controller.items.single);
      expect(chat.starts, 0);
      expect(controller.canStart, true);
      expect((await controller.start())?.value, 'canonical');
      expect(chat.starts, 1);
      chat.available = false;
      await controller.select(controller.items.single);
      expect(controller.canStart, false);
      expect(await controller.start(), isNull);
      expect(chat.starts, 1);
      chat.available = true;
      await controller.select(controller.items.single);
      current = false;
      expect(await controller.start(), isNull);
      expect(chat.starts, 1);
    },
  );
}

class _Chat implements ChatRepository {
  int starts = 0;
  int checks = 0;
  bool available = true;
  Future<ChatAvailability>? heldAvailability;
  Future<void>? heldStart;
  @override
  Future<ChatAvailability> availability(Did recipient) async {
    checks++;
    return await (heldAvailability ??
        Future.value(ChatAvailability.fromMap({'canChat': available})));
  }

  @override
  Future<ChatConversation> start(Did recipient) async {
    starts++;
    await heldStart;
    return ChatConversation.fromMap({
      'id': 'canonical',
      'revision': 'r',
      'status': 'accepted',
      'muted': false,
      'unreadCount': 0,
      'canSend': true,
      'peer': {'did': recipient.value, 'handle': 'peer.test'},
    });
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => throw UnimplementedError();
}

class _Search implements SearchRepository {
  String? query;
  @override
  Future<ProfileSearchPage> searchProfiles({
    required String q,
    int? limit,
    String? cursor,
  }) async {
    query = q;
    return ProfileSearchPage(
      items: [
        ProfileSearchResult(
          did: 'did:plc:peer',
          handle: 'peer.test',
          isCraftskyProfile: true,
          viewerIsFollowing: false,
        ),
        ProfileSearchResult(
          did: 'did:plc:external',
          handle: 'external.test',
          isCraftskyProfile: false,
          viewerIsFollowing: false,
        ),
        ProfileSearchResult(
          did: 'did:plc:owner',
          handle: 'owner.test',
          isCraftskyProfile: true,
          viewerIsFollowing: false,
        ),
      ],
    );
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => throw UnimplementedError();
}

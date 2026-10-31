import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/models/chat_conversation_details.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_message.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/models/text_validation.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:dio/dio.dart';

/// Uses account-bound Dio and fixed AppView consumer operations.
final class ApiChatRepository implements ChatRepository {
  const new(this._dio, {this._cancelToken});
  final Dio _dio;
  final CancelToken? _cancelToken;

  @override
  Future<ChatStatus> status() async =>
      ChatStatus.fromMap(await _request('GET', '/v1/chat/status'));

  @override
  Future<ChatStatus> setConnection({required bool enabled}) async =>
      ChatStatus.fromMap(
        await _request(
          'PUT',
          '/v1/chat/connection',
          data: {'enabled': enabled},
        ),
      );

  @override
  Future<ChatUnreadCounts> unreadCounts() async =>
      ChatUnreadCounts.fromMap(await _request('GET', '/v1/chat/unread-counts'));

  bool _pushValue(Map<String, dynamic> map) {
    final value = map['pushEnabled'];
    if (value is! bool) {
      throw const FormatException('Invalid messaging preference');
    }
    return value;
  }

  @override
  Future<bool> pushPreference() async =>
      _pushValue(await _request('GET', '/v1/chat/push-preferences'));

  @override
  Future<bool> setPushPreference({required bool enabled}) async => _pushValue(
    await _request(
      'PUT',
      '/v1/chat/push-preferences',
      data: {'pushEnabled': enabled},
    ),
  );

  @override
  Future<ChatPolicySetting> incomingPolicy() async => ChatPolicySetting.fromMap(
    await _request('GET', '/v1/chat/incoming-policy'),
  );

  @override
  Future<ChatPolicySetting> setIncomingPolicy(ChatIncomingPolicy value) async =>
      ChatPolicySetting.fromMap(
        await _request(
          'PUT',
          '/v1/chat/incoming-policy',
          data: {'allowIncoming': value.name},
        ),
      );

  @override
  Future<ChatSyncSnapshot> heartbeat(
    ChatView view, {
    ChatConversationId? conversationId,
    int? knownRevision,
  }) async {
    if ((view == ChatView.conversation) != (conversationId != null) ||
        (knownRevision != null && knownRevision < 0)) {
      throw ArgumentError('Invalid messaging view');
    }
    return ChatSyncSnapshot.fromMap(
      await _request(
        'POST',
        '/v1/chat/sync',
        data: {
          'view': view.name,
          'conversationId': ?conversationId?.value,
          'knownRevision': ?knownRevision,
        },
      ),
    );
  }

  @override
  Future<ChatConversationId> notificationOpen(
    ChatOpenReference reference,
  ) async {
    final result = await _request(
      'GET',
      '/v1/chat/notification-opens/${reference.value}',
    );
    if (result['type'] != 'conversation' ||
        result['conversationId'] is! String) {
      throw const FormatException('Invalid messaging notification target');
    }
    return ChatConversationId.parse(result['conversationId'] as String);
  }

  @override
  Future<ChatReceipt> report(
    ChatOperationId operation,
    ChatConversationId id,
    ChatMessageId message,
    String reasonType,
    String? reason,
  ) => _report(operation, id, reasonType, reason, message: message);

  @override
  Future<ChatReceipt> reportConversation(
    ChatOperationId operation,
    ChatConversationId id,
    String reasonType,
    String? reason,
  ) => _report(operation, id, reasonType, reason);

  Future<ChatReceipt> _report(
    ChatOperationId operation,
    ChatConversationId id,
    String reasonType,
    String? reason, {
    ChatMessageId? message,
  }) async {
    if (validateReportInput(reasonType, reason) != null) {
      throw ArgumentError('Invalid messaging report');
    }
    final receipt = ChatReceipt.fromMap(
      await _request(
        'POST',
        '/v1/chat/reports',
        data: {
          'operationId': operation.value,
          'conversationId': id.value,
          if (message != null)
            'messageId': message.value
          else
            'subjectType': 'conversation',
          'reasonType': reasonType,
          'reason': ?reason,
        },
      ),
    );
    if (receipt.operationId != operation || receipt.kind != 'report') {
      throw const FormatException('Invalid messaging receipt');
    }
    return receipt;
  }

  @override
  Future<Uri> authorize({
    required String handoffMode,
    required bool pushEnabled,
  }) async {
    if (handoffMode != 'verified_link' && handoffMode != 'dev_scheme') {
      throw ArgumentError('Invalid messaging handoff mode');
    }
    final result = await _request(
      'POST',
      '/v1/chat/authorization',
      data: {'handoffMode': handoffMode, 'pushEnabled': pushEnabled},
    );
    final raw = result['authUrl'];
    final uri = raw is String ? Uri.tryParse(raw) : null;
    if (uri == null ||
        uri.scheme != 'https' ||
        uri.host.isEmpty ||
        uri.userInfo.isNotEmpty ||
        uri.hasFragment) {
      throw const FormatException('Invalid messaging authorization response');
    }
    return uri;
  }

  String _conversationPath(ChatConversationId id) =>
      '/v1/chat/conversations/${Uri.encodeComponent(id.value)}';

  Future<Map<String, dynamic>> _request(
    String method,
    String path, {
    Map<String, Object>? query,
    Map<String, Object?>? data,
  }) => unwrapApi(() async {
    final response = await _dio.request<Map<String, dynamic>>(
      path,
      options: Options(method: method),
      cancelToken: _cancelToken,
      queryParameters: query,
      data: data,
    );
    final result = response.data;
    if (result == null) {
      throw const FormatException('Invalid messaging response');
    }
    return result;
  });

  @override
  Future<ChatConversation> conversation(ChatConversationId id) async =>
      ChatConversation.fromMap(await _request('GET', _conversationPath(id)));

  @override
  Future<ChatConversationDetails> details(ChatConversationId id) async {
    final result = ChatConversationDetails.fromMap(
      await _request('GET', '${_conversationPath(id)}/details'),
    );
    if (result.conversation.id != id) {
      throw const FormatException('Invalid conversation details identity');
    }
    return result;
  }

  @override
  Future<ChatMessagePage> history(
    ChatConversationId id, {
    ChatCursor? cursor,
    int limit = 50,
  }) async {
    if (limit < 1 || limit > 100) {
      throw ArgumentError('Invalid messaging page size');
    }
    return ChatMessagePage.fromMap(
      await _request(
        'GET',
        '${_conversationPath(id)}/messages',
        query: {'limit': limit, 'cursor': ?cursor?.value},
      ),
    );
  }

  @override
  Future<ChatSendResult> send(
    ChatConversationId id,
    ChatOperationId operation,
    String text,
  ) async {
    if (validateMessageText(text) != null) {
      throw ArgumentError('Invalid messaging text');
    }
    final result = ChatSendResult.fromMap(
      await _request(
        'POST',
        '${_conversationPath(id)}/messages',
        data: {'operationId': operation.value, 'text': text},
      ),
    );
    if (result.receipt.operationId != operation) {
      throw const FormatException('Invalid messaging receipt');
    }
    return result;
  }

  @override
  Future<ChatReceipt> receipt(ChatOperationId operation) async {
    final result = ChatReceipt.fromMap(
      await _request('GET', '/v1/chat/operations/${operation.value}'),
    );
    if (result.operationId != operation) {
      throw const FormatException('Invalid messaging receipt');
    }
    return result;
  }

  @override
  Future<ChatConversation> updateRead(
    ChatConversationId id,
    ChatMessageId message,
  ) async => ChatConversation.fromMap(
    await _request(
      'PUT',
      '${_conversationPath(id)}/read',
      data: {'messageId': message.value},
    ),
  );

  @override
  Future<ChatConversation> accept(ChatConversationId id) async =>
      ChatConversation.fromMap(
        await _request('POST', '${_conversationPath(id)}/accept'),
      );

  @override
  Future<ChatConversation> setMuted(
    ChatConversationId id, {
    required bool muted,
  }) async => ChatConversation.fromMap(
    await _request(
      muted ? 'PUT' : 'DELETE',
      '${_conversationPath(id)}/mute',
    ),
  );

  @override
  Future<void> leave(ChatConversationId id) async {
    if ((await _request('POST', '${_conversationPath(id)}/leave'))['left'] !=
        true) {
      throw const FormatException('Invalid messaging leave result');
    }
  }

  @override
  Future<void> deleteMessage(
    ChatConversationId id,
    ChatMessageId message,
  ) async {
    final result = await _request(
      'DELETE',
      '${_conversationPath(id)}/messages/${Uri.encodeComponent(message.value)}',
    );
    if (result['deleted'] != true) {
      throw const FormatException('Invalid messaging deletion result');
    }
  }

  @override
  Future<ChatAvailability> availability(Did recipient) async =>
      ChatAvailability.fromMap(
        await _request(
          'GET',
          '/v1/chat/availability',
          query: {'recipientDid': recipient.value},
        ),
      );

  @override
  Future<ChatConversation> start(Did recipient) async =>
      ChatConversation.fromMap(
        await _request(
          'POST',
          '/v1/chat/conversations',
          data: {'recipientDid': recipient.value},
        ),
      );

  @override
  Future<ChatConversationPage> conversations({
    required ChatBucket bucket,
    ChatCursor? cursor,
    int limit = 50,
  }) async {
    if (limit < 1 || limit > 100) {
      throw ArgumentError('Invalid messaging page size');
    }
    return await unwrapApi(() async {
      final response = await _dio.get<Map<String, dynamic>>(
        '/v1/chat/conversations',
        cancelToken: _cancelToken,
        queryParameters: {
          'status': bucket.name,
          'limit': limit,
          'cursor': ?cursor?.value,
        },
      );
      return ChatConversationPage.fromMap(response.data!);
    });
  }
}

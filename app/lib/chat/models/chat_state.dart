import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_message.dart';
import 'package:flutter/foundation.dart';

enum ChatOperationState { confirmed, failed, uncertain }

@immutable
final class ChatReceipt {
  const new _({
    required this.operationId,
    required this.kind,
    required this.state,
    this.messageId,
    this.failureCode,
  });

  factory fromMap(Map<String, dynamic> map) {
    final id = map['operationId'];
    final kind = map['kind'];
    final state = map['state'];
    if (id is! String ||
        (kind != 'send' && kind != 'report') ||
        !ChatOperationState.values.any((value) => value.name == state)) {
      throw const FormatException('Invalid messaging receipt');
    }
    final rawMessage = map['messageId'];
    if ((rawMessage != null && rawMessage is! String) ||
        (kind == 'send' && state == 'confirmed' && rawMessage is! String)) {
      throw const FormatException('Invalid messaging receipt');
    }
    final failure = map['failureCode'];
    return ChatReceipt._(
      operationId: ChatOperationId.parse(id),
      kind: kind as String,
      state: ChatOperationState.values.byName(state as String),
      messageId: rawMessage is String ? ChatMessageId.parse(rawMessage) : null,
      failureCode:
          failure is String &&
              RegExp(r'^[a-zA-Z][a-zA-Z0-9_]{0,79}$').hasMatch(failure)
          ? failure
          : null,
    );
  }

  final ChatOperationId operationId;
  final String kind;
  final ChatOperationState state;
  final ChatMessageId? messageId;
  final String? failureCode;

  @override
  String toString() => 'ChatReceipt(<redacted>)';
}

@immutable
final class ChatSendResult {
  const new _({required this.receipt, this.message});

  factory fromMap(Map<String, dynamic> map) {
    final receipt = ChatReceipt.fromMap(map);
    final raw = map['message'];
    final message = raw is Map<String, dynamic>
        ? ChatMessage.fromMap(raw)
        : null;
    if (receipt.kind != 'send' ||
        (receipt.state != ChatOperationState.confirmed && message != null) ||
        (message != null && message.id != receipt.messageId)) {
      throw const FormatException('Invalid messaging send result');
    }
    return ChatSendResult._(receipt: receipt, message: message);
  }

  final ChatReceipt receipt;
  final ChatMessage? message;

  @override
  String toString() => 'ChatSendResult(<redacted>)';
}

@immutable
final class ChatAvailability {
  const new _({required this.canChat, this.conversationId});

  factory fromMap(Map<String, dynamic> map) {
    final canChat = map['canChat'];
    final id = map['conversationId'];
    if (canChat is! bool || (id != null && id is! String)) {
      throw const FormatException('Invalid messaging availability');
    }
    return ChatAvailability._(
      canChat: canChat,
      conversationId: id == null
          ? null
          : ChatConversationId.parse(id as String),
    );
  }

  final bool canChat;
  final ChatConversationId? conversationId;

  @override
  String toString() => 'ChatAvailability(<redacted>)';
}

enum ChatConnectionState {
  disabled,
  authorizationNeeded,
  restricted,
  unavailable,
  ready,
}

enum ChatIncomingPolicy { everyone, following, nobody }

enum ChatView { inbox, conversation, none }

@immutable
final class ChatStatus {
  const new _({
    required this.generation,
    required this.enabled,
    required this.pushEnabled,
    required this.state,
    required this.canRead,
    required this.canSend,
    required this.canReport,
    required this.canSetPolicy,
    this.syncState,
    this.recoveryCode,
    this.lastSuccessAt,
  });

  factory fromMap(Map<String, dynamic> map) {
    final caps = map['capabilities'];
    final state = map['state'];
    final generation = map['generation'];
    const safeStates = {
      'syncing',
      'ready',
      'authorizationNeeded',
      'restricted',
      'unavailable',
      'rateLimited',
      'disabled',
    };
    final sync = map['syncState'];
    final recovery = map['recoveryCode'];
    final rawSuccess = map['lastSuccessAt'];
    final success = rawSuccess is String ? DateTime.tryParse(rawSuccess) : null;
    if ((sync != null && !safeStates.contains(sync)) ||
        (recovery != null && !safeStates.contains(recovery)) ||
        (rawSuccess != null && (success == null || !success.isUtc))) {
      throw const FormatException('Invalid messaging recovery');
    }
    if (generation is! int ||
        generation < 0 ||
        map['enabled'] is! bool ||
        map['pushEnabled'] is! bool ||
        !ChatConnectionState.values.any((value) => value.name == state) ||
        caps is! Map<String, dynamic> ||
        [
          'canRead',
          'canSend',
          'canReport',
          'canSetPolicy',
        ].any((key) => caps[key] is! bool)) {
      throw const FormatException('Invalid messaging status');
    }
    return ChatStatus._(
      generation: generation,
      enabled: map['enabled'] as bool,
      pushEnabled: map['pushEnabled'] as bool,
      state: ChatConnectionState.values.byName(state as String),
      canRead: caps['canRead'] as bool,
      canSend: caps['canSend'] as bool,
      canReport: caps['canReport'] as bool,
      canSetPolicy: caps['canSetPolicy'] as bool,
      syncState: sync as String?,
      recoveryCode: recovery as String?,
      lastSuccessAt: success,
    );
  }

  ChatStatus degraded() => ChatStatus._(
    generation: generation,
    enabled: enabled,
    pushEnabled: pushEnabled,
    state: ChatConnectionState.unavailable,
    canRead: canRead,
    canSend: canSend,
    canReport: canReport,
    canSetPolicy: canSetPolicy,
    syncState: syncState,
    recoveryCode: 'unavailable',
    lastSuccessAt: lastSuccessAt,
  );

  ChatStatus accessLost(ChatConnectionState next) => ChatStatus._(
    generation: generation,
    enabled: next != ChatConnectionState.disabled && enabled,
    pushEnabled: pushEnabled,
    state: next,
    canRead: false,
    canSend: false,
    canReport: false,
    canSetPolicy: false,
    syncState: next.name,
    recoveryCode: next.name,
    lastSuccessAt: lastSuccessAt,
  );

  final bool enabled;
  final int generation;
  final bool pushEnabled;
  final ChatConnectionState state;
  final bool canRead;
  final bool canSend;
  final bool canReport;
  final bool canSetPolicy;
  final String? syncState;
  final String? recoveryCode;
  final DateTime? lastSuccessAt;
}

@immutable
final class ChatUnreadCounts {
  const new _({required this.accepted, required this.requests});

  factory fromMap(Map<String, dynamic> map) {
    final accepted = map['accepted'];
    final requests = map['requests'];
    if (accepted is! int ||
        requests is! int ||
        accepted < 0 ||
        accepted > 100 ||
        requests < 0 ||
        requests > 100) {
      throw const FormatException('Invalid messaging counts');
    }
    return ChatUnreadCounts._(accepted: accepted, requests: requests);
  }

  final int accepted;
  final int requests;
  String get acceptedLabel => accepted == 100 ? '99+' : '$accepted';
  String get requestsLabel => requests == 100 ? '99+' : '$requests';
}

@immutable
final class ChatPolicySetting {
  const new _({required this.value, required this.persisted});

  factory fromMap(Map<String, dynamic> map) {
    final incoming = map['allowIncoming'];
    if (!ChatIncomingPolicy.values.any((value) => value.name == incoming) ||
        map['persisted'] is! bool) {
      throw const FormatException('Invalid messaging policy');
    }
    return ChatPolicySetting._(
      value: ChatIncomingPolicy.values.byName(incoming as String),
      persisted: map['persisted'] as bool,
    );
  }

  final ChatIncomingPolicy value;
  final bool persisted;
}

@immutable
final class ChatSyncSnapshot {
  const new _({
    required this.revision,
    required this.state,
    this.lastSuccessAt,
  });

  factory fromMap(Map<String, dynamic> map) {
    final revision = map['revision'];
    final state = map['state'];
    final rawDate = map['lastSuccessAt'];
    final date = rawDate is String ? DateTime.tryParse(rawDate) : null;
    if (revision is! int ||
        revision < 0 ||
        state is! String ||
        !RegExp(r'^[a-zA-Z]{1,40}$').hasMatch(state) ||
        (rawDate != null && (date == null || !date.isUtc))) {
      throw const FormatException('Invalid messaging sync state');
    }
    return ChatSyncSnapshot._(
      revision: revision,
      state: state,
      lastSuccessAt: date,
    );
  }

  final int revision;
  final String state;
  final DateTime? lastSuccessAt;
}

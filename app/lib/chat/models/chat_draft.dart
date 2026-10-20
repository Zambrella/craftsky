import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/models/text_validation.dart';
import 'package:flutter/foundation.dart';

enum ChatDraftSendState { pending, failed, uncertain }

@immutable
final class ChatDraftAttempt {
  const new({
    required this.operationId,
    required this.text,
    required this.revision,
    required this.state,
  });

  final ChatOperationId operationId;
  final String text;
  final int revision;
  final ChatDraftSendState state;

  ChatDraftAttempt withState(ChatDraftSendState value) => ChatDraftAttempt(
    operationId: operationId,
    text: text,
    revision: revision,
    state: value,
  );

  @override
  String toString() => 'ChatDraftAttempt(<redacted>)';
}

/// Device-local composition state. This reducer never dispatches network work.
@immutable
final class ChatDraft {
  const new _({required this.text, required this.revision, this.pending});
  const new empty() : text = '', revision = 0, pending = null;

  /// Private local-storage representation, never an AppView request body.
  factory fromStorage(Map<String, dynamic> map) {
    final text = map['text'];
    final revision = map['revision'];
    final rawAttempt = map['attempt'];
    if (text is! String ||
        revision is! int ||
        revision < 0 ||
        (rawAttempt != null && rawAttempt is! Map<String, dynamic>)) {
      throw const FormatException('Invalid local messaging draft');
    }
    ChatDraftAttempt? attempt;
    if (rawAttempt is Map<String, dynamic>) {
      final id = rawAttempt['operationId'];
      final submitted = rawAttempt['text'];
      final submittedRevision = rawAttempt['revision'];
      final state = rawAttempt['state'];
      if (id is! String ||
          submitted is! String ||
          submittedRevision is! int ||
          submittedRevision < 0 ||
          submittedRevision > revision ||
          !ChatDraftSendState.values.any((value) => value.name == state)) {
        throw const FormatException('Invalid local messaging attempt');
      }
      attempt = ChatDraftAttempt(
        operationId: ChatOperationId.parse(id),
        text: submitted,
        revision: submittedRevision,
        state: state == 'failed'
            ? ChatDraftSendState.failed
            : ChatDraftSendState.uncertain,
      );
    }
    return ChatDraft._(text: text, revision: revision, pending: attempt);
  }

  Map<String, Object?> toStorage() => {
    'text': text,
    'revision': revision,
    if (pending case final attempt?)
      'attempt': {
        'operationId': attempt.operationId.value,
        'text': attempt.text,
        'revision': attempt.revision,
        'state': attempt.state.name,
      },
  };

  final String text;
  final int revision;
  final ChatDraftAttempt? pending;

  bool get canSend => pending == null && validateMessageText(text) == null;

  ChatDraft edit(String value) => value == text
      ? this
      : ChatDraft._(text: value, revision: revision + 1, pending: pending);

  ChatDraft beginSend(ChatOperationId id) {
    if (!canSend) throw StateError('Messaging draft cannot be submitted');
    return ChatDraft._(
      text: text,
      revision: revision,
      pending: ChatDraftAttempt(
        operationId: id,
        text: text,
        revision: revision,
        state: ChatDraftSendState.pending,
      ),
    );
  }

  ChatDraft markUncertain() => pending == null
      ? this
      : ChatDraft._(
          text: text,
          revision: revision,
          pending: pending!.withState(ChatDraftSendState.uncertain),
        );

  ChatDraft resolve(ChatReceipt receipt) {
    final attempt = pending;
    if (attempt == null ||
        receipt.operationId != attempt.operationId ||
        receipt.kind != 'send') {
      return this;
    }
    if (receipt.state == ChatOperationState.confirmed) {
      return ChatDraft._(
        text: revision == attempt.revision ? '' : text,
        revision: revision + (revision == attempt.revision ? 1 : 0),
      );
    }
    return ChatDraft._(
      text: text,
      revision: revision,
      pending: attempt.withState(
        receipt.state == ChatOperationState.failed
            ? ChatDraftSendState.failed
            : ChatDraftSendState.uncertain,
      ),
    );
  }

  ChatDraft repeatUncertain(ChatOperationId id) {
    final attempt = pending;
    if (attempt == null ||
        attempt.state != ChatDraftSendState.uncertain ||
        id == attempt.operationId ||
        validateMessageText(attempt.text) != null) {
      throw StateError('Messaging attempt cannot be repeated');
    }
    return ChatDraft._(
      text: text,
      revision: revision,
      pending: ChatDraftAttempt(
        operationId: id,
        text: attempt.text,
        revision: attempt.revision,
        state: ChatDraftSendState.pending,
      ),
    );
  }

  /// Called only by an explicit recovery choice after explaining uncertainty.
  ChatDraft releaseAttempt() => ChatDraft._(text: text, revision: revision);

  ChatDraft restoreAttempt(ChatDraftAttempt? attempt) {
    if (attempt != null && attempt.revision > revision) {
      throw StateError('Messaging attempt revision is invalid');
    }
    return ChatDraft._(text: text, revision: revision, pending: attempt);
  }

  ChatDraft discard() => ChatDraft._(text: '', revision: revision + 1);

  @override
  String toString() => 'ChatDraft(<redacted>)';
}

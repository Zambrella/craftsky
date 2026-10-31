import 'package:craftsky_app/chat/models/chat_draft.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:flutter_test/flutter_test.dart';

// UT-009 / FR-005, FR-016, FR-019 / AC-028.
void main() {
  test('UT-009 explicit uncertain resend preserves '
      'original attempt and newer draft', () {
    final first = ChatOperationId.parse('00000000-0000-4000-8000-000000000123');
    final next = ChatOperationId.parse('00000000-0000-4000-8000-000000000456');
    final uncertain = const ChatDraft.empty()
        .edit('private-original')
        .beginSend(first)
        .markUncertain()
        .edit('private-newer');
    expect(() => uncertain.repeatUncertain(first), throwsStateError);
    final retry = uncertain.repeatUncertain(next);
    expect(retry.pending!.operationId, next);
    expect(retry.pending!.text, 'private-original');
    expect(retry.text, 'private-newer');
    expect(
      retry
          .resolve(
            ChatReceipt.fromMap({
              'operationId': next.value,
              'kind': 'send',
              'state': 'confirmed',
              'messageId': 'message-2',
            }),
          )
          .text,
      'private-newer',
    );
  });
  test(
    'confirmation clears only submitted revision and never retries implicitly',
    () {
      var draft = const ChatDraft.empty().edit('private-original');
      final operation = ChatOperationId.parse(
        '00000000-0000-4000-8000-000000000123',
      );
      draft = draft.beginSend(operation);
      expect(draft.pending!.text, 'private-original');
      expect(draft.canSend, isFalse);
      expect(() => draft.beginSend(operation), throwsStateError);
      draft = draft.edit('private-newer');
      draft = draft.resolve(
        ChatReceipt.fromMap({
          'operationId': operation.value,
          'kind': 'send',
          'state': 'confirmed',
          'messageId': 'message-1',
        }),
      );
      expect(draft.text, 'private-newer');
      expect(draft.pending, isNull);
      draft = draft.beginSend(operation);
      draft = draft.markUncertain();
      expect(draft.text, 'private-newer');
      expect(draft.pending!.state, ChatDraftSendState.uncertain);
      expect(draft.canSend, isFalse);
      // Explicit recovery/discard is required before a newly chosen UUID.
      draft = draft.releaseAttempt();
      expect(draft.text, 'private-newer');
      expect(draft.canSend, isTrue);
      final next = ChatOperationId.parse(
        '00000000-0000-4000-8000-000000000456',
      );
      draft = draft
          .beginSend(next)
          .resolve(
            ChatReceipt.fromMap({
              'operationId': next.value,
              'kind': 'send',
              'state': 'failed',
              'failureCode': 'notSubmitted',
            }),
          );
      expect(draft.text, 'private-newer');
      expect(draft.pending!.state, ChatDraftSendState.failed);
      draft = draft
          .releaseAttempt()
          .beginSend(next)
          .resolve(
            ChatReceipt.fromMap({
              'operationId': next.value,
              'kind': 'send',
              'state': 'confirmed',
              'messageId': 'message-2',
            }),
          );
      expect(draft.text, isEmpty);
      expect(draft.canSend, isFalse);
      expect(draft.edit('  ').canSend, isFalse);
      expect(draft.edit('x' * 1001).text.length, 1001);
      expect(draft.edit('x' * 1001).canSend, isFalse);
      expect(draft.edit('private-draft').discard().text, isEmpty);
      expect(
        '${draft.edit('private-draft')}',
        isNot(contains('private-draft')),
      );
    },
  );
}

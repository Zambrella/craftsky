import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/models/text_validation.dart';
import 'package:craftsky_app/chat/services/chat_report_controller.dart';
import 'package:flutter_test/flutter_test.dart';

// UT-005 / FR-010 / AC-010. Submission/evidence belongs to IT-004.
void main() {
  for (final conversation in [false, true]) {
    test(
      'IT-004 uncertain report checks metadata without resubmitting '
      'explanation conversation=$conversation',
      () async {
        final repository = _ReportRepository();
        final controller = ChatReportController(
          account: AccountKey('did:plc:owner'),
          conversationId: ChatConversationId.parse('conversation'),
          messageId: conversation ? null : ChatMessageId.parse('message'),
          repository: repository,
          operation: ActiveAccountOperation.withGuard(
            ownership: null,
            isCurrent: () => true,
          ),
        );
        addTearDown(controller.dispose);
        await controller.submit(
          'com.atproto.moderation.defs#reasonSpam',
          'private-explanation',
        );
        expect(controller.state, ChatOperationState.uncertain);
        await controller.submit(
          'com.atproto.moderation.defs#reasonSpam',
          'private-explanation',
        );
        expect(repository.reports, 1);
        expect(repository.conversationReports, conversation ? 1 : 0);
        await controller.checkReceipt();
        expect(repository.reports, 1);
        expect(repository.checks, 1);
        expect(controller.state, ChatOperationState.confirmed);
        expect(controller.toString(), isNot(contains('private-explanation')));
      },
    );
  }
  test('report input has independent explanation and reason limits', () {
    final byteBoundary = 'a${'\u0301' * 9999}';
    final cases = <(String, String, String?, bool)>[
      ('no explanation', 'com.atproto.moderation.defs#reasonSpam', null, true),
      (
        'empty optional explanation',
        'com.atproto.moderation.defs#reasonOther',
        '',
        true,
      ),
      ('missing reason type', '', null, false),
      (
        'unsupported reason type',
        'example.invalid#private-canary',
        null,
        false,
      ),
      (
        '1999 graphemes',
        'com.atproto.moderation.defs#reasonRude',
        'x' * 1999,
        true,
      ),
      (
        '2000 graphemes',
        'com.atproto.moderation.defs#reasonRude',
        'x' * 2000,
        true,
      ),
      (
        '2001 graphemes',
        'com.atproto.moderation.defs#reasonRude',
        'x' * 2001,
        false,
      ),
      (
        '19999 bytes',
        'com.atproto.moderation.defs#reasonOther',
        byteBoundary,
        true,
      ),
      (
        '20000 bytes',
        'com.atproto.moderation.defs#reasonOther',
        '${byteBoundary}b',
        true,
      ),
      (
        '20001 bytes',
        'com.atproto.moderation.defs#reasonOther',
        '${byteBoundary}bc',
        false,
      ),
      (
        'invalid Unicode',
        'com.atproto.moderation.defs#reasonOther',
        String.fromCharCode(0xd800),
        false,
      ),
    ];
    for (final (name, reasonType, reason, valid) in cases) {
      expect(
        validateReportInput(reasonType, reason) == null,
        valid,
        reason: name,
      );
    }
  });
}

class _ReportRepository implements ChatRepository {
  int reports = 0;
  int conversationReports = 0;
  int checks = 0;
  ChatOperationId? id;
  @override
  Future<ChatReceipt> report(
    ChatOperationId operation,
    ChatConversationId conversation,
    ChatMessageId message,
    String type,
    String? reason,
  ) async {
    reports++;
    id = operation;
    throw StateError('synthetic-drop');
  }

  @override
  Future<ChatReceipt> reportConversation(
    ChatOperationId operation,
    ChatConversationId conversation,
    String type,
    String? reason,
  ) async {
    conversationReports++;
    reports++;
    id = operation;
    throw StateError('synthetic-drop');
  }

  @override
  Future<ChatReceipt> receipt(ChatOperationId operation) async {
    checks++;
    expect(operation, id);
    return ChatReceipt.fromMap({
      'operationId': operation.value,
      'kind': 'report',
      'state': 'confirmed',
    });
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => throw UnimplementedError();
}

import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/models/text_validation.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:flutter/foundation.dart';
import 'package:uuid/uuid.dart';

/// One explicit report. Explanation stays only in the caller's form memory.
final class ChatReportController extends ChangeNotifier {
  new({
    required this.account,
    required this.conversationId,
    required this.messageId,
    required this._repository,
    required this._operation,
    ErrorReporter reporter = const NoopErrorReporter(),
    ChatOperationId Function()? newOperationId,
  }) : _reporter = GuardedErrorReporter(reporter),
       operationId =
           (newOperationId ?? () => ChatOperationId.parse(const Uuid().v4()))();
  final AccountKey account;
  final ChatConversationId conversationId;
  final ChatMessageId? messageId;
  final ChatOperationId operationId;
  final ChatRepository _repository;
  final ActiveAccountOperation _operation;
  final ErrorReporter _reporter;
  ChatOperationState? state;
  bool busy = false;
  bool _submitted = false;
  bool _disposed = false;
  bool get _current => !_disposed && _operation.isCurrent;
  void _publish(void Function() action) {
    if (_current) {
      _operation.publish(() {
        action();
        notifyListeners();
      });
    }
  }

  void _report(Object error, StackTrace stack, String name) {
    unawaited(
      _reporter.captureException(
        error,
        stackTrace: stack,
        context: ReportContext(
          feature: 'Chat',
          operation: name,
          classification: 'chat.failure',
          workflow: PrivateOperationalFailureContext(
            accountDid: account.did.value,
          ),
        ),
      ),
    );
  }

  Future<void> submit(String type, String? reason) async {
    if (!_current ||
        _submitted ||
        busy ||
        validateReportInput(type, reason) != null) {
      return;
    }
    _submitted = true;
    _publish(() => busy = true);
    try {
      final message = messageId;
      final receipt = await _operation.step(
        () => message == null
            ? _repository.reportConversation(
                operationId,
                conversationId,
                type,
                reason,
              )
            : _repository.report(
                operationId,
                conversationId,
                message,
                type,
                reason,
              ),
      );
      _publish(() => state = receipt.state);
    } on AccountOperationDiscarded {
      return;
    } on Object catch (error, stack) {
      if (_current) {
        _report(error, stack, 'report');
        _publish(() => state = ChatOperationState.uncertain);
      }
    } finally {
      _publish(() => busy = false);
    }
  }

  Future<void> checkReceipt() async {
    if (!_current ||
        !_submitted ||
        busy ||
        state == ChatOperationState.confirmed) {
      return;
    }
    _publish(() => busy = true);
    try {
      final receipt = await _operation.step(
        () => _repository.receipt(operationId),
      );
      if (receipt.kind != 'report' || receipt.operationId != operationId) {
        throw const FormatException('Invalid report receipt');
      }
      _publish(() => state = receipt.state);
    } on AccountOperationDiscarded {
      return;
    } on Object catch (error, stack) {
      if (_current) _report(error, stack, 'reportReceipt');
    } finally {
      _publish(() => busy = false);
    }
  }

  @override
  void dispose() {
    _disposed = true;
    state = null;
    super.dispose();
  }

  @override
  String toString() => 'ChatReportController(<redacted>)';
}

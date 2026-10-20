import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:flutter/foundation.dart';

/// One bucket for one captured account activation; never marks or accepts read.
final class ChatInboxController extends ChangeNotifier {
  new({
    required this.account,
    required this.bucket,
    required this._repository,
    required this._operation,
    ErrorReporter reporter = const NoopErrorReporter(),
  }) : _reporter = GuardedErrorReporter(reporter);

  final AccountKey account;
  final ChatBucket bucket;
  final ChatRepository _repository;
  final ActiveAccountOperation _operation;
  final ErrorReporter _reporter;
  List<ChatConversation> items = const [];
  ChatCursor? cursor;
  bool loading = true;
  bool loadingMore = false;
  bool failed = false;
  bool _disposed = false;
  int _snapshot = 0;
  bool _refreshing = false;

  bool get hasMore => cursor != null;
  bool get _current => !_disposed && _operation.isCurrent;

  void _publish(void Function() action) {
    if (!_current) return;
    _operation.publish(() {
      action();
      notifyListeners();
    });
  }

  void _report(Object error, StackTrace stack) {
    unawaited(
      _reporter.captureException(
        error,
        stackTrace: stack,
        context: ReportContext(
          feature: 'Chat',
          operation: 'inbox',
          classification: 'chat.failure',
          workflow: PrivateOperationalFailureContext(
            accountDid: account.did.value,
          ),
        ),
      ),
    );
  }

  Future<void> refresh() async {
    if (!_current || _refreshing) return;
    _refreshing = true;
    final snapshot = ++_snapshot;
    _publish(() => loading = true);
    try {
      final page = await _operation.step(
        () => _repository.conversations(bucket: bucket),
      );
      if (snapshot != _snapshot) return;
      _publish(() {
        items = List.unmodifiable(
          {for (final item in page.items) item.id: item}.values,
        );
        cursor = page.cursor;
        failed = false;
      });
    } on AccountOperationDiscarded {
      return;
    } on Object catch (error, stack) {
      if (!_current) return;
      _report(error, stack);
      _publish(() => failed = true);
    } finally {
      _refreshing = false;
      _publish(() => loading = false);
    }
  }

  Future<void> loadMore() async {
    final continuation = cursor;
    if (!_current || loading || loadingMore || continuation == null) return;
    final snapshot = _snapshot;
    _publish(() => loadingMore = true);
    try {
      final page = await _operation.step(
        () => _repository.conversations(bucket: bucket, cursor: continuation),
      );
      if (snapshot != _snapshot) return;
      _publish(() {
        final merged = {for (final item in items) item.id: item};
        for (final item in page.items) {
          merged[item.id] = item;
        }
        items = List.unmodifiable(merged.values);
        cursor = page.cursor;
        failed = false;
      });
    } on AccountOperationDiscarded {
      return;
    } on Object catch (error, stack) {
      if (!_current) return;
      _report(error, stack);
      _publish(() => failed = true);
    } finally {
      _publish(() => loadingMore = false);
    }
  }

  @override
  void dispose() {
    _disposed = true;
    items = const [];
    cursor = null;
    super.dispose();
  }
}

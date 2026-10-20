import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/chat/data/chat_draft_store.dart';
import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_conversation.dart';
import 'package:craftsky_app/chat/models/chat_draft.dart';
import 'package:craftsky_app/chat/models/chat_history.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_message.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/chat/models/text_validation.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:flutter/foundation.dart';
import 'package:uuid/uuid.dart';

/// In-memory presentation state under one captured account activation.
final class ChatConversationController extends ChangeNotifier {
  new({
    required this.account,
    required this.id,
    required this._repository,
    required this._store,
    required this._draftLease,
    required this._operation,
    this._onSharedStateChanged,
    ChatOperationId Function()? newOperationId,
    ErrorReporter reporter = const NoopErrorReporter(),
  }) : _newOperationId =
           newOperationId ?? (() => ChatOperationId.parse(const Uuid().v4())),
       _reporter = GuardedErrorReporter(reporter);

  final AccountKey account;
  final ChatConversationId id;
  final ChatRepository _repository;
  final ChatDraftStore _store;
  final ChatDraftLease _draftLease;
  final ActiveAccountOperation _operation;
  final Future<void> Function()? _onSharedStateChanged;
  final ChatOperationId Function() _newOperationId;
  final ErrorReporter _reporter;
  ChatConversation? conversation;
  ChatHistory history = const ChatHistory.empty();
  ChatDraft draft = const ChatDraft.empty();
  bool loading = true;
  bool busy = false;
  bool saveFailed = false;
  bool failed = false;
  bool left = false;
  bool loadingOlder = false;
  bool refreshing = false;
  bool _disposed = false;
  bool _draftLoaded = false;
  int _contentEpoch = 0;
  Timer? _saveTimer;
  bool _foreground = false;
  bool _reading = false;
  ChatMessage? _lastRead;
  ChatMessage? _pendingPresented;
  final _historyGaps = <_HistoryGap>[];

  void setForeground({required bool value}) {
    _foreground = value;
    if (!value) _pendingPresented = null;
  }

  int _compareMessage(ChatMessage a, ChatMessage b) {
    final date = a.sentAt.compareTo(b.sentAt);
    return date == 0 ? a.id.value.compareTo(b.id.value) : date;
  }

  /// The view supplies only actually visible message IDs, never fetched IDs.
  Future<void> presented(List<ChatMessageId> visible) async {
    if (!_current || !_foreground || loading) return;
    final epoch = _contentEpoch;
    for (final item in history.messages) {
      if (visible.contains(item.id) &&
          (_lastRead == null || _compareMessage(item, _lastRead!) > 0) &&
          (_pendingPresented == null ||
              _compareMessage(item, _pendingPresented!) > 0)) {
        _pendingPresented = item;
      }
    }
    if (_reading) return;
    _reading = true;
    try {
      while (_current && _foreground && _pendingPresented != null) {
        final target = _pendingPresented!;
        _pendingPresented = null;
        await _operation.step(() => _repository.updateRead(id, target.id));
        if (epoch != _contentEpoch) return;
        final summary = await _operation.step(
          () => _repository.conversation(id),
        );
        if (epoch != _contentEpoch) return;
        _publish(() {
          _lastRead = target;
          conversation = summary;
        });
        await _sharedChanged();
      }
    } on AccountOperationDiscarded {
      return;
    } on Object catch (error, stack) {
      _contentFailure(error, stack, 'read', epoch);
    } finally {
      _reading = false;
      if (_current && _foreground && _pendingPresented != null) {
        unawaited(presented(const []));
      }
    }
  }

  bool get _current => !_disposed && _operation.isCurrent;

  void _contentFailure(
    Object error,
    StackTrace stack,
    String operation,
    int epoch,
  ) {
    if (!_current || epoch != _contentEpoch) return;
    final denied =
        error is ApiUnauthorized ||
        (error is ApiBadRequest &&
            error.code != 'chat_relationship_unavailable' &&
            (const {
                  'chat_conversation_unavailable',
                  'chat_authorization_required',
                  'chat_restricted',
                  'chat_disconnected',
                  'chat_connection_changed',
                }.contains(error.code) ||
                const {403, 404, 410}.contains(error.details.statusCode)));
    if (denied) {
      invalidateAccess();
      return;
    }
    _report(error, stack, operation);
    _publish(() => failed = true);
  }

  void invalidateAccess() {
    _publish(() {
      _contentEpoch++;
      conversation = null;
      history = const ChatHistory.empty();
      _historyGaps.clear();
      _lastRead = null;
      _pendingPresented = null;
      failed = true;
    });
  }

  bool get canSend =>
      _current && !busy && conversation?.canSend == true && draft.canSend;

  bool get canSendAgain {
    final attempt = draft.pending;
    if (!_current || busy || conversation?.canSend != true || attempt == null) {
      return false;
    }
    return switch (attempt.state) {
      ChatDraftSendState.pending => false,
      ChatDraftSendState.failed => validateMessageText(draft.text) == null,
      ChatDraftSendState.uncertain => validateMessageText(attempt.text) == null,
    };
  }

  void _publish(void Function() action) {
    if (!_current) return;
    _operation.publish(() {
      action();
      notifyListeners();
    });
  }

  void _report(Object error, StackTrace stack, String operation) {
    unawaited(
      _reporter.captureException(
        error,
        stackTrace: stack,
        context: ReportContext(
          feature: 'Chat',
          operation: operation,
          classification: 'chat.failure',
          workflow: PrivateOperationalFailureContext(
            accountDid: account.did.value,
          ),
        ),
      ),
    );
  }

  Future<void> load() async {
    final epoch = _contentEpoch;
    try {
      final summary = await _operation.step(() => _repository.conversation(id));
      final page = await _operation.step(() => _repository.history(id));
      final local = await _operation.step(() => _store.load(_draftLease, id));
      if (epoch != _contentEpoch) return;
      _publish(() {
        conversation = summary;
        history = const ChatHistory.empty().merge(page);
        draft = local;
        _draftLoaded = true;
        loading = false;
        failed = false;
      });
    } on AccountOperationDiscarded {
      return;
    } on Object catch (error, stack) {
      if (!_current) return;
      _contentFailure(error, stack, 'load', epoch);
      _publish(() => loading = false);
    }
  }

  Future<void> loadOlder() async {
    final cursor = history.cursor;
    if (!_current || loading || loadingOlder || cursor == null) return;
    final epoch = _contentEpoch;
    _publish(() => loadingOlder = true);
    try {
      final page = await _operation.step(
        () => _repository.history(id, cursor: cursor),
      );
      if (epoch != _contentEpoch) return;
      _publish(() {
        if (history.cursor != cursor) {
          // A newer refresh owns the continuation. The delayed older content
          // can still merge, but must not skip that newly discovered interval.
          history = history.merge(
            ChatMessagePage(items: page.items, cursor: history.cursor),
          );
          failed = false;
          return;
        }
        var next = page.cursor;
        if (_historyGaps.isNotEmpty) {
          final gap = _historyGaps.last;
          if (next == null ||
              page.items.any((message) => gap.olderIds.contains(message.id))) {
            next = gap.olderCursor;
            _historyGaps.removeLast();
          }
        }
        history = history.merge(
          ChatMessagePage(items: page.items, cursor: next),
        );
        failed = false;
      });
    } on AccountOperationDiscarded {
      return;
    } on Object catch (error, stack) {
      if (!_current) return;
      _contentFailure(error, stack, 'history', epoch);
    } finally {
      _publish(() => loadingOlder = false);
    }
  }

  void _mergeNewest(ChatMessagePage page) {
    final olderIds = {for (final message in history.messages) message.id};
    if (olderIds.isEmpty) {
      history = history.merge(page);
    } else if (page.cursor != null &&
        !page.items.any((message) => olderIds.contains(message.id))) {
      // Follow the new page back to the retained segment before resuming its
      // older continuation. A long absence must not leave an invisible gap.
      _historyGaps.add(_HistoryGap(olderIds, history.cursor));
      history = history.merge(page);
    } else {
      history = history.merge(
        ChatMessagePage(items: page.items, cursor: history.cursor),
      );
    }
  }

  Future<void> refresh() async {
    if (!_current || loading || refreshing) return;
    final epoch = _contentEpoch;
    _publish(() => refreshing = true);
    try {
      final summary = await _operation.step(() => _repository.conversation(id));
      final page = await _operation.step(() => _repository.history(id));
      if (epoch != _contentEpoch) return;
      _publish(() {
        conversation = summary;
        _mergeNewest(page);
        failed = false;
      });
    } on AccountOperationDiscarded {
      return;
    } on Object catch (error, stack) {
      if (!_current) return;
      _contentFailure(error, stack, 'refresh', epoch);
    } finally {
      _publish(() => refreshing = false);
    }
  }

  Future<void> _sharedChanged() async {
    final callback = _onSharedStateChanged;
    if (!_current || callback == null) return;
    try {
      await _operation.step(callback);
    } on AccountOperationDiscarded {
      return;
    } on Object catch (error, stack) {
      if (_current) _report(error, stack, 'counts');
    }
  }

  Future<void> _conversationAction(
    String name,
    Future<void> Function() effect,
  ) async {
    if (!_current || busy || loading || left) return;
    final epoch = _contentEpoch;
    _publish(() => busy = true);
    try {
      await _operation.step(effect);
      final summary = await _operation.step(() => _repository.conversation(id));
      if (epoch != _contentEpoch) return;
      _publish(() {
        conversation = summary;
        failed = false;
      });
      await _sharedChanged();
    } on AccountOperationDiscarded {
      return;
    } on Object catch (error, stack) {
      if (!_current) return;
      _contentFailure(error, stack, name, epoch);
    } finally {
      _publish(() => busy = false);
    }
  }

  Future<void> accept() async {
    if (conversation?.bucket != ChatBucket.request) return;
    await _conversationAction('accept', () async {
      await _repository.accept(id);
    });
  }

  Future<void> setMuted({required bool muted}) =>
      _conversationAction('mute', () async {
        await _repository.setMuted(id, muted: muted);
      });

  Future<bool> leave() async {
    if (!_current || busy || loading || left) return false;
    final epoch = _contentEpoch;
    _publish(() => busy = true);
    try {
      await _operation.step(() => _repository.leave(id));
      _publish(() {
        left = true;
        conversation = null;
        history = const ChatHistory.empty();
      });
      await _sharedChanged();
      return _current && left;
    } on AccountOperationDiscarded {
      return false;
    } on Object catch (error, stack) {
      if (_current) {
        _contentFailure(error, stack, 'leave', epoch);
      }
      return false;
    } finally {
      _publish(() => busy = false);
    }
  }

  void edit(String text) {
    if (!_current || !_draftLoaded) return;
    _publish(() => draft = draft.edit(text));
    _saveTimer?.cancel();
    _saveTimer = Timer(
      const Duration(milliseconds: 250),
      () => unawaited(flush()),
    );
  }

  Future<bool> _persist(ChatDraft snapshot, {bool reportFailure = true}) async {
    try {
      await _store.save(_draftLease, id, snapshot);
      _publish(() => saveFailed = false);
      return true;
    } on ChatDraftLeaseExpired {
      return false;
    } on Object catch (error, stack) {
      if (reportFailure) _report(error, stack, 'draftSave');
      _publish(() => saveFailed = true);
      return false;
    }
  }

  Future<void> flush() async {
    _saveTimer?.cancel();
    if (!_draftLoaded) return;
    await _persist(draft);
  }

  Future<void> discardDraft() async {
    if (!_current || busy || !_draftLoaded) return;
    _publish(() => draft = draft.discard());
    await flush();
  }

  Future<void> checkReceipt() async {
    final attempt = draft.pending;
    if (!_current || busy || attempt == null) return;
    final epoch = _contentEpoch;
    _publish(() => busy = true);
    try {
      await _reconcileAttempt(attempt, epoch);
    } finally {
      _publish(() => busy = false);
    }
  }

  // A receipt and an authorized history read are independent evidence. Neither
  // a missing receipt nor matching text in history proves a send's outcome.
  Future<void> _reconcileAttempt(ChatDraftAttempt attempt, int epoch) async {
    if (!_current) return;
    try {
      final receipt = await _operation.step(
        () => _repository.receipt(attempt.operationId),
      );
      _publish(() => draft = draft.resolve(receipt));
      await _persist(draft);
      if (receipt.state == ChatOperationState.confirmed) {
        await _sharedChanged();
      }
    } on AccountOperationDiscarded {
      return;
    } on Object catch (error, stack) {
      if (_current) _report(error, stack, 'receipt');
    }
    if (!_current || epoch != _contentEpoch || conversation == null) return;
    try {
      final page = await _operation.step(() => _repository.history(id));
      if (epoch == _contentEpoch) _publish(() => _mergeNewest(page));
    } on AccountOperationDiscarded {
      return;
    } on Object catch (error, stack) {
      _contentFailure(error, stack, 'history', epoch);
    }
  }

  Future<void> send() async {
    if (!canSend) return;
    final operationId = _newOperationId();
    _publish(() {
      draft = draft.beginSend(operationId);
      busy = true;
    });
    await _submitAttempt();
  }

  Future<void> sendAgain({required bool duplicateRiskAcknowledged}) async {
    var attempt = draft.pending;
    if (!canSendAgain ||
        attempt == null ||
        (attempt.state == ChatDraftSendState.uncertain &&
            !duplicateRiskAcknowledged)) {
      return;
    }
    if (attempt.state == ChatDraftSendState.uncertain) {
      final epoch = _contentEpoch;
      final originalId = attempt.operationId;
      _publish(() => busy = true);
      try {
        await _reconcileAttempt(attempt, epoch);
      } finally {
        _publish(() => busy = false);
      }
      attempt = draft.pending;
      if (!canSendAgain ||
          attempt == null ||
          attempt.operationId != originalId ||
          epoch != _contentEpoch) {
        return;
      }
    }
    final next = _newOperationId();
    if (next == attempt.operationId) return;
    final retryAttempt = attempt;
    _publish(() {
      draft = retryAttempt.state == ChatDraftSendState.uncertain
          ? draft.repeatUncertain(next)
          : draft.releaseAttempt().beginSend(next);
      busy = true;
    });
    await _submitAttempt(restoreIfNotSubmitted: retryAttempt);
  }

  Future<void> _submitAttempt({ChatDraftAttempt? restoreIfNotSubmitted}) async {
    final epoch = _contentEpoch;
    final attempt = draft.pending!;
    final operationId = attempt.operationId;
    // Recovery identity must be durable before a network effect is possible.
    final saved = await _persist(draft);
    if (!_current) return;
    if (!saved) {
      _publish(() => draft = draft.restoreAttempt(restoreIfNotSubmitted));
      // A queued save of the proposed attempt must not outlive its rollback.
      // This repairs local metadata only; no network attempt follows failure.
      await _persist(draft, reportFailure: false);
      _publish(() {
        saveFailed = true;
        busy = false;
      });
      return;
    }
    try {
      final result = await _operation.step(
        () => _repository.send(id, operationId, attempt.text),
      );
      _publish(() {
        draft = draft.resolve(result.receipt);
        final message = result.message;
        if (epoch == _contentEpoch && message != null) {
          history = history.merge(
            ChatMessagePage(items: [message], cursor: history.cursor),
          );
        }
      });
      if (_current) await _persist(draft);
      if (result.receipt.state == ChatOperationState.confirmed) {
        await _sharedChanged();
      } else if (result.receipt.state == ChatOperationState.uncertain) {
        await _reconcileAttempt(attempt, epoch);
      }
    } on AccountOperationDiscarded {
      return;
    } on Object catch (error, stack) {
      if (!_current) return;
      _contentFailure(error, stack, 'send', epoch);
      _publish(() => draft = draft.markUncertain());
      await _persist(draft);
      await _reconcileAttempt(attempt, epoch);
    } finally {
      _publish(() => busy = false);
    }
  }

  @override
  void dispose() {
    if (_disposed) return;
    _saveTimer?.cancel();
    // Keep local text on navigation. Logout invalidates this storage lease.
    if (_draftLoaded) unawaited(_persist(draft));
    _disposed = true;
    history = const ChatHistory.empty();
    conversation = null;
    draft = const ChatDraft.empty();
    _lastRead = null;
    _pendingPresented = null;
    _historyGaps.clear();
    super.dispose();
  }
}

final class _HistoryGap {
  const new(this.olderIds, this.olderCursor);
  final Set<ChatMessageId> olderIds;
  final ChatCursor? olderCursor;
}

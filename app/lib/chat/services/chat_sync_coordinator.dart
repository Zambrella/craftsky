import 'dart:async';
import 'dart:math';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:flutter/foundation.dart';

typedef ChatRevisionCallback = Future<void> Function(
  ChatView view,
  ChatConversationId? conversationId,
  ChatSyncSnapshot snapshot,
);

/// One disposable foreground heartbeat loop for one messaging generation.
/// No message mutation or send retry is performed by this coordinator.
final class ChatSyncCoordinator extends ChangeNotifier {
  new({
    required this.account,
    required this._repository,
    required this._operation,
    required this._onRevision,
    this._onAccessLost,
    ErrorReporter reporter = const NoopErrorReporter(),
    DateTime Function()? now,
  }) : _reporter = GuardedErrorReporter(reporter),
       _now = now ?? DateTime.now;
  final AccountKey account;
  final ChatRepository _repository;
  final ActiveAccountOperation _operation;
  final ChatRevisionCallback _onRevision;
  final void Function(ApiException, ChatConversationId?)? _onAccessLost;
  final ErrorReporter _reporter;
  final DateTime Function() _now;
  final _random = Random();
  int _failures = 0;
  DateTime? _retryNotBefore;
  ChatSyncSnapshot? snapshot;
  bool failed = false;
  bool busy = false;
  bool _disposed = false;
  bool _active = false;
  bool _restart = false;
  bool _forceRefresh = true;
  int _version = 0;
  ChatView _view = ChatView.none;
  ChatConversationId? _id;
  Timer? _timer;
  final _views =
      <Object, ({ChatView view, ChatConversationId? id, bool active})>{};
  void registerView(
    Object owner,
    ChatView view, {
    required bool active,
    ChatConversationId? conversationId,
  }) {
    if (!_current) return;
    if ((view == ChatView.conversation) != (conversationId != null)) {
      throw ArgumentError('Invalid messaging view');
    }
    _views[owner] = (view: view, id: conversationId, active: active);
    _selectView();
  }

  void unregisterView(Object owner) {
    if (!_current) return;
    _views.remove(owner);
    _selectView();
  }

  void _selectView() {
    final selected = _views.values.where((view) => view.active).lastOrNull;
    if (selected == null) {
      setForeground(active: false);
      setView(ChatView.none);
      return;
    }
    setView(selected.view, conversationId: selected.id);
    setForeground(active: true);
  }

  bool get _current => !_disposed && _operation.isCurrent;
  bool get _canPoll => _current && _active && _view != ChatView.none;
  Duration get _cadence =>
      Duration(seconds: _view == ChatView.conversation ? 4 : 30);

  void setView(ChatView view, {ChatConversationId? conversationId}) {
    if (!_current) return;
    if ((view == ChatView.conversation) != (conversationId != null)) {
      throw ArgumentError('Invalid messaging view');
    }
    if (_view == view && _id == conversationId) return;
    _view = view;
    _id = conversationId;
    ++_version;
    _forceRefresh = true;
    _timer?.cancel();
    _timer = null;
    if (_canPoll) retry();
  }

  void setForeground({required bool active}) {
    if (!_current || _active == active) return;
    _active = active;
    ++_version;
    _timer?.cancel();
    _timer = null;
    _restart = false;
    _forceRefresh = true;
    if (_canPoll) retry();
  }

  void retry() {
    if (!_canPoll) return;
    _forceRefresh = true;
    final notBefore = _retryNotBefore;
    if (notBefore != null && _now().isBefore(notBefore)) {
      if (!busy) {
        _timer ??= Timer(
          notBefore.difference(_now()),
          () => unawaited(_poll()),
        );
      }
      return;
    }
    _timer?.cancel();
    _timer = null;
    if (busy) {
      _restart = true;
      return;
    }
    unawaited(_poll());
  }

  Future<void> _poll() async {
    if (!_canPoll || busy) return;
    busy = true;
    _restart = false;
    final version = _version;
    final view = _view;
    final id = _id;
    final force = _forceRefresh;
    _forceRefresh = false;
    try {
      final next = await _operation.step(
        () => _repository.heartbeat(
          view,
          conversationId: id,
          knownRevision: snapshot?.revision,
        ),
      );
      if (!_canPoll || version != _version) return;
      if (next.state == 'authorizationNeeded' || next.state == 'restricted') {
        _onAccessLost?.call(
          ApiBadRequest(
            next.state == 'restricted'
                ? 'chat_restricted'
                : 'chat_authorization_required',
          ),
          id,
        );
        return;
      }
      if (force || failed || next.revision != snapshot?.revision) {
        await _operation.step(() => _onRevision(view, id, next));
      }
      if (!_canPoll || version != _version) return;
      snapshot = next;
      failed = false;
      _failures = 0;
      _retryNotBefore = null;
      notifyListeners();
    } on AccountOperationDiscarded {
      return;
    } on Object catch (error, stack) {
      if (_current) {
        // Backoff belongs to this account/generation, even if the view changed
        // or backgrounded while its request was pending. Presentation remains
        // fenced to the captured view below.
        _failures++;
        final base = min(60, 1 << min(_failures - 1, 6));
        var delay = Duration(
          milliseconds: max(
            1000,
            min(
              60000,
              (base * 1000 * (0.8 + _random.nextDouble() * 0.4)).round(),
            ),
          ),
        );
        final providerDelay = error is ApiException
            ? error.details.retryAfterSeconds
            : null;
        if (providerDelay != null &&
            providerDelay > 0 &&
            Duration(seconds: providerDelay) > delay) {
          delay = Duration(seconds: providerDelay);
        }
        _retryNotBefore = _now().add(delay);
        if (!_canPoll || version != _version) return;
        failed = true;
        notifyListeners();
        if (error is ApiUnauthorized ||
            (error is ApiBadRequest &&
                const {
                  'chat_disconnected',
                  'chat_authorization_required',
                  'chat_restricted',
                  'chat_connection_changed',
                  'chat_conversation_unavailable',
                }.contains(error.code))) {
          _onAccessLost?.call(error as ApiException, id);
          return;
        }
        unawaited(
          _reporter.captureException(
            error,
            stackTrace: stack,
            context: ReportContext(
              feature: 'Chat',
              operation: 'sync',
              classification: 'chat.failure',
              workflow: PrivateOperationalFailureContext(
                accountDid: account.did.value,
              ),
            ),
          ),
        );
      }
    } finally {
      busy = false;
      if (_canPoll) {
        var delay = _restart ? Duration.zero : _cadence;
        final wait = _retryNotBefore?.difference(_now());
        if (wait != null && wait > delay) delay = wait;
        _timer = Timer(delay, () => unawaited(_poll()));
      }
    }
  }

  @override
  void dispose() {
    if (_disposed) return;
    _disposed = true;
    _timer?.cancel();
    _timer = null;
    _id = null;
    _view = ChatView.none;
    snapshot = null;
    _views.clear();
    super.dispose();
  }
}

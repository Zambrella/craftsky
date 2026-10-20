import 'dart:async';

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/chat/data/chat_repository.dart';
import 'package:craftsky_app/chat/models/chat_identifier.dart';
import 'package:craftsky_app/chat/models/chat_state.dart';
import 'package:craftsky_app/search/data/search_repository.dart';
import 'package:craftsky_app/search/models/profile_search_page.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';
import 'package:flutter/foundation.dart';

final class ChatRecipientController extends ChangeNotifier {
  new({
    required this.account,
    required this._repository,
    required this._searchRepository,
    required this._operation,
    ErrorReporter reporter = const NoopErrorReporter(),
  }) : _reporter = GuardedErrorReporter(reporter);
  final AccountKey account;
  final ChatRepository _repository;
  final SearchRepository _searchRepository;
  final ActiveAccountOperation _operation;
  final ErrorReporter _reporter;
  List<ProfileSearchResult> items = const [];
  Did? recipient;
  ChatAvailability? availability;
  bool searching = false;
  bool checking = false;
  bool starting = false;
  bool failed = false;
  bool _disposed = false;
  int _searchVersion = 0;
  int _selectionVersion = 0;
  bool get _current => !_disposed && _operation.isCurrent;
  bool get canStart =>
      _current &&
      !starting &&
      !checking &&
      recipient != null &&
      availability?.canChat == true;
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

  void beginSearch() {
    if (!_current || starting) return;
    ++_searchVersion;
    ++_selectionVersion;
    _publish(() {
      recipient = null;
      availability = null;
      items = const [];
      checking = false;
      searching = true;
      failed = false;
    });
  }

  Future<void> search(String query) async {
    if (!_current) return;
    final version = ++_searchVersion;
    ++_selectionVersion;
    _publish(() {
      recipient = null;
      availability = null;
      items = const [];
      searching = query.trim().isNotEmpty;
      failed = false;
    });
    if (query.trim().isEmpty) return;
    try {
      final page = await _operation.step(
        () => _searchRepository.searchProfiles(q: query.trim(), limit: 50),
      );
      if (version != _searchVersion) return;
      _publish(() {
        items = List.unmodifiable(
          page.items.where(
            (item) => item.isCraftskyProfile && item.did != account.did,
          ),
        );
      });
    } on AccountOperationDiscarded {
      return;
    } on Object catch (error, stack) {
      if (_current && version == _searchVersion) {
        _report(error, stack, 'recipientSearch');
        _publish(() => failed = true);
      }
    } finally {
      if (version == _searchVersion) _publish(() => searching = false);
    }
  }

  Future<void> select(ProfileSearchResult profile) async {
    if (!profile.isCraftskyProfile ||
        !items.any((item) => item.did == profile.did)) {
      return;
    }
    await selectRecipient(profile.did);
  }

  /// Profile entry may refer to an existing external identity; never creates.
  Future<void> selectRecipient(Did did) async {
    if (!_current || starting || did == account.did) return;
    final version = ++_selectionVersion;
    ++_searchVersion;
    _publish(() {
      searching = false;
      recipient = did;
      availability = null;
      checking = true;
      failed = false;
    });
    try {
      final result = await _operation.step(() => _repository.availability(did));
      if (version != _selectionVersion) return;
      _publish(() => availability = result);
    } on AccountOperationDiscarded {
      return;
    } on Object catch (error, stack) {
      if (_current && version == _selectionVersion) {
        _report(error, stack, 'availability');
        _publish(() => failed = true);
      }
    } finally {
      if (version == _selectionVersion) _publish(() => checking = false);
    }
  }

  Future<ChatConversationId?> start() async {
    final did = recipient;
    if (!canStart || did == null) return null;
    _publish(() => starting = true);
    try {
      final conversation = await _operation.step(() => _repository.start(did));
      return _current ? conversation.id : null;
    } on AccountOperationDiscarded {
      return null;
    } on Object catch (error, stack) {
      if (_current) {
        _report(error, stack, 'start');
        _publish(() => failed = true);
      }
      return null;
    } finally {
      _publish(() => starting = false);
    }
  }

  @override
  void dispose() {
    _disposed = true;
    ++_searchVersion;
    ++_selectionVersion;
    items = const [];
    recipient = null;
    availability = null;
    super.dispose();
  }
}

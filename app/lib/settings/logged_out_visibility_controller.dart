import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:flutter/foundation.dart';

final class VisibilityIntent {
  const new({
    required this.hideFromLoggedOut,
    required this.expectedCid,
    required this.operationKey,
  });
  factory fromMap(Map<String, dynamic> value) => VisibilityIntent(
    hideFromLoggedOut: value['hideFromLoggedOut'] as bool,
    expectedCid: value['expectedCid'] as String,
    operationKey: value['operationKey'] as String,
  );
  final bool hideFromLoggedOut;
  final String expectedCid;
  final String operationKey;
  Map<String, dynamic> get body => {
    'hideFromLoggedOut': hideFromLoggedOut,
    'expectedCid': expectedCid,
  };
}

final class VisibilitySetting {
  const new({
    required this.hideFromLoggedOut,
    required this.expectedCid,
    this.pending,
  });
  factory fromMap(Map<String, dynamic> value) => VisibilitySetting(
    hideFromLoggedOut: value['hideFromLoggedOut'] as bool,
    expectedCid: value['expectedCid'] as String,
    pending: value['pending'] == null
        ? null
        : VisibilityIntent.fromMap(
            Map<String, dynamic>.from(value['pending'] as Map),
          ),
  );
  final bool hideFromLoggedOut;
  final String expectedCid;
  final VisibilityIntent? pending;
}

/// Only a confirmed owner response changes the displayed preference.
final class LoggedOutVisibilityController extends ChangeNotifier {
  new({
    required this.read,
    required this.save,
    String Function()? newKey,
    this.isCurrent,
  }) : newKey = newKey ?? newPdsMutationOperationKey;
  final Future<VisibilitySetting> Function() read;
  final Future<VisibilitySetting> Function(VisibilityIntent) save;
  final String Function() newKey;
  final bool Function()? isCurrent;
  VisibilitySetting? value;
  VisibilityIntent? pending;
  bool busy = false;
  String? error;
  String? notice;
  bool _active = true;
  bool get _current => _active && (isCurrent?.call() ?? true);
  @override
  void dispose() {
    _active = false;
    super.dispose();
  }

  Future<void> load() async {
    if (busy || !_current) return;
    busy = true;
    error = null;
    notifyListeners();
    try {
      final next = await read();
      if (_current) {
        value = next;
        pending = next.pending;
      }
    } on Object {
      if (_current) {
        error =
            'Could not read your shared Bluesky preference. Retry to load it.';
      }
    } finally {
      if (_current) {
        busy = false;
        notifyListeners();
      }
    }
  }

  // Matches the SwitchListTile onChanged callback.
  // ignore: avoid_positional_boolean_parameters
  Future<void> setHidden(bool hidden) async {
    if (busy || pending != null || value == null || !_current) return;
    pending = VisibilityIntent(
      hideFromLoggedOut: hidden,
      expectedCid: value!.expectedCid,
      operationKey: newKey(),
    );
    await retry();
  }

  Future<void> retry() async {
    final intent = pending;
    if (intent == null || busy || !_current) return;
    busy = true;
    error = null;
    notice = null;
    notifyListeners();
    try {
      final next = await save(intent);
      if (_current) {
        value = next;
        pending = next.pending;
        notice = next.pending == null
            ? 'Saved. This preference also applies on Bluesky.'
            : null;
      }
    } on PdsMutationAmbiguousException {
      if (_current) {
        error = 'Your save is still being confirmed. Retry the same save.';
      }
    } on ApiBadRequest catch (failure) {
      if (_current) {
        pending = null;
        value = null;
        error = failure.details.statusCode == 409
            ? 'Your profile changed. Reload before changing this preference.'
            : 'Could not save. Sign in again or reload your preference.';
      }
    } on ApiUnauthorized {
      if (_current) {
        pending = null;
        value = null;
        error = 'Sign in again to manage this preference.';
      }
    } on Object {
      if (_current) error = 'Could not confirm your save. Retry the same save.';
    } finally {
      if (_current) {
        busy = false;
        notifyListeners();
      }
    }
  }
}

typedef NotificationTokenDelete = Future<void> Function();
typedef NotificationBindingRemove = Future<void> Function(String did);

final class NotificationSignOutCleanup {
  NotificationSignOutCleanup({
    required this._deleteProviderToken,
    required this._removeRoutingBinding,
  });

  final NotificationTokenDelete _deleteProviderToken;
  final NotificationBindingRemove _removeRoutingBinding;
  final _inFlight = <String, Future<void>>{};

  Future<void> run({required String did, required bool confirmedLogout}) async {
    final existing = _inFlight[did];
    if (existing != null) {
      await existing;
      return;
    }
    final operation = _run(did, confirmedLogout).whenComplete(() {
      _inFlight.removeWhere((key, _) => key == did);
    });
    _inFlight[did] = operation;
    await operation;
  }

  Future<void> _run(String did, bool confirmedLogout) async {
    if (!confirmedLogout) {
      try {
        await _deleteProviderToken();
      } on Object {
        // Provider cleanup is best effort; never retain the local session.
      }
    }
    try {
      await _removeRoutingBinding(did);
    } on Object {
      // A storage failure must not trap the user in an invalid session.
    }
  }
}

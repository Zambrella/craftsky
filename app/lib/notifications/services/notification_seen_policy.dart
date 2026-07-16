final class NotificationSeenGate {
  final _consumed = <int>{};

  bool consume({required int? token, required bool rendered}) =>
      rendered && token != null && _consumed.add(token);

  void release(int token) => _consumed.remove(token);
}

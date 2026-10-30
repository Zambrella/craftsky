import 'dart:async';

/// Owns retention links, never private payloads or additional network work.
final class ChatSessionCache {
  static const maxIdleConversations = 3;
  static const idleLifetime = Duration(minutes: 2);
  final _entries = <ChatCacheRetention>{};
  final _idle = <ChatCacheRetention>{};
  bool _disposed = false;

  ChatCacheRetention retainList(void Function() release) {
    return _retain(release, conversation: false);
  }

  ChatCacheRetention retainConversation(void Function() release) {
    return _retain(release, conversation: true);
  }

  ChatCacheRetention _retain(
    void Function() release, {
    required bool conversation,
  }) {
    final entry = ChatCacheRetention._(this, release, conversation);
    if (_disposed) {
      entry.close();
    } else {
      _entries.add(entry);
    }
    return entry;
  }

  void dispose() {
    if (_disposed) return;
    _disposed = true;
    for (final entry in _entries.toList()) {
      entry.close();
    }
  }
}

final class ChatCacheRetention {
  new _(this._cache, this._release, this._conversation);
  final ChatSessionCache _cache;
  final void Function() _release;
  final bool _conversation;
  Timer? _timer;
  bool _closed = false;

  void markIdle() {
    if (_closed || !_conversation) return;
    _timer?.cancel();
    _cache._idle
      ..remove(this)
      ..add(this);
    _timer = Timer(ChatSessionCache.idleLifetime, close);
    while (_cache._idle.length > ChatSessionCache.maxIdleConversations) {
      _cache._idle.first.close();
    }
  }

  void markActive() {
    _timer?.cancel();
    _timer = null;
    _cache._idle.remove(this);
  }

  void forget() {
    if (_closed) return;
    _closed = true;
    markActive();
    _cache._entries.remove(this);
  }

  void close() {
    if (_closed) return;
    forget();
    _release();
  }
}

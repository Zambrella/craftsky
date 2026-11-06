import 'package:craftsky_app/auth/models/account_operation_canceled.dart';
import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/shared/browsing/browsing_context.dart';
import 'package:craftsky_app/shared/browsing/browsing_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final class ContentReadDiscarded
    implements Exception, AccountOperationCancellation {
  const new();
}

/// Captures the read audience and lifetime without granting mutation authority.
final class ContentReadGuard {
  new withContext(this.context, this._read);
  factory capture(Ref ref) {
    final context = ref.watch(browsingContextProvider);
    final guard = ContentReadGuard.withContext(
      context,
      () => ref.mounted ? ref.read(browsingContextProvider) : null,
    )..requireCurrent();
    return guard;
  }
  static Future<ContentReadGuard> captureWhenReady(Ref ref) async {
    if (ref.read(browsingContextProvider) is BrowsingLoading) {
      await ref.read(sessionRegistryProvider.future);
      if (!ref.mounted) throw const ContentReadDiscarded();
    }
    return ContentReadGuard.capture(ref);
  }

  final BrowsingContext context;
  ActiveAccountLease? get ownership => switch (context) {
    MemberBrowsingContext(:final lease) => lease,
    _ => null,
  };
  final BrowsingContext? Function() _read;
  bool get isCurrent =>
      context.capabilities.canReadPublic && _read() == context;
  void requireCurrent({Exception discarded = const ContentReadDiscarded()}) {
    if (!isCurrent) throw discarded;
  }

  Future<T> step<T>(
    Future<T> Function() action, {
    Exception discarded = const ContentReadDiscarded(),
  }) async {
    requireCurrent(discarded: discarded);
    final value = await action();
    requireCurrent(discarded: discarded);
    return value;
  }

  bool publish(void Function() action) {
    if (!isCurrent) return false;
    action();
    return true;
  }
}

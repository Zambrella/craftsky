import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/shared/browsing/browsing_context.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final browsingContextProvider =
    NotifierProvider<BrowsingContextNotifier, BrowsingContext>(
      BrowsingContextNotifier.new,
    );

class BrowsingContextNotifier extends Notifier<BrowsingContext> {
  int _epoch = 0;
  @override
  BrowsingContext build() {
    final registry = ref.watch(sessionRegistryProvider);
    final epoch = ++_epoch;
    if (registry.isLoading || !registry.hasValue) {
      return BrowsingLoading(epoch: epoch);
    }
    final lease = registry.requireValue.activeLease;
    return lease == null
        ? GuestBrowsingContext(epoch: epoch)
        : MemberBrowsingContext(lease: lease);
  }
}

final isGuestBrowsingProvider = Provider<bool>(
  (ref) => ref.watch(browsingContextProvider) is GuestBrowsingContext,
);

import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'notification_filter_provider.g.dart';

/// In-memory selection survives navigation, but resets when the owner changes.
@Riverpod(keepAlive: true)
class NotificationFilter extends _$NotificationFilter {
  @override
  NotificationCategory? build() {
    ref.watch(
      sessionRegistryProvider.select(
        (registry) => registry.value?.activeLease?.session.account,
      ),
    );
    return null;
  }

  void select(NotificationCategory? category) {
    if (state == category) return;
    state = category;
  }
}

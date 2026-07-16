import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_preferences.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final notificationPreferencesProvider =
    AsyncNotifierProvider<
      NotificationPreferencesNotifier,
      NotificationPreferences
    >(
      NotificationPreferencesNotifier.new,
    );

class NotificationPreferencesNotifier
    extends AsyncNotifier<NotificationPreferences> {
  final _generations =
      <(NotificationCategory, NotificationPreferenceField), int>{};

  @override
  Future<NotificationPreferences> build() =>
      ref.watch(notificationPreferencesRepositoryProvider).load();

  Future<bool> setPushEnabled(
    NotificationCategory category, {
    required bool value,
  }) => _edit(
    NotificationPreferencePatch.pushEnabled(category, value: value),
  );

  Future<bool> setScope(
    NotificationCategory category, {
    required NotificationPreferenceScope value,
  }) => _edit(NotificationPreferencePatch.scope(category, value: value));

  Future<bool> _edit(NotificationPreferencePatch patch) async {
    final current = state.value;
    final previous = current?.known[patch.category];
    if (current == null || previous == null) return false;

    final key = (patch.category, patch.field);
    final generation = (_generations[key] ?? 0) + 1;
    _generations[key] = generation;
    state = AsyncData(current.replace(patch.category, _apply(previous, patch)));

    try {
      final server = await ref
          .read(notificationPreferencesRepositoryProvider)
          .patch(patch);
      if (_generations[key] != generation || !ref.mounted) return true;
      final now = state.value;
      final serverValue = server.known[patch.category];
      if (now != null && serverValue != null) {
        state = AsyncData(
          now.replace(
            patch.category,
            _mergeField(now.known[patch.category]!, serverValue, patch.field),
          ),
        );
      }
      return true;
    } on Object {
      if (_generations[key] == generation && ref.mounted) {
        final now = state.value;
        if (now != null) {
          state = AsyncData(
            now.replace(
              patch.category,
              _mergeField(now.known[patch.category]!, previous, patch.field),
            ),
          );
        }
      }
      return false;
    }
  }

  NotificationPreference _apply(
    NotificationPreference value,
    NotificationPreferencePatch patch,
  ) => switch (patch.field) {
    NotificationPreferenceField.scope => value.copyWith(
      scope: patch.scopeValue,
    ),
    NotificationPreferenceField.pushEnabled => value.copyWith(
      pushEnabled: patch.pushEnabledValue,
    ),
  };

  NotificationPreference _mergeField(
    NotificationPreference current,
    NotificationPreference source,
    NotificationPreferenceField field,
  ) => switch (field) {
    NotificationPreferenceField.scope => current.copyWith(scope: source.scope),
    NotificationPreferenceField.pushEnabled => current.copyWith(
      pushEnabled: source.pushEnabled,
    ),
  };
}

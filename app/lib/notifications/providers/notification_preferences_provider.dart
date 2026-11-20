import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/notifications/data/notification_repository.dart';
import 'package:craftsky_app/notifications/models/notification_category.dart';
import 'package:craftsky_app/notifications/models/notification_preferences.dart';
import 'package:craftsky_app/notifications/providers/notification_repository_provider.dart';
import 'package:craftsky_app/shared/observability/consumed_failure.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'notification_preferences_provider.g.dart';

NotificationPreference _applyPreference(
  NotificationPreference value,
  NotificationPreferencePatch patch,
) => switch (patch.field) {
  NotificationPreferenceField.scope => value.copyWith(scope: patch.scopeValue),
  NotificationPreferenceField.pushEnabled => value.copyWith(
    pushEnabled: patch.pushEnabledValue,
  ),
};

NotificationPreference _mergePreferenceField(
  NotificationPreference current,
  NotificationPreference source,
  NotificationPreferenceField field,
) => switch (field) {
  NotificationPreferenceField.scope => current.copyWith(scope: source.scope),
  NotificationPreferenceField.pushEnabled => current.copyWith(
    pushEnabled: source.pushEnabled,
  ),
};

@Riverpod(keepAlive: true)
class AccountNotificationPreferencesNotifier
    extends _$AccountNotificationPreferencesNotifier {
  _PreferenceEdits _edits = _PreferenceEdits();

  @override
  Future<NotificationPreferences> build(AccountKey account) async {
    _edits = _PreferenceEdits();
    // This watch owns the account repository for the retained preferences.
    // ignore: riverpod_lint/only_use_keep_alive_inside_keep_alive
    final repository = await ref.watch(
      accountNotificationPreferencesRepositoryProvider(account).future,
    );
    return await repository.load();
  }

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

  Future<bool> _edit(NotificationPreferencePatch patch) {
    final operationRef = ref;
    return _edits.apply(
      patch,
      operationRef: operationRef,
      readState: () => state.value,
      update: (value) => state = AsyncData(value),
      accountDid: account.did.value,
      repository: () =>
          // build's watch owns the account repository throughout the edit.
          // ignore: riverpod_lint/only_use_keep_alive_inside_keep_alive
          operationRef.read(
            accountNotificationPreferencesRepositoryProvider(account).future,
          ),
    );
  }
}

@Riverpod(keepAlive: true, name: 'notificationPreferencesProvider')
class NotificationPreferencesNotifier
    extends _$NotificationPreferencesNotifier {
  _PreferenceEdits _edits = _PreferenceEdits();

  @override
  Future<NotificationPreferences> build() {
    _edits = _PreferenceEdits();
    return ref.watch(notificationPreferencesRepositoryProvider).load();
  }

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

  Future<bool> _edit(NotificationPreferencePatch patch) {
    final operationRef = ref;
    final accountDid = operationRef.exists(sessionRegistryProvider)
        ? operationRef
              .read(sessionRegistryProvider)
              .value
              ?.activeLease
              ?.session
              .account
              .did
              .value
        : null;
    return _edits.apply(
      patch,
      operationRef: operationRef,
      readState: () => state.value,
      update: (value) => state = AsyncData(value),
      accountDid: accountDid,
      repository: () => Future.value(
        operationRef.read(notificationPreferencesRepositoryProvider),
      ),
    );
  }
}

final class _PreferenceEdits {
  final _fields =
      <(NotificationCategory, NotificationPreferenceField), _PendingField>{};

  Future<bool> apply(
    NotificationPreferencePatch patch, {
    required Ref operationRef,
    required NotificationPreferences? Function() readState,
    required void Function(NotificationPreferences) update,
    required String? accountDid,
    required Future<NotificationPreferencesRepository> Function() repository,
  }) async {
    final current = readState();
    final previous = current?.known[patch.category];
    if (current == null || previous == null) return false;
    final reporter = operationRef.read(errorReporterProvider);
    final key = (patch.category, patch.field);
    final field = _fields.putIfAbsent(key, () => _PendingField(previous));
    final earlier = field.pending;
    final revision = ++field.revision;
    update(current.replace(patch.category, _applyPreference(previous, patch)));

    Future<bool> save() async {
      if (earlier != null) await earlier;
      if (!operationRef.mounted) return false;
      var dispatched = false;
      try {
        final target = await repository();
        if (!operationRef.mounted) return false;
        dispatched = true;
        final server = await target.patch(patch);
        field.saved = server.known[patch.category] ?? field.saved;
        return true;
      } on Object catch (error, stack) {
        if (dispatched && operationRef.mounted && revision == field.revision) {
          reportConsumedFailure(
            error,
            stack,
            reporter: reporter,
            feature: 'Notifications',
            operation: 'notifications.preferences_patch',
            accountDid: accountDid,
          );
        }
        return false;
      }
    }

    final saving = save();
    field.pending = saving;
    final saved = await saving;
    if (!operationRef.mounted || revision != field.revision) return saved;
    _fields.remove(key);
    final now = readState();
    final nowValue = now?.known[patch.category];
    if (now != null && nowValue != null) {
      update(
        now.replace(
          patch.category,
          _mergePreferenceField(nowValue, field.saved, patch.field),
        ),
      );
    }
    return saved;
  }
}

final class _PendingField {
  new(this.saved);

  NotificationPreference saved;
  Future<bool>? pending;
  int revision = 0;
}

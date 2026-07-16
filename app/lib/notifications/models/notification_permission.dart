enum NotificationPermission { notDetermined, authorized, denied }

enum NotificationPermissionAction { none, request, register }

abstract final class NotificationPermissionPolicy {
  static NotificationPermissionAction actionFor({
    required bool signedIn,
    required bool onboarded,
    required NotificationPermission permission,
  }) {
    if (!signedIn || !onboarded) return NotificationPermissionAction.none;
    return switch (permission) {
      NotificationPermission.notDetermined =>
        NotificationPermissionAction.request,
      NotificationPermission.authorized =>
        NotificationPermissionAction.register,
      NotificationPermission.denied => NotificationPermissionAction.none,
    };
  }
}

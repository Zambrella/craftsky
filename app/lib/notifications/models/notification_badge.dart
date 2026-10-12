final class NotificationBadge {
  const new _({
    required this.count,
    required this.visible,
    required this.label,
  });

  factory fromCount(int count) {
    final safeCount = count < 0 ? 0 : count;
    return NotificationBadge._(
      count: safeCount,
      visible: safeCount > 0,
      label: safeCount > 99 ? '99+' : '$safeCount',
    );
  }

  final int count;
  final bool visible;
  final String label;
}

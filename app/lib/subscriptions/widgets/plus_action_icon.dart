import 'package:craftsky_app/subscriptions/subscription_build_config.dart';
import 'package:craftsky_app/theme/plus_tier_icon.dart';
import 'package:material_ui/material_ui.dart';

/// Keeps an action recognizable while marking its icon as Plus-only.
class PlusActionIcon extends StatelessWidget {
  const new({required this.icon, this.size = 24, super.key});

  final IconData icon;
  final double size;

  @override
  Widget build(BuildContext context) {
    if (!subscriptionsEnabled) {
      return Icon(
        icon,
        size: size,
        color: Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.72),
      );
    }
    return SizedBox.square(
      dimension: size + 6,
      child: Stack(
        clipBehavior: Clip.none,
        children: [
          PositionedDirectional(
            start: 0,
            top: 0,
            child: Icon(icon, size: size),
          ),
          PositionedDirectional(
            end: 0,
            bottom: 0,
            child: PlusTierIcon(
              size: 12,
              color: Theme.of(context).colorScheme.primary,
            ),
          ),
        ],
      ),
    );
  }
}

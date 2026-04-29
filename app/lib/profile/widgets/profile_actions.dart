import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// The action-row variants drawn under the avatar/identity block.
///
/// Modelled as a sealed class so the page just hands a [ProfileActionSet]
/// to [ProfileActions] — switching a visitor profile from `Follow`/`Share`
/// to its own user's `Edit`/`Settings` is a value swap, not a widget
/// swap. New variants (e.g. blocked-user, suspended) slot in here.
sealed class ProfileActionSet {
  const ProfileActionSet();
}

final class SelfProfileActionSet extends ProfileActionSet {
  const SelfProfileActionSet({required this.onEdit, required this.onSettings});

  final VoidCallback onEdit;
  final VoidCallback onSettings;
}

final class VisitorProfileActionSet extends ProfileActionSet {
  const VisitorProfileActionSet({
    required this.isFollowing,
    required this.onFollowToggle,
    required this.onShare,
  });

  final bool isFollowing;
  final VoidCallback onFollowToggle;
  final VoidCallback onShare;
}

class ProfileActions extends StatelessWidget {
  const ProfileActions({required this.actions, super.key});

  final ProfileActionSet actions;

  @override
  Widget build(BuildContext context) {
    return switch (actions) {
      final SelfProfileActionSet a => _SelfActions(actions: a),
      final VisitorProfileActionSet a => _VisitorActions(actions: a),
    };
  }
}

class _SelfActions extends StatelessWidget {
  const _SelfActions({required this.actions});

  final SelfProfileActionSet actions;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final spacing = theme.extension<SpacingTheme>()!;
    return Row(
      // Hug content rather than stretching across the action lane —
      // Edit and Settings are secondary on your own profile, so the
      // group reads better at intrinsic size, anchored to the right
      // by the surrounding Align.
      mainAxisSize: MainAxisSize.min,
      children: [
        ChunkyButton(
          onPressed: actions.onEdit,
          backgroundColor: swatches.paper3,
          foregroundColor: theme.colorScheme.onSurface,
          child: Row(
            mainAxisAlignment: MainAxisAlignment.center,
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(Icons.edit_outlined),
              SizedBox(width: spacing.sp2),
              const Text('Edit profile'),
            ],
          ),
        ),
        SizedBox(width: spacing.sp3),
        _ChunkyIconButton(
          onPressed: actions.onSettings,
          icon: Icons.settings_outlined,
          tooltip: 'Settings',
        ),
      ],
    );
  }
}

class _VisitorActions extends StatelessWidget {
  const _VisitorActions({required this.actions});

  final VisitorProfileActionSet actions;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final spacing = theme.extension<SpacingTheme>()!;
    return Row(
      children: [
        Expanded(
          child: ChunkyButton(
            onPressed: actions.onFollowToggle,
            backgroundColor: actions.isFollowing ? swatches.paper3 : null,
            foregroundColor: actions.isFollowing
                ? theme.colorScheme.onSurface
                : null,
            child: Text(actions.isFollowing ? 'Following' : 'Follow'),
          ),
        ),
        SizedBox(width: spacing.sp3),
        _ChunkyIconButton(
          onPressed: actions.onShare,
          icon: Icons.ios_share_outlined,
          tooltip: 'Share',
        ),
      ],
    );
  }
}

/// Compact circular [ChunkyButton] that carries only an icon. Lets the
/// row pair a prominent text-bearing primary (Edit profile / Follow)
/// with one or more secondary actions that read as buttons but stay
/// visually quiet — same chunky shadow + ink border, just smaller.
class _ChunkyIconButton extends StatelessWidget {
  const _ChunkyIconButton({
    required this.onPressed,
    required this.icon,
    this.tooltip,
  });

  final VoidCallback onPressed;
  final IconData icon;
  final String? tooltip;

  static const double _size = 44;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final button = SizedBox(
      width: _size,
      height: _size,
      child: ChunkyButton(
        onPressed: onPressed,
        backgroundColor: swatches.paper3,
        foregroundColor: theme.colorScheme.onSurface,
        // Strip ChunkyButton's default text-button padding so the icon
        // can centre in the 44×44 hit target without forcing the
        // stadium shape into a long pill.
        style: const ButtonStyle(
          padding: WidgetStatePropertyAll(EdgeInsets.zero),
          minimumSize: WidgetStatePropertyAll(Size(_size, _size)),
          fixedSize: WidgetStatePropertyAll(Size(_size, _size)),
        ),
        child: Icon(icon),
      ),
    );
    final label = tooltip;
    if (label == null) return button;
    return Tooltip(message: label, child: button);
  }
}

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/chunky_icon_button.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// The action-row variants drawn under the avatar/identity block.
///
/// Modelled as a sealed class so the page just hands a [ProfileActionSet]
/// to [ProfileActions] — switching a visitor profile from `Follow`/`Mute`
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
    required this.isBusy,
    required this.onFollowToggle,
    required this.onShare,
    required this.onReport,
    required this.onMuteToggle,
    required this.onBlockToggle,
    this.isMuted = false,
    this.isBlocking = false,
    this.canFollow = true,
    this.canToggleMute = true,
  });

  final bool isFollowing;
  final bool isBusy;
  final VoidCallback onFollowToggle;
  final VoidCallback onShare;
  final VoidCallback onReport;
  final VoidCallback onMuteToggle;
  final VoidCallback onBlockToggle;
  final bool isMuted;
  final bool isBlocking;
  final bool canFollow;
  final bool canToggleMute;
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
    final l10n = AppLocalizations.of(context);
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
              Text(l10n.profileEditAction),
            ],
          ),
        ),
        SizedBox(width: spacing.sp3),
        ChunkyIconButton(
          onPressed: actions.onSettings,
          icon: Icons.settings_outlined,
          tooltip: l10n.profileSettingsAction,
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
    final l10n = AppLocalizations.of(context);
    final blockLabel = actions.isBlocking
        ? l10n.profileUnblockAction
        : l10n.profileBlockAction;
    final muteLabel = actions.isMuted
        ? l10n.profileUnmuteAction
        : l10n.profileMuteAction;
    return Row(
      children: [
        if (actions.canFollow) ...[
          Expanded(
            child: ChunkyButton(
              onPressed: actions.isBusy ? null : actions.onFollowToggle,
              backgroundColor: actions.isFollowing ? swatches.paper3 : null,
              foregroundColor: actions.isFollowing
                  ? theme.colorScheme.onSurface
                  : null,
              child: Text(
                actions.isFollowing
                    ? l10n.profileFollowingAction
                    : l10n.profileFollowAction,
              ),
            ),
          ),
          SizedBox(width: spacing.sp3),
        ],
        if (actions.canToggleMute) ...[
          ChunkyIconButton(
            onPressed: actions.isBusy ? null : actions.onMuteToggle,
            icon: actions.isMuted
                ? Icons.volume_up_outlined
                : Icons.volume_off_outlined,
            tooltip: muteLabel,
          ),
          SizedBox(width: spacing.sp3),
        ],
        CraftskyContextMenuButton(
          tooltip: l10n.profileMoreActions,
          enabled: !actions.isBusy,
          groups: [
            CraftskyContextMenuGroup(
              items: [
                CraftskyContextMenuItem(
                  text: l10n.profileShareAction,
                  icon: Icons.ios_share_outlined,
                  onPressed: actions.onShare,
                ),
              ],
            ),
            CraftskyContextMenuGroup(
              items: [
                CraftskyContextMenuItem(
                  text: blockLabel,
                  icon: Icons.block_outlined,
                  onPressed: actions.onBlockToggle,
                  style: CraftskyContextMenuItemStyle.destructive,
                  semanticHint: actions.isBlocking
                      ? null
                      : l10n.destructiveActionHint,
                ),
                CraftskyContextMenuItem(
                  text: l10n.profileReportAction,
                  icon: Icons.flag_outlined,
                  onPressed: actions.onReport,
                ),
              ],
            ),
          ],
        ),
      ],
    );
  }
}

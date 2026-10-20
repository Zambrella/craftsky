import 'dart:async';

import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/widgets/action_progress.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/chunky_icon_button.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:material_ui/material_ui.dart';

const profileActionSectionMaxWidth = 420.0;

/// The action-row variants drawn under the avatar/identity block.
///
/// Modelled as a sealed class so the page just hands a [ProfileActionSet]
/// to [ProfileActions] — switching a visitor profile from `Follow`/`Mute`
/// to its own user's `Edit`/`Settings` is a value swap, not a widget
/// swap. New variants (e.g. blocked-user, suspended) slot in here.
sealed class ProfileActionSet {
  const new();
}

final class SelfProfileActionSet extends ProfileActionSet {
  const new({
    required this.onEdit,
    required this.onSettings,
    this.onCopyLink,
  });

  final FutureOr<void> Function()? onCopyLink;

  final VoidCallback onEdit;
  final VoidCallback onSettings;
}

final class VisitorProfileActionSet extends ProfileActionSet {
  const new({
    required this.isFollowing,
    required this.isBusy,
    required this.onFollowToggle,
    required this.onShare,
    required this.onReport,
    required this.onMuteToggle,
    required this.onBlockToggle,
    this.onCopyLink,
    this.isFollowingBusy = false,
    this.isBlockingBusy = false,
    this.isMuted = false,
    this.isBlocking = false,
    this.canFollow = true,
    this.canToggleMute = true,
    this.messagingAction,
  });

  final bool isFollowing;
  final bool isBusy;
  final bool isFollowingBusy;
  final bool isBlockingBusy;
  final VoidCallback onFollowToggle;
  final VoidCallback onShare;
  final FutureOr<void> Function()? onCopyLink;
  final VoidCallback onReport;
  final VoidCallback onMuteToggle;
  final VoidCallback onBlockToggle;
  final bool isMuted;
  final bool isBlocking;
  final bool canFollow;
  final bool canToggleMute;
  final Widget? messagingAction;
}

class ProfileActions extends StatelessWidget {
  const new({required this.actions, super.key});

  final ProfileActionSet actions;

  @override
  Widget build(BuildContext context) {
    return switch (actions) {
      final SelfProfileActionSet a => _SelfActions(actions: a),
      final VisitorProfileActionSet a => _VisitorActions(actions: a),
    };
  }
}

/// Centres the richer full-page action row while keeping visitor actions
/// bounded and responsive.
class ProfileActionSection extends StatelessWidget {
  const new({required this.actions, super.key});

  final ProfileActionSet actions;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(
          maxWidth: profileActionSectionMaxWidth,
        ),
        child: Align(
          child: ProfileActions(actions: actions),
        ),
      ),
    );
  }
}

class _SelfActions extends StatelessWidget {
  const new({required this.actions});

  final SelfProfileActionSet actions;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final l10n = AppLocalizations.of(context);
    return Wrap(
      alignment: WrapAlignment.end,
      crossAxisAlignment: WrapCrossAlignment.center,
      spacing: spacing.sp3,
      runSpacing: spacing.sp3,
      children: [
        ChunkyButton(
          onPressed: actions.onEdit,
          variant: ChunkyButtonVariant.secondary,
          child: Wrap(
            alignment: WrapAlignment.center,
            crossAxisAlignment: WrapCrossAlignment.center,
            spacing: spacing.sp2,
            children: [
              const Icon(CraftskyIconsBold.edit),
              Text(l10n.profileEditAction),
            ],
          ),
        ),
        ChunkyIconButton(
          onPressed: actions.onSettings,
          icon: CraftskyIconsBold.settings,
          tooltip: l10n.profileSettingsAction,
        ),
        if (actions.onCopyLink != null) ...[
          CraftskyContextMenuButton(
            tooltip: l10n.profileMoreActions,
            groups: [
              CraftskyContextMenuGroup(
                items: [
                  CraftskyContextMenuItem(
                    text: l10n.recordCopyLinkAction,
                    icon: CraftskyIcons.link,
                    onPressed: actions.onCopyLink,
                  ),
                ],
              ),
            ],
          ),
        ],
      ],
    );
  }
}

class _VisitorActions extends StatelessWidget {
  const new({required this.actions});

  final VisitorProfileActionSet actions;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
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
              variant: actions.isFollowing
                  ? ChunkyButtonVariant.secondary
                  : ChunkyButtonVariant.primary,
              child: ActionProgress(
                isLoading: actions.isFollowingBusy,
                child: Text(
                  actions.isFollowing
                      ? l10n.profileFollowingAction
                      : l10n.profileFollowAction,
                ),
              ),
            ),
          ),
          SizedBox(width: spacing.sp3),
        ],
        if (actions.messagingAction case final action?) ...[
          action,
          SizedBox(width: spacing.sp3),
        ],
        if (actions.canToggleMute) ...[
          ChunkyIconButton(
            onPressed: actions.isBusy ? null : actions.onMuteToggle,
            icon: actions.isMuted
                ? CraftskyIconsBold.unmuted
                : CraftskyIconsBold.muted,
            tooltip: muteLabel,
          ),
          SizedBox(width: spacing.sp3),
        ],
        CraftskyContextMenuButton(
          tooltip: l10n.profileMoreActions,
          enabled: !actions.isBusy,
          isLoading: actions.isBlockingBusy,
          groups: [
            CraftskyContextMenuGroup(
              items: [
                if (actions.onCopyLink != null)
                  CraftskyContextMenuItem(
                    text: l10n.recordCopyLinkAction,
                    icon: CraftskyIcons.link,
                    onPressed: actions.onCopyLink,
                  ),
                CraftskyContextMenuItem(
                  text: l10n.profileShareAction,
                  icon: CraftskyIconsBold.share,
                  onPressed: actions.onShare,
                ),
              ],
            ),
            CraftskyContextMenuGroup(
              items: [
                CraftskyContextMenuItem(
                  text: blockLabel,
                  icon: CraftskyIconsBold.block,
                  onPressed: actions.onBlockToggle,
                  style: CraftskyContextMenuItemStyle.destructive,
                  semanticHint: actions.isBlocking
                      ? null
                      : l10n.destructiveActionHint,
                ),
                CraftskyContextMenuItem(
                  text: l10n.profileReportAction,
                  icon: CraftskyIconsBold.report,
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

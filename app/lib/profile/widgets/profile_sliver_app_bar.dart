import 'dart:math' as math;

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/widgets/profile_actions.dart';
import 'package:craftsky_app/profile/widgets/profile_craft_chips.dart';
import 'package:craftsky_app/profile/widgets/profile_framed_avatar.dart';
import 'package:craftsky_app/profile/widgets/profile_header_background.dart';
import 'package:craftsky_app/profile/widgets/profile_identity.dart';
import 'package:craftsky_app/router/app_shell_drawer.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

/// Collapsing profile header styled like the compact profile card.
///
/// The expanded state shows the themed illustration band, overlapping avatar,
/// and centred identity. It collapses to the same compact identity and trailing
/// action used by the rest of the app.
class ProfileSliverAppBar extends StatelessWidget {
  const ProfileSliverAppBar({
    required this.handle,
    required this.actions,
    this.crafts = const [],
    this.displayName,
    this.avatarUrl,
    this.customisation = ProfileCustomisation.defaults,
    this.isBusiness = false,
    this.onAvatarTap,
    super.key,
  });

  final String handle;
  final ProfileActionSet actions;
  final List<String> crafts;
  final String? displayName;
  final String? avatarUrl;
  final ProfileCustomisation customisation;
  final bool isBusiness;
  final VoidCallback? onAvatarTap;

  static const double backgroundHeight = 128;
  static const double avatarTop = 66;
  static const double identityTop = 200;
  static const double expandedHeight = 268;
  static const double minimumIdentityHeight = 60;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final layout = _resolveLayout(context);
    final hasDrawer = AppShellDrawerScope.maybeOf(context) != null;
    final hasBack = ModalRoute.of(context)?.impliesAppBarDismissal ?? false;
    return SliverAppBar(
      leading: hasDrawer || hasBack
          ? _ProfileLeadingAction(
              showDrawer: hasDrawer,
              expandedHeight: layout.expandedHeight,
            )
          : null,
      pinned: true,
      expandedHeight: layout.expandedHeight,
      backgroundColor: swatches.paper,
      foregroundColor: theme.colorScheme.onSurface,
      surfaceTintColor: Colors.transparent,
      shape: const Border(),
      flexibleSpace: _ProfileFlexibleSpace(
        handle: handle,
        crafts: crafts,
        displayName: displayName,
        avatarUrl: avatarUrl,
        customisation: customisation,
        isBusiness: isBusiness,
        actions: actions,
        onAvatarTap: onAvatarTap,
        expandedHeight: layout.expandedHeight,
        identityHeight: layout.identityHeight,
        craftsTop: layout.craftsTop,
      ),
    );
  }

  _ProfileHeaderLayout _resolveLayout(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final availableWidth = math.max<double>(
      1,
      MediaQuery.sizeOf(context).width - (spacing.sp4 * 2),
    );
    final textScaler = MediaQuery.textScalerOf(context);
    final direction = Directionality.of(context);
    final name = (displayName?.isNotEmpty ?? false) ? displayName! : '@$handle';
    final nameHeight = _measureTextHeight(
      text: name,
      style: theme.textTheme.headlineMedium,
      textScaler: textScaler,
      direction: direction,
      maxWidth: availableWidth,
    );
    final handleHeight = (displayName?.isNotEmpty ?? false)
        ? _measureTextHeight(
            text: '@$handle',
            style: theme.textTheme.bodyMedium,
            textScaler: textScaler,
            direction: direction,
            maxWidth: availableWidth,
          )
        : 0.0;
    final businessLabelHeight = isBusiness
        ? _measureTextHeight(
            text: AppLocalizations.of(context).businessProfileLabel,
            style: theme.textTheme.labelLarge,
            textScaler: textScaler,
            direction: direction,
            maxWidth: availableWidth,
          )
        : 0.0;
    final measuredIdentityHeight =
        nameHeight +
        (businessLabelHeight == 0 ? 0 : 2 + businessLabelHeight) +
        (handleHeight == 0 ? 0 : 2 + handleHeight);
    final identityHeight = math.max(
      minimumIdentityHeight,
      measuredIdentityHeight,
    );
    final craftsTop = identityTop + identityHeight + spacing.sp2;

    if (crafts.isEmpty) {
      return _ProfileHeaderLayout(
        identityHeight: identityHeight,
        craftsTop: craftsTop,
        expandedHeight: craftsTop,
      );
    }

    var rowCount = 1;
    var rowWidth = 0.0;
    var chipHeight = 0.0;

    for (final craft in crafts) {
      final label = craft.isEmpty
          ? craft
          : craft[0].toUpperCase() + craft.substring(1);
      final painter = TextPainter(
        text: TextSpan(text: label, style: theme.textTheme.labelMedium),
        textDirection: direction,
        textScaler: textScaler,
        maxLines: 1,
      )..layout();
      final measuredWidth = painter.width + (spacing.sp3 * 2);
      final width = math.min(measuredWidth, availableWidth);
      chipHeight = math.max(chipHeight, painter.height + 12);

      if (rowWidth > 0 && rowWidth + spacing.sp2 + width > availableWidth) {
        rowCount++;
        rowWidth = width;
      } else {
        rowWidth += (rowWidth == 0 ? 0 : spacing.sp2) + width;
      }
    }

    return _ProfileHeaderLayout(
      identityHeight: identityHeight,
      craftsTop: craftsTop,
      expandedHeight:
          craftsTop +
          (rowCount * chipHeight) +
          ((rowCount - 1) * spacing.sp2) +
          spacing.sp3,
    );
  }

  double _measureTextHeight({
    required String text,
    required TextStyle? style,
    required TextScaler textScaler,
    required TextDirection direction,
    required double maxWidth,
  }) {
    final painter = TextPainter(
      text: TextSpan(text: text, style: style),
      textDirection: direction,
      textScaler: textScaler,
    )..layout(maxWidth: maxWidth);
    return painter.height;
  }
}

class _ProfileLeadingAction extends StatelessWidget {
  const _ProfileLeadingAction({
    required this.showDrawer,
    required this.expandedHeight,
  });

  final bool showDrawer;
  final double expandedHeight;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final settings = context
        .dependOnInheritedWidgetOfExactType<FlexibleSpaceBarSettings>();
    final topPadding = MediaQuery.paddingOf(context).top;
    final maxExtent = settings?.maxExtent ?? expandedHeight;
    final minExtent = settings?.minExtent ?? (kToolbarHeight + topPadding);
    final currentExtent = settings?.currentExtent ?? maxExtent;
    final range = (maxExtent - minExtent).abs();
    final collapsed = range == 0
        ? 0.0
        : ((maxExtent - currentExtent) / range).clamp(0.0, 1.0);
    final backgroundColor = collapsed >= 1
        ? Colors.transparent
        : swatches.paper3.withValues(alpha: 1 - collapsed);
    final style = ButtonStyle(
      backgroundColor: WidgetStatePropertyAll(backgroundColor),
      foregroundColor: WidgetStatePropertyAll(theme.colorScheme.onSurface),
    );

    if (showDrawer) {
      return AppShellDrawerButton(style: style);
    }
    return BackButton(style: style);
  }
}

class _ProfileHeaderLayout {
  const _ProfileHeaderLayout({
    required this.identityHeight,
    required this.craftsTop,
    required this.expandedHeight,
  });

  final double identityHeight;
  final double craftsTop;
  final double expandedHeight;
}

class _ProfileFlexibleSpace extends StatelessWidget {
  const _ProfileFlexibleSpace({
    required this.handle,
    required this.crafts,
    required this.displayName,
    required this.avatarUrl,
    required this.customisation,
    required this.isBusiness,
    required this.actions,
    required this.onAvatarTap,
    required this.expandedHeight,
    required this.identityHeight,
    required this.craftsTop,
  });

  final String handle;
  final List<String> crafts;
  final String? displayName;
  final String? avatarUrl;
  final ProfileCustomisation customisation;
  final bool isBusiness;
  final ProfileActionSet actions;
  final VoidCallback? onAvatarTap;
  final double expandedHeight;
  final double identityHeight;
  final double craftsTop;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final spacing = theme.extension<SpacingTheme>()!;
    final settings = context
        .dependOnInheritedWidgetOfExactType<FlexibleSpaceBarSettings>();
    final topPadding = MediaQuery.paddingOf(context).top;
    final maxExtent = settings?.maxExtent ?? expandedHeight;
    final minExtent = settings?.minExtent ?? (kToolbarHeight + topPadding);
    final currentExtent = settings?.currentExtent ?? maxExtent;
    final range = (maxExtent - minExtent).abs();
    final collapsed = range == 0
        ? 0.0
        : ((maxExtent - currentExtent) / range).clamp(0.0, 1.0);
    final seed = (displayName?.isNotEmpty ?? false) ? displayName! : handle;
    final avatar = ProfileFramedAvatar(
      seed: seed,
      avatarUrl: avatarUrl,
      customisation: customisation,
    );

    return Stack(
      fit: StackFit.expand,
      children: [
        ColoredBox(color: swatches.paper),
        IgnorePointer(
          ignoring: collapsed > 0.5,
          child: Opacity(
            opacity: 1 - collapsed,
            child: Stack(
              fit: StackFit.expand,
              children: [
                Positioned(
                  top: 0,
                  left: 0,
                  right: 0,
                  height: topPadding + ProfileSliverAppBar.backgroundHeight,
                  child: ProfileHeaderBackground(
                    customisation: customisation,
                  ),
                ),
                Positioned(
                  top: topPadding + ProfileSliverAppBar.avatarTop,
                  left: 0,
                  right: 0,
                  child: Center(
                    child: _tappableAvatar(
                      avatarUrl: avatarUrl,
                      onTap: onAvatarTap,
                      child: avatar,
                    ),
                  ),
                ),
                Positioned(
                  top: topPadding + ProfileSliverAppBar.identityTop,
                  left: spacing.sp4,
                  right: spacing.sp4,
                  height: identityHeight,
                  child: Center(
                    child: ProfileIdentity(
                      handle: handle,
                      displayName: displayName,
                      businessLabel: isBusiness
                          ? AppLocalizations.of(context).businessProfileLabel
                          : null,
                      centered: true,
                    ),
                  ),
                ),
                if (crafts.isNotEmpty)
                  Positioned(
                    top: topPadding + craftsTop,
                    left: spacing.sp4,
                    right: spacing.sp4,
                    child: ProfileCraftChips(
                      crafts: crafts,
                      alignment: WrapAlignment.center,
                    ),
                  ),
              ],
            ),
          ),
        ),
        Positioned(
          key: const Key('profile-sliver-collapsed-title'),
          left: 56,
          right: 56,
          top: topPadding,
          height: kToolbarHeight,
          child: Opacity(
            opacity: collapsed,
            child: _CollapsedTitle(
              handle: handle,
              displayName: displayName,
            ),
          ),
        ),
        Positioned(
          right: 4,
          top: topPadding,
          height: kToolbarHeight,
          width: 48,
          child: IgnorePointer(
            ignoring: collapsed < 0.5,
            child: Opacity(
              opacity: collapsed,
              child: _CollapsedTrailingAction(actions: actions),
            ),
          ),
        ),
        Positioned(
          left: 0,
          right: 0,
          bottom: 0,
          child: Opacity(
            key: const Key('profile-sliver-divider'),
            opacity: collapsed,
            child: ColoredBox(
              color: theme.colorScheme.onSurface,
              child: const SizedBox(height: 1.5),
            ),
          ),
        ),
      ],
    );
  }

  Widget _tappableAvatar({
    required String? avatarUrl,
    required VoidCallback? onTap,
    required Widget child,
  }) {
    if (avatarUrl == null || onTap == null) return child;
    return GestureDetector(
      key: const Key('profile-avatar-viewer-target'),
      behavior: HitTestBehavior.opaque,
      onTap: onTap,
      child: child,
    );
  }
}

class _CollapsedTrailingAction extends StatelessWidget {
  const _CollapsedTrailingAction({required this.actions});

  final ProfileActionSet actions;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final style = ButtonStyle(
      backgroundColor: const WidgetStatePropertyAll(Colors.transparent),
      foregroundColor: WidgetStatePropertyAll(
        Theme.of(context).colorScheme.onSurface,
      ),
    );
    return switch (actions) {
      SelfProfileActionSet(:final onSettings) => IconButton(
        tooltip: l10n.profileSettingsAction,
        icon: const Icon(CraftskyIconsBold.settings),
        onPressed: onSettings,
        style: style,
      ),
      VisitorProfileActionSet(
        :final isMuted,
        :final isBusy,
        :final canToggleMute,
        :final onMuteToggle,
      ) =>
        canToggleMute
            ? IconButton(
                tooltip: isMuted
                    ? l10n.profileUnmuteAction
                    : l10n.profileMuteAction,
                icon: Icon(
                  isMuted ? CraftskyIconsBold.unmuted : CraftskyIconsBold.muted,
                ),
                onPressed: isBusy ? null : onMuteToggle,
                style: style,
              )
            : const SizedBox.shrink(),
    };
  }
}

class _CollapsedTitle extends StatelessWidget {
  const _CollapsedTitle({
    required this.handle,
    required this.displayName,
  });

  final String handle;
  final String? displayName;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final name = (displayName?.isNotEmpty ?? false) ? displayName! : '@$handle';
    final showSubtitle = displayName?.isNotEmpty ?? false;

    return Align(
      alignment: Alignment.centerLeft,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Text(
            name,
            style: theme.textTheme.titleMedium,
            overflow: TextOverflow.ellipsis,
          ),
          if (showSubtitle)
            Text(
              '@$handle',
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
              overflow: TextOverflow.ellipsis,
            ),
        ],
      ),
    );
  }
}

import 'dart:async';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/subscription_build_config.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// A discoverable Plus control. The server remains the access authority.
class PlusFeatureLock extends StatelessWidget {
  const PlusFeatureLock({
    required this.feature,
    required this.access,
    required this.onUnlocked,
    required this.child,
    this.onLearnMore,
    this.onRetry,
    this.showPlusBadge = true,
    super.key,
  });

  final String feature;
  final AsyncValue<SubscriptionAccess>? access;
  final VoidCallback onUnlocked;
  final Widget child;
  final VoidCallback? onLearnMore;
  final VoidCallback? onRetry;

  /// Disable when the child already badges its own action icon.
  final bool showPlusBadge;

  @override
  Widget build(BuildContext context) {
    if (access case AsyncData(
      :final value,
      isLoading: false,
      hasError: false,
    ) when subscriptionsEnabled && value.allowsPlus) {
      return InkWell(onTap: onUnlocked, child: child);
    }
    final l10n = AppLocalizations.of(context);
    final free = switch (access) {
      AsyncData(isLoading: false, hasError: false) => true,
      _ => false,
    };
    final label = !subscriptionsEnabled
        ? l10n.plusFeatureComingSoon(feature)
        : free
        ? l10n.plusFeatureRequired(feature)
        : l10n.plusFeatureAccessUnavailable(feature);
    return Semantics(
      button: true,
      label: label,
      excludeSemantics: true,
      child: InkWell(
        onTap: () => unawaited(
          showPlusFeaturePrompt(
            context,
            feature,
            free: free,
            onLearnMore: onLearnMore,
            onRetry: onRetry,
          ),
        ),
        child: !subscriptionsEnabled
            ? Opacity(opacity: 0.72, child: IgnorePointer(child: child))
            : showPlusBadge
            ? Stack(
                children: [
                  IgnorePointer(child: child),
                  PositionedDirectional(
                    end: 0,
                    top: 0,
                    child: Icon(
                      CraftskyIcons.plusTier,
                      size: 16,
                      color: Theme.of(context).colorScheme.primary,
                    ),
                  ),
                ],
              )
            : IgnorePointer(child: child),
      ),
    );
  }
}

Future<void> showPlusFeaturePrompt(
  BuildContext context,
  String feature, {
  bool free = true,
  VoidCallback? onLearnMore,
  VoidCallback? onRetry,
}) async {
  final l10n = AppLocalizations.of(context);
  if (!subscriptionsEnabled) {
    await showCraftskyModal<void>(
      context,
      builder: (dialogContext) => CraftskyDialog(
        title: l10n.plusFeatureComingSoon(feature),
        body: Text(l10n.plusFeatureComingSoonExplanation(feature)),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(),
            child: Text(l10n.dialogOkDefault),
          ),
        ],
      ),
    );
    return;
  }
  await showCraftskyModal<void>(
    context,
    builder: (dialogContext) => CraftskyDialog(
      title: free
          ? l10n.plusFeatureRequired(feature)
          : l10n.subscriptionsTierUnavailable,
      body: Text(
        free
            ? l10n.plusFeatureExplanation(feature)
            : l10n.plusFeatureAccessUnavailable(feature),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(dialogContext).pop(),
          child: Text(l10n.actionCancel),
        ),
        TextButton(
          onPressed: () {
            Navigator.of(dialogContext).pop();
            if (free) {
              if (onLearnMore case final callback?) {
                callback();
              } else {
                const SubscriptionsRoute().go(context);
              }
            } else {
              onRetry?.call();
            }
          },
          child: Text(
            free ? l10n.plusFeatureLearnMore : l10n.dialogOkDefault,
          ),
        ),
      ],
    ),
  );
}

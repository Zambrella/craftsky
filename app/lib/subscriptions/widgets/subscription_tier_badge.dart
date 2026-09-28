import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/subscriptions/models/subscription_access.dart';
import 'package:craftsky_app/subscriptions/providers/subscription_access_provider.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class LeaseSubscriptionTierBadge extends ConsumerWidget {
  const LeaseSubscriptionTierBadge({required this.lease, super.key});

  final AccountSessionLease lease;

  @override
  Widget build(BuildContext context, WidgetRef ref) => SubscriptionTierBadge(
    state: ref.watch(subscriptionAccessProvider(lease)),
  );
}

class SubscriptionTierBadge extends StatelessWidget {
  const SubscriptionTierBadge({required this.state, super.key});

  final AsyncValue<SubscriptionAccess> state;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    if (state case AsyncData(
      :final value,
    ) when value.effectiveTier == SubscriptionTier.free) {
      return const SizedBox.shrink();
    }
    final label = switch (state) {
      AsyncData(:final value) => switch (value.effectiveTier) {
        SubscriptionTier.free => '',
        SubscriptionTier.plus => l10n.subscriptionsTierPlus,
        SubscriptionTier.business => l10n.subscriptionsTierBusiness,
      },
      AsyncLoading() => l10n.subscriptionsTierLoading,
      AsyncError() => l10n.subscriptionsTierUnavailable,
    };
    return Semantics(
      label: label,
      child: DecoratedBox(
        decoration: BoxDecoration(
          color: Theme.of(context).colorScheme.secondaryContainer,
          borderRadius: BorderRadius.circular(999),
        ),
        child: MediaQuery.withClampedTextScaling(
          maxScaleFactor: 1.5,
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
            child: Text(
              label,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: Theme.of(context).textTheme.labelSmall,
            ),
          ),
        ),
      ),
    );
  }
}

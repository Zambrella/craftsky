import 'dart:async';

import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/data/crafts_catalog.dart';
import 'package:craftsky_app/profile/pages/edit_profile_dialog.dart';
import 'package:craftsky_app/profile/providers/recent_members_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/profile/widgets/profile_account_list_tile.dart';
import 'package:craftsky_app/projects/providers/preferred_craft_links_provider.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/widgets/craft_icon.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

/// Mounted only at the shared shell's existing desktop breakpoint.
class DesktopDiscoverySidebar extends ConsumerWidget {
  const new({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final lease = ref.watch(sessionRegistryProvider).value?.activeLease;
    final crafts = lease == null
        ? null
        : ref.watch(preferredCraftLinksProvider(lease));
    final members = lease == null
        ? null
        : ref.watch(recentMembersProvider(lease));
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    final l10n = AppLocalizations.of(context);
    return SizedBox(
      key: const Key('desktop-discovery-sidebar'),
      width: 300 + spacing.sp5 * 2,
      child: SingleChildScrollView(
        padding: EdgeInsets.all(spacing.sp5),
        child: Column(
          key: ValueKey(lease),
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            _DiscoverySection(
              key: const Key('discovery-crafts-section'),
              title: l10n.discoveryYourCrafts,
              child:
                  crafts?.when(
                    skipLoadingOnRefresh: false,
                    skipLoadingOnReload: false,
                    loading: () => const _SectionLoading(),
                    error: (_, _) => _SectionError(
                      message: l10n.discoveryCraftsError,
                      onRetry: () async {
                        final provider = userProfileProvider(
                          lease!.session.account.did,
                        );
                        ref.invalidate(provider);
                        await ref.read(provider.future);
                      },
                    ),
                    data: (options) => Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        if (options.isEmpty)
                          TextButton(
                            onPressed: () =>
                                unawaited(showEditProfileDialog(context)),
                            child: Text(l10n.discoveryChooseCrafts),
                          ),
                        for (final option in options)
                          TextButton(
                            key: ValueKey(
                              'discovery-craft-${option.value.split('#').last}',
                            ),
                            style: TextButton.styleFrom(
                              padding: EdgeInsets.symmetric(
                                horizontal: spacing.sp4,
                                vertical: spacing.sp3,
                              ),
                            ),
                            onPressed: () => FeedRoute(
                              craft: option.value.split('#').last,
                            ).go(context),
                            child: Align(
                              alignment: AlignmentDirectional.centerStart,
                              child: CraftIconLabel(
                                craft: option.value,
                                label: craftLabel(
                                  Craft.fromId(option.value.split('#').last)!,
                                  l10n,
                                ),
                                flexibleLabel: true,
                              ),
                            ),
                          ),
                      ],
                    ),
                  ) ??
                  const SizedBox.shrink(),
            ),
            SizedBox(height: spacing.sp5),
            _DiscoverySection(
              key: const Key('discovery-members-section'),
              title: l10n.discoveryNewMembers,
              bordered: true,
              child:
                  members?.when(
                    skipLoadingOnRefresh: members.hasError,
                    skipLoadingOnReload: members.hasError,
                    loading: () => const _SectionLoading(),
                    error: (_, _) => _SectionError(
                      message: l10n.discoveryMembersError,
                      onRetry: () async {
                        final provider = recentMembersProvider(lease!);
                        ref.invalidate(provider);
                        await ref.read(provider.future);
                      },
                    ),
                    data: (accounts) => accounts.isEmpty
                        ? Padding(
                            padding: EdgeInsets.all(spacing.sp4),
                            child: Text(l10n.discoveryMembersEmpty),
                          )
                        : Column(
                            children: [
                              for (final account in accounts)
                                ProfileAccountListTile(
                                  key: ValueKey(
                                    'discovery-member-${account.did}',
                                  ),
                                  account: account,
                                ),
                            ],
                          ),
                  ) ??
                  const SizedBox.shrink(),
            ),
          ],
        ),
      ),
    );
  }
}

class _DiscoverySection extends StatelessWidget {
  const new({
    required this.title,
    required this.child,
    this.bordered = false,
    super.key,
  });
  final String title;
  final Widget child;
  final bool bordered;

  @override
  Widget build(BuildContext context) {
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    final content = Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Padding(
          padding: EdgeInsets.all(spacing.sp4),
          child: Semantics(
            header: true,
            child: Text(
              title,
              style: Theme.of(context).textTheme.titleMedium,
            ),
          ),
        ),
        child,
      ],
    );
    if (!bordered) return content;
    final swatches = Theme.of(context).extension<BrandSwatchTheme>()!;
    final radii = Theme.of(context).extension<RadiusTheme>()!;
    return DecoratedBox(
      decoration: BoxDecoration(
        border: Border.all(color: swatches.borderHair),
        borderRadius: BorderRadius.circular(radii.r3),
      ),
      child: content,
    );
  }
}

class _SectionLoading extends StatelessWidget {
  const new();
  @override
  Widget build(BuildContext context) => const Padding(
    padding: EdgeInsets.all(16),
    child: Center(child: StitchProgressIndicator()),
  );
}

class _SectionError extends StatefulWidget {
  const new({required this.message, required this.onRetry});
  final String message;
  final Future<void> Function() onRetry;
  @override
  State<_SectionError> createState() => _SectionErrorState();
}

class _SectionErrorState extends State<_SectionError> {
  bool _retrying = false;

  Future<void> _retry() async {
    setState(() => _retrying = true);
    try {
      await widget.onRetry();
    } on Object {
      // The provider observer owns this failure; retain local recovery UI.
    } finally {
      if (mounted) setState(() => _retrying = false);
    }
  }

  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.all(16),
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(widget.message),
        TextButton(
          onPressed: _retrying ? null : () => unawaited(_retry()),
          child: _retrying
              ? const StitchProgressIndicator(size: 18)
              : Text(AppLocalizations.of(context).retryButton),
        ),
      ],
    ),
  );
}

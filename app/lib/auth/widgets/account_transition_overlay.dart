import 'package:craftsky_app/auth/providers/account_transition_provider.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/auth/widgets/account_avatar.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class AccountTransitionOverlay extends ConsumerWidget {
  const AccountTransitionOverlay({required this.child, super.key});

  final Widget child;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final transition = ref.watch(accountTransitionStateProvider);
    final session = transition == null
        ? null
        : ref
              .watch(sessionRegistryProvider)
              .value
              ?.sessions[transition.target.account.did];
    return Stack(
      fit: StackFit.expand,
      children: [
        ExcludeSemantics(
          excluding: transition != null,
          child: AbsorbPointer(absorbing: transition != null, child: child),
        ),
        if (transition != null)
          Semantics(
            scopesRoute: true,
            namesRoute: true,
            explicitChildNodes: true,
            label: l10n.accountSwitchingLabel,
            child: ColoredBox(
              color: Theme.of(context).colorScheme.surface,
              child: Center(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    AccountAvatar(
                      avatarUrl: session?.cachedAvatarUrl,
                      size: 64,
                      selected: true,
                    ),
                    const SizedBox(height: 16),
                    Text(
                      _identityLabel(
                        session?.cachedDisplayName,
                        session?.handle.value,
                        fallback: l10n.accountIdentityFallback,
                      ),
                      style: Theme.of(context).textTheme.titleMedium,
                    ),
                    const SizedBox(height: 16),
                    const CircularProgressIndicator(),
                  ],
                ),
              ),
            ),
          ),
      ],
    );
  }

  String _identityLabel(
    String? displayName,
    String? handle, {
    required String fallback,
  }) {
    final name = displayName?.trim();
    if (name != null && name.isNotEmpty) return name;
    if (handle != null && handle.isNotEmpty) return handle;
    return fallback;
  }
}

import 'dart:async';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/settings/providers/relationship_list_provider.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

export 'package:craftsky_app/settings/providers/relationship_list_provider.dart'
    show RelationshipListKind;

class RelationshipListPage extends ConsumerWidget {
  const RelationshipListPage({required this.kind, super.key});

  final RelationshipListKind kind;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final provider = relationshipListProvider(kind);
    final listAsync = ref.watch(provider);
    final title = switch (kind) {
      RelationshipListKind.muted => l10n.settingsMutedAccounts,
      RelationshipListKind.blocked => l10n.settingsBlockedAccounts,
    };
    return Scaffold(
      appBar: AppBar(title: Text(title)),
      body: switch (listAsync) {
        AsyncValue(:final value?) => _RelationshipListBody(
          kind: kind,
          state: value,
          isLoadingMore: listAsync.isLoading,
          onLoadMore: () => unawaited(
            ref.read(provider.notifier).loadMore(),
          ),
          onReverse: (account) => _reverse(context, ref, provider, account),
        ),
        AsyncError() => _RelationshipListError(
          message: switch (kind) {
            RelationshipListKind.muted => l10n.settingsMutedAccountsError,
            RelationshipListKind.blocked => l10n.settingsBlockedAccountsError,
          },
          onRetry: () => ref.invalidate(provider),
        ),
        _ => const Center(child: StitchProgressIndicator()),
      },
    );
  }

  Future<void> _reverse(
    BuildContext context,
    WidgetRef ref,
    RelationshipListProvider provider,
    ProfileAccountSummary account,
  ) async {
    if (kind == RelationshipListKind.blocked) {
      final l10n = AppLocalizations.of(context);
      final confirmed = await showDialog<bool>(
        context: context,
        builder: (dialogContext) => AlertDialog(
          title: Text(l10n.profileUnblockConfirmTitle),
          content: Text(l10n.profileUnblockConfirmBody),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(dialogContext).pop(false),
              child: Text(l10n.actionCancel),
            ),
            TextButton(
              onPressed: () => Navigator.of(dialogContext).pop(true),
              child: Text(l10n.profileUnblockAction),
            ),
          ],
        ),
      );
      if (confirmed != true || !context.mounted) return;
    }

    try {
      await ref.read(provider.notifier).reverse(account);
    } on Object {
      if (!context.mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            AppLocalizations.of(context).relationshipListMutationError,
          ),
        ),
      );
    }
  }
}

class _RelationshipListBody extends StatelessWidget {
  const _RelationshipListBody({
    required this.kind,
    required this.state,
    required this.isLoadingMore,
    required this.onLoadMore,
    required this.onReverse,
  });

  final RelationshipListKind kind;
  final RelationshipListState state;
  final bool isLoadingMore;
  final VoidCallback onLoadMore;
  final Future<void> Function(ProfileAccountSummary account) onReverse;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    if (state.items.isEmpty) {
      return Center(
        child: Text(
          switch (kind) {
            RelationshipListKind.muted => l10n.settingsMutedAccountsEmpty,
            RelationshipListKind.blocked => l10n.settingsBlockedAccountsEmpty,
          },
        ),
      );
    }
    return ListView.builder(
      itemCount: state.items.length + (state.hasMore ? 1 : 0),
      itemBuilder: (context, index) {
        if (index == state.items.length) {
          return Center(
            child: isLoadingMore
                ? const Padding(
                    padding: EdgeInsets.all(16),
                    child: StitchProgressIndicator(),
                  )
                : TextButton(
                    onPressed: onLoadMore,
                    child: Text(l10n.relationshipListLoadMore),
                  ),
          );
        }
        final account = state.items[index];
        final did = account.did.toString();
        return ListTile(
          title: Text(
            account.displayName?.isNotEmpty ?? false
                ? account.displayName!
                : account.handle.toString(),
          ),
          subtitle: Text('@${account.handle}'),
          onTap: () => unawaited(
            UserProfileRoute(
              handle: account.handle.toString(),
            ).push<void>(context),
          ),
          trailing: TextButton(
            onPressed: state.mutatingDids.contains(did)
                ? null
                : () => unawaited(onReverse(account)),
            child: Text(
              switch (kind) {
                RelationshipListKind.muted => l10n.relationshipListUnmute,
                RelationshipListKind.blocked => l10n.relationshipListUnblock,
              },
            ),
          ),
        );
      },
    );
  }
}

class _RelationshipListError extends StatelessWidget {
  const _RelationshipListError({required this.message, required this.onRetry});

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) => Center(
    child: Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(message),
        TextButton(
          onPressed: onRetry,
          child: Text(AppLocalizations.of(context).relationshipListRetry),
        ),
      ],
    ),
  );
}

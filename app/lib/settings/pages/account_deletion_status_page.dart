import 'dart:async';

import 'package:craftsky_app/auth/models/account_deletion.dart';
import 'package:craftsky_app/auth/providers/deletion_status_registry_provider.dart'
    show deletionStatusRegistryProvider;
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/settings/providers/account_deletion_controller.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class AccountDeletionStatusPage extends ConsumerStatefulWidget {
  const AccountDeletionStatusPage({
    required this.jobId,
    this.autoRefresh = true,
    super.key,
  });

  final String jobId;
  final bool autoRefresh;

  @override
  ConsumerState<AccountDeletionStatusPage> createState() =>
      _AccountDeletionStatusPageState();
}

class _AccountDeletionStatusPageState
    extends ConsumerState<AccountDeletionStatusPage> {
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    if (widget.autoRefresh) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) return;
        unawaited(_refresh());
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final entry = ref
        .watch(deletionStatusRegistryProvider)
        .value?[widget.jobId];
    if (entry == null) {
      return Scaffold(
        appBar: AppBar(title: Text(l10n.deleteAccountAction)),
        body: Center(child: Text(l10n.accountDeletionDeleted)),
      );
    }
    final attention = entry.status == AccountDeletionStatus.needsAttention;
    return Scaffold(
      appBar: AppBar(title: Text(l10n.deleteAccountAction)),
      body: ListView(
        padding: const EdgeInsets.all(24),
        children: [
          Text(
            attention
                ? l10n.accountDeletionNeedsAttention
                : _phaseLabel(l10n, entry),
            style: Theme.of(context).textTheme.headlineSmall,
          ),
          const SizedBox(height: 8),
          Text('@${entry.handle.value}'),
          if (attention) ...[
            const SizedBox(height: 16),
            Text(_phaseLabel(l10n, entry)),
          ],
          const SizedBox(height: 24),
          if (entry.canRetry)
            FilledButton(
              onPressed: _busy ? null : () => _run(() => _retry(entry)),
              child: Text(l10n.accountDeletionRetry),
            ),
          if (entry.needsReauthentication)
            OutlinedButton(
              onPressed: _busy
                  ? null
                  : () => _run(
                      () => ref
                          .read(accountDeletionControllerProvider.notifier)
                          .startReplacementReauthentication(widget.jobId),
                    ),
              child: Text(l10n.accountDeletionReauthenticate),
            ),
          TextButton(
            onPressed: _busy
                ? null
                : () => _run(
                    () async {
                      final launched = await launchExternalLink(
                        Uri.parse('https://craftsky.social/support'),
                      );
                      if (!launched) throw StateError('support launch failed');
                    },
                  ),
            child: Text(l10n.accountDeletionSupport),
          ),
        ],
      ),
    );
  }

  Future<void> _refresh() async {
    if (_busy || !mounted) return;
    try {
      await ref
          .read(accountDeletionControllerProvider.notifier)
          .refresh(widget.jobId);
    } on Object {
      // Polling is best-effort. The durable status row remains visible and a
      // later timer/app resume can recover without exposing a raw error.
    }
  }

  Future<void> _retry(DeletionStatusEntry _) =>
      ref.read(accountDeletionControllerProvider.notifier).retry(widget.jobId);

  Future<void> _run(Future<void> Function() action) async {
    if (_busy) return;
    setState(() => _busy = true);
    try {
      await action();
    } on Object {
      if (mounted) {
        context.showError(AppLocalizations.of(context).errorActionFailed);
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }
}

String _phaseLabel(
  AppLocalizations l10n,
  DeletionStatusEntry entry,
) {
  if (entry.status == AccountDeletionStatus.retrying) {
    return l10n.accountDeletionRetrying;
  }
  return switch (entry.phase) {
    AccountDeletionPhase.preparing => l10n.accountDeletionPreparing,
    AccountDeletionPhase.removingPrivateData =>
      l10n.accountDeletionRemovingPrivateData,
    AccountDeletionPhase.removingCraftskyRecords =>
      l10n.accountDeletionRemovingRecords,
    AccountDeletionPhase.waitingForCraftsky =>
      l10n.accountDeletionWaitingForCraftsky,
    AccountDeletionPhase.finalizing => l10n.accountDeletionFinalizing,
    AccountDeletionPhase.deleted => l10n.accountDeletionDeleted,
  };
}

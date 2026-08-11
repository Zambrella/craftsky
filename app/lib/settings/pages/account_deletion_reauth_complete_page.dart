import 'dart:async';

import 'package:craftsky_app/auth/models/account_deletion.dart';
import 'package:craftsky_app/auth/providers/deletion_status_registry_provider.dart'
    show deletionStatusRegistryProvider;
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/settings/models/delete_account_confirmation.dart';
import 'package:craftsky_app/settings/providers/account_deletion_controller.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class AccountDeletionReauthCompletePage extends ConsumerStatefulWidget {
  const AccountDeletionReauthCompletePage({
    required this.jobId,
    required this.proof,
    this.onCancel,
    super.key,
  });

  final String jobId;
  final String proof;
  final Future<void> Function()? onCancel;

  @override
  ConsumerState<AccountDeletionReauthCompletePage> createState() =>
      _AccountDeletionReauthCompletePageState();
}

class _AccountDeletionReauthCompletePageState
    extends ConsumerState<AccountDeletionReauthCompletePage> {
  final _controller = TextEditingController();
  bool _busy = false;
  bool _replacementRefreshStarted = false;
  bool _submissionStarted = false;
  bool _cancelStarted = false;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final entry = ref
        .watch(deletionStatusRegistryProvider)
        .value?[widget.jobId];
    final canComplete = ref
        .read(accountDeletionControllerProvider.notifier)
        .canComplete(widget.jobId);
    if (entry != null &&
        entry.status != AccountDeletionStatus.intent &&
        !canComplete) {
      if (!_replacementRefreshStarted) {
        _replacementRefreshStarted = true;
        WidgetsBinding.instance.addPostFrameCallback((_) async {
          try {
            await ref
                .read(accountDeletionControllerProvider.notifier)
                .refresh(widget.jobId);
          } on Object {
            // The status page retains safe retry/support actions.
          }
          if (!context.mounted) return;
          AccountDeletionStatusRoute(jobId: widget.jobId).go(context);
        });
      }
      return _guardIntent(
        Scaffold(
          appBar: AppBar(title: Text(l10n.deleteAccountConfirmTitle)),
          body: Center(child: Text(l10n.accountDeletionPreparing)),
        ),
      );
    }
    if (entry == null || !canComplete) {
      return _guardIntent(
        Scaffold(
          appBar: AppBar(title: Text(l10n.deleteAccountConfirmTitle)),
          body: Center(child: Text(l10n.errorActionFailed)),
        ),
      );
    }
    final requiredHandle = '@${entry.handle.value}';
    final matches = matchesDeletionConfirmationHandle(
      requiredHandle: requiredHandle,
      input: _controller.text,
    );
    return _guardIntent(
      Scaffold(
        appBar: AppBar(title: Text(l10n.deleteAccountConfirmTitle)),
        body: ListView(
          padding: const EdgeInsets.all(24),
          children: [
            Text(l10n.deleteAccountConfirmationPrompt(requiredHandle)),
            const SizedBox(height: 16),
            TextField(
              controller: _controller,
              autocorrect: false,
              enableSuggestions: false,
              decoration: InputDecoration(
                labelText: l10n.deleteAccountTypeHandleLabel,
              ),
              onChanged: (_) => setState(() {}),
            ),
            const SizedBox(height: 24),
            FilledButton(
              onPressed: !_busy && matches
                  ? () => _confirm(requiredHandle)
                  : null,
              child: _busy
                  ? const SizedBox.square(
                      dimension: 20,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : Text(l10n.deleteAccountAction),
            ),
          ],
        ),
      ),
    );
  }

  Widget _guardIntent(Widget child) => PopScope(
    onPopInvokedWithResult: (didPop, _) {
      if (didPop && !_submissionStarted && !_cancelStarted) {
        _cancelStarted = true;
        unawaited(
          widget.onCancel?.call() ??
              ref
                  .read(accountDeletionControllerProvider.notifier)
                  .cancelPendingIntent(widget.jobId),
        );
      }
    },
    child: child,
  );

  Future<void> _confirm(String handle) async {
    setState(() {
      _busy = true;
      _submissionStarted = true;
    });
    final succeeded = await ref
        .read(accountDeletionControllerProvider.notifier)
        .confirm(
          jobId: widget.jobId,
          reauthProof: widget.proof,
          confirmationHandle: handle,
        );
    if (!mounted) return;
    if (!succeeded) {
      setState(() => _busy = false);
      context.showError(AppLocalizations.of(context).errorActionFailed);
    }
  }
}

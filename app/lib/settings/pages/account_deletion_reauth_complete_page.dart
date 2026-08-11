import 'dart:async';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
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
    final deletion = ref.read(accountDeletionControllerProvider.notifier);
    final requiredHandle = deletion.requiredHandle(widget.jobId);
    if (requiredHandle == null || !deletion.canComplete(widget.jobId)) {
      return _guardIntent(
        Scaffold(
          appBar: AppBar(title: Text(l10n.deleteAccountConfirmTitle)),
          body: Center(child: Text(l10n.errorActionFailed)),
        ),
      );
    }
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

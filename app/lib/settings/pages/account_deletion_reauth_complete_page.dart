import 'dart:async';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/settings/models/delete_account_confirmation.dart';
import 'package:craftsky_app/settings/providers/account_deletion_controller.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
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
    final colors = Theme.of(context).colorScheme;
    final deletion = ref.read(accountDeletionControllerProvider.notifier);
    final confirmationDid = deletion.confirmationDid(widget.jobId);
    if (confirmationDid == null || !deletion.canComplete(widget.jobId)) {
      return _guardIntent(
        Scaffold(
          appBar: AppBar(title: Text(l10n.deleteAccountConfirmTitle)),
          body: Center(child: Text(l10n.errorActionFailed)),
        ),
      );
    }
    final matches = matchesDeletionConfirmationDid(
      confirmationDid: confirmationDid,
      input: _controller.text,
    );
    final confirmationPrompt = l10n.deleteAccountDidConfirmationPrompt(
      confirmationDid,
    );
    final didOffset = confirmationPrompt.indexOf(confirmationDid);
    return _guardIntent(
      Scaffold(
        appBar: AppBar(title: Text(l10n.deleteAccountConfirmTitle)),
        body: ListView(
          padding: const EdgeInsets.all(24),
          children: [
            Text.rich(
              TextSpan(
                children: didOffset < 0
                    ? [TextSpan(text: confirmationPrompt)]
                    : [
                        TextSpan(
                          text: confirmationPrompt.substring(0, didOffset),
                        ),
                        TextSpan(
                          text: confirmationDid,
                          style: const TextStyle(fontWeight: FontWeight.bold),
                        ),
                        TextSpan(
                          text: confirmationPrompt.substring(
                            didOffset + confirmationDid.length,
                          ),
                        ),
                      ],
              ),
            ),
            const SizedBox(height: 16),
            BrandTextField(
              label: l10n.deleteAccountTypeDidLabel,
              controller: _controller,
              autocorrect: false,
              enableSuggestions: false,
              onChanged: (_) => setState(() {}),
            ),
            const SizedBox(height: 24),
            ChunkyButton(
              backgroundColor: colors.error,
              foregroundColor: colors.onError,
              onPressed: !_busy && matches
                  ? () => _confirm(confirmationDid)
                  : null,
              child: _busy
                  ? StitchProgressIndicator(size: 18, color: colors.onError)
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

  Future<void> _confirm(String confirmationDid) async {
    setState(() {
      _busy = true;
      _submissionStarted = true;
    });
    final succeeded = await ref
        .read(accountDeletionControllerProvider.notifier)
        .confirm(
          jobId: widget.jobId,
          reauthProof: widget.proof,
          confirmationDid: confirmationDid,
        );
    if (!mounted) return;
    if (!succeeded) {
      setState(() => _busy = false);
      context.showError(AppLocalizations.of(context).errorActionFailed);
    }
  }
}

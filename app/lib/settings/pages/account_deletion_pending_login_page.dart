import 'package:craftsky_app/auth/models/account_deletion.dart';
import 'package:craftsky_app/auth/providers/deletion_status_registry_provider.dart';
import 'package:craftsky_app/auth/providers/pending_auth_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class AccountDeletionPendingLoginPage extends ConsumerStatefulWidget {
  const AccountDeletionPendingLoginPage({
    required this.jobId,
    required this.statusToken,
    required this.did,
    required this.handle,
    super.key,
  });

  final String jobId;
  final String statusToken;
  final String did;
  final String handle;

  @override
  ConsumerState<AccountDeletionPendingLoginPage> createState() =>
      _AccountDeletionPendingLoginPageState();
}

class _AccountDeletionPendingLoginPageState
    extends ConsumerState<AccountDeletionPendingLoginPage> {
  Object? _error;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _persistAndOpen());
  }

  Future<void> _persistAndOpen() async {
    try {
      await ref
          .read(deletionStatusRegistryProvider.notifier)
          .upsert(
            DeletionStatusEntry(
              jobId: widget.jobId,
              did: widget.did,
              handle: widget.handle,
              statusToken: widget.statusToken,
              status: AccountDeletionStatus.active,
              phase: AccountDeletionPhase.preparing,
              canRetry: false,
              needsReauthentication: false,
            ),
          );
      if (!mounted) return;
      ref.read(pendingAuthProvider.notifier).clear();
      AccountDeletionStatusRoute(jobId: widget.jobId).go(context);
    } on Object catch (error) {
      if (mounted) setState(() => _error = error);
    }
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    appBar: AppBar(
      title: Text(AppLocalizations.of(context).accountDeletionPreparing),
    ),
    body: Center(
      child: _error == null
          ? const CircularProgressIndicator()
          : Text(AppLocalizations.of(context).errorActionFailed),
    ),
  );
}

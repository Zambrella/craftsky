import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_account.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_suggestion.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_verification.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_account_provider.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_imports_provider.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_suggestions_provider.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_verification_provider.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_import_parser.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_json_file_picker.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/craftsky_text_inputs.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final instagramDmLauncherProvider = Provider<ExternalLinkLauncher>(
  (_) => launchExternalLink,
);

class InstagramMigrationPage extends ConsumerWidget {
  const InstagramMigrationPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final registry = ref.watch(sessionRegistryProvider);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.instagramMigrationTitle)),
      body: registry.when(
        loading: () => const Center(child: StitchProgressIndicator()),
        error: (_, _) => _CenteredMessage(l10n.instagramMigrationLoadError),
        data: (value) {
          final lease = value.activeLease;
          if (lease == null) {
            return _CenteredMessage(l10n.instagramMigrationNoActiveAccount);
          }
          return KeyedSubtree(
            key: ValueKey(lease),
            child: _InstagramMigrationBody(lease: lease),
          );
        },
      ),
    );
  }
}

class _InstagramMigrationBody extends ConsumerWidget {
  const _InstagramMigrationBody({required this.lease});

  final ActiveAccountLease lease;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final account = ref.watch(instagramAccountProvider(lease));
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    return RefreshIndicator(
      onRefresh: () async {
        await ref.read(instagramAccountProvider(lease).notifier).refresh();
        if (!_current(ref, lease)) return;
        await ref.read(instagramImportsProvider(lease).notifier).refresh();
        if (!_current(ref, lease)) return;
        await ref.read(instagramSuggestionsProvider(lease).notifier).refresh();
      },
      child: ListView(
        padding: EdgeInsets.fromLTRB(
          spacing.sp4,
          spacing.sp4,
          spacing.sp4,
          spacing.sp7,
        ),
        children: [
          Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 720),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  account.when(
                    loading: () => const _LoadingCard(),
                    error: (_, _) => _ErrorCard(
                      onRetry: () => ref
                          .read(instagramAccountProvider(lease).notifier)
                          .refresh(),
                    ),
                    data: (value) => _AccountAndVerificationCard(
                      lease: lease,
                      status: value,
                    ),
                  ),
                  SizedBox(height: spacing.sp4),
                  _ImportComposerCard(lease: lease),
                  SizedBox(height: spacing.sp4),
                  _ImportsCard(lease: lease),
                  SizedBox(height: spacing.sp4),
                  _SuggestionsCard(lease: lease),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _AccountAndVerificationCard extends ConsumerStatefulWidget {
  const _AccountAndVerificationCard({
    required this.lease,
    required this.status,
  });

  final ActiveAccountLease lease;
  final InstagramAccountStatus status;

  @override
  ConsumerState<_AccountAndVerificationCard> createState() =>
      _AccountAndVerificationCardState();
}

class _AccountAndVerificationCardState
    extends ConsumerState<_AccountAndVerificationCard> {
  bool? _confirmDiscoverable;
  String? _choiceVerificationId;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final l10n = AppLocalizations.of(context);
    final account = widget.status.account;
    return CraftskyCard(
      key: const Key('instagram-account-card'),
      padding: EdgeInsets.all(spacing.sp4),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _CardHeading(
            icon: account == null
                ? Icons.verified_outlined
                : Icons.link_outlined,
            title: account == null
                ? l10n.instagramVerificationTitle
                : l10n.instagramAccountTitle,
          ),
          SizedBox(height: spacing.sp2),
          if (account != null)
            _LinkedAccountControls(lease: widget.lease, account: account)
          else if (!widget.status.integrationAvailable) ...[
            Text(l10n.instagramVerificationUnavailable),
            SizedBox(height: spacing.sp1),
            Text(
              l10n.instagramVerificationUnavailableImports,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ] else
            _verificationFlow(l10n),
        ],
      ),
    );
  }

  Widget _verificationFlow(AppLocalizations l10n) {
    final flow = ref.watch(instagramVerificationProvider(widget.lease));
    final notifier = ref.read(
      instagramVerificationProvider(widget.lease).notifier,
    );
    final attempt = flow.attempt;
    if (attempt == null) {
      return Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(l10n.instagramVerificationDescription),
          const SizedBox(height: 12),
          FilledButton.icon(
            onPressed: flow.isBusy ? null : notifier.create,
            icon: const Icon(Icons.verified_outlined),
            label: Text(l10n.instagramVerificationStart),
          ),
          if (flow.hasError) ...[
            const SizedBox(height: 8),
            Text(l10n.instagramActionError),
          ],
        ],
      );
    }
    if (attempt.state == InstagramVerificationState.pendingConfirmation &&
        _choiceVerificationId != attempt.verificationId) {
      _choiceVerificationId = attempt.verificationId;
      _confirmDiscoverable = true;
    }
    return switch (attempt.state) {
      InstagramVerificationState.pendingDm ||
      InstagramVerificationState.processing => _ChallengeControls(
        lease: widget.lease,
        attempt: attempt,
        isBusy: flow.isBusy,
        hasError: flow.hasError,
      ),
      InstagramVerificationState.pendingConfirmation => Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _InstagramHandleText(
            username:
                attempt.candidateUsername ?? l10n.instagramUnknownUsername,
            localizedText: l10n.instagramVerificationCandidate(
              attempt.candidateUsername ?? l10n.instagramUnknownUsername,
            ),
          ),
          const SizedBox(height: 8),
          SegmentedButton<bool>(
            segments: [
              ButtonSegment(
                value: true,
                label: Text(l10n.instagramDiscoverableAllow),
              ),
              ButtonSegment(
                value: false,
                label: Text(l10n.instagramDiscoverablePrivate),
              ),
            ],
            selected: {?_confirmDiscoverable},
            onSelectionChanged: flow.isBusy
                ? null
                : (value) => setState(
                    () => _confirmDiscoverable = value.single,
                  ),
          ),
          const SizedBox(height: 8),
          Text(
            _confirmDiscoverable == false
                ? l10n.instagramDiscoverablePrivateDescription
                : l10n.instagramDiscoverableDescription,
          ),
          const SizedBox(height: 8),
          Text(l10n.instagramVerificationCandidateWarning),
          const SizedBox(height: 8),
          FilledButton(
            onPressed: flow.isBusy || _confirmDiscoverable == null
                ? null
                : () => notifier.confirm(
                    discoverable: _confirmDiscoverable!,
                  ),
            child: Text(l10n.instagramVerificationConfirm),
          ),
          TextButton(
            onPressed: flow.isBusy ? null : notifier.cancel,
            child: Text(l10n.instagramCancelVerification),
          ),
          if (flow.hasError) ...[
            const SizedBox(height: 8),
            Text(l10n.instagramActionError),
          ],
        ],
      ),
      InstagramVerificationState.confirmed => Text(
        l10n.instagramVerificationConfirmed,
      ),
      InstagramVerificationState.expired => _RetryVerification(
        message: l10n.instagramVerificationExpired,
        onRetry: notifier.create,
      ),
      InstagramVerificationState.cancelled ||
      InstagramVerificationState.superseded => _RetryVerification(
        message: l10n.instagramVerificationCancelled,
        onRetry: notifier.create,
      ),
      InstagramVerificationState.rejected => _RetryVerification(
        message: _verificationRetryMessage(l10n, attempt.retryCode),
        onRetry: notifier.create,
      ),
      InstagramVerificationState.conflicted => _RetryVerification(
        message: l10n.instagramVerificationConflict,
        onRetry: notifier.create,
      ),
      InstagramVerificationState.unknown => _RetryVerification(
        message: l10n.instagramActionError,
        onRetry: notifier.create,
      ),
    };
  }
}

class _InstagramHandleText extends StatelessWidget {
  const _InstagramHandleText({
    required this.username,
    required this.localizedText,
  });

  final String username;
  final String localizedText;

  @override
  Widget build(BuildContext context) {
    final handle = '@$username';
    final handleStart = localizedText.indexOf(handle);
    if (handleStart < 0) return Text(localizedText);
    final handleEnd = handleStart + handle.length;
    return Text.rich(
      TextSpan(
        children: [
          TextSpan(text: localizedText.substring(0, handleStart)),
          TextSpan(
            text: handle,
            style: const TextStyle(fontWeight: FontWeight.bold),
          ),
          if (handleEnd < localizedText.length)
            TextSpan(text: localizedText.substring(handleEnd)),
        ],
      ),
    );
  }
}

class _ChallengeControls extends ConsumerWidget {
  const _ChallengeControls({
    required this.lease,
    required this.attempt,
    required this.isBusy,
    required this.hasError,
  });

  final ActiveAccountLease lease;
  final InstagramVerificationAttempt attempt;
  final bool isBusy;
  final bool hasError;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final challenge = attempt.challenge;
    final dmUrl = attempt.dmUrl;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(l10n.instagramVerificationSendChallenge),
        const SizedBox(height: 8),
        Semantics(
          label: l10n.instagramVerificationChallengeLabel,
          child: SelectableText(
            challenge ?? l10n.instagramVerificationProcessing,
            style: Theme.of(context).textTheme.headlineSmall,
            textAlign: TextAlign.center,
          ),
        ),
        const SizedBox(height: 8),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            OutlinedButton.icon(
              onPressed: challenge == null
                  ? null
                  : () async {
                      await Clipboard.setData(ClipboardData(text: challenge));
                      if (!context.mounted || !_current(ref, lease)) return;
                      context.showInfo(l10n.instagramChallengeCopied);
                    },
              icon: const Icon(Icons.copy_outlined),
              label: Text(l10n.instagramCopyChallenge),
            ),
            FilledButton.icon(
              onPressed: dmUrl == null
                  ? null
                  : () async {
                      await ref.read(instagramDmLauncherProvider)(dmUrl);
                      if (!context.mounted || !_current(ref, lease)) return;
                    },
              icon: const Icon(Icons.open_in_new),
              label: Text(l10n.instagramOpenDm),
            ),
            TextButton(
              onPressed: isBusy
                  ? null
                  : () => ref
                        .read(instagramVerificationProvider(lease).notifier)
                        .cancel(),
              child: Text(l10n.instagramCancelVerification),
            ),
          ],
        ),
        if (hasError) ...[
          const SizedBox(height: 8),
          Text(l10n.instagramActionError),
          TextButton(
            onPressed: () =>
                ref.read(instagramVerificationProvider(lease).notifier).poll(),
            child: Text(l10n.instagramRetry),
          ),
        ],
      ],
    );
  }
}

class _LinkedAccountControls extends ConsumerWidget {
  const _LinkedAccountControls({required this.lease, required this.account});

  final ActiveAccountLease lease;
  final InstagramAccountLink account;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final notifier = ref.read(instagramAccountProvider(lease).notifier);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _InstagramHandleText(
          username: account.username,
          localizedText: l10n.instagramLinkedAs(account.username),
        ),
        if (account.conflictPending) ...[
          const SizedBox(height: 8),
          Text(l10n.instagramConflictPending),
        ],
        if (account.reactivationRequired ||
            account.state == InstagramAccountLinkState.membershipInactive) ...[
          Text(l10n.instagramReactivateAccountDisclosure),
          FilledButton(
            onPressed: notifier.reactivate,
            child: Text(l10n.instagramReactivateAccount),
          ),
        ] else
          SwitchListTile(
            contentPadding: EdgeInsets.zero,
            title: Text(l10n.instagramDiscoverableLabel),
            // subtitle: Text(l10n.instagramDiscoverableDescription),
            value: account.discoverable,
            onChanged: (value) => notifier.setDiscoverable(value: value),
          ),
        TextButton.icon(
          onPressed: notifier.revoke,
          icon: const Icon(Icons.link_off),
          label: Text(l10n.instagramRevokeAccount),
        ),
      ],
    );
  }
}

enum _ImportInputKind { manual, json }

class _ImportComposerCard extends ConsumerStatefulWidget {
  const _ImportComposerCard({required this.lease});

  final ActiveAccountLease lease;

  @override
  ConsumerState<_ImportComposerCard> createState() =>
      _ImportComposerCardState();
}

class _ImportComposerCardState extends ConsumerState<_ImportComposerCard> {
  final _manualController = TextEditingController();
  _ImportInputKind _kind = _ImportInputKind.manual;
  bool _retainUnmatched = false;
  bool _busy = false;
  InstagramImportParseResult? _preview;
  InstagramImportParseErrorCode? _parseError;
  bool _filePickerFailed = false;

  @override
  void dispose() {
    _manualController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final l10n = AppLocalizations.of(context);
    return CraftskyCard(
      key: const Key('instagram-import-composer-card'),
      padding: EdgeInsets.all(spacing.sp4),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _CardHeading(
            icon: Icons.person_search_outlined,
            title: l10n.instagramImportTitle,
          ),
          SizedBox(height: spacing.sp2),
          Text(
            l10n.instagramImportLocalDisclosure,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
          SizedBox(height: spacing.sp3),
          SegmentedButton<_ImportInputKind>(
            segments: [
              ButtonSegment(
                value: _ImportInputKind.manual,
                label: Text(l10n.instagramImportManual),
                icon: const Icon(Icons.edit_outlined),
              ),
              ButtonSegment(
                value: _ImportInputKind.json,
                label: Text(l10n.instagramImportJson),
                icon: const Icon(Icons.file_open_outlined),
              ),
            ],
            selected: {_kind},
            onSelectionChanged: (value) => setState(() {
              _kind = value.single;
              _preview = null;
              _parseError = null;
              _filePickerFailed = false;
            }),
          ),
          SizedBox(height: spacing.sp3),
          if (_kind == _ImportInputKind.manual) ...[
            CraftskyMultilineTextInput(
              key: const Key('instagram-manual-handles'),
              controller: _manualController,
              label: l10n.instagramImportHandles,
              hintText: l10n.instagramImportHandlesHint,
              enabled: !_busy,
            ),
            SizedBox(height: spacing.sp2),
            OutlinedButton(
              onPressed: _busy ? null : _parseManual,
              child: Text(l10n.instagramImportPreview),
            ),
          ] else
            OutlinedButton.icon(
              onPressed: _busy ? null : _pickJson,
              icon: const Icon(Icons.file_open_outlined),
              label: Text(l10n.instagramImportSelectJson),
            ),
          if (_parseError != null) ...[
            const SizedBox(height: 8),
            Text(_parseErrorMessage(l10n, _parseError!)),
          ],
          if (_filePickerFailed) ...[
            const SizedBox(height: 8),
            Text(l10n.instagramImportFilePickerError),
          ],
          if (_preview case final preview?) ...[
            const SizedBox(height: 12),
            Text(
              l10n.instagramImportFollowingPreviewCount(preview.entries.length),
            ),
            if (preview.ignoredEntryCount > 0)
              Text(
                l10n.instagramImportIgnoredCount(
                  preview.ignoredEntryCount,
                ),
              ),
            if (preview.duplicateEntryCount > 0)
              Text(
                l10n.instagramImportDuplicateCount(
                  preview.duplicateEntryCount,
                ),
              ),
            SwitchListTile(
              contentPadding: EdgeInsets.zero,
              title: Text(l10n.instagramImportRetention),
              subtitle: Text(l10n.instagramImportRetentionDescription),
              value: _retainUnmatched,
              onChanged: _busy
                  ? null
                  : (value) => setState(() => _retainUnmatched = value),
            ),
            FilledButton(
              onPressed: preview.entries.isEmpty || _busy
                  ? null
                  : _uploadPreview,
              child: Text(l10n.instagramImportUpload),
            ),
          ],
        ],
      ),
    );
  }

  void _parseManual() {
    try {
      final result = const InstagramImportParser().parseManual(
        _manualController.text,
      );
      setState(() {
        _preview = result;
        _parseError = null;
        _filePickerFailed = false;
      });
    } on InstagramImportParseException catch (error) {
      setState(() {
        _preview = null;
        _parseError = error.code;
        _filePickerFailed = false;
      });
    }
  }

  Future<void> _pickJson() async {
    final capturedLease = widget.lease;
    setState(() => _busy = true);
    try {
      final bytes = await ref.read(instagramJsonFilePickerProvider)();
      if (!mounted ||
          capturedLease != widget.lease ||
          !_current(ref, capturedLease)) {
        return;
      }
      if (bytes == null) return;
      final result = const InstagramImportParser().parseJson(bytes);
      setState(() {
        _preview = result;
        _parseError = null;
        _filePickerFailed = false;
      });
    } on InstagramImportParseException catch (error) {
      if (!mounted || !_current(ref, capturedLease)) return;
      setState(() {
        _preview = null;
        _parseError = error.code;
        _filePickerFailed = false;
      });
    } on Object {
      if (!mounted || !_current(ref, capturedLease)) return;
      setState(() {
        _preview = null;
        _parseError = null;
        _filePickerFailed = true;
      });
    } finally {
      if (mounted && capturedLease == widget.lease) {
        setState(() => _busy = false);
      }
    }
  }

  Future<void> _uploadPreview() async {
    final preview = _preview;
    if (preview == null) return;
    final lease = widget.lease;
    setState(() => _busy = true);
    final result = await ref
        .read(instagramImportsProvider(lease).notifier)
        .create(
          InstagramImportRequest(
            sourceType: _kind == _ImportInputKind.manual
                ? InstagramImportSourceType.manual
                : InstagramImportSourceType.instagramJson,
            retainUnmatched: _retainUnmatched,
            entries: preview.entries,
          ),
        );
    if (!mounted || !_current(ref, lease)) return;
    setState(() {
      _busy = false;
      if (result != null) {
        _preview = null;
        _manualController.clear();
      }
    });
    final l10n = AppLocalizations.of(context);
    if (result == null) {
      context.showError(l10n.instagramImportUploadError);
    } else {
      context.showInfo(l10n.instagramImportUploadSuccess);
    }
  }
}

class _ImportsCard extends ConsumerWidget {
  const _ImportsCard({required this.lease});

  final ActiveAccountLease lease;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final l10n = AppLocalizations.of(context);
    final imports = ref.watch(instagramImportsProvider(lease));
    return CraftskyCard(
      key: const Key('instagram-imports-card'),
      padding: EdgeInsets.all(spacing.sp4),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _CardHeading(
            icon: Icons.inventory_2_outlined,
            title: l10n.instagramImportsTitle,
          ),
          SizedBox(height: spacing.sp2),
          imports.when(
            loading: () => const Center(child: StitchProgressIndicator()),
            error: (_, _) => _InlineRetry(
              message: l10n.instagramImportsLoadError,
              onRetry: () =>
                  ref.read(instagramImportsProvider(lease).notifier).refresh(),
            ),
            data: (page) => page.items.isEmpty
                ? Text(l10n.instagramImportsEmpty)
                : Column(
                    children: [
                      for (final item in page.items)
                        _ImportRow(lease: lease, item: item),
                      if (page.cursor != null)
                        TextButton(
                          onPressed: () => ref
                              .read(instagramImportsProvider(lease).notifier)
                              .loadMore(),
                          child: Text(l10n.instagramLoadMore),
                        ),
                    ],
                  ),
          ),
        ],
      ),
    );
  }
}

class _ImportRow extends ConsumerWidget {
  const _ImportRow({required this.lease, required this.item});

  final ActiveAccountLease lease;
  final InstagramImportSummary item;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final notifier = ref.read(instagramImportsProvider(lease).notifier);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            _importSourceLabel(l10n, item.sourceType),
            style: Theme.of(context).textTheme.titleMedium,
          ),
          Text(
            l10n.instagramImportCounts(item.followingCount),
          ),
          if (item.retentionExpiresAt case final expiresAt?)
            Text(
              l10n.instagramImportRetainedUntil(
                MaterialLocalizations.of(context).formatMediumDate(expiresAt),
              ),
            ),
          if (item.state == InstagramImportState.membershipInactive) ...[
            const SizedBox(height: 4),
            Text(l10n.instagramImportReactivationDisclosure),
            Align(
              alignment: Alignment.centerLeft,
              child: FilledButton.tonal(
                onPressed: () => _runImportAction(
                  context,
                  ref,
                  lease,
                  () => notifier.reactivate(item.importId),
                ),
                child: Text(l10n.instagramImportReactivate),
              ),
            ),
          ],
          Wrap(
            spacing: 4,
            children: [
              if (item.retainUnmatched) ...[
                TextButton(
                  onPressed: () => _runImportAction(
                    context,
                    ref,
                    lease,
                    () => notifier.renewRetention(item.importId),
                  ),
                  child: Text(l10n.instagramImportRenewRetention),
                ),
                TextButton(
                  onPressed: () => _runImportAction(
                    context,
                    ref,
                    lease,
                    () => notifier.withdrawRetention(item.importId),
                  ),
                  child: Text(l10n.instagramImportWithdrawRetention),
                ),
              ],
              TextButton(
                onPressed: () => _runImportAction(
                  context,
                  ref,
                  lease,
                  () => notifier.delete(item.importId),
                ),
                child: Text(l10n.instagramImportDelete),
              ),
            ],
          ),
          if (!item.retainUnmatched)
            Text(l10n.instagramImportRetentionDiscarded),
          const Divider(),
        ],
      ),
    );
  }
}

class _SuggestionsCard extends ConsumerWidget {
  const _SuggestionsCard({required this.lease});

  final ActiveAccountLease lease;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final l10n = AppLocalizations.of(context);
    final suggestions = ref.watch(instagramSuggestionsProvider(lease));
    return CraftskyCard(
      key: const Key('instagram-suggestions-card'),
      padding: EdgeInsets.all(spacing.sp4),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _CardHeading(
            icon: Icons.group_add_outlined,
            title: l10n.instagramSuggestionsTitle,
          ),
          SizedBox(height: spacing.sp2),
          suggestions.when(
            loading: () => const Center(child: StitchProgressIndicator()),
            error: (_, _) => _InlineRetry(
              message: l10n.instagramSuggestionsLoadError,
              onRetry: () => ref
                  .read(instagramSuggestionsProvider(lease).notifier)
                  .refresh(),
            ),
            data: (value) => _SuggestionReview(
              lease: lease,
              value: value,
            ),
          ),
        ],
      ),
    );
  }
}

class _SuggestionReview extends ConsumerWidget {
  const _SuggestionReview({required this.lease, required this.value});

  final ActiveAccountLease lease;
  final InstagramSuggestionReviewState value;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final notifier = ref.read(instagramSuggestionsProvider(lease).notifier);
    if (value.items.isEmpty) return Text(l10n.instagramSuggestionsEmpty);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Wrap(
          spacing: 8,
          children: [
            OutlinedButton(
              onPressed: notifier.selectAllReviewed,
              child: Text(l10n.instagramSuggestionsSelectAll),
            ),
            TextButton(
              onPressed: value.selectedIds.isEmpty
                  ? null
                  : notifier.clearSelection,
              child: Text(l10n.instagramSuggestionsClearSelection),
            ),
          ],
        ),
        for (final suggestion in value.items)
          _SuggestionRow(
            lease: lease,
            suggestion: suggestion,
            selected: value.selectedIds.contains(suggestion.suggestionId),
            busy: value.busyIds.contains(suggestion.suggestionId),
          ),
        if (value.cursor != null)
          TextButton(
            onPressed: notifier.loadMore,
            child: Text(l10n.instagramLoadMore),
          ),
        FilledButton(
          onPressed: value.selectedIds.isEmpty ? null : notifier.acceptSelected,
          child: Text(
            l10n.instagramSuggestionsAcceptSelected(value.selectedIds.length),
          ),
        ),
        if (value.hasActionError) ...[
          const SizedBox(height: 8),
          Text(l10n.instagramSuggestionsActionError),
          TextButton(
            onPressed: notifier.refresh,
            child: Text(l10n.instagramRetry),
          ),
        ],
      ],
    );
  }
}

class _SuggestionRow extends ConsumerWidget {
  const _SuggestionRow({
    required this.lease,
    required this.suggestion,
    required this.selected,
    required this.busy,
  });

  final ActiveAccountLease lease;
  final InstagramSuggestion suggestion;
  final bool selected;
  final bool busy;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final notifier = ref.read(instagramSuggestionsProvider(lease).notifier);
    final pending = suggestion.state == InstagramSuggestionState.pending;
    final reason = _suggestionReason(l10n, suggestion.reason);
    return Column(
      children: [
        CheckboxListTile(
          contentPadding: EdgeInsets.zero,
          value: selected,
          onChanged: !pending || busy
              ? null
              : (value) => notifier.select(
                  suggestion.suggestionId,
                  selected: value ?? false,
                ),
          title: Text(
            suggestion.profile.displayName ?? suggestion.profile.handle,
          ),
          subtitle: Text('@${suggestion.profile.handle}\n$reason'),
          isThreeLine: true,
        ),
        Wrap(
          spacing: 8,
          children: [
            TextButton(
              onPressed: !pending || busy
                  ? null
                  : () => notifier.accept(suggestion.suggestionId),
              child: Text(l10n.instagramSuggestionAccept),
            ),
            TextButton(
              onPressed: !pending || busy
                  ? null
                  : () => notifier.dismiss(suggestion.suggestionId),
              child: Text(l10n.instagramSuggestionDismiss),
            ),
          ],
        ),
      ],
    );
  }
}

class _RetryVerification extends StatelessWidget {
  const _RetryVerification({required this.message, required this.onRetry});

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) => Column(
    crossAxisAlignment: CrossAxisAlignment.stretch,
    children: [
      Text(message),
      const SizedBox(height: 8),
      FilledButton(
        onPressed: onRetry,
        child: Text(AppLocalizations.of(context).instagramRetry),
      ),
    ],
  );
}

class _CardHeading extends StatelessWidget {
  const _CardHeading({required this.icon, required this.title});

  final IconData icon;
  final String title;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    return Row(
      children: [
        Icon(icon, color: theme.colorScheme.primary),
        SizedBox(width: spacing.sp2),
        Expanded(child: Text(title, style: theme.textTheme.titleLarge)),
      ],
    );
  }
}

class _InlineRetry extends StatelessWidget {
  const _InlineRetry({required this.message, required this.onRetry});

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) => Column(
    children: [
      Text(message),
      TextButton(
        onPressed: onRetry,
        child: Text(AppLocalizations.of(context).instagramRetry),
      ),
    ],
  );
}

class _LoadingCard extends StatelessWidget {
  const _LoadingCard();

  @override
  Widget build(BuildContext context) {
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    return CraftskyCard(
      padding: EdgeInsets.all(spacing.sp5),
      child: const Center(child: StitchProgressIndicator()),
    );
  }
}

class _ErrorCard extends StatelessWidget {
  const _ErrorCard({required this.onRetry});

  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    return CraftskyCard(
      padding: EdgeInsets.all(spacing.sp4),
      child: _InlineRetry(
        message: AppLocalizations.of(context).instagramMigrationLoadError,
        onRetry: onRetry,
      ),
    );
  }
}

class _CenteredMessage extends StatelessWidget {
  const _CenteredMessage(this.message);

  final String message;

  @override
  Widget build(BuildContext context) => Center(
    child: Padding(
      padding: const EdgeInsets.all(24),
      child: Text(message, textAlign: TextAlign.center),
    ),
  );
}

bool _current(WidgetRef ref, ActiveAccountLease lease) =>
    ref.read(sessionRegistryProvider).value?.isCurrent(lease) ?? false;

Future<void> _runImportAction(
  BuildContext context,
  WidgetRef ref,
  ActiveAccountLease lease,
  Future<bool> Function() action,
) async {
  final succeeded = await action();
  if (!context.mounted || !_current(ref, lease) || succeeded) return;
  context.showError(AppLocalizations.of(context).instagramActionError);
}

String _parseErrorMessage(
  AppLocalizations l10n,
  InstagramImportParseErrorCode code,
) => switch (code) {
  InstagramImportParseErrorCode.invalidJson => l10n.instagramImportInvalidJson,
  InstagramImportParseErrorCode.unsupportedShape =>
    l10n.instagramImportUnsupportedShape,
  InstagramImportParseErrorCode.unsupportedFormat =>
    l10n.instagramImportUnsupportedFormat,
  InstagramImportParseErrorCode.fileTooLarge =>
    l10n.instagramImportFileTooLarge,
  InstagramImportParseErrorCode.tooManyEntries =>
    l10n.instagramImportTooManyEntries,
};

String _importSourceLabel(
  AppLocalizations l10n,
  InstagramImportSourceType source,
) => switch (source) {
  InstagramImportSourceType.manual => l10n.instagramImportManualSource,
  InstagramImportSourceType.instagramJson => l10n.instagramImportJsonSource,
  InstagramImportSourceType.unknown => l10n.instagramImportUnknownSource,
};

String _verificationRetryMessage(
  AppLocalizations l10n,
  InstagramVerificationRetryCode? code,
) => switch (code) {
  InstagramVerificationRetryCode.profileLookupUnavailable =>
    l10n.instagramVerificationProfileUnavailable,
  InstagramVerificationRetryCode.invalidProfileResponse =>
    l10n.instagramVerificationProfileInvalid,
  InstagramVerificationRetryCode.membershipInactive =>
    l10n.instagramVerificationMembershipInactive,
  InstagramVerificationRetryCode.unknown ||
  null => l10n.instagramVerificationRejected,
};

String _suggestionReason(
  AppLocalizations l10n,
  InstagramSuggestionReason reason,
) => switch (reason) {
  InstagramSuggestionReason.verifiedInstagramFollow =>
    l10n.instagramSuggestionReason,
  InstagramSuggestionReason.unknown => l10n.instagramSuggestionUnknownReason,
};

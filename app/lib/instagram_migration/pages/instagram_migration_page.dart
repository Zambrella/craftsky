import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_account.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_import.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_suggestion.dart';
import 'package:craftsky_app/instagram_migration/models/instagram_verification.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_account_provider.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_imports_provider.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_suggestions_provider.dart';
import 'package:craftsky_app/instagram_migration/providers/instagram_verification_provider.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_export_file_picker.dart';
import 'package:craftsky_app/instagram_migration/services/instagram_import_parser.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/widgets/compact_profile_identity.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:craftsky_app/shared/widgets/action_progress.dart';
import 'package:craftsky_app/shared/widgets/async_action_builder.dart';
import 'package:craftsky_app/shared/widgets/craftsky_empty_state.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:craftsky_app/theme/yarn_refresh_scroll_view.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';

final instagramDmLauncherProvider = Provider<ExternalLinkLauncher>(
  (_) => launchExternalLink,
);

final instagramHelpLauncherProvider = Provider<ExternalLinkLauncher>(
  (_) => launchExternalLink,
);

enum _InstagramAction {
  create,
  confirm,
  cancel,
  reactivate,
  delete,
  follow,
  dismiss,
  more,
}

enum InstagramMigrationView { landing, following, suggestions, account }

class InstagramMigrationPage extends ConsumerWidget {
  const new({this.view = InstagramMigrationView.landing, super.key});

  final InstagramMigrationView view;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final registry = ref.watch(sessionRegistryProvider);
    return Scaffold(
      appBar: AppBar(
        title: Text(switch (view) {
          InstagramMigrationView.landing => l10n.instagramMigrationTitle,
          InstagramMigrationView.following => l10n.instagramImportTitle,
          InstagramMigrationView.suggestions => l10n.instagramSuggestionsTitle,
          InstagramMigrationView.account => l10n.instagramManageAccount,
        }),
      ),
      body: registry.when(
        loading: () => const Center(child: StitchProgressIndicator()),
        error: (_, _) => _CenteredMessage(l10n.instagramMigrationLoadError),
        data: (value) {
          final lease = value.activeLease;
          if (lease == null) {
            return _CenteredMessage(l10n.instagramMigrationNoActiveAccount);
          }
          return KeyedSubtree(
            key: ValueKey((lease, view)),
            child: _InstagramMigrationBody(lease: lease, view: view),
          );
        },
      ),
    );
  }
}

class _InstagramMigrationBody extends ConsumerWidget {
  const new({required this.lease, required this.view});

  final ActiveAccountLease lease;
  final InstagramMigrationView view;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final accountProvider = instagramAccountProvider(lease);
    final account = ref.watch(accountProvider);
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    return YarnRefreshScrollView.list(
      showSuccess: false,
      onRefresh: () async {
        await ref.read(accountProvider.notifier).refresh();
        if (!context.mounted ||
            !_current(ref, lease) ||
            ref.read(accountProvider).value?.account == null) {
          return;
        }
        switch (view) {
          case InstagramMigrationView.following:
            await ref.read(instagramImportsProvider(lease).notifier).refresh();
          case InstagramMigrationView.landing:
          case InstagramMigrationView.suggestions:
            await ref
                .read(instagramSuggestionsProvider(lease).notifier)
                .refresh();
          case InstagramMigrationView.account:
            break;
        }
      },
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
            child: account.when(
              loading: () => const _LoadingCard(),
              error: (_, _) => _ErrorCard(
                onRetry: () => ref.read(accountProvider.notifier).refresh(),
              ),
              data: (status) => _content(context, ref, status),
            ),
          ),
        ),
      ],
    );
  }

  Widget _content(
    BuildContext context,
    WidgetRef ref,
    InstagramAccountStatus status,
  ) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final account = status.account;
    if (account == null) {
      if (view != InstagramMigrationView.landing) {
        // The verification view owns the challenge. Wait until this route is
        // visible so a confirmation dialog can finish closing after revocation.
        final router = GoRouter.maybeOf(context);
        if (router != null &&
            TickerMode.valuesOf(context).enabled &&
            ModalRoute.isCurrentOf(context) == true) {
          final childLocation = GoRouterState.of(context).matchedLocation;
          WidgetsBinding.instance.addPostFrameCallback((_) {
            if (!context.mounted ||
                !_current(ref, lease) ||
                ModalRoute.isCurrentOf(context) != true ||
                router.state.matchedLocation != childLocation) {
              return;
            }
            final latest = ref.read(instagramAccountProvider(lease)).asData;
            if (latest != null && latest.value.account == null) {
              const InstagramMigrationRoute().go(context);
            }
          });
        }
        return Text(l10n.instagramVerificationRequiredForImport);
      }
      return Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _AccountAndVerificationCard(lease: lease, status: status),
          SizedBox(height: spacing.sp4),
          Text(l10n.instagramVerificationRequiredForImport),
        ],
      );
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (!status.integrationAvailable) ...[
          Text(l10n.instagramVerificationUnavailableLinked),
          SizedBox(height: spacing.sp4),
        ],
        ...switch (view) {
          InstagramMigrationView.landing => [
            _LinkedInstagramLanding(lease: lease, account: account),
          ],
          InstagramMigrationView.following => [
            if (account.allowsImport) ...[
              Align(
                alignment: AlignmentDirectional.centerStart,
                child: ChunkyButton(
                  key: const Key('instagram-import-help'),
                  backgroundColor: theme.extension<BrandSwatchTheme>()!.lilac,
                  foregroundColor: theme.colorScheme.onSurface,
                  onPressed: () => unawaited(
                    confirmAndLaunchExternalLink(
                      context,
                      uri: Uri.https('craftsky.social', '/instagram-import'),
                      launchUrl: ref.read(instagramHelpLauncherProvider),
                      confirmOpenLink: (_, _) async => true,
                    ),
                  ),
                  child: Semantics(
                    label: l10n.instagramImportHelp,
                    excludeSemantics: true,
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      spacing: spacing.sp2,
                      children: [
                        const Icon(CraftskyIcons.help, size: 20),
                        Flexible(child: Text(l10n.instagramImportHelpShort)),
                        const Icon(CraftskyIcons.externalLink, size: 16),
                      ],
                    ),
                  ),
                ),
              ),
              SizedBox(height: spacing.sp4),
              _ImportComposerCard(lease: lease),
            ] else ...[
              Text(l10n.instagramImportAccountBlocked),
              Align(
                alignment: AlignmentDirectional.centerStart,
                child: TextButton(
                  onPressed: () => unawaited(
                    const InstagramAccountRoute().push<void>(context),
                  ),
                  child: Text(l10n.instagramManageAccount),
                ),
              ),
            ],
            SizedBox(height: spacing.sp4),
            _ImportsCard(lease: lease),
          ],
          InstagramMigrationView.suggestions => [
            _SuggestionsCard(
              lease: lease,
              onSuggestionTap: (suggestion) => unawaited(
                UserProfileRoute(
                  did: Did.parse(suggestion.target.did),
                ).push<void>(context),
              ),
            ),
          ],
          InstagramMigrationView.account => [
            _AccountAndVerificationCard(lease: lease, status: status),
            SizedBox(height: spacing.sp6),
            _RevokeInstagramVerificationButton(lease: lease),
          ],
        },
      ],
    );
  }
}

class _LinkedInstagramLanding extends ConsumerWidget {
  const new({required this.lease, required this.account});

  final ActiveAccountLease lease;
  final InstagramAccountLink account;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final suggestions = ref.watch(instagramSuggestionsProvider(lease)).value;
    final count = suggestions?.items.length ?? 0;
    final countText = MaterialLocalizations.of(context).formatDecimal(count);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        CraftskyCard(
          padding: EdgeInsets.all(spacing.sp3),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              OverflowBar(
                alignment: MainAxisAlignment.spaceBetween,
                spacing: spacing.sp2,
                children: [
                  Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(
                        CraftskyIcons.verifiedAccount,
                        color: theme.colorScheme.primary,
                      ),
                      SizedBox(width: spacing.sp2),
                      Flexible(
                        child: _InstagramHandleText(
                          username: account.username,
                          localizedText: l10n.instagramLinkedAs(
                            account.username,
                          ),
                        ),
                      ),
                    ],
                  ),
                  TextButton(
                    key: const Key('instagram-account-manage'),
                    onPressed: () => unawaited(
                      const InstagramAccountRoute().push<void>(context),
                    ),
                    child: Text(l10n.instagramManageAccount),
                  ),
                ],
              ),
              if (_hasOwnershipConflict(account))
                Padding(
                  padding: EdgeInsets.only(top: spacing.sp2),
                  child: Text(l10n.instagramConflictPending),
                ),
              if (_needsReactivation(account))
                Padding(
                  padding: EdgeInsets.only(top: spacing.sp2),
                  child: Text(l10n.instagramAccountReactivationRequired),
                ),
            ],
          ),
        ),
        SizedBox(height: spacing.sp4),
        _InstagramTaskTile(
          key: const Key('instagram-following-import-open'),
          icon: CraftskyIcons.addPeople,
          title: l10n.instagramImportTitle,
          onTap: () => unawaited(
            const InstagramFollowingImportRoute().push<void>(context),
          ),
        ),
        const Divider(height: 1),
        _InstagramTaskTile(
          key: const Key('instagram-post-import-open'),
          icon: CraftskyIcons.image,
          title: l10n.instagramPostImportTitle,
          onTap: () => unawaited(
            const InstagramPostImportRoute().push<void>(context),
          ),
        ),
        const Divider(height: 1),
        _InstagramTaskTile(
          key: const Key('instagram-suggestions-open'),
          icon: CraftskyIcons.people,
          title: l10n.instagramSuggestionsTitle,
          count: count == 0
              ? null
              : '$countText${suggestions?.cursor == null ? '' : '+'}',
          onTap: () => unawaited(
            const InstagramSuggestionsRoute().push<void>(context),
          ),
        ),
      ],
    );
  }
}

class _InstagramTaskTile extends StatelessWidget {
  const new({
    required this.icon,
    required this.title,
    required this.onTap,
    this.count,
    super.key,
  });

  final IconData icon;
  final String title;
  final String? count;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return ListTile(
      leading: Icon(icon),
      title: Text(title),
      trailing: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (count case final label?)
            Padding(
              padding: const EdgeInsetsDirectional.only(end: 8),
              child: Text(
                label,
                style: theme.textTheme.labelLarge?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
            ),
          Icon(
            Directionality.of(context) == TextDirection.ltr
                ? CraftskyIconsBold.next
                : CraftskyIconsBold.previous,
          ),
        ],
      ),
      onTap: onTap,
    );
  }
}

bool _hasOwnershipConflict(InstagramAccountLink account) =>
    account.conflictPending ||
    account.state == InstagramAccountLinkState.disputed;

bool _needsReactivation(InstagramAccountLink account) =>
    account.reactivationRequired ||
    account.state == InstagramAccountLinkState.membershipInactive;

class _AccountAndVerificationCard extends ConsumerStatefulWidget {
  const new({
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
      clipBehavior: Clip.none,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _CardHeading(
            icon: account == null
                ? CraftskyIcons.verifiedAccount
                : CraftskyIcons.link,
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

    return AsyncActionBuilder(
      builder: (context, actions) {
        final attempt = flow.attempt;
        if (attempt == null) {
          return Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(l10n.instagramVerificationDescription),
              const SizedBox(height: 12),
              FilledButton.icon(
                onPressed: flow.isBusy || actions.isBusy
                    ? null
                    : () =>
                          actions.run(_InstagramAction.create, notifier.create),
                icon: ActionProgress(
                  isLoading: actions.isRunning(_InstagramAction.create),
                  child: const Icon(CraftskyIconsBold.verifiedAccount),
                ),
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
            isBusy: flow.isBusy || actions.isBusy,
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
                onPressed:
                    flow.isBusy ||
                        actions.isBusy ||
                        _confirmDiscoverable == null
                    ? null
                    : () => actions.run(
                        _InstagramAction.confirm,
                        () => notifier.confirm(
                          discoverable: _confirmDiscoverable!,
                        ),
                      ),
                child: ActionProgress(
                  isLoading: actions.isRunning(_InstagramAction.confirm),
                  child: Text(l10n.instagramVerificationConfirm),
                ),
              ),
              TextButton(
                onPressed: flow.isBusy || actions.isBusy
                    ? null
                    : () =>
                          actions.run(_InstagramAction.cancel, notifier.cancel),
                child: ActionProgress(
                  isLoading: actions.isRunning(_InstagramAction.cancel),
                  child: Text(l10n.instagramCancelVerification),
                ),
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
      },
    );
  }
}

class _InstagramHandleText extends StatelessWidget {
  const new({
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
            style: const TextStyle(fontWeight: FontWeight.w600),
          ),
          if (handleEnd < localizedText.length)
            TextSpan(text: localizedText.substring(handleEnd)),
        ],
      ),
    );
  }
}

class _ChallengeControls extends ConsumerWidget {
  const new({
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
    return AsyncActionBuilder(
      builder: (context, actions) {
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
            Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                OutlinedButton.icon(
                  onPressed: challenge == null
                      ? null
                      : () async {
                          await Clipboard.setData(
                            ClipboardData(text: challenge),
                          );
                          if (!context.mounted || !_current(ref, lease)) return;
                          context.showInfo(l10n.instagramChallengeCopied);
                        },
                  icon: const Icon(CraftskyIconsBold.copy),
                  label: Text(l10n.instagramCopyChallenge),
                ),
                const SizedBox(height: 8),
                FilledButton.icon(
                  onPressed: dmUrl == null
                      ? null
                      : () async {
                          await ref.read(instagramDmLauncherProvider)(dmUrl);
                          if (!context.mounted || !_current(ref, lease)) return;
                        },
                  icon: const Icon(CraftskyIconsBold.externalLink),
                  label: Text(l10n.instagramOpenDm),
                ),
                const SizedBox(height: 8),
                TextButton(
                  onPressed: isBusy || actions.isBusy
                      ? null
                      : () => actions.run(
                          _InstagramAction.cancel,
                          () => ref
                              .read(
                                instagramVerificationProvider(lease).notifier,
                              )
                              .cancel(),
                        ),
                  child: ActionProgress(
                    isLoading: actions.isRunning(_InstagramAction.cancel),
                    child: Text(l10n.instagramCancelVerification),
                  ),
                ),
              ],
            ),
            if (hasError) ...[
              const SizedBox(height: 8),
              Text(l10n.instagramActionError),
              TextButton(
                onPressed: () => ref
                    .read(instagramVerificationProvider(lease).notifier)
                    .poll(),
                child: Text(l10n.instagramRetry),
              ),
            ],
          ],
        );
      },
    );
  }
}

class _LinkedAccountControls extends ConsumerWidget {
  const new({required this.lease, required this.account});

  final ActiveAccountLease lease;
  final InstagramAccountLink account;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final l10n = AppLocalizations.of(context);
    final notifier = ref.read(instagramAccountProvider(lease).notifier);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        _InstagramHandleText(
          username: account.username,
          localizedText: l10n.instagramLinkedAs(account.username),
        ),
        if (_hasOwnershipConflict(account)) ...[
          const SizedBox(height: 8),
          Text(l10n.instagramConflictPending),
        ],
        if (_needsReactivation(account)) ...[
          SizedBox(height: spacing.sp2),
          Text(l10n.instagramReactivateAccountDisclosure),
          FilledButton(
            onPressed: notifier.reactivate,
            child: Text(l10n.instagramReactivateAccount),
          ),
        ] else ...[
          SizedBox(height: spacing.sp2),
          MergeSemantics(
            child: Row(
              children: [
                Expanded(
                  child: Text(
                    l10n.instagramDiscoverableLabel,
                    style: theme.textTheme.titleMedium,
                  ),
                ),
                SizedBox(width: spacing.sp3),
                Switch(
                  key: const Key('instagram-discoverable-switch'),
                  value: account.discoverable,
                  onChanged: (value) => notifier.setDiscoverable(value: value),
                ),
              ],
            ),
          ),
        ],
      ],
    );
  }
}

class _RevokeInstagramVerificationButton extends ConsumerWidget {
  const new({required this.lease});

  final ActiveAccountLease lease;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context);
    final notifier = ref.read(instagramAccountProvider(lease).notifier);
    return TextButton.icon(
      key: const Key('instagram-revoke-verification'),
      onPressed: () => showCraftskyDestructiveConfirmDialog(
        context,
        title: l10n.instagramRevokeAccountConfirmTitle,
        message: l10n.instagramRevokeAccountConfirmMessage,
        confirmLabel: l10n.instagramRevokeAccount,
        onConfirm: () async {
          final revoked = await notifier.revoke();
          if (!revoked) {
            throw DiagnosticStateError(
              'Instagram verification revocation failed',
            );
          }
        },
      ),
      style: TextButton.styleFrom(
        foregroundColor: theme.colorScheme.error,
        iconColor: theme.colorScheme.error,
      ),
      icon: const Icon(CraftskyIconsBold.unlink),
      label: Text(l10n.instagramRevokeAccount),
    );
  }
}

class _ImportComposerCard extends ConsumerStatefulWidget {
  const new({required this.lease});

  final ActiveAccountLease lease;

  @override
  ConsumerState<_ImportComposerCard> createState() =>
      _ImportComposerCardState();
}

class _ImportComposerCardState extends ConsumerState<_ImportComposerCard> {
  bool _busy = false;
  InstagramImportParseErrorCode? _parseError;
  bool _filePickerFailed = false;

  @override
  Widget build(BuildContext context) {
    final importsProvider = instagramImportsProvider(widget.lease);
    final imports = ref.watch(importsProvider);
    final ready = imports.hasValue;
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final l10n = AppLocalizations.of(context);
    return CraftskyCard(
      key: const Key('instagram-import-composer-card'),
      padding: EdgeInsets.all(spacing.sp4),
      clipBehavior: Clip.none,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            l10n.instagramImportJsonDescription,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
          SizedBox(height: spacing.sp3),
          FilledButton.icon(
            onPressed: ready && !_busy ? _pickExport : null,
            icon: ActionProgress(
              isLoading: _busy,
              child: const Icon(CraftskyIconsBold.openFile),
            ),
            label: Text(l10n.instagramImportSelectJson),
          ),
          if (imports.hasError) ...[
            SizedBox(height: spacing.sp2),
            Text(l10n.instagramImportsLoadError),
            TextButton(
              key: const Key('instagram-import-readiness-retry'),
              onPressed: () => unawaited(
                ref.read(importsProvider.notifier).refresh(),
              ),
              child: Text(l10n.instagramRetry),
            ),
          ],
          if (_parseError case final error?) ...[
            const SizedBox(height: 8),
            Text(_parseErrorMessage(l10n, error)),
          ],
          if (_filePickerFailed) ...[
            const SizedBox(height: 8),
            Text(l10n.instagramImportFilePickerError),
          ],
          SizedBox(height: spacing.sp2),
          Text(
            l10n.instagramImportSuggestionDisclosure,
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
        ],
      ),
    );
  }

  Future<void> _pickExport() async {
    final capturedLease = widget.lease;
    if (_busy || !_current(ref, capturedLease)) return;
    setState(() => _busy = true);
    try {
      final parsed = await ref.read(instagramExportFilePickerProvider)();
      if (!mounted ||
          capturedLease != widget.lease ||
          !_current(ref, capturedLease)) {
        return;
      }
      if (parsed == null) return;
      setState(() {
        _parseError = null;
        _filePickerFailed = false;
      });
      final result = await ref
          .read(instagramImportsProvider(capturedLease).notifier)
          .create(
            InstagramImportRequest(
              sourceType: InstagramImportSourceType.instagramJson,
              entries: parsed.entries,
            ),
          );
      if (!mounted || !_current(ref, capturedLease)) return;
      final l10n = AppLocalizations.of(context);
      if (result == null) {
        context.showError(l10n.instagramImportUploadError);
      } else {
        context.showInfo(l10n.instagramImportUploadSuccess);
      }
    } on InstagramImportParseException catch (error) {
      if (!mounted || !_current(ref, capturedLease)) return;
      setState(() {
        _parseError = error.code;
        _filePickerFailed = false;
      });
    } on Object {
      if (!mounted || !_current(ref, capturedLease)) return;
      setState(() {
        _parseError = null;
        _filePickerFailed = true;
      });
    } finally {
      if (mounted && capturedLease == widget.lease) {
        setState(() => _busy = false);
      }
    }
  }
}

class _ImportsCard extends ConsumerWidget {
  const new({required this.lease});

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
      clipBehavior: Clip.none,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _CardHeading(
            icon: CraftskyIcons.projectCount,
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
                ? CraftskyEmptyState(
                    icon: CraftskyIcons.history,
                    title: l10n.instagramImportsEmpty,
                    subtitle: l10n.instagramImportCounts(0),
                  )
                : Column(
                    children: [
                      for (final item in page.items)
                        _ImportRow(
                          key: ValueKey(item.importId),
                          lease: lease,
                          item: item,
                        ),
                      if (page.cursor != null)
                        AsyncActionBuilder(
                          builder: (context, pagination) => TextButton(
                            onPressed: pagination.isBusy
                                ? null
                                : () => pagination.run(
                                    _InstagramAction.more,
                                    () async {
                                      await ref
                                          .read(
                                            instagramImportsProvider(
                                              lease,
                                            ).notifier,
                                          )
                                          .loadMore();
                                    },
                                  ),
                            child: ActionProgress(
                              isLoading: pagination.isBusy,
                              child: Text(l10n.instagramLoadMore),
                            ),
                          ),
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
  const new({required this.lease, required this.item, super.key});

  final ActiveAccountLease lease;
  final InstagramImportSummary item;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return AsyncActionBuilder(
      builder: (context, actions) {
        final theme = Theme.of(context);
        final l10n = AppLocalizations.of(context);
        final notifier = ref.read(instagramImportsProvider(lease).notifier);
        return Padding(
          padding: const EdgeInsets.symmetric(vertical: 8),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          _importSourceLabel(l10n, item.sourceType),
                          style: theme.textTheme.titleMedium,
                        ),
                        Text(
                          l10n.instagramImportCounts(item.followingCount),
                        ),
                        if (item.state ==
                            InstagramImportState.membershipInactive) ...[
                          const SizedBox(height: 4),
                          Text(l10n.instagramImportReactivationDisclosure),
                          FilledButton.tonal(
                            onPressed: actions.isBusy
                                ? null
                                : () => actions.run(
                                    _InstagramAction.reactivate,
                                    () => _runImportAction(
                                      context,
                                      ref,
                                      lease,
                                      () => notifier.reactivate(item.importId),
                                    ),
                                  ),
                            child: ActionProgress(
                              isLoading: actions.isRunning(
                                _InstagramAction.reactivate,
                              ),
                              child: Text(l10n.instagramImportReactivate),
                            ),
                          ),
                        ],
                      ],
                    ),
                  ),
                  IconButton(
                    onPressed: actions.isBusy
                        ? null
                        : () => actions.run(
                            _InstagramAction.delete,
                            () => _runImportAction(
                              context,
                              ref,
                              lease,
                              () => notifier.delete(item.importId),
                            ),
                          ),
                    style: IconButton.styleFrom(
                      foregroundColor: theme.colorScheme.error,
                    ),
                    icon: ActionProgress(
                      isLoading: actions.isRunning(_InstagramAction.delete),
                      child: const Icon(CraftskyIconsBold.delete),
                    ),
                    tooltip: l10n.instagramImportDelete,
                  ),
                ],
              ),
              const Divider(),
            ],
          ),
        );
      },
    );
  }
}

class _SuggestionsCard extends ConsumerWidget {
  const new({required this.lease, this.onSuggestionTap});

  final ActiveAccountLease lease;
  final ValueChanged<InstagramSuggestion>? onSuggestionTap;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final l10n = AppLocalizations.of(context);
    final suggestions = ref.watch(instagramSuggestionsProvider(lease));
    return CraftskyCard(
      key: const Key('instagram-suggestions-card'),
      padding: EdgeInsets.all(spacing.sp4),
      clipBehavior: Clip.none,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            l10n.instagramSuggestionsDescription,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
          SizedBox(height: spacing.sp3),
          suggestions.when(
            loading: () => const Center(child: StitchProgressIndicator()),
            error: (_, _) => _InlineRetry(
              message: l10n.instagramSuggestionsLoadError,
              onRetry: () => ref
                  .read(instagramSuggestionsProvider(lease).notifier)
                  .refresh(),
            ),
            data: (value) => Column(
              children: [
                if (value.items.isEmpty)
                  CraftskyEmptyState(
                    icon: CraftskyIcons.findPeople,
                    title: l10n.instagramSuggestionsEmpty,
                    subtitle: l10n.instagramSuggestionsEmptyHint,
                  )
                else
                  for (final suggestion in value.items)
                    _SuggestionRow(
                      key: ValueKey(suggestion.suggestionId),
                      lease: lease,
                      suggestion: suggestion,
                      busy: value.busyIds.contains(suggestion.suggestionId),
                      onTap: onSuggestionTap,
                    ),
                if (value.cursor != null)
                  AsyncActionBuilder(
                    builder: (context, pagination) => TextButton(
                      onPressed: pagination.isBusy
                          ? null
                          : () =>
                                pagination.run(_InstagramAction.more, () async {
                                  await ref
                                      .read(
                                        instagramSuggestionsProvider(
                                          lease,
                                        ).notifier,
                                      )
                                      .loadMore();
                                }),
                      child: ActionProgress(
                        isLoading: pagination.isBusy,
                        child: Text(l10n.instagramLoadMore),
                      ),
                    ),
                  ),
                if (value.hasActionError) ...[
                  SizedBox(height: spacing.sp2),
                  Text(l10n.instagramSuggestionsActionError),
                ],
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _SuggestionRow extends ConsumerWidget {
  const new({
    required this.lease,
    required this.suggestion,
    required this.busy,
    this.onTap,
    super.key,
  });

  final ActiveAccountLease lease;
  final InstagramSuggestion suggestion;
  final bool busy;
  final ValueChanged<InstagramSuggestion>? onTap;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return AsyncActionBuilder(
      builder: (context, actions) {
        final theme = Theme.of(context);
        final spacing = theme.extension<SpacingTheme>()!;
        final l10n = AppLocalizations.of(context);
        final target = suggestion.target;
        final notifier = ref.read(instagramSuggestionsProvider(lease).notifier);
        return Padding(
          key: ValueKey('instagram-suggestion-${suggestion.suggestionId}'),
          padding: EdgeInsets.only(bottom: spacing.sp3),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              InkWell(
                onTap: busy || actions.isBusy || onTap == null
                    ? null
                    : () {
                        if (!_current(ref, lease)) return;
                        onTap!(suggestion);
                      },
                child: Row(
                  children: [
                    ProfileAvatar(
                      seed: target.displayLabel,
                      avatarUrl: target.avatar,
                      size: ProfileAvatarSize.small,
                    ),
                    SizedBox(width: spacing.sp3),
                    Expanded(
                      child: CompactProfileIdentity(
                        handle: target.handle,
                        displayName: target.displayName,
                        crafts: target.crafts,
                      ),
                    ),
                  ],
                ),
              ),
              SizedBox(height: spacing.sp2),
              Row(
                children: [
                  Expanded(
                    child: FilledButton(
                      onPressed: busy || actions.isBusy
                          ? null
                          : () => actions.run(
                              _InstagramAction.follow,
                              () => _runSuggestionAction(
                                context,
                                ref,
                                lease,
                                () => notifier.accept(suggestion.suggestionId),
                              ),
                            ),
                      child: ActionProgress(
                        isLoading: actions.isRunning(_InstagramAction.follow),
                        child: Text(l10n.instagramSuggestionFollow),
                      ),
                    ),
                  ),
                  SizedBox(width: spacing.sp2),
                  TextButton(
                    onPressed: busy || actions.isBusy
                        ? null
                        : () => actions.run(
                            _InstagramAction.dismiss,
                            () => _runSuggestionAction(
                              context,
                              ref,
                              lease,
                              () => notifier.dismiss(suggestion.suggestionId),
                            ),
                          ),
                    child: ActionProgress(
                      isLoading: actions.isRunning(_InstagramAction.dismiss),
                      child: Text(l10n.instagramSuggestionDismiss),
                    ),
                  ),
                ],
              ),
              const Divider(),
            ],
          ),
        );
      },
    );
  }
}

class _RetryVerification extends StatelessWidget {
  const new({required this.message, required this.onRetry});

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
  const new({required this.icon, required this.title});

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
  const new({required this.message, required this.onRetry});

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
  const new();

  @override
  Widget build(BuildContext context) {
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    return CraftskyCard(
      padding: EdgeInsets.all(spacing.sp5),
      clipBehavior: Clip.none,
      child: const Center(child: StitchProgressIndicator()),
    );
  }
}

class _ErrorCard extends StatelessWidget {
  const new({required this.onRetry});

  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    return CraftskyCard(
      padding: EdgeInsets.all(spacing.sp4),
      clipBehavior: Clip.none,
      child: _InlineRetry(
        message: AppLocalizations.of(context).instagramMigrationLoadError,
        onRetry: onRetry,
      ),
    );
  }
}

class _CenteredMessage extends StatelessWidget {
  const new(this.message);

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
    ActiveAccountOperation.fromWidget(ref, lease).isCurrent;

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

Future<void> _runSuggestionAction(
  BuildContext context,
  WidgetRef ref,
  ActiveAccountLease lease,
  Future<bool> Function() action,
) async {
  final succeeded = await action();
  if (!context.mounted || !_current(ref, lease) || succeeded) return;
  context.showError(
    AppLocalizations.of(context).instagramSuggestionsActionError,
  );
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
  InstagramImportParseErrorCode.invalidArchive =>
    l10n.instagramImportInvalidArchive,
  InstagramImportParseErrorCode.archiveTooLarge =>
    l10n.instagramImportArchiveTooLarge,
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

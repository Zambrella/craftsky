import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/active_account_initialization_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/onboarding/models/onboarding_action_state.dart';
import 'package:craftsky_app/onboarding/models/onboarding_flow_state.dart';
import 'package:craftsky_app/onboarding/providers/onboarding_flow_provider.dart';
import 'package:craftsky_app/onboarding/widgets/onboarding_bottom_action.dart';
import 'package:craftsky_app/onboarding/widgets/onboarding_crafts_step.dart';
import 'package:craftsky_app/onboarding/widgets/onboarding_instagram_step.dart';
import 'package:craftsky_app/onboarding/widgets/onboarding_profile_step.dart';
import 'package:craftsky_app/onboarding/widgets/onboarding_progress.dart';
import 'package:craftsky_app/profile/data/profile_field_constraints.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class OnboardingPage extends ConsumerWidget {
  const OnboardingPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final lease = ref.watch(activeAccountInitializationProvider).value?.lease;
    if (lease == null) {
      return const Scaffold(body: Center(child: StitchProgressIndicator()));
    }
    final provider = onboardingFlowProvider(lease);
    final flow = ref.watch(provider);
    final prefillAppBar = AppBar(
      title: Text(l10n.onboardingTitle),
      actions: [
        TextButton(
          onPressed: () => unawaited(ref.read(provider.notifier).complete()),
          child: Text(l10n.onboardingSkip),
        ),
      ],
    );
    return switch (flow) {
      AsyncData(:final value) => _OnboardingFlowScaffold(
        lease: lease,
        state: value,
      ),
      AsyncError() => Scaffold(
        appBar: prefillAppBar,
        body: Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(l10n.onboardingLoadError),
              TextButton(
                onPressed: () => ref.invalidate(onboardingFlowProvider(lease)),
                child: Text(l10n.onboardingRetry),
              ),
            ],
          ),
        ),
      ),
      _ => Scaffold(
        appBar: prefillAppBar,
        body: const Center(child: StitchProgressIndicator()),
      ),
    };
  }
}

class _OnboardingFlowScaffold extends ConsumerWidget {
  const _OnboardingFlowScaffold({required this.lease, required this.state});

  final ActiveAccountLease lease;
  final OnboardingFlowState state;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final provider = onboardingFlowProvider(lease);
    final notifier = ref.read(provider.notifier);
    final dirty = switch (state.step) {
      OnboardingStep.profile => state.identityDirty,
      OnboardingStep.crafts => state.craftsDirty,
      OnboardingStep.instagram => false,
    };
    final valid =
        state.identity.displayName.length <= profileDisplayNameMaxLength &&
        state.identity.bio.length <= profileBioMaxLength &&
        !state.uploadingAvatar &&
        !state.avatarUploadFailed;
    final action = deriveOnboardingActionState(
      step: state.step,
      dirty: dirty,
      valid: valid,
      saving: state.saving,
    );

    void previous() {
      if (action.canGoBack) notifier.previous();
    }

    return PopScope<Object?>(
      canPop: false,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) previous();
      },
      child: Scaffold(
        appBar: AppBar(
          leading: state.step == OnboardingStep.profile
              ? null
              : IconButton(
                  onPressed: action.canGoBack ? previous : null,
                  icon: const Icon(CraftskyIconsBold.back),
                ),
          title: Text(l10n.onboardingTitle),
          actions: [
            TextButton(
              onPressed: action.canSkip
                  ? () => unawaited(notifier.complete())
                  : null,
              child: Text(l10n.onboardingSkip),
            ),
          ],
        ),
        body: SafeArea(
          top: false,
          child: Column(
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(24, 12, 24, 0),
                child: OnboardingProgress(step: state.step),
              ),
              Expanded(
                child: SingleChildScrollView(
                  padding: EdgeInsets.fromLTRB(
                    24,
                    24,
                    24,
                    24 + MediaQuery.viewInsetsOf(context).bottom,
                  ),
                  child: Center(
                    child: ConstrainedBox(
                      constraints: const BoxConstraints(maxWidth: 720),
                      child: switch (state.step) {
                        OnboardingStep.profile => OnboardingProfileStep(
                          state: state,
                          onDisplayNameChanged: (value) =>
                              notifier.updateIdentity(displayName: value),
                          onBioChanged: (value) =>
                              notifier.updateIdentity(bio: value),
                          onPickAvatar: () => unawaited(notifier.pickAvatar()),
                        ),
                        OnboardingStep.crafts => OnboardingCraftsStep(
                          state: state,
                          onToggle: (craft) => notifier.toggleCraft(craft.id),
                        ),
                        OnboardingStep.instagram => OnboardingInstagramStep(
                          lease: lease,
                        ),
                      },
                    ),
                  ),
                ),
              ),
              if (state.saveError != null)
                Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 24),
                  child: Text(
                    l10n.onboardingSaveError,
                    style: TextStyle(
                      color: Theme.of(context).colorScheme.error,
                    ),
                  ),
                ),
              OnboardingBottomAction(
                state: action,
                onPressed: () {
                  switch (action.kind) {
                    case OnboardingActionKind.next:
                      notifier.next();
                    case OnboardingActionKind.saveAndNext:
                      unawaited(notifier.saveAndNext());
                    case OnboardingActionKind.finish:
                      unawaited(notifier.complete());
                  }
                },
              ),
            ],
          ),
        ),
      ),
    );
  }
}

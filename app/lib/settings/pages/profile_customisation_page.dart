import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/auth/providers/unsaved_work_guard_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/profile/providers/profile_customisation_provider.dart';
import 'package:craftsky_app/profile/widgets/profile_avatar.dart';
import 'package:craftsky_app/profile/widgets/profile_customisation_theme.dart';
import 'package:craftsky_app/profile/widgets/profile_header_background.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class ProfileCustomisationPage extends ConsumerWidget {
  const ProfileCustomisationPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final editor = ref.watch(profileCustomisationEditorProvider);
    ref.listen(profileCustomisationEditorProvider, (previous, next) {
      if (previous?.isLoading != true || previous?.value == null) return;
      if (next.hasError) {
        context.showError(l10n.profileCustomisationSaveError);
      } else if (next.hasValue && !next.isLoading) {
        context.showInfo(l10n.profileCustomisationSaved);
      }
    });

    final value = editor.value;
    if (value == null) {
      return Scaffold(
        appBar: AppBar(title: Text(l10n.profileCustomisationTitle)),
        body: editor.hasError
            ? Center(
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Text(l10n.profileCustomisationLoadError),
                    const SizedBox(height: 12),
                    FilledButton(
                      onPressed: () =>
                          ref.invalidate(profileCustomisationEditorProvider),
                      child: Text(l10n.profileCustomisationRetry),
                    ),
                  ],
                ),
              )
            : const Center(child: StitchProgressIndicator()),
      );
    }

    return _LoadedCustomisationPage(value: value, isSaving: editor.isLoading);
  }
}

class _LoadedCustomisationPage extends ConsumerStatefulWidget {
  const _LoadedCustomisationPage({required this.value, required this.isSaving});

  final ProfileCustomisationEditorState value;
  final bool isSaving;

  @override
  ConsumerState<_LoadedCustomisationPage> createState() =>
      _LoadedCustomisationPageState();
}

class _LoadedCustomisationPageState
    extends ConsumerState<_LoadedCustomisationPage> {
  AccountSessionLease? _unsavedOwner;
  UnsavedWorkRegistration? _unsavedRegistration;
  late final UnsavedWorkGuard _unsavedGuard;

  @override
  void initState() {
    super.initState();
    _unsavedGuard = ref.read(unsavedWorkGuardProvider);
  }

  @override
  void dispose() {
    _unsavedGuard.unregister(_unsavedRegistration);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    _ensureUnsavedWorkRegistration();
    final l10n = AppLocalizations.of(context);
    final notifier = ref.read(profileCustomisationEditorProvider.notifier);
    final draft = widget.value.draft;
    return PopScope<Object?>(
      canPop: !widget.value.isDirty || widget.isSaving,
      onPopInvokedWithResult: (didPop, _) async {
        if (didPop) return;
        await _confirmDiscardAndClose();
      },
      child: Scaffold(
        appBar: AppBar(title: Text(l10n.profileCustomisationTitle)),
        body: ListView(
          padding: const EdgeInsets.fromLTRB(16, 8, 16, 32),
          children: [
            Semantics(
              label: l10n.profileCustomisationPreview,
              container: true,
              child: ProfileCustomisationTheme(
                customisation: draft,
                child: SizedBox(
                  height: 160,
                  child: Stack(
                    fit: StackFit.expand,
                    alignment: Alignment.center,
                    children: [
                      ProfileHeaderBackground(customisation: draft),
                      Center(
                        child: ProfileAvatar(
                          seed: 'Craftsky',
                          size: ProfileAvatarSize.large,
                          customisation: draft,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
            const SizedBox(height: 24),
            _ChoiceGroup(
              label: l10n.profileCustomisationColour,
              values: profileColourCatalogue,
              selected: draft.colour,
              labels: _colourLabels(l10n),
              onSelected: notifier.selectColour,
            ),
            const SizedBox(height: 20),
            _ChoiceGroup(
              label: l10n.profileCustomisationBorder,
              values: profileBorderCatalogue,
              selected: draft.border,
              labels: _borderLabels(l10n),
              onSelected: notifier.selectBorder,
            ),
            const SizedBox(height: 20),
            _ChoiceGroup(
              label: l10n.profileCustomisationBackground,
              values: profileBackgroundCatalogue,
              selected: draft.background,
              labels: _backgroundLabels(l10n),
              onSelected: notifier.selectBackground,
            ),
            const SizedBox(height: 28),
            FilledButton(
              onPressed: widget.value.isDirty && !widget.isSaving
                  ? () => unawaited(notifier.save())
                  : null,
              child: widget.isSaving
                  ? const SizedBox.square(
                      dimension: 20,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : Text(l10n.profileCustomisationSave),
            ),
          ],
        ),
      ),
    );
  }

  void _ensureUnsavedWorkRegistration() {
    final owner = ref.read(sessionRegistryProvider).value?.activeLease?.session;
    if (owner == null || owner == _unsavedOwner) return;
    _unsavedOwner = owner;
    _unsavedRegistration = _unsavedGuard.replace(
      _unsavedRegistration,
      owner: owner,
      isDirty: () => mounted && widget.value.isDirty,
      confirmAndClose: _confirmDiscardAndClose,
    );
  }

  Future<bool> _confirmDiscardAndClose() async {
    if (!mounted) return true;
    final l10n = AppLocalizations.of(context);
    final discard = await showCraftskyConfirmDialog(
      context,
      title: l10n.profileCustomisationDiscardTitle,
      message: l10n.profileCustomisationDiscardMessage,
      confirmLabel: l10n.editProfileDiscardConfirm,
      cancelLabel: l10n.editProfileDiscardCancel,
    );
    if (!discard || !mounted) return false;
    ref.read(profileCustomisationEditorProvider.notifier).discard();
    Navigator.of(context).pop();
    await Future<void>.delayed(Duration.zero);
    return true;
  }
}

class _ChoiceGroup extends StatelessWidget {
  const _ChoiceGroup({
    required this.label,
    required this.values,
    required this.selected,
    required this.onSelected,
    this.labels = const {},
  });

  final String label;
  final List<String> values;
  final String selected;
  final ValueChanged<String> onSelected;
  final Map<String, String> labels;

  @override
  Widget build(BuildContext context) => Semantics(
    label: label,
    container: true,
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(label, style: Theme.of(context).textTheme.titleMedium),
        const SizedBox(height: 8),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            for (final value in values)
              ChoiceChip(
                label: Text(labels[value] ?? value),
                selected: selected == value,
                onSelected: (_) => onSelected(value),
              ),
          ],
        ),
      ],
    ),
  );
}

Map<String, String> _colourLabels(AppLocalizations l10n) => {
  'cobalt': l10n.profileCustomisationColourCobalt,
  'orchid': l10n.profileCustomisationColourOrchid,
  'rose': l10n.profileCustomisationColourRose,
  'amber': l10n.profileCustomisationColourAmber,
  'lime': l10n.profileCustomisationColourLime,
  'teal': l10n.profileCustomisationColourTeal,
};

Map<String, String> _borderLabels(AppLocalizations l10n) => {
  'thin': l10n.profileCustomisationBorderThin,
  'medium': l10n.profileCustomisationBorderMedium,
  'thick': l10n.profileCustomisationBorderThick,
};

Map<String, String> _backgroundLabels(AppLocalizations l10n) => {
  'none': l10n.profileCustomisationNone,
  'bayerdark': l10n.profileCustomisationBackgroundDither,
  'cubedark': l10n.profileCustomisationBackgroundGrid,
  'dotcrossdark': l10n.profileCustomisationBackgroundCrossStitch,
  'scallopdark': l10n.profileCustomisationBackgroundScallops,
  'skewdark': l10n.profileCustomisationBackgroundDiagonalWeave,
  'x2': l10n.profileCustomisationBackgroundCrosshatch,
};

import 'dart:async';
import 'dart:typed_data';

import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/data/crafts_catalog.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/profile_image_picker_provider.dart';
import 'package:craftsky_app/profile/providers/save_profile_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/profile/widgets/edit_profile_banner_avatar.dart';
import 'package:craftsky_app/profile/widgets/edit_profile_crafts_picker.dart';
import 'package:craftsky_app/profile/widgets/profile_page_error.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_form_builder/flutter_form_builder.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:form_builder_validators/form_builder_validators.dart';

/// Field names used by the [FormBuilder] state. Centralised so the
/// build site, the validators, and the save-time reads can't drift
/// from one another.
const _fieldDisplayName = 'displayName';
const _fieldBio = 'bio';
const _fieldCrafts = 'crafts';

/// Bsky's `app.bsky.actor.profile.displayName` lexicon caps display
/// names at 64 graphemes. We approximate with code-unit length here —
/// the server has the final say.
const _displayNameMaxLength = 64;

/// Bsky's `app.bsky.actor.profile.description` lexicon caps the bio at
/// 256 graphemes. Same code-unit caveat as [_displayNameMaxLength].
const _bioMaxLength = 256;

/// Opens the profile-edit screen as a full-screen Material dialog.
///
/// Uses `MaterialPageRoute(fullscreenDialog: true)` for two reasons:
/// 1. The AppBar's auto-injected leading becomes a `CloseButton` (X)
///    instead of a back arrow — the "close button for free" that
///    matches Material's intent for temporary task screens.
/// 2. The transition slides up from the bottom rather than the
///    platform's default page-push, signalling "this is a modal task,
///    not a navigation step".
///
/// Profile editing isn't a core navigation surface, so it doesn't
/// live as a named route — callers reach it through this helper.
/// Discard semantics still flow through [PopScope] inside the dialog,
/// so the close button and system-back with unsaved changes prompt a
/// confirm dialog before the pop completes.
Future<void> showEditProfileDialog(BuildContext context) {
  return Navigator.of(context, rootNavigator: true).push<void>(
    MaterialPageRoute<void>(
      fullscreenDialog: true,
      builder: (_) => const EditProfileDialog(),
    ),
  );
}

/// Full-screen profile-edit screen. Pushed as a `fullscreenDialog`
/// `MaterialPageRoute` (see [showEditProfileDialog]); not a named
/// route. Resolves the signed-in user's handle, loads their profile,
/// and hands a snapshot to [_EditProfileForm] which owns the form
/// state.
///
/// Loading and error states render a minimal Scaffold + AppBar so the
/// route's auto-injected `CloseButton` (the "free" close affordance
/// from `fullscreenDialog: true`) is reachable even before the profile
/// resolves.
class EditProfileDialog extends ConsumerWidget {
  const EditProfileDialog({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final auth = ref.watch(authSessionProvider);
    final myHandle = switch (auth) {
      AsyncData(value: SignedIn(:final handle)) => handle,
      _ => null,
    };

    if (myHandle == null) return const _EditProfileLoadingScaffold();

    final profileAsync = ref.watch(userProfileProvider(myHandle));
    return switch (profileAsync) {
      AsyncValue(:final value?) => _EditProfileForm(profile: value),
      AsyncError(:final error) => Scaffold(
        appBar: AppBar(),
        body: ProfilePageError(
          error: error,
          onRetry: () => ref.invalidate(userProfileProvider(myHandle)),
        ),
      ),
      _ => const _EditProfileLoadingScaffold(),
    };
  }
}

class _EditProfileLoadingScaffold extends StatelessWidget {
  const _EditProfileLoadingScaffold();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(),
      body: const Center(child: StitchProgressIndicator()),
    );
  }
}

/// Stateful body of the edit sheet. Wires a [FormBuilder] over the
/// three editable fields (display name, bio, crafts) so validation,
/// dirty tracking, and read-time access all flow through a single
/// [GlobalKey<FormBuilderState>].
class _EditProfileForm extends ConsumerStatefulWidget {
  const _EditProfileForm({required this.profile});

  /// Snapshot of the signed-in user's profile at the time the sheet
  /// opened. Treated as the "original" against which the form's diff
  /// is computed — we don't re-seed the form if the underlying provider
  /// changes mid-edit, since that would clobber the user's typing.
  final Profile profile;

  @override
  ConsumerState<_EditProfileForm> createState() => _EditProfileFormState();
}

class _EditProfileFormState extends ConsumerState<_EditProfileForm> {
  final _formKey = GlobalKey<FormBuilderState>();

  /// Text controllers are kept in state because [BrandTextField] needs
  /// one to seed its initial display text — `TextField` doesn't expose
  /// an `initialValue` parameter. The form value is kept in sync via
  /// each field's `onChanged: field.didChange`.
  late final TextEditingController _displayNameController;
  late final TextEditingController _bioController;

  /// Focus nodes are owned at the page level and handed to **both** the
  /// [FormBuilderField] and the [BrandTextField] for each text input.
  /// Sharing the node is critical: `FormBuilderFieldState.validate`
  /// auto-grabs focus when a field becomes invalid unless one of the
  /// form's registered fields already has focus, and it checks each
  /// field's own `effectiveFocusNode` for that. If we let the form
  /// allocate its own node and BrandTextField create its own internally,
  /// they're disjoint, and every keystroke that fails validation steals
  /// focus mid-typing.
  late final FocusNode _displayNameFocusNode;
  late final FocusNode _bioFocusNode;

  /// Initial set of selected crafts — captured once at mount and used
  /// by [_hasChanges] to decide whether the form is dirty. The current
  /// selection lives in the form's value under [_fieldCrafts].
  late final Set<Craft> _initialSelectedCrafts;

  /// Crafts on the profile that don't map to any [Craft] enum entry.
  /// Preserved verbatim and re-attached on save so a viewer on an older
  /// build can't accidentally drop tags it doesn't recognise from a
  /// newer server.
  late final List<String> _unknownCrafts;

  _ProfileImageDraft _avatarDraft = const _ProfileImageDraft();
  _ProfileImageDraft _bannerDraft = const _ProfileImageDraft();

  @override
  void initState() {
    super.initState();
    _displayNameController = TextEditingController(
      text: widget.profile.displayName,
    );
    _bioController = TextEditingController(
      text: widget.profile.description,
    );
    _displayNameFocusNode = FocusNode(debugLabel: _fieldDisplayName);
    _bioFocusNode = FocusNode(debugLabel: _fieldBio);

    final selected = <Craft>{};
    final unknown = <String>[];
    for (final id in widget.profile.crafts) {
      final craft = Craft.fromId(id);
      if (craft != null) {
        selected.add(craft);
      } else {
        unknown.add(id);
      }
    }
    _initialSelectedCrafts = Set.unmodifiable(selected);
    _unknownCrafts = List.unmodifiable(unknown);
  }

  @override
  void dispose() {
    _displayNameController.dispose();
    _bioController.dispose();
    _displayNameFocusNode.dispose();
    _bioFocusNode.dispose();
    super.dispose();
  }

  /// Currently-typed values, as the [FormBuilder] sees them. Returns an
  /// empty map before the form has built so callers can fall back to
  /// "no changes" without null-checking.
  Map<String, dynamic> get _formValues =>
      _formKey.currentState?.instantValue ?? const {};

  bool get _hasChanges {
    final values = _formValues;
    if (values.isEmpty) return false;

    final initialDisplayName = widget.profile.displayName ?? '';
    final initialBio = widget.profile.description ?? '';
    if ((values[_fieldDisplayName] as String? ?? '') != initialDisplayName) {
      return true;
    }
    if ((values[_fieldBio] as String? ?? '') != initialBio) return true;

    final initialIds = _initialSelectedCrafts.map((c) => c.id).toSet();
    final currentIds = (values[_fieldCrafts] as Set<Craft>? ?? const <Craft>{})
        .map((c) => c.id)
        .toSet();
    if (currentIds.length != initialIds.length) return true;
    if (!currentIds.containsAll(initialIds)) return true;
    if (_avatarDraft.changed || _bannerDraft.changed) return true;
    return false;
  }

  bool get _imageUploadInFlight =>
      _avatarDraft.isUploading || _bannerDraft.isUploading;

  bool get _imageUploadHasError =>
      _avatarDraft.hasError || _bannerDraft.hasError;

  /// Validates the form and dispatches the save. Sends the **full**
  /// current form state, not a diff — atproto profile records are
  /// atomic, so any field absent from the PUT gets cleared on the PDS.
  /// See `ProfileApiClient.updateMyProfile` for the rationale.
  Future<void> _save() async {
    final form = _formKey.currentState;
    if (form == null) return;
    if (!form.saveAndValidate()) return;
    if (!_hasChanges) return;

    final values = form.value;
    final selectedCrafts =
        (values[_fieldCrafts] as Set<Craft>?) ?? const <Craft>{};

    // Re-attach the preserved unknowns so a save from this client
    // doesn't strip tags a newer server has added.
    final craftsPayload = <String>[
      ...selectedCrafts.map((c) => c.id),
      ..._unknownCrafts,
    ];
    final description = (values[_fieldBio] as String? ?? '').trim();

    unawaited(
      ref
          .read(saveProfileProvider.notifier)
          .save(
            displayName: (values[_fieldDisplayName] as String? ?? '').trim(),
            description: description,
            crafts: craftsPayload,
            avatar: _avatarDraft.uploaded?.blob,
            banner: _bannerDraft.uploaded?.blob,
          ),
    );
  }

  Future<void> _pickProfileImage(_ProfileImageKind kind) async {
    if (_imageUploadInFlight) return;
    final l10n = AppLocalizations.of(context);
    try {
      final result = await ref
          .read(profileImagePickerProvider)
          .pickAndUpload(
            onPreviewReady: (bytes) {
              if (!mounted) return;
              setState(
                () => _setImageDraft(
                  kind,
                  _ProfileImageDraft.uploading(bytes),
                ),
              );
            },
          );
      if (result == null || !mounted) return;
      setState(
        () => _setImageDraft(
          kind,
          _ProfileImageDraft.uploaded(result.previewBytes, result.uploaded),
        ),
      );
    } on Object {
      if (!mounted) return;
      setState(() => _setImageDraft(kind, _ProfileImageDraft.failed()));
      context.showError(l10n.editProfilePhotoUploadError);
    }
  }

  void _setImageDraft(_ProfileImageKind kind, _ProfileImageDraft draft) {
    switch (kind) {
      case _ProfileImageKind.avatar:
        _avatarDraft = draft;
      case _ProfileImageKind.banner:
        _bannerDraft = draft;
    }
  }

  Future<bool> _confirmDiscard() async {
    final l10n = AppLocalizations.of(context);
    return showCraftskyConfirmDialog(
      context,
      title: l10n.editProfileDiscardTitle,
      message: l10n.editProfileDiscardMessage,
      confirmLabel: l10n.editProfileDiscardConfirm,
      cancelLabel: l10n.editProfileDiscardCancel,
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final l10n = AppLocalizations.of(context);
    final saveState = ref.watch(saveProfileProvider);
    final isSaving = saveState is AsyncLoading;

    // Handle the (loading -> data) and (loading -> error) transitions.
    // Success closes the sheet; failure surfaces a snackbar and leaves
    // the form untouched so the user can retry. Per the Riverpod rules,
    // listeners go in build (not initState).
    ref.listen<AsyncValue<Profile?>>(saveProfileProvider, (prev, next) {
      switch ((prev, next)) {
        case (AsyncLoading(), AsyncData(value: final _?)):
          if (Navigator.of(context).canPop()) {
            Navigator.of(context).pop();
          }
        case (AsyncLoading(), AsyncError()):
          context.showError(l10n.editProfileSaveError);
        case _:
          break;
      }
    });

    // `isValid` is true on first build (no errors yet) and flips false
    // as soon as a validator fails — autovalidateMode keeps it in sync
    // with the user's typing. Combined with `_hasChanges` it gives the
    // save button the strict "dirty + valid + not in flight" gate.
    final isFormValid = _formKey.currentState?.isValid ?? true;
    final canSave =
        _hasChanges &&
        isFormValid &&
        !isSaving &&
        !_imageUploadInFlight &&
        !_imageUploadHasError;

    return PopScope<Object?>(
      canPop: !_hasChanges || isSaving,
      onPopInvokedWithResult: (didPop, _) async {
        if (didPop) return;
        final discard = await _confirmDiscard();
        if (!discard) return;
        if (!context.mounted) return;
        Navigator.of(context).pop();
      },
      child: Scaffold(
        backgroundColor: swatches.paper,
        // Inside a `fullscreenDialog: true` route, the AppBar's
        // automatically-implied leading resolves to a `CloseButton`
        // (X) instead of a back arrow — that's the "close button for
        // free" Material gives us for temporary task screens.
        appBar: AppBar(
          title: Text(l10n.editProfileTitle),
          actions: [
            Padding(
              padding: EdgeInsets.only(right: spacing.sp3),
              child: _SaveAction(
                onPressed: canSave ? () => unawaited(_save()) : null,
                isSaving: isSaving,
              ),
            ),
          ],
        ),
        body: FormBuilder(
          key: _formKey,
          // Re-render on any field change so the save button picks up
          // the dirty/valid state without us hand-rolling listeners
          // per controller.
          onChanged: () {
            if (mounted) setState(() {});
          },
          // Validate as the user types so length errors surface
          // beneath the field they refer to, not only on save.
          autovalidateMode: AutovalidateMode.onUserInteraction,
          child: SingleChildScrollView(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                EditProfileBannerAvatar(
                  profile: widget.profile,
                  bannerColor: swatches.clay,
                  avatarPreviewBytes: _avatarDraft.previewBytes,
                  bannerPreviewBytes: _bannerDraft.previewBytes,
                  avatarUploading: _avatarDraft.isUploading,
                  bannerUploading: _bannerDraft.isUploading,
                  avatarError: _avatarDraft.hasError,
                  bannerError: _bannerDraft.hasError,
                  onPickAvatar: isSaving
                      ? null
                      : () => unawaited(
                          _pickProfileImage(_ProfileImageKind.avatar),
                        ),
                  onPickBanner: isSaving
                      ? null
                      : () => unawaited(
                          _pickProfileImage(_ProfileImageKind.banner),
                        ),
                ),
                Padding(
                  padding: EdgeInsets.fromLTRB(
                    spacing.sp4,
                    spacing.sp4,
                    spacing.sp4,
                    spacing.sp6,
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      FormBuilderField<String>(
                        name: _fieldDisplayName,
                        // Shared node — see _displayNameFocusNode docstring
                        // for why this matters.
                        focusNode: _displayNameFocusNode,
                        initialValue: widget.profile.displayName ?? '',
                        // `checkNullOrEmpty: false` — form_builder_validators
                        // 11.x's BaseValidator treats empty/null as a hard
                        // failure by default, which is wrong for a
                        // length-cap validator (empty is a valid display
                        // name).
                        validator: FormBuilderValidators.maxLength(
                          _displayNameMaxLength,
                          errorText: l10n.editProfileDisplayNameTooLong,
                          checkNullOrEmpty: false,
                        ),
                        builder: (field) => BrandTextField(
                          label: l10n.editProfileDisplayNameLabel,
                          controller: _displayNameController,
                          focusNode: _displayNameFocusNode,
                          hintText: l10n.editProfileDisplayNameHint,
                          textInputAction: TextInputAction.next,
                          enabled: !isSaving,
                          onChanged: field.didChange,
                          errorText: field.errorText,
                        ),
                      ),
                      SizedBox(height: spacing.sp5),
                      FormBuilderField<String>(
                        name: _fieldBio,
                        focusNode: _bioFocusNode,
                        initialValue: widget.profile.description ?? '',
                        validator: FormBuilderValidators.maxLength(
                          _bioMaxLength,
                          errorText: l10n.editProfileBioTooLong,
                          checkNullOrEmpty: false,
                        ),
                        builder: (field) => BrandTextField(
                          label: l10n.editProfileBioLabel,
                          controller: _bioController,
                          focusNode: _bioFocusNode,
                          hintText: l10n.editProfileBioHint,
                          minLines: 3,
                          maxLines: 6,
                          keyboardType: TextInputType.multiline,
                          textInputAction: TextInputAction.newline,
                          enabled: !isSaving,
                          onChanged: field.didChange,
                          errorText: field.errorText,
                        ),
                      ),
                      SizedBox(height: spacing.sp5),
                      Text(
                        l10n.editProfileCraftsLabel,
                        style: theme.textTheme.titleMedium?.copyWith(
                          fontWeight: FontWeight.w800,
                        ),
                      ),
                      SizedBox(height: spacing.sp1),
                      Text(
                        l10n.editProfileCraftsHelper,
                        style: theme.textTheme.bodyMedium?.copyWith(
                          color: theme.colorScheme.onSurfaceVariant,
                        ),
                      ),
                      SizedBox(height: spacing.sp3),
                      FormBuilderField<Set<Craft>>(
                        name: _fieldCrafts,
                        initialValue: _initialSelectedCrafts.toSet(),
                        builder: (field) {
                          final selected = field.value ?? const <Craft>{};
                          return EditProfileCraftsPicker(
                            selected: selected,
                            onToggle: isSaving
                                ? (_) {}
                                : (craft) {
                                    final next = selected.toSet();
                                    if (next.contains(craft)) {
                                      next.remove(craft);
                                    } else {
                                      next.add(craft);
                                    }
                                    field.didChange(next);
                                  },
                          );
                        },
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

/// Save button shown in the app-bar trailing slot. Plain `TextButton` —
/// flat, no chunky styling, just a bold text label that picks up
/// `colorScheme.primary` for the accent colour. Disabled (`null`
/// callback) when the form is clean / invalid / a save is in flight;
/// the in-flight state replaces the label with a small spinner so the
/// user has feedback during the network round trip.
///
/// Styling is delivered via `TextButton.styleFrom(textStyle: ...)`
/// rather than an explicit style on the inner [Text] so the
/// `WidgetStateProperty<Color>` foreground resolver gets to choose
/// between enabled and disabled colours unimpeded — passing a baked-in
/// colour on the child Text overrides the resolver and the disabled
/// state ends up looking identical to the enabled one.
class _SaveAction extends StatelessWidget {
  const _SaveAction({required this.onPressed, required this.isSaving});

  final VoidCallback? onPressed;
  final bool isSaving;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);

    return TextButton(
      onPressed: onPressed,
      style: TextButton.styleFrom(
        textStyle: const TextStyle(fontWeight: FontWeight.w800),
      ),
      child: isSaving
          ? const StitchProgressIndicator(size: 18)
          : Text(l10n.editProfileSave),
    );
  }
}

enum _ProfileImageKind { avatar, banner }

class _ProfileImageDraft {
  const _ProfileImageDraft({
    this.previewBytes,
    this.uploaded,
    this.isUploading = false,
    this.hasError = false,
  });

  factory _ProfileImageDraft.uploading(Uint8List previewBytes) {
    return _ProfileImageDraft(previewBytes: previewBytes, isUploading: true);
  }

  factory _ProfileImageDraft.uploaded(
    Uint8List previewBytes,
    UploadedImageBlob uploaded,
  ) {
    return _ProfileImageDraft(previewBytes: previewBytes, uploaded: uploaded);
  }

  factory _ProfileImageDraft.failed() {
    return const _ProfileImageDraft(hasError: true);
  }

  final Uint8List? previewBytes;
  final UploadedImageBlob? uploaded;
  final bool isUploading;
  final bool hasError;

  bool get changed => previewBytes != null || uploaded != null || hasError;
}

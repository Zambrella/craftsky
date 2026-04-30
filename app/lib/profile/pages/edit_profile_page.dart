import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/data/crafts_catalog.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/providers/save_profile_provider.dart';
import 'package:craftsky_app/profile/providers/user_profile_provider.dart';
import 'package:craftsky_app/profile/widgets/edit_profile_banner_avatar.dart';
import 'package:craftsky_app/profile/widgets/edit_profile_crafts_picker.dart';
import 'package:craftsky_app/profile/widgets/profile_page_error.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Fullscreen profile-edit screen reached via `/profile/edit`. Resolves
/// the signed-in user's handle, loads their profile, and hands a
/// snapshot to [_EditProfileScaffold] which owns the form state.
class EditProfilePage extends ConsumerWidget {
  const EditProfilePage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final auth = ref.watch(authSessionProvider).value;
    final myHandle = switch (auth) {
      SignedIn(:final handle) => handle,
      _ => null,
    };

    if (myHandle == null) {
      // Either auth is loading or the user signed out while sitting on
      // this page — the router will resolve the redirect; show a
      // neutral progress state in the meantime.
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }

    final profileAsync = ref.watch(userProfileProvider(myHandle));
    return Scaffold(
      body: switch (profileAsync) {
        AsyncValue(:final value?) => _EditProfileScaffold(profile: value),
        AsyncError(:final error) => ProfilePageError(
          error: error,
          onRetry: () => ref.invalidate(userProfileProvider(myHandle)),
        ),
        _ => const Center(child: CircularProgressIndicator()),
      },
    );
  }
}

/// Stateful body of the edit page. Holds the draft form state
/// (controllers + selected crafts), the originals (for diffing), and
/// the listener that closes the page on a successful save.
class _EditProfileScaffold extends ConsumerStatefulWidget {
  const _EditProfileScaffold({required this.profile});

  /// Snapshot of the signed-in user's profile at the time the page
  /// opened. Treated as the "original" against which the form's diff
  /// is computed — we don't re-seed the form if the underlying provider
  /// changes mid-edit, since that would clobber the user's typing.
  final Profile profile;

  @override
  ConsumerState<_EditProfileScaffold> createState() =>
      _EditProfileScaffoldState();
}

class _EditProfileScaffoldState extends ConsumerState<_EditProfileScaffold> {
  late final TextEditingController _displayNameController;
  late final TextEditingController _bioController;
  late final Set<Craft> _selectedCrafts;

  /// Crafts on the profile that don't map to any [Craft] enum entry.
  /// Preserved verbatim and re-attached on save so a viewer on an older
  /// build can't accidentally drop tags it doesn't recognise from a
  /// newer server.
  late final List<String> _unknownCrafts;

  @override
  void initState() {
    super.initState();
    _displayNameController = TextEditingController(
      text: widget.profile.displayName,
    );
    _bioController = TextEditingController(
      text: widget.profile.description,
    );

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
    _selectedCrafts = selected;
    _unknownCrafts = List.unmodifiable(unknown);

    // Save-button enabled state depends on whether any field differs
    // from its initial value, so re-render whenever the text changes.
    _displayNameController.addListener(_onTextChanged);
    _bioController.addListener(_onTextChanged);
  }

  @override
  void dispose() {
    _displayNameController
      ..removeListener(_onTextChanged)
      ..dispose();
    _bioController
      ..removeListener(_onTextChanged)
      ..dispose();
    super.dispose();
  }

  void _onTextChanged() {
    if (mounted) setState(() {});
  }

  bool get _hasChanges {
    final initialDisplayName = widget.profile.displayName ?? '';
    final initialBio = widget.profile.description ?? '';
    if (_displayNameController.text != initialDisplayName) return true;
    if (_bioController.text != initialBio) return true;

    final initialKnownCraftIds = widget.profile.crafts
        .where((id) => Craft.fromId(id) != null)
        .toSet();
    final currentCraftIds = _selectedCrafts.map((c) => c.id).toSet();
    if (currentCraftIds.length != initialKnownCraftIds.length) return true;
    if (!currentCraftIds.containsAll(initialKnownCraftIds)) return true;
    return false;
  }

  void _toggleCraft(Craft craft) {
    setState(() {
      if (_selectedCrafts.contains(craft)) {
        _selectedCrafts.remove(craft);
      } else {
        _selectedCrafts.add(craft);
      }
    });
  }

  /// Dispatches the save. Sends the **full** current form state, not a
  /// diff — atproto profile records are atomic, so any field absent
  /// from the PUT gets cleared on the PDS. See
  /// `ProfileApiClient.updateMyProfile` for the rationale.
  void _save() {
    if (!_hasChanges) return;

    // Re-attach the preserved unknowns so a save from this client
    // doesn't strip tags a newer server has added.
    final craftsPayload = <String>[
      ..._selectedCrafts.map((c) => c.id),
      ..._unknownCrafts,
    ];

    ref
        .read(saveProfileProvider.notifier)
        .save(
          displayName: _displayNameController.text.trim(),
          description: _bioController.text.trim(),
          crafts: craftsPayload,
        );
  }

  Future<bool> _confirmDiscard() async {
    final l10n = AppLocalizations.of(context);
    final discard = await showDialog<bool>(
      context: context,
      builder: (dialogContext) {
        return AlertDialog(
          title: Text(l10n.editProfileDiscardTitle),
          content: Text(l10n.editProfileDiscardMessage),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(dialogContext).pop(false),
              child: Text(l10n.editProfileDiscardCancel),
            ),
            TextButton(
              onPressed: () => Navigator.of(dialogContext).pop(true),
              child: Text(l10n.editProfileDiscardConfirm),
            ),
          ],
        );
      },
    );
    return discard ?? false;
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
    // Success closes the page; failure surfaces a snackbar and leaves
    // the form untouched so the user can retry. Per the Riverpod rules,
    // listeners go in build (not initState).
    ref.listen<AsyncValue<Profile?>>(saveProfileProvider, (prev, next) {
      switch ((prev, next)) {
        case (AsyncLoading(), AsyncData(value: final updated?)):
          if (Navigator.of(context).canPop()) {
            Navigator.of(context).pop();
          }
          // Drop the result on the floor — userProfileProvider is
          // already invalidated inside the notifier, so the profile
          // page picks up the change on its own.
          // ignore: unused_local_variable
          final _ = updated;
        case (AsyncLoading(), AsyncError()):
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text(l10n.editProfileSaveError)),
          );
        case _:
          break;
      }
    });

    final canSave = _hasChanges && !isSaving;

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
        appBar: AppBar(
          leading: CloseButton(
            // Tooltip surfaces the cancel intent on hover/long-press;
            // the icon itself is the close X (not a back arrow) since
            // this is a modal-style edit page.
            onPressed: () => Navigator.of(context).maybePop(),
          ),
          title: Text(l10n.editProfileTitle),
          actions: [
            Padding(
              padding: EdgeInsets.only(right: spacing.sp3),
              child: _SaveAction(
                onPressed: canSave ? _save : null,
                isSaving: isSaving,
              ),
            ),
          ],
        ),
        body: SafeArea(
          top: false,
          bottom: false,
          child: SingleChildScrollView(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                EditProfileBannerAvatar(
                  profile: widget.profile,
                  bannerColor: swatches.clay,
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
                      BrandTextField(
                        label: l10n.editProfileDisplayNameLabel,
                        controller: _displayNameController,
                        hintText: l10n.editProfileDisplayNameHint,
                        textInputAction: TextInputAction.next,
                        enabled: !isSaving,
                      ),
                      SizedBox(height: spacing.sp5),
                      BrandTextField(
                        label: l10n.editProfileBioLabel,
                        controller: _bioController,
                        hintText: l10n.editProfileBioHint,
                        minLines: 3,
                        maxLines: 6,
                        keyboardType: TextInputType.multiline,
                        textInputAction: TextInputAction.newline,
                        enabled: !isSaving,
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
                      EditProfileCraftsPicker(
                        selected: _selectedCrafts,
                        onToggle: isSaving ? (_) {} : _toggleCraft,
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
/// callback) when the form is unchanged or a save is in flight; the
/// in-flight state replaces the label with a small spinner so the user
/// has feedback during the network round trip.
class _SaveAction extends StatelessWidget {
  const _SaveAction({required this.onPressed, required this.isSaving});

  final VoidCallback? onPressed;
  final bool isSaving;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context);

    return TextButton(
      onPressed: onPressed,
      child: isSaving
          ? SizedBox(
              width: 18,
              height: 18,
              child: CircularProgressIndicator(
                strokeWidth: 2.4,
                valueColor: AlwaysStoppedAnimation<Color>(
                  theme.colorScheme.primary,
                ),
              ),
            )
          : Text(
              l10n.editProfileSave,
              style: theme.textTheme.labelLarge?.copyWith(
                fontWeight: FontWeight.w800,
              ),
            ),
    );
  }
}

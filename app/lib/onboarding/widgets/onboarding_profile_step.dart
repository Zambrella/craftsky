import 'dart:typed_data';

import 'package:craftsky_app/keys.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/onboarding/models/onboarding_flow_state.dart';
import 'package:craftsky_app/profile/data/profile_field_constraints.dart';
import 'package:craftsky_app/profile/models/profile_handle.dart';
import 'package:craftsky_app/profile/models/profile_image_failure.dart';
import 'package:craftsky_app/profile/models/profile_link.dart';
import 'package:craftsky_app/profile/widgets/profile_links_editor.dart';
import 'package:craftsky_app/shared/text/record_text_limits.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/craftsky_text_inputs.dart';
import 'package:material_ui/material_ui.dart';

class OnboardingProfileStep extends StatefulWidget {
  const new({
    required this.state,
    required this.onDisplayNameChanged,
    required this.onPronounsChanged,
    required this.onBioChanged,
    required this.onLinksChanged,
    required this.onPickAvatar,
    super.key,
  });

  final OnboardingFlowState state;
  final ValueChanged<String> onDisplayNameChanged;
  final ValueChanged<String> onPronounsChanged;
  final ValueChanged<String> onBioChanged;
  final ValueChanged<List<ProfileLink>> onLinksChanged;
  final VoidCallback onPickAvatar;

  @override
  State<OnboardingProfileStep> createState() => _OnboardingProfileStepState();
}

class _OnboardingProfileStepState extends State<OnboardingProfileStep> {
  late final TextEditingController _name;
  late final TextEditingController _pronouns;
  late final TextEditingController _bio;

  @override
  void initState() {
    super.initState();
    _name = TextEditingController(text: widget.state.identity.displayName);
    _pronouns = TextEditingController(text: widget.state.identity.pronouns);
    _bio = TextEditingController(text: widget.state.identity.bio);
  }

  @override
  void didUpdateWidget(covariant OnboardingProfileStep oldWidget) {
    super.didUpdateWidget(oldWidget);
    _syncController(_name, widget.state.identity.displayName);
    _syncController(_pronouns, widget.state.identity.pronouns);
    _syncController(_bio, widget.state.identity.bio);
  }

  void _syncController(TextEditingController controller, String value) {
    if (controller.text == value) return;
    controller.value = TextEditingValue(
      text: value,
      selection: TextSelection.collapsed(offset: value.length),
    );
  }

  @override
  void dispose() {
    _name.dispose();
    _pronouns.dispose();
    _bio.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final profile = widget.state.baseline;
    final onPickAvatar = widget.state.saving || widget.state.uploadingAvatar
        ? null
        : widget.onPickAvatar;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(
          l10n.onboardingProfileTitle,
          style: Theme.of(context).textTheme.headlineMedium,
        ),
        const SizedBox(height: 8),
        Text(l10n.onboardingProfileDescription),
        const SizedBox(height: 24),
        Center(
          child: Semantics(
            button: true,
            label: l10n.editProfileChangeAvatar,
            child: InkWell(
              onTap: onPickAvatar,
              borderRadius: BorderRadius.circular(56),
              child: CircleAvatar(
                radius: 52,
                backgroundImage: _avatarImage(
                  widget.state.avatarPreview,
                  profile.avatar,
                ),
                child:
                    profile.avatar == null && widget.state.avatarPreview == null
                    ? const Icon(CraftskyIconsBold.addPhoto, size: 32)
                    : null,
              ),
            ),
          ),
        ),
        Center(
          child: TextButton(
            onPressed: onPickAvatar,
            child: Text(l10n.editProfileChangeAvatar),
          ),
        ),
        if (widget.state.uploadingAvatar) ...[
          const SizedBox(height: 12),
          Semantics(
            liveRegion: true,
            child: Column(
              children: [
                Text(l10n.onboardingAvatarUploading),
                const SizedBox(height: 8),
                const LinearProgressIndicator(),
              ],
            ),
          ),
        ] else if (widget.state.avatarUploadFailed) ...[
          const SizedBox(height: 12),
          Semantics(
            liveRegion: true,
            child: Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Icon(
                  CraftskyIcons.error,
                  color: Theme.of(context).colorScheme.error,
                ),
                const SizedBox(width: 8),
                Flexible(
                  child: Text(
                    switch (widget.state.avatarFailure) {
                      ProfileImageFailure.tooManyPixels =>
                        l10n.profilePhotoTooManyPixels,
                      ProfileImageFailure.fileTooLarge =>
                        l10n.profilePhotoFileTooLarge,
                      ProfileImageFailure.unknown =>
                        l10n.onboardingAvatarUploadFailed,
                    },
                    style: TextStyle(
                      color: Theme.of(context).colorScheme.error,
                    ),
                  ),
                ),
              ],
            ),
          ),
        ],
        const SizedBox(height: 12),
        Center(
          child: Text(
            ProfileHandle(profile.handle).isAvailable
                ? l10n.onboardingHandleLabel(profile.handle.value)
                : l10n.handleUnavailable,
          ),
        ),
        const SizedBox(height: 24),
        CraftskyTextInput(
          label: l10n.editProfileDisplayNameLabel,
          textFieldKey: const Key('onboarding-display-name'),
          controller: _name,
          enabled: !widget.state.saving,
          maxLength: profileDisplayNameMaxLength,
          maxBytes: RecordTextLimits.displayName.bytes,
          textCapitalization: TextCapitalization.words,
          textInputAction: TextInputAction.next,
          onChanged: widget.onDisplayNameChanged,
        ),
        const SizedBox(height: 12),
        CraftskyTextInput(
          label: l10n.editProfilePronounsLabel,
          hintText: l10n.editProfilePronounsHint,
          textFieldKey: const Key('onboarding-pronouns'),
          controller: _pronouns,
          enabled: !widget.state.saving,
          maxLength: profilePronounsMaxLength,
          maxBytes: RecordTextLimits.pronouns.bytes,
          textInputAction: TextInputAction.next,
          onChanged: widget.onPronounsChanged,
        ),
        const SizedBox(height: 12),
        CraftskyMultilineTextInput(
          label: l10n.editProfileBioLabel,
          textFieldKey: keys.onboarding.bio,
          controller: _bio,
          enabled: !widget.state.saving,
          maxLength: profileBioMaxLength,
          maxBytes: RecordTextLimits.bio.bytes,
          onChanged: widget.onBioChanged,
        ),
        const SizedBox(height: 24),
        ProfileLinksEditor(
          links: widget.state.identity.links,
          enabled: !widget.state.saving,
          onChanged: widget.onLinksChanged,
        ),
      ],
    );
  }

  ImageProvider<Object>? _avatarImage(Uint8List? preview, String? avatar) {
    if (preview != null) return MemoryImage(preview);
    return avatar == null ? null : NetworkImage(avatar);
  }
}

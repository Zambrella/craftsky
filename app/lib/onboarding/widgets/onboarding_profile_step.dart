import 'dart:typed_data';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/onboarding/models/onboarding_flow_state.dart';
import 'package:craftsky_app/profile/data/profile_field_constraints.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/craftsky_text_inputs.dart';
import 'package:flutter/material.dart';

class OnboardingProfileStep extends StatefulWidget {
  const OnboardingProfileStep({
    required this.state,
    required this.onDisplayNameChanged,
    required this.onBioChanged,
    required this.onPickAvatar,
    super.key,
  });

  final OnboardingFlowState state;
  final ValueChanged<String> onDisplayNameChanged;
  final ValueChanged<String> onBioChanged;
  final VoidCallback onPickAvatar;

  @override
  State<OnboardingProfileStep> createState() => _OnboardingProfileStepState();
}

class _OnboardingProfileStepState extends State<OnboardingProfileStep> {
  late final TextEditingController _name;
  late final TextEditingController _bio;

  @override
  void initState() {
    super.initState();
    _name = TextEditingController(text: widget.state.identity.displayName);
    _bio = TextEditingController(text: widget.state.identity.bio);
  }

  @override
  void didUpdateWidget(covariant OnboardingProfileStep oldWidget) {
    super.didUpdateWidget(oldWidget);
    _syncController(_name, widget.state.identity.displayName);
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
    _bio.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final profile = widget.state.baseline;
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
              onTap: widget.state.saving || widget.state.uploadingAvatar
                  ? null
                  : widget.onPickAvatar,
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
                    l10n.onboardingAvatarUploadFailed,
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
        Center(child: Text(l10n.onboardingHandleLabel(profile.handle.value))),
        const SizedBox(height: 24),
        CraftskyTextInput(
          label: l10n.editProfileDisplayNameLabel,
          textFieldKey: const Key('onboarding-display-name'),
          controller: _name,
          enabled: !widget.state.saving,
          maxLength: profileDisplayNameMaxLength,
          textInputAction: TextInputAction.next,
          onChanged: widget.onDisplayNameChanged,
        ),
        const SizedBox(height: 12),
        CraftskyMultilineTextInput(
          label: l10n.editProfileBioLabel,
          textFieldKey: const Key('onboarding-bio'),
          controller: _bio,
          enabled: !widget.state.saving,
          maxLength: profileBioMaxLength,
          onChanged: widget.onBioChanged,
        ),
      ],
    );
  }

  ImageProvider<Object>? _avatarImage(Uint8List? preview, String? avatar) {
    if (preview != null) return MemoryImage(preview);
    return avatar == null ? null : NetworkImage(avatar);
  }
}

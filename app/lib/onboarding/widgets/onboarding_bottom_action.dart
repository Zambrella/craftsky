import 'package:craftsky_app/keys.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/onboarding/models/onboarding_action_state.dart';
import 'package:craftsky_app/shared/widgets/composer_bottom_action.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:material_ui/material_ui.dart';

class OnboardingBottomAction extends StatelessWidget {
  const new({
    required this.state,
    required this.onPressed,
    this.keyboardVisible = false,
    super.key,
  });

  final OnboardingActionState state;
  final VoidCallback onPressed;
  final bool keyboardVisible;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    final label = switch (state.kind) {
      OnboardingActionKind.next => l10n.onboardingNext,
      OnboardingActionKind.saveAndNext => l10n.onboardingSaveNext,
      OnboardingActionKind.finish => l10n.onboardingFinish,
    };
    return SafeArea(
      top: false,
      minimum: EdgeInsets.fromLTRB(spacing.sp5, spacing.sp3, spacing.sp5, 0),
      child: Padding(
        padding: EdgeInsets.only(bottom: spacing.sp4),
        child: Semantics(
          key: const Key('onboarding-primary-action-semantics'),
          excludeSemantics: true,
          button: true,
          enabled: keyboardVisible || state.canSubmit,
          label: keyboardVisible ? l10n.postComposeDone : label,
          value: !keyboardVisible && state.busy ? l10n.loading : null,
          liveRegion: !keyboardVisible && state.busy,
          onTap: keyboardVisible
              ? () => FocusScope.of(context).unfocus()
              : state.canSubmit
              ? onPressed
              : null,
          child: SizedBox(
            width: double.infinity,
            child: ComposerBottomAction(
              keyboardVisible: keyboardVisible,
              doneKey: const Key('onboarding-keyboard-done'),
              actionKey: keys.onboarding.next(state.kind),
              label: label,
              onPressed: state.canSubmit ? onPressed : null,
              child: state.busy
                  ? const StitchProgressIndicator(size: 20)
                  : Text(label),
            ),
          ),
        ),
      ),
    );
  }
}

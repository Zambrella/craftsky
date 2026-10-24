import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:material_ui/material_ui.dart';

/// Swaps the composer submit action for keyboard dismissal while typing.
class ComposerBottomAction extends StatelessWidget {
  const new({
    required this.keyboardVisible,
    required this.doneKey,
    required this.child,
    super.key,
  });

  // Read above the Scaffold, which removes keyboard insets from its body.
  final bool keyboardVisible;
  final Key doneKey;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final durations = Theme.of(context).extension<DurationTheme>()!;
    return AnimatedSwitcher(
      duration: MediaQuery.disableAnimationsOf(context)
          ? Duration.zero
          : durations.medium,
      switchInCurve: durations.ease,
      switchOutCurve: durations.ease,
      layoutBuilder: (currentChild, previousChildren) => Stack(
        fit: StackFit.passthrough,
        alignment: AlignmentDirectional.centerEnd,
        children: [
          for (final child in previousChildren)
            ExcludeSemantics(child: IgnorePointer(child: child)),
          ?currentChild,
        ],
      ),
      child: keyboardVisible
          ? Align(
              alignment: AlignmentDirectional.centerEnd,
              child: ChunkyButton(
                key: doneKey,
                variant: ChunkyButtonVariant.secondary,
                onPressed: () => FocusScope.of(context).unfocus(),
                child: Text(AppLocalizations.of(context).postComposeDone),
              ),
            )
          : child,
    );
  }
}

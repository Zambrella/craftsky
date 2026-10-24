import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/animated_chunky_button.dart';
import 'package:material_ui/material_ui.dart';

/// Morphs the submit action into a compact keyboard-dismissal button.
class ComposerBottomAction extends StatelessWidget {
  const new({
    required this.keyboardVisible,
    required this.doneKey,
    required this.actionKey,
    required this.label,
    required this.onPressed,
    this.focusNode,
    this.style,
    this.child,
    super.key,
  });

  // Read above the Scaffold, which removes keyboard insets from its body.
  final bool keyboardVisible;
  final Key doneKey;
  final Key actionKey;
  final String label;
  final VoidCallback? onPressed;
  final FocusNode? focusNode;
  final ButtonStyle? style;
  final Widget? child;

  @override
  Widget build(BuildContext context) => AnimatedChunkyButton(
    compact: keyboardVisible,
    label: label,
    compactLabel: AppLocalizations.of(context).postComposeDone,
    actionKey: keyboardVisible ? doneKey : actionKey,
    onPressed: keyboardVisible
        ? () => FocusScope.of(context).unfocus()
        : onPressed,
    focusNode: keyboardVisible ? null : focusNode,
    style: style,
    child: child,
  );
}

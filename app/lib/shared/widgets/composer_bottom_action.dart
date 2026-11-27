import 'dart:async';

import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/animated_chunky_button.dart';
import 'package:flutter/services.dart';
import 'package:material_ui/material_ui.dart';

void dismissComposerKeyboard(BuildContext context) {
  FocusScope.of(context).unfocus();
  // The platform keyboard can still be visible after the input loses focus.
  unawaited(SystemChannels.textInput.invokeMethod<void>('TextInput.hide'));
}

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
        ? () => dismissComposerKeyboard(context)
        : onPressed,
    focusNode: keyboardVisible ? null : focusNode,
    style: style,
    child: child,
  );
}

import 'package:craftsky_app/shared/widgets/composer_bottom_action.dart';
import 'package:craftsky_app/shared/widgets/form_keyboard_scope.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:material_ui/material_ui.dart';

/// Keeps the form action reachable while its fields scroll underneath it.
/// Read [keyboardVisible] above the Scaffold, before it consumes view insets.
class FormActionLayout extends StatelessWidget {
  const new({
    required this.child,
    required this.keyboardVisible,
    required this.label,
    required this.onPressed,
    required this.actionKey,
    this.overlayAction = false,
    this.doneKey = const Key('form-keyboard-done'),
    this.actionChild,
    this.actionStyle,
    super.key,
  });

  final Widget child;
  final bool keyboardVisible;
  final String label;
  final VoidCallback? onPressed;
  final Key actionKey;

  /// Lets content scroll behind the action. The scrollable supplies clearance
  /// at its end so the last field can still scroll above the action.
  final bool overlayAction;
  final Key doneKey;
  final Widget? actionChild;
  final ButtonStyle? actionStyle;

  @override
  Widget build(BuildContext context) {
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    return FormKeyboardScope(
      child: Stack(
        fit: StackFit.expand,
        children: [
          if (overlayAction)
            child
          else
            Padding(
              padding: EdgeInsets.only(
                bottom: spacing.sp9 + MediaQuery.paddingOf(context).bottom,
              ),
              child: child,
            ),
          PositionedDirectional(
            start: spacing.sp4,
            end: spacing.sp4,
            bottom: 0,
            child: SafeArea(
              top: false,
              child: Padding(
                padding: EdgeInsets.only(bottom: spacing.sp4),
                child: ComposerBottomAction(
                  keyboardVisible: keyboardVisible,
                  doneKey: doneKey,
                  actionKey: actionKey,
                  label: label,
                  onPressed: onPressed,
                  style: actionStyle,
                  child: actionChild,
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

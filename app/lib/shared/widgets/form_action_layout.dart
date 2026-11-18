import 'package:craftsky_app/shared/widgets/composer_bottom_action.dart';
import 'package:craftsky_app/shared/widgets/form_keyboard_scope.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:material_ui/material_ui.dart';

/// Keeps the form action reachable while its fields scroll underneath it.
/// Read [keyboardVisible] above the Scaffold, before it consumes view insets.
class FormActionLayout extends StatelessWidget {
  const new({
    required this.contentBuilder,
    required this.keyboardVisible,
    required this.label,
    required this.onPressed,
    required this.actionKey,
    this.doneKey = const Key('form-keyboard-done'),
    this.actionChild,
    this.actionStyle,
    super.key,
  });

  /// Apply the supplied clearance at the end of the scrollable content.
  final Widget Function(BuildContext context, double actionClearance)
  contentBuilder;
  final bool keyboardVisible;
  final String label;
  final VoidCallback? onPressed;
  final Key actionKey;
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
          contentBuilder(
            context,
            spacing.sp9 + MediaQuery.paddingOf(context).bottom,
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

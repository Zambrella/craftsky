import 'package:craftsky_app/router/app_shell_drawer.dart';
import 'package:material_ui/material_ui.dart';

/// Roots use the shell drawer; explicit child close actions keep platform Back.
Widget? chatAppBarLeading(BuildContext context, VoidCallback? onClose) {
  if (onClose != null) return BackButton(onPressed: onClose);
  return AppShellDrawerScope.maybeOf(context) == null
      ? null
      : const AppShellDrawerButton();
}

import 'package:material_ui/material_ui.dart';

/// Dismisses only this form's focus when its route actually closes. A rejected
/// pop (including a cancelled discard prompt) keeps the editor's focus intact.
class FormKeyboardScope extends StatefulWidget {
  const new({required this.child, super.key});

  final Widget child;

  @override
  State<FormKeyboardScope> createState() => _FormKeyboardScopeState();
}

class _FormKeyboardScopeState extends State<FormKeyboardScope> {
  final _focusScope = FocusScopeNode(debugLabel: 'formKeyboard');

  @override
  void dispose() {
    _focusScope.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => PopScope<Object?>(
    onPopInvokedWithResult: (didPop, _) {
      if (didPop && _focusScope.hasFocus) _focusScope.unfocus();
    },
    child: FocusScope(node: _focusScope, child: widget.child),
  );
}

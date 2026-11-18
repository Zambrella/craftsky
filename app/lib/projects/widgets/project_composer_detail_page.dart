import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/shared/widgets/form_action_layout.dart';
import 'package:craftsky_app/shared/widgets/form_keyboard_scope.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:material_ui/material_ui.dart';

class ProjectDetailActionTile extends StatelessWidget {
  const new({
    required this.focusNode,
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.enabled,
    required this.onTap,
    this.errorText,
    super.key,
  });

  final FocusNode focusNode;
  final IconData icon;
  final String title;
  final String subtitle;
  final String? errorText;
  final bool enabled;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    return Semantics(
      button: true,
      enabled: enabled,
      child: ListTile(
        contentPadding: EdgeInsets.zero,
        focusNode: focusNode,
        leading: Icon(icon),
        title: Text(title),
        subtitle: Text(
          errorText ?? subtitle,
          style: errorText == null ? null : TextStyle(color: colors.error),
        ),
        trailing: const Icon(CraftskyIconsBold.next),
        enabled: enabled,
        onTap: enabled ? onTap : null,
      ),
    );
  }
}

class ProjectComposerDetailPage extends StatelessWidget {
  const new({
    required this.title,
    required this.child,
    this.errorText,
    super.key,
  });

  final String title;
  final Widget child;
  final String? errorText;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    return FormKeyboardScope(
      child: Scaffold(
        backgroundColor: swatches.paper,
        appBar: AppBar(title: Text(title, style: theme.textTheme.titleLarge)),
        body: FormActionLayout(
          keyboardVisible: MediaQuery.viewInsetsOf(context).bottom > 0,
          label: AppLocalizations.of(context).postComposeDone,
          actionKey: const Key('composer-details-done'),
          onPressed: () => Navigator.of(context).maybePop(),
          child: GestureDetector(
            behavior: HitTestBehavior.translucent,
            onTap: () => FocusManager.instance.primaryFocus?.unfocus(),
            child: SafeArea(
              top: false,
              bottom: false,
              child: SingleChildScrollView(
                padding: EdgeInsets.fromLTRB(
                  spacing.sp4,
                  spacing.sp4,
                  spacing.sp4,
                  spacing.sp4 + MediaQuery.paddingOf(context).bottom,
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    if (errorText case final message?) ...[
                      Text(
                        message,
                        style: theme.textTheme.bodySmall?.copyWith(
                          color: theme.colorScheme.error,
                        ),
                      ),
                      SizedBox(height: spacing.sp4),
                    ],
                    child,
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/settings/models/settings_row.dart';
import 'package:craftsky_app/shared/widgets/action_progress.dart';
import 'package:craftsky_app/subscriptions/widgets/plus_action_icon.dart';
import 'package:material_ui/material_ui.dart';

class SettingsRowTile extends StatelessWidget {
  const new({
    required this.descriptor,
    required this.label,
    required this.leading,
    this.onTap,
    this.subtitle,
    this.locked = false,
    this.isLoading = false,
    super.key,
  });

  final SettingsRowDescriptor descriptor;
  final String label;
  final IconData leading;
  final VoidCallback? onTap;
  final String? subtitle;
  final bool locked;
  final bool isLoading;

  @override
  Widget build(BuildContext context) {
    final direction = Directionality.of(context);
    final trailingIcon = descriptor.trailingIcon(direction);
    final available = onTap != null && !isLoading;
    final interactive = descriptor.kind != SettingsRowKind.readOnly;
    final destructive = descriptor.kind == SettingsRowKind.destructiveAction;
    final foreground = destructive ? Theme.of(context).colorScheme.error : null;

    return Semantics(
      container: true,
      button: interactive,
      enabled: interactive ? available : null,
      label: label,
      value: isLoading ? AppLocalizations.of(context).loading : null,
      excludeSemantics: true,
      child: ConstrainedBox(
        constraints: const BoxConstraints(minHeight: 48),
        child: ListTile(
          leading: IconTheme.merge(
            data: IconThemeData(color: foreground),
            child: ActionProgress(
              isLoading: isLoading,
              child: locked ? PlusActionIcon(icon: leading) : Icon(leading),
            ),
          ),
          title: Text(label, style: TextStyle(color: foreground)),
          subtitle: subtitle == null ? null : Text(subtitle!),
          textColor: foreground,
          enabled: !interactive || available,
          trailing: trailingIcon == null ? null : Icon(trailingIcon),
          onTap: interactive && available ? onTap : null,
        ),
      ),
    );
  }
}

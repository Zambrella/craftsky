import 'package:craftsky_app/settings/models/settings_row.dart';
import 'package:craftsky_app/subscriptions/widgets/plus_action_icon.dart';
import 'package:flutter/material.dart';

class SettingsRowTile extends StatelessWidget {
  const SettingsRowTile({
    required this.descriptor,
    required this.label,
    required this.leading,
    this.onTap,
    this.subtitle,
    this.locked = false,
    super.key,
  });

  final SettingsRowDescriptor descriptor;
  final String label;
  final IconData leading;
  final VoidCallback? onTap;
  final String? subtitle;
  final bool locked;

  @override
  Widget build(BuildContext context) {
    final direction = Directionality.of(context);
    final trailingIcon = descriptor.trailingIcon(direction);
    final interactive = descriptor.kind != SettingsRowKind.readOnly;
    final destructive = descriptor.kind == SettingsRowKind.destructiveAction;
    final foreground = destructive ? Theme.of(context).colorScheme.error : null;

    return Semantics(
      container: true,
      button: interactive,
      enabled: interactive ? onTap != null : null,
      label: label,
      excludeSemantics: true,
      child: ConstrainedBox(
        constraints: const BoxConstraints(minHeight: 48),
        child: ListTile(
          leading: IconTheme(
            data: IconThemeData(color: foreground),
            child: locked ? PlusActionIcon(icon: leading) : Icon(leading),
          ),
          title: Text(label, style: TextStyle(color: foreground)),
          subtitle: subtitle == null ? null : Text(subtitle!),
          textColor: foreground,
          enabled: !interactive || onTap != null,
          trailing: trailingIcon == null ? null : Icon(trailingIcon),
          onTap: interactive ? onTap : null,
        ),
      ),
    );
  }
}

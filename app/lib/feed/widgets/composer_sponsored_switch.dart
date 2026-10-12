import 'package:material_ui/material_ui.dart';

class ComposerSponsoredSwitch extends StatelessWidget {
  const new({
    required this.value,
    required this.title,
    required this.description,
    required this.onChanged,
    super.key,
  });

  final bool value;
  final String title;
  final String description;
  final ValueChanged<bool>? onChanged;

  @override
  Widget build(BuildContext context) => SwitchListTile.adaptive(
    contentPadding: EdgeInsets.zero,
    title: Text(title),
    subtitle: Text(description),
    value: value,
    onChanged: onChanged,
  );
}

import 'dart:async';

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/theme/theme_notifier.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class AppearancePage extends ConsumerWidget {
  const AppearancePage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final selectedMode = ref.watch(themeModeProvider);

    return Scaffold(
      appBar: AppBar(title: Text(l10n.appearanceTitle)),
      body: RadioGroup<ThemeMode>(
        groupValue: selectedMode,
        onChanged: (mode) {
          if (mode != null) {
            unawaited(ref.read(themeModeProvider.notifier).setMode(mode));
          }
        },
        child: ListView(
          children: [
            RadioListTile<ThemeMode>(
              value: ThemeMode.system,
              secondary: const Icon(Icons.brightness_auto_outlined),
              title: Text(l10n.appearanceUseDeviceSetting),
            ),
            RadioListTile<ThemeMode>(
              value: ThemeMode.light,
              secondary: const Icon(Icons.light_mode_outlined),
              title: Text(l10n.appearanceLight),
            ),
            RadioListTile<ThemeMode>(
              value: ThemeMode.dark,
              secondary: const Icon(Icons.dark_mode_outlined),
              title: Text(l10n.appearanceDark),
            ),
          ],
        ),
      ),
    );
  }
}

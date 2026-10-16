// Run with: flutter run -d web-server -t test/fixtures/publishing_animation_preview.dart
import 'package:craftsky_app/feed/widgets/submission_blocking_overlay.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/publishing_progress_indicator.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

void main() => runApp(const ProviderScope(child: _Preview()));

class _Preview extends ConsumerStatefulWidget {
  const new();

  @override
  ConsumerState<_Preview> createState() => _PreviewState();
}

class _PreviewState extends ConsumerState<_Preview> {
  bool _dark = false;
  bool _reduceMotion = false;

  @override
  Widget build(BuildContext context) => MaterialApp(
    theme: AppTheme.lightThemeData,
    darkTheme: AppTheme.darkThemeData,
    themeMode: _dark ? ThemeMode.dark : ThemeMode.light,
    localizationsDelegates: appLocalizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: Builder(
      builder: (context) => MediaQuery(
        data: MediaQuery.of(context).copyWith(disableAnimations: _reduceMotion),
        child: Scaffold(
          appBar: AppBar(
            title: const Text('Publishing animation preview'),
            actions: [
              Padding(
                padding: const EdgeInsets.only(right: 16),
                child: Text(switch (ref.watch(
                  publishingAnimationFileProvider,
                )) {
                  AsyncData() => 'Animation ready',
                  AsyncError() => 'Fallback indicator',
                  _ => 'Loading animation',
                }),
              ),
            ],
          ),
          body: Column(
            children: [
              SwitchListTile(
                title: const Text('Dark mode'),
                value: _dark,
                onChanged: (value) => setState(() => _dark = value),
              ),
              SwitchListTile(
                title: const Text('Reduced motion'),
                value: _reduceMotion,
                onChanged: (value) => setState(() => _reduceMotion = value),
              ),
              const Expanded(
                child: Stack(
                  children: [SubmissionBlockingOverlay(scheduling: false)],
                ),
              ),
            ],
          ),
        ),
      ),
    ),
  );
}

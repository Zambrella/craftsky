import 'dart:async';

import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/settings/logged_out_visibility_provider.dart';
import 'package:craftsky_app/shared/observability/error_reporting_consent_tile.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

class PrivacyPage extends ConsumerStatefulWidget {
  const new({super.key});
  @override
  ConsumerState<PrivacyPage> createState() => _PrivacyPageState();
}

class _PrivacyPageState extends ConsumerState<PrivacyPage> {
  Object? _loadedModel;
  @override
  Widget build(BuildContext context) {
    final model = ref.watch(loggedOutVisibilityControllerProvider);
    if (!identical(_loadedModel, model)) {
      _loadedModel = model;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) unawaited(model.load());
      });
    }
    return Scaffold(
      appBar: AppBar(title: Text(AppLocalizations.of(context).privacyTitle)),
      body: ListenableBuilder(
        listenable: model,
        builder: (context, _) => ListView(
          children: [
            const ErrorReportingConsentTile(),
            if (model.value case final value?)
              SwitchListTile(
                title: const Text(
                  'Ask apps and sites not to show my account to '
                  'logged-out users',
                ),
                subtitle: const Text(
                  'CraftSky and Bluesky will not show your account '
                  'to logged-out users and will ask other apps to '
                  'do the same. Other apps may not honour this '
                  "request. It doesn't make your account private.",
                ),
                value: value.hideFromLoggedOut,
                onChanged: model.busy || model.pending != null
                    ? null
                    : model.setHidden,
              )
            else
              ListTile(
                title: Text(
                  model.busy ? 'Loading preference…' : 'Preference unavailable',
                ),
              ),
            const Padding(
              padding: EdgeInsets.all(16),
              child: Text(
                'This is the shared Bluesky profile preference. '
                'Changing it here also changes it on Bluesky.',
              ),
            ),
            const Padding(
              padding: EdgeInsets.all(16),
              child: Text(
                'CraftSky is part of the Atmosphere, an open '
                'public network. Other apps and websites may '
                'choose not to honour this request. Your public '
                'content may still appear elsewhere. Previously '
                'cached link previews may remain visible.',
              ),
            ),
            if (model.error case final error?)
              Padding(
                padding: const EdgeInsets.all(16),
                child: Semantics(liveRegion: true, child: Text(error)),
              ),
            if (model.notice case final notice?)
              Padding(
                padding: const EdgeInsets.all(16),
                child: Semantics(liveRegion: true, child: Text(notice)),
              ),
            if (model.pending != null) ...[
              const ListTile(title: Text('Save pending confirmation.')),
              TextButton(
                onPressed: model.busy ? null : model.retry,
                child: const Text('Retry save'),
              ),
            ] else if (model.error != null)
              TextButton(
                onPressed: model.busy ? null : model.load,
                child: const Text('Reload preference'),
              ),
          ],
        ),
      ),
    );
  }
}

import 'dart:async';

import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/safety/guide_data.dart';
import 'package:craftsky_app/shared/link/external_link.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:go_router/go_router.dart';
import 'package:material_ui/material_ui.dart';

class SafetyGuidePage extends StatelessWidget {
  const new({super.key});

  @override
  Widget build(BuildContext context) {
    Widget heading(String key) => Padding(
      padding: const EdgeInsets.only(top: 24, bottom: 12),
      child: Semantics(
        header: true,
        child: Text(
          safetyGuide[key] as String,
          style: Theme.of(context).textTheme.titleLarge,
        ),
      ),
    );
    Widget paragraph(String key) => Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: Text(safetyGuide[key] as String),
    );
    return Scaffold(
      appBar: AppBar(
        title: Text(safetyGuide['title'] as String),
        leading: IconButton(
          tooltip: MaterialLocalizations.of(context).backButtonTooltip,
          icon: const Icon(Icons.arrow_back),
          onPressed: () =>
              context.canPop() ? context.pop() : const FeedRoute().go(context),
        ),
      ),
      body: Align(
        alignment: Alignment.topCenter,
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 820),
          child: ListView(
            padding: const EdgeInsets.all(24),
            children: [
              paragraph('intro'),
              paragraph('urgent'),
              heading('supportTitle'),
              paragraph('supportIntro'),
              for (final resource
                  in (safetyGuide['resources'] as List)
                      .cast<Map<String, dynamic>>())
                Padding(
                  padding: const EdgeInsets.only(bottom: 12),
                  child: CraftskyCard(
                    child: Padding(
                      padding: const EdgeInsets.all(16),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.stretch,
                        children: [
                          Text(
                            resource['scope'] as String,
                            style: Theme.of(context).textTheme.labelMedium,
                          ),
                          TextButton.icon(
                            style: TextButton.styleFrom(
                              alignment: AlignmentDirectional.centerStart,
                            ),
                            onPressed: () => unawaited(
                              confirmAndLaunchExternalAction(
                                context,
                                uri: Uri.parse(resource['url'] as String),
                                launchUrl: launchExternalLink,
                              ),
                            ),
                            icon: const Icon(Icons.open_in_new),
                            label: Text(resource['title'] as String),
                          ),
                          Text(resource['body'] as String),
                        ],
                      ),
                    ),
                  ),
                ),
              paragraph('outsideUK'),
              heading('youngTitle'),
              paragraph('youngBody'),
              heading('stepsTitle'),
              for (final step
                  in (safetyGuide['steps'] as List)
                      .cast<Map<String, dynamic>>())
                Padding(
                  padding: const EdgeInsets.only(bottom: 12),
                  child: CraftskyCard(
                    child: Padding(
                      padding: const EdgeInsets.all(16),
                      child: Row(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          ExcludeSemantics(
                            child: CircleAvatar(
                              child: Text(step['symbol'] as String),
                            ),
                          ),
                          const SizedBox(width: 16),
                          Expanded(
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Semantics(
                                  header: true,
                                  child: Text(
                                    step['title'] as String,
                                    style: Theme.of(context)
                                        .textTheme
                                        .titleMedium,
                                  ),
                                ),
                                const SizedBox(height: 8),
                                Text(step['body'] as String),
                              ],
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
              heading('messagesTitle'),
              paragraph('messagesBody'),
              heading('parentsTitle'),
              paragraph('parentsBody'),
              paragraph('limits'),
              Text(
                AppLocalizations.of(context)
                    .safetyResourcesReviewed(safetyGuide['reviewed'] as String),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

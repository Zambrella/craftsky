import 'dart:async';

import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/router/router.dart';
import 'package:craftsky_app/safety/guide_data.dart';
import 'package:craftsky_app/safety/support_search.dart';
import 'package:material_ui/material_ui.dart';

class SearchSupportNotice extends StatefulWidget {
  const new({required this.query, super.key});
  final String query;
  @override
  State<SearchSupportNotice> createState() => _SearchSupportNoticeState();
}

class _SearchSupportNoticeState extends State<SearchSupportNotice> {
  bool _dismissed = false;
  @override
  void didUpdateWidget(covariant SearchSupportNotice oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.query != widget.query) _dismissed = false;
  }

  @override
  Widget build(BuildContext context) {
    if (_dismissed || !shouldOfferSearchSupport(widget.query)) {
      return const SizedBox.shrink();
    }
    final l10n = AppLocalizations.of(context);
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Semantics(
            liveRegion: true,
            child: Text(safetyGuide['searchNotice'] as String),
          ),
          Wrap(
            spacing: 12,
            children: [
              TextButton(
                onPressed: () =>
                    unawaited(const SafetyRoute().push<void>(context)),
                child: Text(l10n.safetyFindSupport),
              ),
              TextButton(
                onPressed: () => setState(() => _dismissed = true),
                child: Text(l10n.messengerDismiss),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

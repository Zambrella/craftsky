import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/service_status/models/service_status_document.dart';
import 'package:craftsky_app/service_status/widgets/status_message_image.dart';
import 'package:craftsky_app/shared/widgets/action_progress.dart';
import 'package:craftsky_app/theme/chunky_button.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:material_ui/material_ui.dart';

class AnnouncementModal extends StatelessWidget {
  const new({
    required this.document,
    required this.onDismiss,
    this.dismissing = false,
    this.activating = false,
    this.actionFailed = false,
    this.onActivate,
    super.key,
  });
  final StatusMessage document;
  final VoidCallback onDismiss;
  final bool dismissing;
  final bool activating;
  final bool actionFailed;
  final VoidCallback? onActivate;

  @override
  Widget build(BuildContext context) {
    final spacing = Theme.of(context).extension<SpacingTheme>()!;
    final l10n = AppLocalizations.of(context);
    return SafeArea(
      child: Semantics(
        scopesRoute: true,
        explicitChildNodes: true,
        namesRoute: true,
        label: document.title,
        child: CraftskyDialog(
          title: document.title,
          body: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(document.description),
              if (document.imageUrl != null)
                Padding(
                  padding: EdgeInsets.only(
                    top: spacing.sp5,
                    bottom: spacing.sp2,
                  ),
                  child: StatusMessageImage(
                    url: document.imageUrl!,
                    key: ValueKey(document.imageUrl),
                  ),
                ),
              if (actionFailed) Text(l10n.serviceStatusActionFailed),
            ],
          ),
          actions: [
            SizedBox(
              width: double.infinity,
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  if (document.cta != null) ...[
                    ChunkyButton(
                      onPressed: activating || dismissing ? null : onActivate,
                      child: ActionProgress(
                        isLoading: activating,
                        child: Text(document.cta!.text),
                      ),
                    ),
                    SizedBox(height: spacing.sp3),
                  ],
                  TextButton(
                    autofocus: true,
                    onPressed: dismissing ? null : onDismiss,
                    child: ActionProgress(
                      isLoading: dismissing,
                      child: Text(l10n.serviceStatusDismiss),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/data/language_catalogue.dart';
import 'package:craftsky_app/languages/models/post_language_selection.dart';
import 'package:craftsky_app/languages/widgets/post_language_selector.dart';
import 'package:craftsky_app/theme/craftsky_context_menu.dart';
import 'package:craftsky_app/theme/craftsky_icons.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';

class ComposerMetadataControls extends StatelessWidget {
  const ComposerMetadataControls({
    required this.languages,
    required this.onLanguagesChanged,
    required this.sponsored,
    required this.onSponsoredChanged,
    required this.scheduledAtLocal,
    required this.onSchedulePressed,
    this.showSponsored = true,
    this.showSchedule = true,
    super.key,
  });

  final PostLanguageSelection languages;
  final ValueChanged<PostLanguageSelection>? onLanguagesChanged;
  final bool sponsored;
  final ValueChanged<bool>? onSponsoredChanged;
  final DateTime? scheduledAtLocal;
  final ValueChanged<BuildContext>? onSchedulePressed;
  final bool showSponsored;
  final bool showSchedule;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final spacing = theme.extension<SpacingTheme>()!;
    return Padding(
      padding: EdgeInsets.only(bottom: spacing.sp2),
      child: Wrap(
        key: const Key('composer-metadata-controls'),
        spacing: 4,
        runSpacing: 4,
        crossAxisAlignment: WrapCrossAlignment.center,
        children: [
          Builder(
            builder: (menuContext) => Semantics(
              label: l10n.postLanguagesSemantics,
              value: languages.values.map(languageLabel).join(', '),
              button: true,
              enabled: onLanguagesChanged != null,
              excludeSemantics: true,
              child: IconButton(
                key: const Key('composer-language-control'),
                tooltip: l10n.postLanguagesSemantics,
                color: colors.onSurface,
                onPressed: onLanguagesChanged == null
                    ? null
                    : () => _showLanguageContext(menuContext),
                icon: Badge.count(
                  count: languages.values.length,
                  backgroundColor: colors.onSurface,
                  textColor: colors.surface,
                  child: const Icon(CraftskyIcons.language),
                ),
              ),
            ),
          ),
          if (showSponsored)
            Builder(
              builder: (menuContext) => Semantics(
                label: l10n.postSponsoredToggleTitle,
                button: true,
                enabled: onSponsoredChanged != null,
                selected: sponsored,
                excludeSemantics: true,
                child: IconButton(
                  key: const Key('composer-sponsored-control'),
                  tooltip: l10n.postSponsoredToggleTitle,
                  color: sponsored ? colors.primary : colors.onSurface,
                  onPressed: onSponsoredChanged == null
                      ? null
                      : () => _showSponsoredContext(menuContext),
                  icon: const Icon(CraftskyIcons.sponsored),
                ),
              ),
            ),
          if (showSchedule)
            Builder(
              builder: (menuContext) {
                final compactLabel = switch (scheduledAtLocal) {
                  final value? => _compactScheduleLabel(context, value),
                  _ => l10n.scheduledPostNow,
                };
                return Semantics(
                  label: l10n.scheduledPostWhenTitle,
                  value: compactLabel,
                  button: true,
                  enabled: onSchedulePressed != null,
                  selected: scheduledAtLocal != null,
                  excludeSemantics: true,
                  child: switch (scheduledAtLocal) {
                    final scheduledAt? => TextButton.icon(
                      key: const Key('composer-schedule-control'),
                      style: TextButton.styleFrom(
                        foregroundColor: colors.primary,
                        minimumSize: const Size(0, 48),
                      ),
                      onPressed: onSchedulePressed == null
                          ? null
                          : () => onSchedulePressed!(menuContext),
                      icon: Icon(
                        CraftskyIcons.schedule,
                        color: colors.primary,
                      ),
                      label: Text(_compactScheduleLabel(context, scheduledAt)),
                    ),
                    _ => IconButton(
                      key: const Key('composer-schedule-control'),
                      tooltip: l10n.scheduledPostWhenTitle,
                      color: colors.onSurface,
                      onPressed: onSchedulePressed == null
                          ? null
                          : () => onSchedulePressed!(menuContext),
                      icon: const Icon(CraftskyIcons.schedule),
                    ),
                  },
                );
              },
            ),
        ],
      ),
    );
  }

  Future<void> _showLanguageContext(BuildContext context) =>
      showCraftskyContextMenu(
        context,
        position: craftskyContextMenuAnchorPosition(context),
        groups: [
          CraftskyContextMenuGroup(
            items: [
              for (final language in languages.values)
                CraftskyContextMenuItem(
                  text: languageLabel(language),
                  icon: CraftskyIcons.language,
                  isSelected: true,
                  onPressed: languages.values.length == 1
                      ? null
                      : () => onLanguagesChanged?.call(
                          languages.remove(language),
                        ),
                ),
            ],
          ),
          CraftskyContextMenuGroup(
            items: [
              CraftskyContextMenuItem(
                text: AppLocalizations.of(context).postLanguageAdd,
                icon: CraftskyIconsBold.add,
                onPressed: languages.values.length == 3
                    ? null
                    : () async {
                        final language = await showPostLanguagePicker(
                          context,
                          excluded: languages.values.toSet(),
                        );
                        if (language != null) {
                          onLanguagesChanged?.call(languages.add(language));
                        }
                      },
              ),
            ],
          ),
        ],
      );

  Future<void> _showSponsoredContext(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return showCraftskyContextMenu(
      context,
      position: craftskyContextMenuAnchorPosition(context),
      groups: [
        CraftskyContextMenuGroup(
          items: [
            CraftskyContextMenuItem(
              text: l10n.postSponsoredToggleTitle,
              description: l10n.postSponsoredToggleDescription,
              icon: CraftskyIcons.sponsored,
              switchValue: sponsored,
              onPressed: () => onSponsoredChanged?.call(!sponsored),
            ),
          ],
        ),
      ],
    );
  }

  String _compactScheduleLabel(BuildContext context, DateTime value) {
    final localizations = MaterialLocalizations.of(context);
    return AppLocalizations.of(context).scheduledPostCompactTime(
      localizations.formatShortMonthDay(value),
      localizations.formatTimeOfDay(TimeOfDay.fromDateTime(value)),
    );
  }
}

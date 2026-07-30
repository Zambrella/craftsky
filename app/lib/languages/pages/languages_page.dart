import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/data/language_catalogue.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:craftsky_app/theme/craftsky_card.dart';
import 'package:craftsky_app/theme/craftsky_select_inputs.dart';
import 'package:craftsky_app/theme/stitch_progress_indicator.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final List<CraftskySelectOption<String>> _languageOptions = List.unmodifiable(
  <CraftskySelectOption<String>>[
    for (final tag in supportedLanguageTags)
      CraftskySelectOption(
        value: tag,
        label: languageLabel(tag),
        description: tag,
      ),
  ]..sort((left, right) => left.label.compareTo(right.label)),
);

class LanguagesPage extends ConsumerWidget {
  const LanguagesPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final l10n = AppLocalizations.of(context);
    final session = ref.watch(authSessionProvider);
    final activeDID = switch (session.value) {
      SignedIn(:final did) => did,
      _ => null,
    };
    return Scaffold(
      appBar: AppBar(title: Text(l10n.languagesTitle)),
      body: SingleChildScrollView(
        padding: EdgeInsets.fromLTRB(
          spacing.sp4,
          spacing.sp4,
          spacing.sp4,
          spacing.sp7,
        ),
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 720),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                const _AppLanguageField(),
                SizedBox(height: spacing.sp4),
                if (activeDID == null)
                  Padding(
                    padding: EdgeInsets.all(spacing.sp5),
                    child: const Center(child: StitchProgressIndicator()),
                  )
                else
                  _AccountLanguageSections(
                    account: AccountKey(activeDID.value),
                  ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _AccountLanguageSections extends ConsumerStatefulWidget {
  const _AccountLanguageSections({required this.account});

  final AccountKey account;

  @override
  ConsumerState<_AccountLanguageSections> createState() =>
      _AccountLanguageSectionsState();
}

class _AccountLanguageSectionsState
    extends ConsumerState<_AccountLanguageSections> {
  var _saving = false;

  Future<void> _replace(LanguagePreferences candidate) async {
    setState(() => _saving = true);
    final success = await ref
        .read(
          accountLanguagePreferencesProvider(widget.account).notifier,
        )
        .replace(candidate);
    if (!mounted) return;
    setState(() => _saving = false);
    if (!success) {
      context.showError(AppLocalizations.of(context).languageSaveError);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final l10n = AppLocalizations.of(context);
    final provider = accountLanguagePreferencesProvider(widget.account);
    return switch (ref.watch(provider)) {
      AsyncData(:final value) => Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          CraftskyCard(
            key: const Key('primary-language-card'),
            clipBehavior: Clip.none,
            padding: EdgeInsets.all(spacing.sp4),
            child: CraftskySingleSelectInput<String>(
              key: const Key('primary-language-input'),
              keyPrefix: 'primary-language',
              label: l10n.primaryLanguageTitle,
              helperText: l10n.primaryLanguageDescription,
              value: value.primaryLanguage,
              options: _languageOptions,
              enabled: !_saving,
              searchHintText: l10n.languageSearchHint,
              onChanged: (language) async {
                if (language == null || language == value.primaryLanguage) {
                  return;
                }
                await _replace(
                  value.copyWith(primaryLanguage: language),
                );
              },
            ),
          ),
          SizedBox(height: spacing.sp4),
          CraftskyCard(
            key: const Key('content-languages-card'),
            clipBehavior: Clip.none,
            padding: EdgeInsets.all(spacing.sp4),
            child: CraftskySearchableMultiSelectInput<String>(
              key: const Key('content-languages-input'),
              keyPrefix: 'content-languages',
              label: l10n.contentLanguagesTitle,
              helperText: l10n.contentLanguagesDescription,
              values: value.contentLanguages,
              options: _languageOptions,
              enabled: !_saving,
              searchHintText: l10n.languageAddMore,
              onChanged: (languages) async {
                await _replace(
                  value.copyWith(contentLanguages: languages),
                );
              },
            ),
          ),
        ],
      ),
      AsyncError() => CraftskyCard(
        key: const Key('language-preferences-error-card'),
        clipBehavior: Clip.none,
        padding: EdgeInsets.all(spacing.sp4),
        child: Column(
          children: [
            Icon(
              Icons.cloud_off_outlined,
              color: theme.colorScheme.primary,
            ),
            SizedBox(height: spacing.sp3),
            FilledButton(
              onPressed: () => ref.invalidate(provider),
              child: Text(l10n.retryButton),
            ),
          ],
        ),
      ),
      _ => Padding(
        padding: EdgeInsets.all(spacing.sp5),
        child: const Center(child: StitchProgressIndicator()),
      ),
    };
  }
}

class _AppLanguageField extends StatelessWidget {
  const _AppLanguageField();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final l10n = AppLocalizations.of(context);
    return CraftskyCard(
      key: const Key('app-language-card'),
      clipBehavior: Clip.none,
      padding: EdgeInsets.all(spacing.sp4),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          CraftskySingleSelectInput<String>(
            key: const Key('app-language-input'),
            keyPrefix: 'app-language',
            label: l10n.appLanguageTitle,
            helperText: l10n.appLanguageDescription,
            value: 'en',
            options: [
              CraftskySelectOption(
                value: 'en',
                label: l10n.appLanguageEnglish,
              ),
            ],
            onChanged: (_) {},
          ),
          SizedBox(height: spacing.sp3),
          Row(
            children: [
              Icon(
                Icons.translate,
                size: 18,
                color: theme.colorScheme.primary,
              ),
              SizedBox(width: spacing.sp2),
              Expanded(
                child: Text(
                  l10n.appLanguageMoreComing,
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

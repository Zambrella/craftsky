import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/active_account_initialization_provider.dart';
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
    final activeLease = ref
        .watch(activeAccountInitializationProvider)
        .requireValue
        ?.lease;
    if (activeLease == null) {
      throw StateError('Language settings require an initialized account');
    }
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
                _AccountLanguageSections(lease: activeLease),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _AccountLanguageSections extends ConsumerWidget {
  const _AccountLanguageSections({required this.lease});

  final ActiveAccountLease lease;

  Future<void> _replace(
    WidgetRef ref,
    LanguagePreferences candidate,
  ) async {
    await ref
        .read(
          accountLanguagePreferencesProvider(lease).notifier,
        )
        .replace(candidate);
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final l10n = AppLocalizations.of(context);
    final provider = accountLanguagePreferencesProvider(lease);
    final preferences = ref.watch(provider);
    ref.listen(provider, (previous, next) {
      if (previous?.value?.replacement.isLoading == true &&
          next.value?.replacement.hasError == true) {
        context.showError(l10n.languageSaveError);
      }
    });
    if (preferences.hasValue) {
      final accountState = preferences.requireValue;
      final value = accountState.preferences;
      return Column(
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
              enabled: !accountState.replacement.isLoading,
              searchHintText: l10n.languageSearchHint,
              onChanged: (language) async {
                if (language == null || language == value.primaryLanguage) {
                  return;
                }
                await _replace(
                  ref,
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
              enabled: !accountState.replacement.isLoading,
              searchHintText: l10n.languageAddMore,
              onChanged: (languages) async {
                await _replace(
                  ref,
                  value.copyWith(contentLanguages: languages),
                );
              },
            ),
          ),
        ],
      );
    }
    return switch (preferences) {
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

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/languages/data/language_catalogue.dart';
import 'package:craftsky_app/languages/models/language_preferences.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/shared/messaging/context_messenger_extension.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class LanguagesPage extends ConsumerWidget {
  const LanguagesPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context);
    final session = ref.watch(authSessionProvider);
    final activeDID = switch (session.value) {
      SignedIn(:final did) => did,
      _ => null,
    };
    return Scaffold(
      appBar: AppBar(title: Text(l10n.languagesTitle)),
      body: ListView(
        padding: const EdgeInsets.symmetric(vertical: 16),
        children: [
          _Section(
            title: l10n.appLanguageTitle,
            description: l10n.appLanguageDescription,
            child: const _AppLanguageField(),
          ),
          const Divider(height: 32),
          if (activeDID == null)
            const Center(child: CircularProgressIndicator())
          else
            _AccountLanguageSections(
              account: AccountKey(activeDID.value),
            ),
        ],
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
    final l10n = AppLocalizations.of(context);
    final provider = accountLanguagePreferencesProvider(widget.account);
    return switch (ref.watch(provider)) {
      AsyncData(:final value) => Column(
        children: [
          _Section(
            title: l10n.primaryLanguageTitle,
            description: l10n.primaryLanguageDescription,
            child: SizedBox(
              width: double.infinity,
              child: OutlinedButton(
                onPressed: _saving
                    ? null
                    : () async {
                        final language = await showDialog<String>(
                          context: context,
                          builder: (context) => _PrimaryLanguageDialog(
                            selected: value.primaryLanguage,
                          ),
                        );
                        if (language == null ||
                            language == value.primaryLanguage) {
                          return;
                        }
                        await _replace(
                          value.copyWith(primaryLanguage: language),
                        );
                      },
                child: Row(
                  children: [
                    Expanded(
                      child: Text(languageLabel(value.primaryLanguage)),
                    ),
                    const Icon(Icons.unfold_more),
                  ],
                ),
              ),
            ),
          ),
          const Divider(height: 32),
          _Section(
            title: l10n.contentLanguagesTitle,
            description: l10n.contentLanguagesDescription,
            child: _ContentLanguagesField(
              preferences: value,
              enabled: !_saving,
              onChanged: (languages) => _replace(
                value.copyWith(contentLanguages: languages),
              ),
            ),
          ),
        ],
      ),
      AsyncError() => Center(
        child: FilledButton(
          onPressed: () => ref.invalidate(provider),
          child: Text(l10n.retryButton),
        ),
      ),
      _ => const Center(child: CircularProgressIndicator()),
    };
  }
}

class _AppLanguageField extends StatelessWidget {
  const _AppLanguageField();

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        DropdownButtonFormField<String>(
          initialValue: 'en',
          items: [
            DropdownMenuItem(
              value: 'en',
              child: Text(l10n.appLanguageEnglish),
            ),
          ],
          onChanged: (_) {},
        ),
        const SizedBox(height: 8),
        Text(l10n.appLanguageMoreComing),
      ],
    );
  }
}

class _ContentLanguagesField extends StatelessWidget {
  const _ContentLanguagesField({
    required this.preferences,
    required this.onChanged,
    required this.enabled,
  });

  final LanguagePreferences preferences;
  final Future<void> Function(List<String>) onChanged;
  final bool enabled;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Wrap(
          spacing: 8,
          runSpacing: 4,
          children: [
            for (final language in preferences.contentLanguages)
              InputChip(
                label: Text(languageLabel(language)),
                onDeleted: enabled
                    ? () async => onChanged([
                        for (final value in preferences.contentLanguages)
                          if (value != language) value,
                      ])
                    : null,
              ),
          ],
        ),
        TextButton.icon(
          onPressed: enabled
              ? () async {
                  final selected = await showDialog<List<String>>(
                    context: context,
                    builder: (context) => _ContentLanguageDialog(
                      initial: preferences.contentLanguages,
                    ),
                  );
                  if (selected != null) await onChanged(selected);
                }
              : null,
          icon: const Icon(Icons.add),
          label: Text(AppLocalizations.of(context).languageAddMore),
        ),
      ],
    );
  }
}

class _ContentLanguageDialog extends StatefulWidget {
  const _ContentLanguageDialog({required this.initial});

  final List<String> initial;

  @override
  State<_ContentLanguageDialog> createState() => _ContentLanguageDialogState();
}

class _ContentLanguageDialogState extends State<_ContentLanguageDialog> {
  late final Set<String> _selected = widget.initial.toSet();
  var _query = '';

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final values =
        supportedLanguageTags
            .where(
              (tag) =>
                  tag.contains(_query.toLowerCase()) ||
                  languageLabel(
                    tag,
                  ).toLowerCase().contains(_query.toLowerCase()),
            )
            .toList()
          ..sort(
            (left, right) =>
                languageLabel(left).compareTo(languageLabel(right)),
          );
    return AlertDialog(
      title: Text(l10n.contentLanguagesTitle),
      content: SizedBox(
        width: 420,
        height: 480,
        child: Column(
          children: [
            TextField(
              autofocus: true,
              decoration: InputDecoration(
                hintText: l10n.languageSearchHint,
                prefixIcon: const Icon(Icons.search),
              ),
              onChanged: (value) => setState(() => _query = value),
            ),
            Expanded(
              child: ListView(
                children: [
                  for (final tag in values)
                    CheckboxListTile(
                      value: _selected.contains(tag),
                      title: Text(languageLabel(tag)),
                      subtitle: Text(tag),
                      onChanged: (checked) => setState(() {
                        if (checked ?? false) {
                          _selected.add(tag);
                        } else {
                          _selected.remove(tag);
                        }
                      }),
                    ),
                ],
              ),
            ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: Text(l10n.languageCancel),
        ),
        FilledButton(
          onPressed: () => Navigator.pop(context, _selected.toList()),
          child: Text(l10n.languageDone),
        ),
      ],
    );
  }
}

class _Section extends StatelessWidget {
  const _Section({
    required this.title,
    required this.description,
    required this.child,
  });

  final String title;
  final String description;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(title, style: Theme.of(context).textTheme.titleLarge),
          const SizedBox(height: 8),
          Text(description),
          const SizedBox(height: 12),
          child,
        ],
      ),
    );
  }
}

class _PrimaryLanguageDialog extends StatefulWidget {
  const _PrimaryLanguageDialog({required this.selected});

  final String selected;

  @override
  State<_PrimaryLanguageDialog> createState() => _PrimaryLanguageDialogState();
}

class _PrimaryLanguageDialogState extends State<_PrimaryLanguageDialog> {
  var _query = '';

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final query = _query.trim().toLowerCase();
    final values =
        supportedLanguageTags.where((tag) {
          if (query.isEmpty) return true;
          return tag.contains(query) ||
              languageLabel(tag).toLowerCase().contains(query);
        }).toList()..sort(
          (left, right) => languageLabel(left).compareTo(languageLabel(right)),
        );
    return AlertDialog(
      title: Text(l10n.primaryLanguageTitle),
      content: SizedBox(
        width: 420,
        height: 480,
        child: Column(
          children: [
            TextField(
              autofocus: true,
              decoration: InputDecoration(
                hintText: l10n.languageSearchHint,
                prefixIcon: const Icon(Icons.search),
              ),
              onChanged: (value) => setState(() => _query = value),
            ),
            Expanded(
              child: ListView(
                children: [
                  for (final tag in values)
                    ListTile(
                      selected: tag == widget.selected,
                      title: Text(languageLabel(tag)),
                      subtitle: Text(tag),
                      trailing: tag == widget.selected
                          ? const Icon(Icons.check)
                          : null,
                      onTap: () => Navigator.pop(context, tag),
                    ),
                ],
              ),
            ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: Text(l10n.languageCancel),
        ),
      ],
    );
  }
}

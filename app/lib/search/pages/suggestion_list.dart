part of 'search_page.dart';

class _SuggestionList {
  const _SuggestionList._();

  static List<Widget> slivers({
    required BuildContext context,
    required WidgetRef ref,
    required String query,
    required bool isWaitingForDebounce,
    required ValueChanged<ProfileSearchResult> onOpenProfile,
    required ValueChanged<String> onOpenHashtag,
    required VoidCallback onViewAllProfiles,
    required VoidCallback onViewAllHashtags,
  }) {
    final l10n = AppLocalizations.of(context);
    final spacing =
        Theme.of(context).extension<SpacingTheme>() ?? const SpacingTheme();
    if (isWaitingForDebounce || query.isEmpty) {
      return const [
        SliverFillRemaining(
          hasScrollBody: false,
          child: Center(child: StitchProgressIndicator()),
        ),
      ];
    }
    final suggestionsAsync = ref.watch(
      searchSuggestionsProvider(SearchSuggestionQuery(q: query)),
    );
    return switch (suggestionsAsync) {
      AsyncValue(:final value?) => [
        SliverPadding(
          padding: EdgeInsets.fromLTRB(
            spacing.sp4,
            spacing.sp2,
            spacing.sp4,
            spacing.sp5,
          ),
          sliver: SliverList.list(
            children: [
              _SuggestionProfileSection(
                suggestions: value,
                onOpenProfile: onOpenProfile,
                onViewAll: onViewAllProfiles,
              ),
              SizedBox(height: spacing.sp3),
              _SuggestionHashtagSection(
                suggestions: value,
                onOpenHashtag: onOpenHashtag,
                onViewAll: onViewAllHashtags,
              ),
            ],
          ),
        ),
      ],
      _ when suggestionsAsync.hasError => [
        _ErrorView(
          message: l10n.searchLoadError,
          onRetry: () => ref.invalidate(
            searchSuggestionsProvider(SearchSuggestionQuery(q: query)),
          ),
        ),
      ],
      _ => const [
        SliverFillRemaining(
          hasScrollBody: false,
          child: Center(child: StitchProgressIndicator()),
        ),
      ],
    };
  }
}

class _SuggestionProfileSection extends StatelessWidget {
  const _SuggestionProfileSection({
    required this.suggestions,
    required this.onOpenProfile,
    required this.onViewAll,
  });

  final SearchSuggestions suggestions;
  final ValueChanged<ProfileSearchResult> onOpenProfile;
  final VoidCallback onViewAll;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return _SuggestionSectionScaffold(
      title: l10n.searchProfilesHeading,
      hasMore: suggestions.profiles.hasMore,
      onViewAll: onViewAll,
      children: [
        for (final profile in suggestions.profiles.items)
          _ProfileResultTile(
            profile: profile,
            onTap: () => onOpenProfile(profile),
          ),
      ],
    );
  }
}

class _SuggestionHashtagSection extends StatelessWidget {
  const _SuggestionHashtagSection({
    required this.suggestions,
    required this.onOpenHashtag,
    required this.onViewAll,
  });

  final SearchSuggestions suggestions;
  final ValueChanged<String> onOpenHashtag;
  final VoidCallback onViewAll;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return _SuggestionSectionScaffold(
      title: l10n.searchHashtagsHeading,
      hasMore: suggestions.hashtags.hasMore,
      onViewAll: onViewAll,
      children: [
        for (final hashtag in suggestions.hashtags.items)
          _HashtagResultTile(
            hashtag: hashtag,
            onTap: () => onOpenHashtag(hashtag.tag),
          ),
      ],
    );
  }
}

class _SuggestionSectionScaffold extends StatelessWidget {
  const _SuggestionSectionScaffold({
    required this.title,
    required this.hasMore,
    required this.onViewAll,
    required this.children,
  });

  final String title;
  final bool hasMore;
  final VoidCallback onViewAll;
  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    if (children.isEmpty) return const SizedBox.shrink();
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Row(
          children: [
            Expanded(
              child: Text(
                title,
                style: Theme.of(context).textTheme.titleMedium,
              ),
            ),
            if (hasMore)
              TextButton(
                onPressed: onViewAll,
                child: Text(l10n.searchViewAllAction),
              ),
          ],
        ),
        ...children,
      ],
    );
  }
}

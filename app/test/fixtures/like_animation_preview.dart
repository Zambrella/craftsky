// Local-only preview: like toggles update in memory, without network writes.
import 'package:craftsky_app/auth/models/session_registry.dart' as registry;
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/feed/widgets/post_like_reaction.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

void main() => runApp(
  ProviderScope(
    overrides: [
      secureSessionRegistryStorageProvider.overrideWithValue(
        _PreviewRegistry(),
      ),
    ],
    child: const _Preview(),
  ),
);

class _PreviewRegistry implements SessionRegistryStorage {
  @override
  Future<registry.SessionRegistry> read() async =>
      registry.SessionRegistry.empty();

  @override
  Future<void> write(registry.SessionRegistry value) async {}
}

class _Preview extends ConsumerStatefulWidget {
  const new();

  @override
  ConsumerState<_Preview> createState() => _PreviewState();
}

class _PreviewState extends ConsumerState<_Preview> {
  bool _liked = false;
  bool _dark = false;
  bool _reduced = false;

  final _post = Post(
    uri: 'at://did:plc:preview/social.craftsky.feed.post/preview',
    cid: 'bafy123',
    rkey: 'preview',
    text:
        'Fresh off the needles.\n\nA few quiet evenings, a favourite skein, '
        'and one more row. This little piece is finally ready to wear.',
    tags: const [],
    createdAt: DateTime.now(),
    indexedAt: DateTime.now(),
    author: PostAuthor(
      did: 'did:plc:preview',
      handle: 'maker.craftsky.social',
      displayName: 'A happy maker',
    ),
    likeCount: 12,
    repostCount: 0,
    replyCount: 0,
    viewerHasLiked: false,
    viewerHasReposted: false,
    viewerHasSaved: false,
    sponsored: false,
  );

  @override
  Widget build(BuildContext context) => MaterialApp(
    theme: AppTheme.lightThemeData,
    darkTheme: AppTheme.darkThemeData,
    themeMode: _dark ? ThemeMode.dark : ThemeMode.light,
    localizationsDelegates: appLocalizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: Builder(
      builder: (context) => MediaQuery(
        data: MediaQuery.of(context).copyWith(disableAnimations: _reduced),
        child: Scaffold(
          appBar: AppBar(
            title: const Text('Stitched-heart like preview'),
            actions: [
              Padding(
                padding: const EdgeInsets.only(right: 16),
                child: Text(switch (ref.watch(likeAnimationFileProvider)) {
                  AsyncData() => 'Animation ready',
                  AsyncError() => 'Fallback heart',
                  _ => 'Loading animation',
                }),
              ),
            ],
          ),
          body: ListView(
            children: [
              SwitchListTile(
                title: const Text('Dark mode'),
                value: _dark,
                onChanged: (value) => setState(() => _dark = value),
              ),
              SwitchListTile(
                title: const Text('Reduced motion'),
                value: _reduced,
                onChanged: (value) => setState(() => _reduced = value),
              ),
              Center(
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 520),
                  child: PostCard(
                    post: _post.copyWith(
                      viewerHasLiked: _liked,
                      likeCount: _liked ? 13 : 12,
                    ),
                    onLike: () => setState(() => _liked = !_liked),
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    ),
  );
}

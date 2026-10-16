// Local UI preview; the refresh callback only waits and updates a counter.
import 'package:craftsky_app/auth/models/session_registry.dart' as registry;
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/feed/widgets/post_card.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:craftsky_app/theme/yarn_refresh_scroll_view.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

void main() => runApp(
  ProviderScope(
    overrides: [
      secureSessionRegistryStorageProvider.overrideWithValue(_Storage()),
    ],
    child: const _Preview(),
  ),
);

class _Storage implements SessionRegistryStorage {
  @override
  Future<registry.SessionRegistry> read() async =>
      registry.SessionRegistry.empty();
  @override
  Future<void> write(registry.SessionRegistry value) async {}
}

class _Preview extends StatefulWidget {
  const new();
  @override
  State<_Preview> createState() => _PreviewState();
}

class _PreviewState extends State<_Preview> {
  int _refreshes = 0;
  bool _dark = false;
  bool _reduced = false;
  Post post(int i) => Post(
    uri: 'at://did:plc:preview/social.craftsky.feed.post/demo$i',
    cid: 'bafy123',
    rkey: 'demo$i',
    text: i == 0
        ? 'A quiet morning with a favourite skein.\n\n'
              'Pull down to unwind the yarn above this post.'
        : 'Fresh off the needles. Just one more row…',
    tags: const [],
    createdAt: DateTime(2026, 10, 16),
    indexedAt: DateTime(2026, 10, 16),
    author: PostAuthor(
      did: 'did:plc:preview',
      handle: 'alice.craftsky.social',
      displayName: 'Alice',
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
    scrollBehavior: const MaterialScrollBehavior().copyWith(
      dragDevices: {
        PointerDeviceKind.touch,
        PointerDeviceKind.mouse,
        PointerDeviceKind.stylus,
        PointerDeviceKind.trackpad,
      },
    ),
    theme: AppTheme.lightThemeData,
    darkTheme: AppTheme.darkThemeData,
    themeMode: _dark ? ThemeMode.dark : ThemeMode.light,
    localizationsDelegates: appLocalizationsDelegates,
    supportedLocales: AppLocalizations.supportedLocales,
    home: Builder(
      builder: (context) => MediaQuery(
        data: MediaQuery.of(context).copyWith(disableAnimations: _reduced),
        child: Scaffold(
          body: YarnRefreshScrollView(
            onRefresh: () async {
              await Future<void>.delayed(const Duration(milliseconds: 1500));
              if (mounted) setState(() => _refreshes++);
            },
            headerSliverCount: 1,
            slivers: [
              SliverAppBar(
                title: Text('Feed · $_refreshes refreshes'),
                pinned: true,
                actions: [
                  IconButton(
                    tooltip: 'Toggle dark mode',
                    onPressed: () => setState(() => _dark = !_dark),
                    icon: const Icon(Icons.dark_mode),
                  ),
                  IconButton(
                    tooltip: 'Toggle reduced motion',
                    onPressed: () => setState(() => _reduced = !_reduced),
                    icon: const Icon(Icons.accessibility_new),
                  ),
                ],
              ),
              SliverList.list(
                children: [for (var i = 0; i < 6; i++) PostCard(post: post(i))],
              ),
            ],
          ),
        ),
      ),
    ),
  );
}

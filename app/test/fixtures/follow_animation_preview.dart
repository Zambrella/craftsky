// Local preview: full profile, in-memory follow writes.
import 'package:craftsky_app/auth/models/session_registry.dart' as registry;
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/auth/providers/secure_token_storage.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/pages/profile_page.dart';
import 'package:craftsky_app/profile/providers/profile_repository_provider.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

import '../fakes/auth_session_fakes.dart';
import '../profile/fakes/fake_profile_repository.dart';

void main() {
  var profile = Profile(
    did: 'did:plc:previewmaker',
    handle: 'alice.craftsky.social',
    displayName: 'Alice',
    crafts: const [],
    followerCount: 42,
  );
  runApp(
    ProviderScope(
      overrides: [
        secureSessionRegistryStorageProvider.overrideWithValue(
          _PreviewRegistry(),
        ),
        authSessionProvider.overrideWith(SignedInAuthSession.new),
        profileRepositoryProvider.overrideWithValue(
          FakeProfileRepository(
            onFetch: (_) async => profile,
            onFollow: (_) async {
              await Future<void>.delayed(const Duration(milliseconds: 250));
              return profile = profile.copyWith(
                viewerIsFollowing: true,
                followerCount: 43,
              );
            },
            onUnfollow: (_) async => profile = profile.copyWith(
              viewerIsFollowing: false,
              followerCount: 42,
            ),
          ),
        ),
      ],
      child: MaterialApp(
        theme: AppTheme.lightThemeData,
        localizationsDelegates: appLocalizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
        home: Scaffold(
          body: ProfilePage(did: profile.did),
        ),
      ),
    ),
  );
}

class _PreviewRegistry implements SessionRegistryStorage {
  @override
  Future<registry.SessionRegistry> read() async =>
      registry.SessionRegistry.empty();
  @override
  Future<void> write(registry.SessionRegistry value) async {}
}

import 'package:craftsky_app/auth/models/session_registry.dart';
import 'package:craftsky_app/languages/providers/language_preferences_provider.dart';
import 'package:craftsky_app/shared/messaging/messenger_scope.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../fakes/recording_messenger.dart';
import 'language_overrides.dart';
import 'post_fixtures.dart';
import 'provider_container.dart';
import 'session_registry_storage.dart';
import 'widget_pump.dart';

void main() {
  test('in-memory session storage retains the latest registry', () async {
    final initial = SessionRegistry.empty();
    final replacement = SessionRegistry.empty();
    final storage = InMemorySessionRegistryStorage(initial);

    expect(await storage.read(), same(initial));

    await storage.write(replacement);

    expect(await storage.read(), same(replacement));
  });

  test('post model and wire fixtures share canonical defaults', () {
    final post = postFixture(rkey: 'fixture', text: 'Fixture post');
    final wire = postWirePayload(rkey: 'fixture', text: 'Fixture post');

    expect(post.uri.toString(), wire['uri']);
    expect(post.cid.toString(), wire['cid']);
    expect(post.rkey.toString(), wire['rkey']);
    expect(post.text, wire['text']);
    expect(post.author.did.toString(), (wire['author'] as Map)['did']);
  });

  test('test container composes common provider overrides', () {
    final container = createTestContainer(
      overrides: [englishLanguagePreferencesOverride],
    );

    expect(
      container.read(activeLanguagePreferencesProvider),
      englishLanguagePreferences,
    );
  });

  testWidgets('widget pump composes explicit app options', (tester) async {
    final messenger = RecordingMessenger();
    late Size mediaSize;
    late Locale locale;
    late bool hasMessenger;

    await pumpCraftskyWidget(
      tester,
      Consumer(
        builder: (context, ref, child) {
          mediaSize = MediaQuery.sizeOf(context);
          locale = Localizations.localeOf(context);
          hasMessenger = MessengerScope.of(context) == messenger;
          return Text(
            ref.watch(activeLanguagePreferencesProvider).primaryLanguage,
          );
        },
      ),
      overrides: [englishLanguagePreferencesOverride],
      surfaceSize: const Size(320, 640),
      messenger: messenger,
    );

    expect(find.text('en'), findsOneWidget);
    expect(mediaSize, const Size(320, 640));
    expect(locale, const Locale('en'));
    expect(hasMessenger, isTrue);
  });
}

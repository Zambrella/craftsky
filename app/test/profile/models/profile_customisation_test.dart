import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/search/models/profile_search_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  setUpAll(initializeMappers);

  test(
    'profile customisation catalogues and defaults are closed and stable',
    () {
      expect(profileColourCatalogue, [
        'cobalt',
        'orchid',
        'rose',
        'amber',
        'lime',
        'teal',
      ]);
      expect(profileBorderCatalogue, ['thin', 'medium', 'thick']);
      expect(profileBackgroundCatalogue, [
        'none',
        'bayerdark',
        'cubedark',
        'dotcrossdark',
        'scallopdark',
        'skewdark',
        'x2',
      ]);

      expect(profileColourCatalogue.toSet(), hasLength(6));
      expect(profileBorderCatalogue.toSet(), hasLength(3));
      expect(profileBackgroundCatalogue.toSet(), hasLength(7));
      expect(ProfileCustomisation.defaults.toMap(), {
        'colour': 'cobalt',
        'profileBorder': 'medium',
        'profileBackground': 'none',
      });
    },
  );

  test('colour keys resolve to the approved local theme bundles', () {
    expect(profileColourBundles.keys, profileColourCatalogue);
    expect(profileColourBundles, {
      'cobalt': const ProfileColourBundle(
        base: '#1535D6',
        foreground: '#FFFFFF',
        hover: '#122EBA',
        pressed: '#0F279E',
        softContainer: '#D8DDF9',
        textureTint: '#FFFFFF',
        textureOpacity: 0.18,
      ),
      'orchid': const ProfileColourBundle(
        base: '#B615D6',
        foreground: '#FFFFFF',
        hover: '#9E12BA',
        pressed: '#860F9E',
        softContainer: '#F3D8F9',
        textureTint: '#FFFFFF',
        textureOpacity: 0.18,
      ),
      'rose': const ProfileColourBundle(
        base: '#D61535',
        foreground: '#FFFFFF',
        hover: '#BA122E',
        pressed: '#9E0F27',
        softContainer: '#F9D8DD',
        textureTint: '#FFFFFF',
        textureOpacity: 0.18,
      ),
      'amber': const ProfileColourBundle(
        base: '#D6B615',
        foreground: '#111318',
        hover: '#BA9E12',
        pressed: '#9E860F',
        softContainer: '#F9F3D8',
        textureTint: '#111318',
        textureOpacity: 0.18,
      ),
      'lime': const ProfileColourBundle(
        base: '#35D615',
        foreground: '#111318',
        hover: '#2EBA12',
        pressed: '#279E0F',
        softContainer: '#DDF9D8',
        textureTint: '#111318',
        textureOpacity: 0.18,
      ),
      'teal': const ProfileColourBundle(
        base: '#15D6B6',
        foreground: '#111318',
        hover: '#12BA9E',
        pressed: '#0F9E86',
        softContainer: '#D8F9F3',
        textureTint: '#111318',
        textureOpacity: 0.18,
      ),
    });
  });

  test('every approved foreground and container pair meets AA contrast', () {
    for (final bundle in profileColourBundles.values) {
      final foreground = _colour(bundle.foreground);
      for (final background in [bundle.base, bundle.hover, bundle.pressed]) {
        expect(
          _contrast(foreground, _colour(background)),
          greaterThanOrEqualTo(4.5),
          reason: '${bundle.foreground} on $background',
        );
      }
      expect(
        _contrast(const Color(0xFF111318), _colour(bundle.softContainer)),
        greaterThanOrEqualTo(4.5),
        reason: '#111318 on ${bundle.softContainer}',
      );
    }
  });

  test(
    'unknown values fall back independently without discarding siblings',
    () {
      final customisation = ProfileCustomisation.fromMap(const {
        'colour': 'future-colour',
        'profileBorder': 'thick',
        'profileBackground': 'cubedark',
      });

      expect(
        customisation,
        const ProfileCustomisation(border: 'thick', background: 'cubedark'),
      );
      expect(
        ProfileCustomisation.fromMap(const {
          'colour': 'teal',
          'profileBorder': 'future-border',
          'profileBackground': 'x2',
        }),
        const ProfileCustomisation(colour: 'teal', background: 'x2'),
      );
      expect(
        ProfileCustomisation.fromMap(const {
          'colour': 'rose',
          'profileBorder': 'thin',
          'profileBackground': 'future-background',
        }),
        const ProfileCustomisation(colour: 'rose', border: 'thin'),
      );
    },
  );

  test('identity models default absent customisation', () {
    final profile = ProfileMapper.fromMap({
      'did': 'did:plc:alice',
      'handle': 'alice.example',
      'crafts': <String>[],
    });
    final author = PostAuthorMapper.fromMap({
      'did': 'did:plc:alice',
      'handle': 'alice.example',
    });
    final account = ProfileAccountSummaryMapper.fromMap({
      'did': 'did:plc:alice',
      'handle': 'alice.example',
      'isCraftskyProfile': true,
    });
    final actor = NotificationActor.fromMap({
      'did': 'did:plc:alice',
      'handle': 'alice.example',
    });
    final search = ProfileSearchResultMapper.fromMap({
      'did': 'did:plc:alice',
      'handle': 'alice.example',
      'isCraftskyProfile': true,
      'viewerIsFollowing': false,
    });

    expect(profile.customisation, ProfileCustomisation.defaults);
    expect(author.customisation, ProfileCustomisation.defaults);
    expect(account.customisation, ProfileCustomisation.defaults);
    expect(actor.customisation, ProfileCustomisation.defaults);
    expect(search.customisation, ProfileCustomisation.defaults);
    expect(search.summary.customisation, ProfileCustomisation.defaults);
  });

  test('identity models decode customisation with per-field fallback', () {
    final profile = ProfileMapper.fromMap({
      'did': 'did:plc:alice',
      'handle': 'alice.example',
      'crafts': <String>[],
      'customisation': {
        'colour': 'teal',
        'profileBorder': 'future-border',
        'profileBackground': 'x2',
      },
    });
    final malformed = PostAuthorMapper.fromMap({
      'did': 'did:plc:bob',
      'handle': 'bob.example',
      'customisation': 'not-an-object',
    });

    expect(
      profile.customisation,
      const ProfileCustomisation(colour: 'teal', background: 'x2'),
    );
    expect(malformed.customisation, ProfileCustomisation.defaults);
  });
}

Color _colour(String hex) =>
    Color(int.parse(hex.substring(1), radix: 16) | 0xFF000000);

double _contrast(Color a, Color b) {
  final first = a.computeLuminance();
  final second = b.computeLuminance();
  final lighter = first > second ? first : second;
  final darker = first > second ? second : first;
  return (lighter + 0.05) / (darker + 0.05);
}

import 'package:craftsky_app/bootstrap.dart';
import 'package:craftsky_app/feed/models/post.dart';
import 'package:craftsky_app/notifications/models/craftsky_notification.dart';
import 'package:craftsky_app/profile/models/profile.dart';
import 'package:craftsky_app/profile/models/profile_account_summary.dart';
import 'package:craftsky_app/profile/models/profile_customisation.dart';
import 'package:craftsky_app/search/models/profile_search_page.dart';
import 'package:craftsky_app/theme/brand_colors.dart';
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
        'ink',
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

      expect(profileColourCatalogue.toSet(), hasLength(7));
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
        darkAccent: '#7890FF',
        darkForeground: '#171513',
        darkHover: '#8EA0FF',
        darkPressed: '#6A81ED',
        softContainer: '#D8DDF9',
        textureTint: '#FFFFFF',
        textureOpacity: 0.18,
      ),
      'orchid': const ProfileColourBundle(
        base: '#B615D6',
        foreground: '#FFFFFF',
        hover: '#9E12BA',
        pressed: '#860F9E',
        darkAccent: '#DF7AF4',
        darkForeground: '#171513',
        darkHover: '#E98CF8',
        darkPressed: '#C966DF',
        softContainer: '#F3D8F9',
        textureTint: '#FFFFFF',
        textureOpacity: 0.18,
      ),
      'rose': const ProfileColourBundle(
        base: '#D61535',
        foreground: '#FFFFFF',
        hover: '#BA122E',
        pressed: '#9E0F27',
        darkAccent: '#FF7088',
        darkForeground: '#171513',
        darkHover: '#FF8799',
        darkPressed: '#E85E76',
        softContainer: '#F9D8DD',
        textureTint: '#FFFFFF',
        textureOpacity: 0.18,
      ),
      'amber': const ProfileColourBundle(
        base: '#766200',
        foreground: '#FFFFFF',
        hover: '#655300',
        pressed: '#544500',
        darkAccent: '#F7D46A',
        darkForeground: '#171513',
        darkHover: '#FFE083',
        darkPressed: '#DDBB55',
        softContainer: '#F9F3D8',
        textureTint: '#FFFFFF',
        textureOpacity: 0.18,
      ),
      'lime': const ProfileColourBundle(
        base: '#23770F',
        foreground: '#FFFFFF',
        hover: '#1D650C',
        pressed: '#175309',
        darkAccent: '#8FD36F',
        darkForeground: '#171513',
        darkHover: '#A2DF83',
        darkPressed: '#77BC59',
        softContainer: '#DDF9D8',
        textureTint: '#FFFFFF',
        textureOpacity: 0.18,
      ),
      'teal': const ProfileColourBundle(
        base: '#007663',
        foreground: '#FFFFFF',
        hover: '#006454',
        pressed: '#005146',
        darkAccent: '#5BD0BC',
        darkForeground: '#171513',
        darkHover: '#73DCCB',
        darkPressed: '#43B9A5',
        softContainer: '#D8F9F3',
        textureTint: '#FFFFFF',
        textureOpacity: 0.18,
      ),
      'ink': const ProfileColourBundle(
        base: '#161210',
        foreground: '#FFFFFF',
        hover: '#3E3733',
        pressed: '#0B0908',
        darkAccent: '#F5EFE4',
        darkForeground: '#161210',
        darkHover: '#FFFFFF',
        darkPressed: '#CFC6BB',
        softContainer: '#EFE7D6',
        textureTint: '#FFFFFF',
        textureOpacity: 0.18,
      ),
    });

    final ink = profileColourBundles['ink']!;
    expect(_colour(ink.base), BrandColors.ink);
    expect(_colour(ink.hover), BrandColors.ink2);
    expect(_colour(ink.softContainer), BrandColors.paper2);
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

  test('profile accent and link states meet AA contrast on paper surfaces', () {
    for (final bundle in profileColourBundles.values) {
      for (final accent in [bundle.base, bundle.hover, bundle.pressed]) {
        for (final surface in [BrandColors.paper, BrandColors.paper3]) {
          expect(
            _contrast(_colour(accent), surface),
            greaterThanOrEqualTo(4.5),
            reason: '$accent on ${surface.toARGB32().toRadixString(16)}',
          );
        }
      }
    }
  });

  test('dark profile accent states meet AA contrast in both directions', () {
    const darkSurfaces = [Color(0xFF171513), Color(0xFF24201D)];

    for (final bundle in profileColourBundles.values) {
      final foreground = _colour(bundle.darkForeground);
      for (final accent in [
        bundle.darkAccent,
        bundle.darkHover,
        bundle.darkPressed,
      ]) {
        final accentColour = _colour(accent);
        expect(
          _contrast(foreground, accentColour),
          greaterThanOrEqualTo(4.5),
          reason: '${bundle.darkForeground} on $accent',
        );
        for (final surface in darkSurfaces) {
          expect(
            _contrast(accentColour, surface),
            greaterThanOrEqualTo(4.5),
            reason: '$accent on ${surface.toARGB32().toRadixString(16)}',
          );
        }
      }
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

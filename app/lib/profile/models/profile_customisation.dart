import 'package:dart_mappable/dart_mappable.dart';
import 'package:flutter/foundation.dart';

const profileColourCatalogue = <String>[
  'cobalt',
  'orchid',
  'rose',
  'amber',
  'lime',
  'teal',
  'ink',
];

const profileBorderCatalogue = <String>['thin', 'medium', 'thick'];

const profileBackgroundCatalogue = <String>[
  'none',
  'bayerdark',
  'cubedark',
  'dotcrossdark',
  'scallopdark',
  'skewdark',
  'x2',
];

@immutable
class ProfileColourBundle {
  const ProfileColourBundle({
    required this.base,
    required this.foreground,
    required this.hover,
    required this.pressed,
    required this.softContainer,
    required this.textureTint,
    required this.textureOpacity,
  });

  final String base;
  final String foreground;
  final String hover;
  final String pressed;
  final String softContainer;
  final String textureTint;
  final double textureOpacity;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ProfileColourBundle &&
          base == other.base &&
          foreground == other.foreground &&
          hover == other.hover &&
          pressed == other.pressed &&
          softContainer == other.softContainer &&
          textureTint == other.textureTint &&
          textureOpacity == other.textureOpacity;

  @override
  int get hashCode => Object.hash(
    base,
    foreground,
    hover,
    pressed,
    softContainer,
    textureTint,
    textureOpacity,
  );
}

const profileColourBundles = <String, ProfileColourBundle>{
  'cobalt': ProfileColourBundle(
    base: '#1535D6',
    foreground: '#FFFFFF',
    hover: '#122EBA',
    pressed: '#0F279E',
    softContainer: '#D8DDF9',
    textureTint: '#FFFFFF',
    textureOpacity: 0.18,
  ),
  'orchid': ProfileColourBundle(
    base: '#B615D6',
    foreground: '#FFFFFF',
    hover: '#9E12BA',
    pressed: '#860F9E',
    softContainer: '#F3D8F9',
    textureTint: '#FFFFFF',
    textureOpacity: 0.18,
  ),
  'rose': ProfileColourBundle(
    base: '#D61535',
    foreground: '#FFFFFF',
    hover: '#BA122E',
    pressed: '#9E0F27',
    softContainer: '#F9D8DD',
    textureTint: '#FFFFFF',
    textureOpacity: 0.18,
  ),
  'amber': ProfileColourBundle(
    base: '#766200',
    foreground: '#FFFFFF',
    hover: '#655300',
    pressed: '#544500',
    softContainer: '#F9F3D8',
    textureTint: '#FFFFFF',
    textureOpacity: 0.18,
  ),
  'lime': ProfileColourBundle(
    base: '#23770F',
    foreground: '#FFFFFF',
    hover: '#1D650C',
    pressed: '#175309',
    softContainer: '#DDF9D8',
    textureTint: '#FFFFFF',
    textureOpacity: 0.18,
  ),
  'teal': ProfileColourBundle(
    base: '#007663',
    foreground: '#FFFFFF',
    hover: '#006454',
    pressed: '#005146',
    softContainer: '#D8F9F3',
    textureTint: '#FFFFFF',
    textureOpacity: 0.18,
  ),
  'ink': ProfileColourBundle(
    base: '#161210',
    foreground: '#FFFFFF',
    hover: '#3E3733',
    pressed: '#0B0908',
    softContainer: '#EFE7D6',
    textureTint: '#FFFFFF',
    textureOpacity: 0.18,
  ),
};

@immutable
class ProfileCustomisation {
  const ProfileCustomisation({
    this.colour = 'cobalt',
    this.border = 'medium',
    this.background = 'none',
  });

  factory ProfileCustomisation.fromMap(Map<String, dynamic>? map) {
    return ProfileCustomisation(
      colour: _effectiveKey(
        map?['colour'],
        profileColourCatalogue,
        defaults.colour,
      ),
      border: _effectiveKey(
        map?['profileBorder'],
        profileBorderCatalogue,
        defaults.border,
      ),
      background: _effectiveKey(
        map?['profileBackground'],
        profileBackgroundCatalogue,
        defaults.background,
      ),
    );
  }

  static const defaults = ProfileCustomisation();

  final String colour;
  final String border;
  final String background;

  Map<String, String> toMap() => {
    'colour': colour,
    'profileBorder': border,
    'profileBackground': background,
  };

  ProfileCustomisation copyWith({
    String? colour,
    String? border,
    String? background,
  }) => ProfileCustomisation(
    colour: colour ?? this.colour,
    border: border ?? this.border,
    background: background ?? this.background,
  );

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ProfileCustomisation &&
          colour == other.colour &&
          border == other.border &&
          background == other.background;

  @override
  int get hashCode => Object.hash(colour, border, background);
}

class ProfileCustomisationMapper extends SimpleMapper<ProfileCustomisation> {
  const ProfileCustomisationMapper();

  @override
  ProfileCustomisation decode(dynamic value) {
    if (value is! Map) return ProfileCustomisation.defaults;
    return ProfileCustomisation.fromMap(Map<String, dynamic>.from(value));
  }

  @override
  Object encode(ProfileCustomisation self) => self.toMap();
}

String _effectiveKey(Object? raw, List<String> catalogue, String fallback) {
  return raw is String && catalogue.contains(raw) ? raw : fallback;
}

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
  'terracotta',
  'denim',
  'forest',
  'plum',
  'seafoam',
  'dusty-pink',
  'chocolate',
];

const profileBackgroundCatalogue = <String>[
  'none',
  'bayerdark',
  'cubedark',
  'dotcrossdark',
  'scallopdark',
  'skewdark',
  'x2',
  'craft-sewing',
  'craft-knitting',
  'craft-crochet',
  'craft-quilting',
  'craft-embroidery',
  'animated-flow',
  'animated-waves',
  'animated-weave',
  'animated-grain',
  'animated-marble',
  'animated-textile',
];

const profileShaderBackgrounds = <String, int>{
  'animated-flow': 0,
  'animated-waves': 1,
  'animated-weave': 2,
  'animated-grain': 3,
  'animated-marble': 4,
  'animated-textile': 5,
};

const profileCraftBackgrounds = <String, String>{
  'craft-sewing': 'sewing',
  'craft-knitting': 'knitting',
  'craft-crochet': 'crochet',
  'craft-quilting': 'quilting',
  'craft-embroidery': 'embroidery',
};

@immutable
class ProfileColourBundle {
  const new({
    required this.base,
    required this.foreground,
    required this.hover,
    required this.pressed,
    required this.darkAccent,
    required this.darkForeground,
    required this.darkHover,
    required this.darkPressed,
    required this.softContainer,
    required this.textureTint,
    required this.textureOpacity,
  });

  final String base;
  final String foreground;
  final String hover;
  final String pressed;
  final String darkAccent;
  final String darkForeground;
  final String darkHover;
  final String darkPressed;
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
          darkAccent == other.darkAccent &&
          darkForeground == other.darkForeground &&
          darkHover == other.darkHover &&
          darkPressed == other.darkPressed &&
          softContainer == other.softContainer &&
          textureTint == other.textureTint &&
          textureOpacity == other.textureOpacity;

  @override
  int get hashCode => Object.hash(
    base,
    foreground,
    hover,
    pressed,
    darkAccent,
    darkForeground,
    darkHover,
    darkPressed,
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
    darkAccent: '#7890FF',
    darkForeground: '#171513',
    darkHover: '#8EA0FF',
    darkPressed: '#6A81ED',
    softContainer: '#D8DDF9',
    textureTint: '#FFFFFF',
    textureOpacity: 0.18,
  ),
  'orchid': ProfileColourBundle(
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
  'rose': ProfileColourBundle(
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
  'amber': ProfileColourBundle(
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
  'lime': ProfileColourBundle(
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
  'teal': ProfileColourBundle(
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
  'ink': ProfileColourBundle(
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
  'terracotta': ProfileColourBundle(
    base: '#9A442D',
    foreground: '#FFFFFF',
    hover: '#863B27',
    pressed: '#723221',
    darkAccent: '#ECA68E',
    darkForeground: '#171513',
    darkHover: '#EFB29E',
    darkPressed: '#D0927D',
    softContainer: '#F4DDD4',
    textureTint: '#FFFFFF',
    textureOpacity: 0.18,
  ),
  'denim': ProfileColourBundle(
    base: '#365B80',
    foreground: '#FFFFFF',
    hover: '#2F4F6F',
    pressed: '#28435F',
    darkAccent: '#9FBDDC',
    darkForeground: '#171513',
    darkHover: '#ACC6E1',
    darkPressed: '#8CA6C2',
    softContainer: '#DFE8F2',
    textureTint: '#FFFFFF',
    textureOpacity: 0.18,
  ),
  'forest': ProfileColourBundle(
    base: '#286044',
    foreground: '#FFFFFF',
    hover: '#23543B',
    pressed: '#1E4732',
    darkAccent: '#97C8A9',
    darkForeground: '#171513',
    darkHover: '#A6D0B5',
    darkPressed: '#85B095',
    softContainer: '#DDEDE2',
    textureTint: '#FFFFFF',
    textureOpacity: 0.18,
  ),
  'plum': ProfileColourBundle(
    base: '#744369',
    foreground: '#FFFFFF',
    hover: '#653A5B',
    pressed: '#56324E',
    darkAccent: '#CCA4C4',
    darkForeground: '#171513',
    darkHover: '#D3B1CC',
    darkPressed: '#B490AC',
    softContainer: '#EEDFEB',
    textureTint: '#FFFFFF',
    textureOpacity: 0.18,
  ),
  'seafoam': ProfileColourBundle(
    base: '#246B60',
    foreground: '#FFFFFF',
    hover: '#1F5D54',
    pressed: '#1B4F47',
    darkAccent: '#9BD4C6',
    darkForeground: '#171513',
    darkHover: '#A9DACE',
    darkPressed: '#88BBAE',
    softContainer: '#DDF0EA',
    textureTint: '#FFFFFF',
    textureOpacity: 0.18,
  ),
  'dusty-pink': ProfileColourBundle(
    base: '#975165',
    foreground: '#FFFFFF',
    hover: '#834658',
    pressed: '#703C4B',
    darkAccent: '#DFAABC',
    darkForeground: '#171513',
    darkHover: '#E3B6C5',
    darkPressed: '#C496A5',
    softContainer: '#F3E0E7',
    textureTint: '#FFFFFF',
    textureOpacity: 0.18,
  ),
  'chocolate': ProfileColourBundle(
    base: '#694634',
    foreground: '#FFFFFF',
    hover: '#5B3D2D',
    pressed: '#4E3426',
    darkAccent: '#D2B29A',
    darkForeground: '#171513',
    darkHover: '#D8BDA8',
    darkPressed: '#B99D88',
    softContainer: '#EDE1D5',
    textureTint: '#FFFFFF',
    textureOpacity: 0.18,
  ),
};

@immutable
class ProfileCustomisation {
  const new({
    this.colour = 'cobalt',
    this.background = 'none',
  });

  factory fromMap(Map<String, dynamic>? map) {
    return ProfileCustomisation(
      colour: _effectiveKey(
        map?['colour'],
        profileColourCatalogue,
        defaults.colour,
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
  final String background;

  Map<String, String> toMap() => {
    'colour': colour,
    'profileBackground': background,
  };

  ProfileCustomisation copyWith({
    String? colour,
    String? background,
  }) => ProfileCustomisation(
    colour: colour ?? this.colour,
    background: background ?? this.background,
  );

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is ProfileCustomisation &&
          colour == other.colour &&
          background == other.background;

  @override
  int get hashCode => Object.hash(colour, background);
}

class ProfileCustomisationMapper extends SimpleMapper<ProfileCustomisation> {
  const new();

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

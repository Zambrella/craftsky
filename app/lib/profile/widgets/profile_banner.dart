import 'package:flutter/material.dart';

/// Flat coloured banner that sits behind the profile header. Future
/// iterations will paint cutout shapes (per the design-system mockups)
/// or render a user-supplied banner image; for now it's a solid swatch.
///
/// Height is fixed so the avatar overlap math in `ProfileHeaderHero`
/// stays predictable.
class ProfileBanner extends StatelessWidget {
  const ProfileBanner({
    required this.color,
    this.bannerUrl,
    this.height = ProfileBanner.defaultHeight,
    super.key,
  });

  static const double defaultHeight = 160;

  final Color color;
  final String? bannerUrl;
  final double height;

  @override
  Widget build(BuildContext context) {
    return Container(
      height: height,
      width: double.infinity,
      decoration: BoxDecoration(
        color: color,
        image: bannerUrl == null
            ? null
            : DecorationImage(
                image: NetworkImage(bannerUrl!),
                fit: BoxFit.cover,
              ),
      ),
    );
  }
}

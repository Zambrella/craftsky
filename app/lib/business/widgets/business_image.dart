import 'package:cached_network_image/cached_network_image.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:flutter/material.dart';

class BusinessImage extends StatelessWidget {
  const BusinessImage({
    required this.image,
    required this.networkUrl,
    this.fit,
    super.key,
  });

  final BusinessImageView image;
  final String networkUrl;
  final BoxFit? fit;

  @override
  Widget build(BuildContext context) {
    if (image.previewBytes case final bytes?) {
      return Image.memory(bytes, fit: fit);
    }
    return CachedNetworkImage(
      imageUrl: networkUrl,
      fit: fit,
      errorWidget: (_, _, _) => const Icon(Icons.image_not_supported_outlined),
    );
  }
}

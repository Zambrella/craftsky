enum SupportedImageFormat { jpeg, png, webp }

class ImagePreparationInput {
  const ImagePreparationInput({
    required this.format,
    required this.metadata,
    this.hasTransparency = false,
  });

  final SupportedImageFormat format;
  final Map<String, String> metadata;
  final bool hasTransparency;
}

class PreparedImage {
  const PreparedImage({
    required this.format,
    required this.metadata,
    required this.hasTransparency,
  });

  final SupportedImageFormat format;
  final Map<String, String> metadata;
  final bool hasTransparency;
}

const _metadataKeysToStrip = <String>{
  'gpsLatitude',
  'gpsLongitude',
  'cameraMake',
  'cameraModel',
  'captureTimestamp',
  'lens',
  'software',
  'userComment',
  'comment',
};

PreparedImage stripNonEssentialMetadata(ImagePreparationInput input) {
  final retained = <String, String>{
    for (final entry in input.metadata.entries)
      if (!_metadataKeysToStrip.contains(entry.key)) entry.key: entry.value,
  };

  return PreparedImage(
    format: input.format,
    metadata: retained,
    hasTransparency: input.format == SupportedImageFormat.png
        ? input.hasTransparency
        : false,
  );
}

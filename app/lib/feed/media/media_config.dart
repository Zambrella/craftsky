class MediaConfig {
  const MediaConfig({
    required this.maxImages,
    required this.maxImageBytes,
    required this.maxAltTextCharacters,
  });

  final int maxImages;
  final int maxImageBytes;
  final int maxAltTextCharacters;
}

const mediaConfig = MediaConfig(
  maxImages: 4,
  maxImageBytes: 15 * 1024 * 1024,
  maxAltTextCharacters: 300,
);

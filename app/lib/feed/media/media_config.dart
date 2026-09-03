const defaultMaxImageAspectRatio = 20;

class MediaConfig {
  const MediaConfig({
    required this.maxImages,
    required this.maxImageBytes,
    required this.maxAltTextCharacters,
    this.maxSourceImageBytes = 50000000,
    this.maxSourceImageSide = 8192,
    this.maxSourceImagePixels = 16000000,
    this.targetImageBytes = 1950000,
    this.maxImageWidth = 4000,
    this.maxImageHeight = 4000,
    this.maxImageAspectRatio = defaultMaxImageAspectRatio,
    this.minJpegQuality = 45,
    this.maxJpegQuality = 85,
    this.maxJpegQualityAttempts = 6,
    this.maxResizeRounds = 3,
  });

  final int maxImages;
  final int maxImageBytes;
  final int maxAltTextCharacters;
  final int maxSourceImageBytes;
  final int maxSourceImageSide;
  final int maxSourceImagePixels;
  final int targetImageBytes;
  final int maxImageWidth;
  final int maxImageHeight;
  final int maxImageAspectRatio;
  final int minJpegQuality;
  final int maxJpegQuality;
  final int maxJpegQualityAttempts;
  final int maxResizeRounds;
}

const mediaConfig = MediaConfig(
  maxImages: 4,
  maxImageBytes: 2000000,
  maxAltTextCharacters: 300,
);

import 'package:flutter_riverpod/flutter_riverpod.dart';

final videoUploadsEnabledProvider = Provider<bool>(
  (_) => const bool.fromEnvironment(
    'CRAFTSKY_ENABLE_VIDEO_UPLOADS',
  ),
);

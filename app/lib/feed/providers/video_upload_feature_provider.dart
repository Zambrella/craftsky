import 'package:flutter_riverpod/flutter_riverpod.dart';

const videoUploadsAvailableInThisRelease = false;

final videoUploadsEnabledProvider = Provider<bool>(
  (_) => videoUploadsAvailableInThisRelease,
);

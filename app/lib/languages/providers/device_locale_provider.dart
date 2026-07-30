import 'dart:ui';

import 'package:flutter_riverpod/flutter_riverpod.dart';

final deviceLocalesProvider = Provider<List<Locale>>(
  (ref) => List.unmodifiable(PlatformDispatcher.instance.locales),
);

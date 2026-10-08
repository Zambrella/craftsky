import 'package:craftsky_app/auth/services/handle_typeahead_service.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Override this provider to avoid public network calls in widget tests.
// The concrete auto-dispose provider type is inferred by Riverpod.
// ignore: specify_nonobvious_property_types
final handleTypeaheadServiceProvider =
    Provider.autoDispose<HandleTypeaheadService>((ref) {
      final service = HandleTypeaheadService();
      ref.onDispose(service.close);
      return service;
    });

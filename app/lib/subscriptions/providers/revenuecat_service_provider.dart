import 'package:craftsky_app/subscriptions/services/revenuecat_service.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final revenueCatServiceProvider = Provider<RevenueCatService>(
  (ref) => const UnavailableRevenueCatService(),
  name: 'revenueCatServiceProvider',
);

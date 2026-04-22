import 'package:craftsky_app/shared/api/craftsky_api_client.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'api_client_provider.g.dart';

@Riverpod(keepAlive: true)
CraftskyApiClient craftskyApiClient(Ref ref) =>
    CraftskyApiClient(ref.watch(dioProvider));

// Task 14c adds handoffApiClient(token) here — a family provider that
// constructs a short-lived HandoffApiClient with the Bearer token baked
// into BaseOptions.headers.

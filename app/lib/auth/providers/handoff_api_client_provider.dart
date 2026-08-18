import 'package:craftsky_app/auth/data/handoff_api_client.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';

part 'handoff_api_client_provider.g.dart';

/// Uses the anonymous client for code redemption. The pending bearer is passed
/// directly to the one confirmation request, never retained in provider
/// identity or diagnostics.
@Riverpod(keepAlive: true)
HandoffApiClient handoffApiClient(Ref ref) =>
    HandoffApiClient(ref.watch(anonymousDioProvider));

import 'package:craftsky_app/auth/models/account_key.dart';
import 'package:craftsky_app/languages/data/api_language_preferences_repository.dart';
import 'package:craftsky_app/languages/data/language_preferences_api_client.dart';
import 'package:craftsky_app/languages/data/language_preferences_repository.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

// The concrete family type is intentionally inferred by Riverpod.
// ignore: specify_nonobvious_property_types
final languagePreferencesRepositoryProvider = FutureProvider.autoDispose
    .family<LanguagePreferencesRepository, AccountKey>((ref, account) async {
      final dio = await ref.watch(accountDioProvider(account).future);
      return ApiLanguagePreferencesRepository(
        LanguagePreferencesApiClient(dio),
      );
    });

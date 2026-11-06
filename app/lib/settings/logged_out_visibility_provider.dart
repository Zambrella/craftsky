import 'package:craftsky_app/settings/logged_out_visibility_controller.dart';
import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/api/api_unwrap.dart';
import 'package:craftsky_app/shared/api/pds_mutation_contract.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:craftsky_app/shared/browsing/browsing_context.dart';
import 'package:craftsky_app/shared/browsing/browsing_provider.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

final Provider<LoggedOutVisibilityController>
loggedOutVisibilityControllerProvider =
    Provider.autoDispose<LoggedOutVisibilityController>((ref) {
      final browsing = ref.watch(browsingContextProvider);
      if (browsing is! MemberBrowsingContext) throw const ApiUnauthorized();
      const path = '/v1/preferences/logged-out-visibility';
      bool current() =>
          ref.mounted && ref.read(browsingContextProvider) == browsing;
      final controller = LoggedOutVisibilityController(
        isCurrent: current,
        read: () => unwrapApi(() async {
          final dio = await ref.read(
            accountDioProvider(browsing.lease.session.account).future,
          );
          if (!current()) throw const ApiCanceled();
          final response = await dio.get<Map<String, dynamic>>(path);
          return VisibilitySetting.fromMap(response.data!);
        }),
        save: (intent) => unwrapPdsMutationApi(() async {
          final dio = await ref.read(
            accountDioProvider(browsing.lease.session.account).future,
          );
          if (!current()) throw const ApiCanceled();
          final response = await dio.put<Object?>(
            path,
            data: intent.body,
            options: Options(headers: {'Idempotency-Key': intent.operationKey}),
          );
          return await parsePdsMutationResponse(
            response,
            accepted: (data) => VisibilitySetting.fromMap(
              Map<String, dynamic>.from(data! as Map),
            ),
          );
        }),
      );
      ref.onDispose(controller.dispose);
      return controller;
    });

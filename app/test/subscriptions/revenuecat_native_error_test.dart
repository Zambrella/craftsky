import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:craftsky_app/subscriptions/services/revenuecat_service_native.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  const service = NativeRevenueCatService();
  const purchases = MethodChannel('purchases_flutter');
  const ui = MethodChannel('purchases_ui_flutter');
  final messenger =
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;

  setUp(() {
    Future<Object?> fail(MethodCall call) async => throw PlatformException(
      code: 'sdk_failure',
      message: 'PRIVATE_PROVIDER_MESSAGE',
      details: {'payment': 'PRIVATE_PAYMENT_DETAILS'},
    );
    messenger
      ..setMockMethodCallHandler(purchases, fail)
      ..setMockMethodCallHandler(ui, fail);
  });
  tearDown(() {
    messenger
      ..setMockMethodCallHandler(purchases, null)
      ..setMockMethodCallHandler(ui, null);
  });

  for (final (code, expected) in [
    ('1', true),
    ('10', true),
    ('3', false),
    ('-1', false),
    ('sdk_failure', false),
  ]) {
    test('SDK cause code $code expected=$expected', () {
      final error = RevenueCatServiceException(
        'restore_failed',
        diagnosticCause: PlatformException(code: code),
      );
      expect(error.expectedFailure, expected);
    });
  }

  for (final operation in <String, Future<Object?> Function()>{
    'identity_read_failed': service.currentIdentity,
    'identity_write_failed': () => service.identify('PRIVATE_BILLING_ID'),
    'offering_read_failed': () => service.getOffering('plus'),
    'restore_failed': service.restorePurchases,
    'customer_center_failed': () => service.presentCustomerCenter((_) {}),
  }.entries) {
    test('${operation.key} preserves the SDK cause and caught stack', () async {
      Object? caught;
      try {
        await operation.value();
      } on Object catch (error) {
        caught = error;
      }
      expect(caught, isA<RevenueCatServiceException>());
      expect(caught, isA<DiagnosticFailureCause>());
      final diagnostic = caught! as DiagnosticFailureCause;
      expect(diagnostic.diagnosticCause, isA<PlatformException>());
      final original = diagnostic.diagnosticCause! as PlatformException;
      expect(original.code, 'sdk_failure');
      expect(original.message, 'PRIVATE_PROVIDER_MESSAGE');
      expect(original.details, {'payment': 'PRIVATE_PAYMENT_DETAILS'});
      expect(diagnostic.diagnosticStack, isNotNull);
      expect(diagnostic.diagnosticStack.toString(), contains('_invokeMethod'));
      expect(caught.toString(), isNot(contains('PRIVATE_')));
    });
  }
}

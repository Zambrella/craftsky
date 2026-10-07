import 'package:craftsky_app/shared/observability/diagnostic_text.dart';
import 'package:craftsky_app/shared/observability/error_reporter.dart';

final class SentrySanitizer {
  const SentrySanitizer._();

  static const _allowedContextKeys = {
    'appErrorKind',
    'failureStage',
    'itemCount',
    'suppressedCount',
    'providerStatus',
    'providerCode',
    'retryable',
    'attempt',
    'outcome',
    'byteBand',
    'severity',
    'feature',
    'operation',
    'classification',
    'appViewRequestId',
    'appViewError',
    'httpStatus',
    'authState',
    'platform',
    'environment',
    'release',
  };

  static const _allowedBreadcrumbCategories = {
    'navigation',
    'feature',
    'lifecycle',
    'ui.action',
    'log',
    'device.connectivity',
    'app.lifecycle',
  };

  static const _allowedBreadcrumbDataKeys = {
    'routeName',
    'feature',
    'lifecycleState',
    'action',
    'connectivity',
    'state',
    'logger',
  };

  static final RegExp _sensitivePattern = RegExp(
    r'(did:|https?://|bearer\s+|token=|[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,})',
    caseSensitive: false,
  );

  static Map<String, Object?> sanitizeContext(Map<String, Object?> context) {
    final selected = <String, Object?>{
      for (final entry in context.entries)
        if (_allowedContextKeys.contains(entry.key) &&
            _isSafeValue(entry.value))
          entry.key: entry.value,
    };
    final code = context['appViewError'];
    if (code is String && RegExp(r'^[a-z][a-z0-9_]{0,79}$').hasMatch(code)) {
      selected['appViewError'] = code;
    }
    final method = context['httpMethod'];
    if (method is String &&
        const {
          'GET',
          'POST',
          'PUT',
          'PATCH',
          'DELETE',
          'HEAD',
          'OPTIONS',
        }.contains(method)) {
      selected['httpMethod'] = method;
    }
    return selected;
  }

  static SafeBreadcrumb? sanitizeBreadcrumb(SafeBreadcrumb breadcrumb) {
    if (!_allowedBreadcrumbCategories.contains(breadcrumb.category)) {
      return null;
    }
    final data = <String, Object?>{
      for (final entry in breadcrumb.data.entries)
        if (_allowedBreadcrumbDataKeys.contains(entry.key) &&
            _isSafeValue(entry.value) &&
            (entry.value is! String ||
                RegExp(
                  r'^[A-Za-z0-9_.]{1,160}$',
                ).hasMatch(entry.value! as String)))
          entry.key: entry.value,
    };

    // Native observer names originate from GoRouter's static route patterns.
    // Concrete locations, query strings and route arguments are never selected.
    if (breadcrumb.category == 'navigation') {
      for (final key in ['from', 'to']) {
        final name = breadcrumb.data[key];
        if (name is String &&
            RegExp(
              r'^(?:/[A-Za-z0-9_:/.-]*|[A-Za-z0-9_.-]+)$',
            ).hasMatch(name) &&
            name.length <= 160 &&
            !name.contains('://')) {
          data[key] = name;
        }
      }
    }
    return SafeBreadcrumb(
      category: breadcrumb.category,
      message: switch (breadcrumb.category) {
        'device.connectivity' => 'Connectivity changed',
        'app.lifecycle' => 'Application lifecycle changed',
        'navigation' => 'Navigation',
        _ => boundDiagnosticText(breadcrumb.message, 256),
      },
      data: data,
    );
  }

  static bool _isSafeValue(Object? value) {
    return switch (value) {
      null => false,
      String() => _isSafeString(value),
      int() || double() || bool() => true,
      _ => false,
    };
  }

  static bool _isSafeString(String value) {
    if (value.isEmpty || value.length > 160) return false;
    return RegExp(r'^[A-Za-z0-9_.:-]{1,160}$').hasMatch(value) &&
        !_sensitivePattern.hasMatch(value);
  }
}

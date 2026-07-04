import 'package:craftsky_app/shared/observability/error_reporter.dart';

final class SentrySanitizer {
  const SentrySanitizer._();

  static const _allowedContextKeys = {
    'appErrorKind',
    'severity',
    'feature',
    'operation',
    'classification',
    'appViewRequestId',
    'appViewError',
    'httpStatus',
    'endpointCategory',
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
  };

  static const _allowedBreadcrumbDataKeys = {
    'routeName',
    'feature',
    'lifecycleState',
    'action',
  };

  static final RegExp _sensitivePattern = RegExp(
    r'(did:|https?://|bearer\s+|token=|[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,})',
    caseSensitive: false,
  );

  static Map<String, Object?> sanitizeContext(Map<String, Object?> context) {
    return {
      for (final entry in context.entries)
        if (_allowedContextKeys.contains(entry.key) &&
            _isSafeValue(entry.value))
          entry.key: entry.value,
    };
  }

  static SafeBreadcrumb? sanitizeBreadcrumb(SafeBreadcrumb breadcrumb) {
    if (!_allowedBreadcrumbCategories.contains(breadcrumb.category)) {
      return null;
    }
    if (!_isSafeString(breadcrumb.message)) return null;

    final data = <String, Object?>{
      for (final entry in breadcrumb.data.entries)
        if (_allowedBreadcrumbDataKeys.contains(entry.key) &&
            _isSafeValue(entry.value))
          entry.key: entry.value,
    };

    return SafeBreadcrumb(
      category: breadcrumb.category,
      message: breadcrumb.message,
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
    return !_sensitivePattern.hasMatch(value);
  }
}

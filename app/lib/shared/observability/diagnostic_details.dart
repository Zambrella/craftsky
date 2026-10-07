import 'package:craftsky_app/shared/api/api_exception.dart';
import 'package:craftsky_app/shared/observability/diagnostic_failure.dart';
import 'package:craftsky_app/shared/observability/diagnostic_text.dart';
import 'package:dio/dio.dart';
import 'package:flutter/services.dart';

Map<String, Object?> selectedCause(Object error) {
  final causes = selectedCauses(error);
  return {
    ...causes.first,
    if (causes.length > 1) 'causes': causes.skip(1).toList(),
  };
}

List<Map<String, Object?>> selectedCauses(Object error) {
  final causes = <Map<String, Object?>>[];
  final visited = Set<Object>.identity();
  Object? current = error;
  while (current != null && causes.length < 8 && visited.add(current)) {
    causes.add({
      'type': boundDiagnosticText(current.runtimeType.toString(), 256),
      'message': switch (current) {
        ApiUnauthorized() || ApiCanceled() => current.toString(),
        DioException(:final response)
            when response?.statusCode != null &&
                response!.statusCode! >= 100 &&
                response.statusCode! <= 599 =>
          'HTTP request failed (HTTP ${response.statusCode})',
        _ => 'Operation failed',
      },
    });
    if (current is PlatformException &&
        const {
          'read_error',
          'write_error',
          'storage_unavailable',
          'Missing Parameter',
          'Unexpected security result code',
          'Exception encountered',
        }.contains(current.code)) {
      causes.last['code'] = current.code;
      final status =
          current.code == 'Unexpected security result code' &&
              current.details is int &&
              (current.details as int).abs() <= 999999
          ? current.details
          : null;
      causes.last['message'] =
          'Platform operation failed (code ${current.code}'
          "${status == null ? '' : ', status $status'})";
      if (status != null) causes.last['status'] = status;
    }
    current = switch (current) {
      DiagnosticFailureCause(:final diagnosticCause) => diagnosticCause,
      ApiException(:final details) => details.cause,
      DioException(:final error) => error,
      _ => null,
    };
  }
  if (current != null) {
    if (causes.length == 8) causes.removeLast();
    causes.add({
      'type': 'DiagnosticOmission',
      'message': '[OMITTED: cyclic or excess causes]',
    });
  }
  return causes;
}

final _frame = RegExp(
  r'^#\d+\s+([A-Za-z0-9_.$<>]+)\s+\((.*?):(\d+)(?::\d+)?\)$',
);
List<Map<String, Object?>> selectedStack(StackTrace stack) {
  final frames = <Map<String, Object?>>[];
  const omission = <String, Object?>{'omitted': '[OMITTED: excess frames]'};
  for (final line in stack.toString().split('\n')) {
    final match = _frame.firstMatch(line.trim());
    if (match == null) continue;
    final frame = <String, Object?>{
      'function': boundDiagnosticText(match[1]!, 160),
      'file': boundDiagnosticText(
        match[2]!.replaceAll(r'\', '/').split('/').last,
        80,
      ),
      'line': int.tryParse(match[3]!),
    };
    if (frames.length >= 8) {
      frames.add(omission);
      break;
    }
    frames.add(frame);
  }
  return frames;
}

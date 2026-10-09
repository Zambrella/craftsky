import 'dart:async';

import 'package:intl/date_symbol_data_local.dart';
import 'package:intl/intl.dart';

var _initialized = false;

String formatServiceStatusEstimate(
  DateTime instant,
  String locale, {
  DateTime Function(DateTime)? localize,
}) {
  if (!_initialized) {
    // intl's bundled-data initializer registers symbols synchronously; its
    // completed Future does not gate app/account startup or maintenance text.
    unawaited(initializeDateFormatting());
    _initialized = true;
  }
  return DateFormat.yMMMd(locale).add_jm().format(
    localize == null ? instant.toLocal() : localize(instant),
  );
}

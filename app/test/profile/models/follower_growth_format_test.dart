import 'package:craftsky_app/profile/models/follower_growth.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:intl/date_symbol_data_local.dart';

void main() {
  setUpAll(initializeDateFormatting);

  test('formats counts and UTC dates with the requested locale', () {
    final date = DateTime.utc(2026, 8, 25);

    expect(formatFollowerCount(1234567, 'en_US'), '1,234,567');
    expect(formatFollowerCount(1234567, 'de_DE'), '1.234.567');
    expect(formatFollowerGrowthDate(date, 'en_US'), '8/25/2026');
    expect(formatFollowerGrowthDate(date, 'de_DE'), '25.8.2026');
    expect(date, DateTime.utc(2026, 8, 25));
    expect(date.isUtc, isTrue);
  });
}

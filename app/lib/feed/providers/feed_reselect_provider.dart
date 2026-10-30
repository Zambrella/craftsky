import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Signals an explicit tap on Home while the feed is already open.
final feedReselectProvider = NotifierProvider<FeedReselect, int>(
  FeedReselect.new,
);

class FeedReselect extends Notifier<int> {
  @override
  int build() => 0;

  void reselect() => state++;
}

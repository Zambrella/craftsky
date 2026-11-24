enum VideoThumbnailFrame {
  start(0),
  quarter(25),
  midpoint(50),
  threeQuarters(75);

  new(this.percent);
  final int percent;

  static VideoThumbnailFrame fromPercent(Object? value) => values.firstWhere(
    (frame) => frame.percent == value,
    orElse: () => midpoint,
  );

  int timestampMilliseconds(Duration duration) =>
      duration.inMilliseconds * percent ~/ 100;
}

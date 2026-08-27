import 'dart:math' as math;

import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/profile/models/follower_growth.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:fl_chart/fl_chart.dart';
import 'package:flutter/material.dart';
import 'package:intl/intl.dart' show DateFormat;

class FollowerGrowthChart extends StatelessWidget {
  const FollowerGrowthChart({
    required this.growth,
    required this.period,
    super.key,
  });

  final FollowerGrowth growth;
  final FollowerGrowthPeriod period;

  @override
  Widget build(BuildContext context) {
    final chartPoints = _chartPoints(growth.points);
    final observed = chartPoints.where((point) => point.count != null).toList();
    if (observed.isEmpty) return const SizedBox.shrink();

    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>() ?? const SpacingTheme();
    final locale = Localizations.localeOf(context).toString();
    final counts = observed.map((point) => point.count!).toList();
    final minimum = counts.reduce(math.min).toDouble();
    final maximum = counts.reduce(math.max).toDouble();
    final spread = maximum - minimum;
    final yInterval = math.max(1, (spread / 4).ceil()).toDouble();
    final minimumY = math.max(
      0,
      ((minimum - yInterval) / yInterval).floor() * yInterval,
    );
    final maximumY = ((maximum + yInterval) / yInterval).ceil() * yInterval;
    final missingCount = growth.points.length - observed.length;
    final label = [
      _periodLabel(l10n),
      l10n.growthLatestCount(
        formatFollowerCount(observed.last.count!, locale),
      ),
      _changeLabel(l10n, locale),
      l10n.growthChartRange(
        formatFollowerGrowthDate(growth.rangeStart, locale),
        formatFollowerGrowthDate(growth.rangeEnd, locale),
      ),
      l10n.growthFreshnessCopy,
      if (missingCount > 0) l10n.growthMissingDays(missingCount),
    ].join('. ');
    final lineColor = theme.colorScheme.primary;

    return Semantics(
      label: label,
      container: true,
      child: ExcludeSemantics(
        child: Directionality(
          textDirection: TextDirection.ltr,
          child: LayoutBuilder(
            builder: (context, constraints) {
              final labelStyle = theme.textTheme.labelSmall;
              final textScaler = MediaQuery.textScalerOf(context);
              double measure(String value) {
                final painter = TextPainter(
                  text: TextSpan(text: value, style: labelStyle),
                  textDirection: TextDirection.ltr,
                  textScaler: textScaler,
                )..layout();
                return painter.width;
              }

              final maximumReserved = constraints.maxWidth * 0.45;
              final useCompactLabels =
                  math.max(
                        measure(formatFollowerCount(minimum.round(), locale)),
                        measure(formatFollowerCount(maximum.round(), locale)),
                      ) +
                      spacing.sp4 >
                  maximumReserved;
              String formatAxisCount(double value) => useCompactLabels
                  ? formatCompactFollowerCount(value.round(), locale)
                  : formatFollowerCount(value.round(), locale);
              final leftReserved = math
                  .min(
                    maximumReserved,
                    math.max(
                      54,
                      math.max(
                            measure(formatAxisCount(minimum)),
                            measure(formatAxisCount(maximum)),
                          ) +
                          spacing.sp4,
                    ),
                  )
                  .toDouble();
              return SizedBox(
                height: 240,
                child: LineChart(
                  LineChartData(
                    minX: 0,
                    maxX: math.max(1, chartPoints.length - 1).toDouble(),
                    minY: minimumY.toDouble(),
                    maxY: maximumY,
                    lineTouchData: const LineTouchData(enabled: false),
                    borderData: FlBorderData(
                      show: true,
                      border: Border.all(
                        color: theme.colorScheme.outlineVariant,
                      ),
                    ),
                    gridData: FlGridData(
                      drawVerticalLine: false,
                      horizontalInterval: yInterval,
                      getDrawingHorizontalLine: (_) => FlLine(
                        color: theme.colorScheme.outlineVariant.withValues(
                          alpha: 0.5,
                        ),
                        strokeWidth: 1,
                      ),
                    ),
                    titlesData: FlTitlesData(
                      topTitles: const AxisTitles(),
                      rightTitles: const AxisTitles(),
                      leftTitles: AxisTitles(
                        sideTitles: SideTitles(
                          showTitles: true,
                          reservedSize: leftReserved,
                          interval: yInterval,
                          getTitlesWidget: (value, meta) => SideTitleWidget(
                            meta: meta,
                            fitInside: SideTitleFitInsideData.fromTitleMeta(
                              meta,
                            ),
                            child: SizedBox(
                              key: const ValueKey(
                                'follower-growth-y-axis-label',
                              ),
                              width: math.max(0, leftReserved - spacing.sp2),
                              child: FittedBox(
                                fit: BoxFit.scaleDown,
                                alignment: Alignment.centerRight,
                                child: Text(
                                  formatAxisCount(value),
                                  maxLines: 1,
                                  softWrap: false,
                                  style: theme.textTheme.labelSmall,
                                ),
                              ),
                            ),
                          ),
                        ),
                      ),
                      bottomTitles: AxisTitles(
                        sideTitles: SideTitles(
                          showTitles: true,
                          reservedSize: 42,
                          interval: 1,
                          getTitlesWidget: (value, meta) {
                            final index = value.round();
                            final middle = chartPoints.length ~/ 2;
                            if (index != 0 &&
                                index != middle &&
                                index != chartPoints.length - 1) {
                              return const SizedBox.shrink();
                            }
                            if (index < 0 || index >= chartPoints.length) {
                              return const SizedBox.shrink();
                            }
                            return SideTitleWidget(
                              meta: meta,
                              fitInside: SideTitleFitInsideData.fromTitleMeta(
                                meta,
                              ),
                              child: Text(
                                _formatAxisDate(
                                  chartPoints[index].date,
                                  locale,
                                ),
                                style: theme.textTheme.labelSmall,
                              ),
                            );
                          },
                        ),
                      ),
                    ),
                    lineBarsData: [
                      LineChartBarData(
                        spots: [
                          for (final (index, point) in chartPoints.indexed)
                            point.count == null
                                ? FlSpot.nullSpot
                                : FlSpot(
                                    index.toDouble(),
                                    point.count!.toDouble(),
                                  ),
                        ],
                        color: lineColor,
                        barWidth: 3,
                        isStrokeCapRound: true,
                        dotData: FlDotData(
                          getDotPainter: (spot, percent, bar, index) =>
                              FlDotCirclePainter(
                                radius: 4,
                                color: theme.colorScheme.surface,
                                strokeWidth: 2,
                                strokeColor: lineColor,
                              ),
                        ),
                      ),
                    ],
                  ),
                  duration: Duration.zero,
                ),
              );
            },
          ),
        ),
      ),
    );
  }

  List<FollowerGrowthPoint> _chartPoints(
    List<FollowerGrowthPoint> points,
  ) {
    if (period != FollowerGrowthPeriod.oneYear) return points;

    final monthly = <FollowerGrowthPoint>[];
    DateTime? month;
    int? latestCount;
    for (final point in points) {
      final pointMonth = DateTime.utc(point.date.year, point.date.month);
      if (month != null && pointMonth != month) {
        monthly.add(FollowerGrowthPoint(date: month, count: latestCount));
        latestCount = null;
      }
      month = pointMonth;
      if (point.count != null) latestCount = point.count;
    }
    if (month != null) {
      monthly.add(FollowerGrowthPoint(date: month, count: latestCount));
    }
    return monthly;
  }

  String _formatAxisDate(DateTime date, String locale) =>
      period == FollowerGrowthPeriod.oneYear
      ? DateFormat.yMMM(locale).format(date)
      : formatFollowerGrowthDate(date, locale);

  String _periodLabel(AppLocalizations l10n) => switch (period) {
    FollowerGrowthPeriod.sevenDays => l10n.growthPeriodSevenDays,
    FollowerGrowthPeriod.thirtyDays => l10n.growthPeriodThirtyDays,
    FollowerGrowthPeriod.oneYear => l10n.growthPeriodOneYear,
  };

  String _changeLabel(AppLocalizations l10n, String locale) =>
      switch (growth.netChange) {
        null => l10n.growthInsufficientHistory,
        final change when change > 0 => l10n.growthChangeUp(
          formatFollowerCount(change, locale),
        ),
        final change when change < 0 => l10n.growthChangeDown(
          formatFollowerCount(-change, locale),
        ),
        _ => l10n.growthNoChange,
      };
}

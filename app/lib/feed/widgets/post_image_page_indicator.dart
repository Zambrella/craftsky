import 'package:flutter/material.dart';
import 'package:smooth_page_indicator/smooth_page_indicator.dart';

class PostImagePageIndicator extends StatelessWidget {
  const PostImagePageIndicator({
    required this.controller,
    required this.count,
    this.indicatorKey,
    super.key,
  });

  final PageController controller;
  final int count;
  final Key? indicatorKey;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: DecoratedBox(
        key: indicatorKey,
        decoration: BoxDecoration(
          color: Colors.black.withValues(alpha: 0.58),
          borderRadius: BorderRadius.circular(999),
          border: Border.all(color: Colors.white.withValues(alpha: 0.18)),
        ),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
          child: SmoothPageIndicator(
            controller: controller,
            count: count,
            effect: const WormEffect(
              dotWidth: 7,
              dotHeight: 7,
              spacing: 5,
              radius: 4,
              dotColor: Color(0x66FFFFFF),
              activeDotColor: Colors.white,
            ),
          ),
        ),
      ),
    );
  }
}

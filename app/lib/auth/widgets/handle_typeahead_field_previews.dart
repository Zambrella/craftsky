import 'package:craftsky_app/auth/providers/handle_typeahead_provider.dart';
import 'package:craftsky_app/auth/services/handle_typeahead_service.dart';
import 'package:craftsky_app/auth/widgets/handle_typeahead_field.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';
import 'package:craftsky_app/shared/atproto/identifiers.dart';
import 'package:craftsky_app/theme/app_theme.dart';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/widget_previews.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

@Preview(name: 'Login handle suggestions', group: 'Auth', size: Size(390, 600))
Widget handleTypeaheadFieldPreview() => const _Preview();

class _PreviewSearch extends HandleTypeaheadService {
  @override
  Future<List<HandleSuggestion>> search(
    String query, {
    CancelToken? cancelToken,
  }) async => [
    HandleSuggestion(
      handle: Handle.parse('alice.bsky.social'),
      displayName: 'Alice — knitting & crochet',
    ),
    HandleSuggestion(
      handle: Handle.parse('alice.example.com'),
      displayName: 'Alice',
    ),
  ];
}

class _Preview extends StatefulWidget {
  const _Preview();

  @override
  State<_Preview> createState() => _PreviewState();
}

class _PreviewState extends State<_Preview> {
  final _controller = TextEditingController();
  final _search = _PreviewSearch();

  @override
  void dispose() {
    _controller.dispose();
    _search.close();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => ProviderScope(
    overrides: [handleTypeaheadServiceProvider.overrideWithValue(_search)],
    child: MaterialApp(
      theme: AppTheme.lightThemeData,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: Scaffold(
        body: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: HandleTypeaheadField(
            controller: _controller,
            onSubmitted: (_) {},
          ),
        ),
      ),
    ),
  );
}

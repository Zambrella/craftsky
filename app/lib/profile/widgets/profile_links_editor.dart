import 'dart:convert';

import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/data/profile_link_policy.dart';
import 'package:craftsky_app/profile/models/profile_link.dart';
import 'package:craftsky_app/profile/widgets/profile_links.dart';
import 'package:craftsky_app/shared/api/providers/dio_provider.dart';
import 'package:craftsky_app/shared/media/blob_api_client_provider.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:craftsky_app/theme/brand_text_field.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:flutter/semantics.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:material_ui/material_ui.dart';

class ProfileLinksEditor extends StatelessWidget {
  const new({
    required this.links,
    required this.onChanged,
    this.enabled = true,
    super.key,
  });
  final List<ProfileLink> links;
  final ValueChanged<List<ProfileLink>> onChanged;
  final bool enabled;

  Future<void> _edit(BuildContext context, ProfileLink? link) async {
    final result = await showDialog<ProfileLink>(
      context: context,
      builder: (_) => _LinkForm(
        link: link,
        others: links.where((other) => other != link).toList(),
        onRemove: link == null
            ? null
            : () => onChanged([...links]..remove(link)),
      ),
    );
    if (result == null || !context.mounted) return;
    final next = List<ProfileLink>.of(links);
    if (link == null) {
      next.add(result);
    } else {
      next[next.indexOf(link)] = result;
    }
    onChanged(next);
  }

  void _move(int from, int to) {
    if (from == to || to < 0 || to >= links.length) return;
    final next = List<ProfileLink>.of(links);
    next.insert(to, next.removeAt(from));
    onChanged(next);
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          l10n.profileLinksTitle,
          style: Theme.of(context).textTheme.titleMedium,
        ),
        const SizedBox(height: 4),
        Text(l10n.profileLinksHelper),
        const SizedBox(height: 8),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            for (var i = 0; i < links.length; i++)
              DragTarget<ProfileLink>(
                key: ValueKey('profile-link-editor-${links[i].recordKey}'),
                onWillAcceptWithDetails: (details) =>
                    enabled && details.data != links[i],
                onAcceptWithDetails: (details) =>
                    _move(links.indexOf(details.data), i),
                builder: (context, candidates, rejected) =>
                    LongPressDraggable<ProfileLink>(
                      data: links[i],
                      maxSimultaneousDrags: enabled ? 1 : 0,
                      feedback: Material(
                        child: ProfileLinkPill(link: links[i]),
                      ),
                      child: CallbackShortcuts(
                        bindings: enabled
                            ? {
                                const SingleActivator(
                                  LogicalKeyboardKey.arrowLeft,
                                  alt: true,
                                ): () =>
                                    _move(i, i - 1),
                                const SingleActivator(
                                  LogicalKeyboardKey.arrowRight,
                                  alt: true,
                                ): () =>
                                    _move(i, i + 1),
                              }
                            : const {},
                        child: Semantics(
                          customSemanticsActions: enabled
                              ? {
                                  if (i > 0)
                                    CustomSemanticsAction(
                                      label: l10n.profileLinkMoveEarlier,
                                    ): () =>
                                        _move(i, i - 1),
                                  if (i + 1 < links.length)
                                    CustomSemanticsAction(
                                      label: l10n.profileLinkMoveLater,
                                    ): () =>
                                        _move(i, i + 1),
                                }
                              : null,
                          child: ProfileLinkPill(
                            link: links[i],
                            onTap: enabled
                                ? () => _edit(context, links[i])
                                : null,
                          ),
                        ),
                      ),
                    ),
              ),
            if (links.length < maxProfileLinks)
              ActionChip(
                key: const Key('profile-link-add'),
                avatar: const Icon(Icons.add, size: 18),
                label: Text(l10n.profileLinkAdd),
                onPressed: enabled ? () => _edit(context, null) : null,
              ),
          ],
        ),
      ],
    );
  }
}

class _LinkForm extends ConsumerStatefulWidget {
  const new({required this.link, required this.others, this.onRemove});
  final ProfileLink? link;
  final List<ProfileLink> others;
  final VoidCallback? onRemove;
  @override
  ConsumerState<_LinkForm> createState() => _LinkFormState();
}

class _LinkFormState extends ConsumerState<_LinkForm> {
  late final _url = TextEditingController(text: widget.link?.url);
  late final _title = TextEditingController(text: widget.link?.title);
  bool _busy = false;
  bool _unreachable = false;
  String? _error;
  UploadedBlob? _icon;
  String? _previewIcon;

  @override
  void dispose() {
    _url.dispose();
    _title.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    final l10n = AppLocalizations.of(context);
    final normalized = normalizeProfileLink(_url.text.trim());
    if (normalized == null) {
      setState(() => _error = l10n.profileLinkInvalid);
      return;
    }
    if (widget.others.any(
      (link) => normalizeProfileLink(link.url) == normalized,
    )) {
      setState(() => _error = l10n.profileLinkDuplicate);
      return;
    }
    final owner = ActiveAccountOperation.fromWidget(
      ref,
      ref.read(sessionRegistryProvider).value?.activeLease,
    );
    final url = normalized.toString();
    final sameDestination = url == widget.link?.url;
    if (!_unreachable && !sameDestination) {
      setState(() {
        _busy = true;
        _error = null;
      });
      try {
        final response = await ref
            .read(dioProvider)
            .post<Map<String, dynamic>>(
              '/v1/profile-link-previews',
              data: {'url': url},
            );
        if (!mounted || !owner.isCurrent) return;
        if (response.data?['reachability'] == 'unreachable') {
          setState(() {
            _unreachable = true;
            _busy = false;
          });
          return;
        }
        final icon = response.data?['icon'];
        if (profileLinkSupportProvider(url) == null && icon is Map) {
          final bytes = base64Decode(icon['bytes'] as String);
          final uploaded = await ref
              .read(blobApiClientProvider)
              .uploadImage(bytes: bytes, mimeType: icon['mimeType'] as String);
          if (!mounted || !owner.isCurrent) return;
          _icon = uploaded.blob;
          _previewIcon = 'data:${uploaded.mime};base64,${base64Encode(bytes)}';
        }
      } on Object {
        // Icons and reachability checks are advisory, like Bluesky's editor.
      }
    }
    if (!mounted || !owner.isCurrent) return;
    final now = DateTime.now().toUtc();
    final title = _title.text.trim().isEmpty ? null : _title.text.trim();
    final unchanged = sameDestination && title == widget.link?.title;
    Navigator.of(context).pop(
      ProfileLink(
        url: url,
        title: title,
        uri: widget.link?.uri,
        cid: widget.link?.cid,
        rkey: widget.link?.rkey ?? (widget.link == null ? _newTid(now) : null),
        createdAt: unchanged
            ? widget.link?.createdAt ?? now.toIso8601String()
            : now.toIso8601String(),
        iconBlob: sameDestination ? widget.link?.iconBlob : _icon,
        icon: sameDestination ? widget.link?.icon : _previewIcon,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    return PopScope(
      canPop: !_busy,
      child: CraftskyDialog(
        title: widget.link == null ? l10n.profileLinkAdd : l10n.profileLinkEdit,
        body: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            BrandTextField(
              label: l10n.profileLinkUrl,
              controller: _url,
              keyboardType: TextInputType.url,
              autocorrect: false,
              enabled: !_busy,
              errorText: _error,
              onChanged: (_) => setState(() {
                _unreachable = false;
                _error = null;
              }),
            ),
            const SizedBox(height: 16),
            BrandTextField(
              label: l10n.profileLinkTitle,
              controller: _title,
              maxLength: maxProfileLinkTitle,
              maxBytes: 320,
              enabled: !_busy,
            ),
            if (widget.onRemove != null) ...[
              const SizedBox(height: 16),
              TextButton.icon(
                onPressed: _busy
                    ? null
                    : () {
                        widget.onRemove!();
                        Navigator.of(context).pop();
                      },
                icon: const Icon(Icons.delete_outline),
                label: Text(l10n.profileLinkRemove),
              ),
            ],
            if (_unreachable) ...[
              const SizedBox(height: 16),
              Text(l10n.profileLinkUnreachable),
            ],
          ],
        ),
        actions: [
          TextButton(
            onPressed: _busy ? null : () => Navigator.of(context).pop(),
            child: Text(l10n.actionCancel),
          ),
          TextButton(
            key: const Key('profile-link-save'),
            onPressed: _busy ? null : _save,
            child: Text(
              _busy
                  ? l10n.profileLinkChecking
                  : _unreachable
                  ? l10n.profileLinkSaveAnyway
                  : l10n.editProfileSave,
            ),
          ),
        ],
      ),
    );
  }
}

String _newTid(DateTime now) {
  const alphabet = '234567abcdefghijklmnopqrstuvwxyz';
  var value = BigInt.from(now.microsecondsSinceEpoch) << 10;
  final buffer = List<String>.filled(13, '2');
  for (var i = 12; i >= 0; i--) {
    buffer[i] = alphabet[(value & BigInt.from(31)).toInt()];
    value >>= 5;
  }
  return buffer.join();
}

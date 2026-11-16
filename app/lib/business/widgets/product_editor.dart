import 'dart:async';

import 'package:craftsky_app/auth/models/account_session_lease.dart';
import 'package:craftsky_app/auth/providers/account_operation_guard.dart';
import 'package:craftsky_app/auth/providers/session_registry_provider.dart';
import 'package:craftsky_app/business/models/business_drafts.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/feed/models/create_post_image.dart';
import 'package:craftsky_app/feed/providers/composer_image_state.dart';
import 'package:craftsky_app/feed/widgets/composer_image_attachment_section.dart';
import 'package:craftsky_app/l10n/app_localizations.dart';
import 'package:craftsky_app/profile/providers/profile_image_picker_provider.dart';
import 'package:craftsky_app/router/responsive_modal_navigation.dart';
import 'package:craftsky_app/shared/image/basic_image_editor.dart';
import 'package:craftsky_app/shared/media/blob_api_client_provider.dart';
import 'package:craftsky_app/shared/media/image_pick_error_reporting.dart';
import 'package:craftsky_app/shared/media/uploaded_image_blob.dart';
import 'package:craftsky_app/shared/observability/error_reporter_provider.dart';
import 'package:craftsky_app/shared/text/record_text_limits.dart';
import 'package:craftsky_app/shared/widgets/composer_bottom_action.dart';
import 'package:craftsky_app/theme/craftsky_dialog.dart';
import 'package:craftsky_app/theme/craftsky_select_inputs.dart';
import 'package:craftsky_app/theme/craftsky_text_inputs.dart';
import 'package:craftsky_app/theme/theme_extensions.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';
import 'package:l10n_currencies/l10n_currencies.dart';
import 'package:material_ui/material_ui.dart';
import 'package:sealed_countries/sealed_countries.dart';
import 'package:uuid/uuid.dart';

typedef ProductImagePicker = Future<PreparedProfileImage?> Function(
  ImageSource source,
  void Function(Uint8List bytes) onPreviewReady,
);

typedef ProductImagesPicker = Future<List<PreparedProfileImage>> Function(
  int limit,
);

Future<ProductDraft?> showProductEditorSheet(
  BuildContext context, {
  ProductDraft? initial,
  ProductImagePicker? pickImage,
  Future<bool> Function(ProductDraft product)? persist,
  bool Function(String destination)? destinationExists,
}) {
  return responsiveModalNavigator(context).push<ProductDraft>(
    FullscreenModalRoute<ProductDraft>(
      builder: (routeContext) => ProductEditor(
        initial: initial,
        pickImage: pickImage,
        persist: persist,
        destinationExists: destinationExists,
        onCancel: () => Navigator.of(routeContext).pop(),
        onSave: (product) => Navigator.of(routeContext).pop(product),
      ),
    ),
  );
}

class ProductEditor extends ConsumerStatefulWidget {
  const new({
    required this.onSave,
    this.initial,
    this.pickImage,
    this.pickImages,
    this.editImage,
    this.uploadImage,
    this.onCancel,
    this.locale,
    this.persist,
    this.destinationExists,
    super.key,
  });

  final ProductDraft? initial;
  final ValueChanged<ProductDraft> onSave;
  final ProductImagePicker? pickImage;
  final ProductImagesPicker? pickImages;
  final BasicImageEditor? editImage;
  final Future<UploadedImageBlob> Function(PreparedProfileImage image)?
  uploadImage;
  final VoidCallback? onCancel;
  final Locale? locale;
  final Future<bool> Function(ProductDraft product)? persist;
  final bool Function(String destination)? destinationExists;

  @override
  ConsumerState<ProductEditor> createState() => _ProductEditorState();
}

class _ProductEditorState extends ConsumerState<ProductEditor> {
  final _formKey = GlobalKey<FormState>();
  final _photos = <_ProductPhoto>[];
  int _photoSequence = 0;
  late final TextEditingController _title;
  late final TextEditingController _destination;
  late final TextEditingController _amount;
  late String _currency;
  late List<CraftskySelectOption<String>> _currencyOptions;
  bool _preparingImage = false;
  bool _showImageError = false;
  bool _showPriceError = false;
  String? _uploadError;
  late final ActiveAccountLease? _owner;
  bool _dirty = false;
  bool _currencyInitialized = false;
  bool _saving = false;
  String? _saveError;

  @override
  void initState() {
    super.initState();
    _owner = ref.read(sessionRegistryProvider).value?.activeLease;
    final initial = widget.initial;
    _title = TextEditingController(text: initial?.title);
    _destination = TextEditingController(text: initial?.destination);
    _amount = TextEditingController(
      text: initial == null
          ? null
          : _editableAmount(initial.amount, initial.currency),
    );
    _currency = initial?.currency ?? '';
    for (final image in initial?.photos ?? <BusinessImageDraft>[]) {
      if (!image.hasImage) continue;
      _photos.add(
        _ProductPhoto(id: _newPhotoId(), image: image, alt: image.alt),
      );
    }
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (_currencyInitialized) return;
    _currencyInitialized = true;
    final locale = widget.locale ?? View.of(context).platformDispatcher.locale;
    final localeName = locale.toString();
    final localized = CurrenciesLocaleMapper().localize(
      businessProductCurrencies,
      mainLocale: localeName,
      fallbackLocale: 'en',
    );
    final names = <String, String>{};
    for (final entry in localized.entries) {
      names.putIfAbsent(entry.key.isoCode, () => entry.value);
    }
    _currencyOptions =
        businessProductCurrencies
            .map(
              (code) => CraftskySelectOption(
                value: code,
                label: '$code - ${names[code] ?? code}',
              ),
            )
            .toList()
          ..sort((a, b) => a.label.compareTo(b.label));
    if (widget.initial == null) {
      final countryCode = locale.countryCode;
      _currency = countryCode == null
          ? ''
          : WorldCountry.maybeFromCodeShort(
                  countryCode,
                )?.currencies?.firstOrNull?.code ??
                '';
    }
  }

  @override
  void dispose() {
    _title.dispose();
    _destination.dispose();
    _amount.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context);
    final theme = Theme.of(context);
    final spacing = theme.extension<SpacingTheme>()!;
    final swatches = theme.extension<BrandSwatchTheme>()!;
    final keyboardVisible = MediaQuery.viewInsetsOf(context).bottom > 0;
    return PopScope<Object?>(
      canPop: !_saving && !_dirty,
      onPopInvokedWithResult: (didPop, _) async {
        if (_saving) return;
        if (!didPop) await _cancel();
      },
      child: Scaffold(
        backgroundColor: swatches.paper,
        appBar: AppBar(
          leading: CloseButton(onPressed: _saving ? null : _cancel),
          title: Text(
            widget.initial == null
                ? l10n.businessProductEditorAddTitle
                : l10n.businessProductEditorEditTitle,
          ),
        ),
        body: Form(
          key: _formKey,
          onChanged: _markDirty,
          child: Stack(
            children: [
              SingleChildScrollView(
                padding: EdgeInsets.fromLTRB(
                  spacing.sp4,
                  spacing.sp4,
                  spacing.sp4,
                  0,
                ),
                child: Center(
                  child: ConstrainedBox(
                    constraints: const BoxConstraints(maxWidth: 720),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        CraftskyTextFormField(
                          textFieldKey: const ValueKey('product-title'),
                          controller: _title,
                          enabled: !_saving,
                          label: l10n.businessProductTitleLabel,
                          required: true,
                          maxLength: businessProductTitleLimit,
                          maxBytes: RecordTextLimits.productTitle.bytes,
                          textCapitalization: TextCapitalization.words,
                          validator: (value) {
                            if (value == null || value.trim().isEmpty) {
                              return l10n.businessProductTitleRequired;
                            }
                            return null;
                          },
                        ),
                        SizedBox(height: spacing.sp5),
                        CraftskyTextFormField(
                          textFieldKey: const ValueKey('product-destination'),
                          controller: _destination,
                          enabled: !_saving,
                          label: l10n.businessProductDestinationLabel,
                          hintText: l10n.businessProductDestinationHint,
                          required: true,
                          keyboardType: TextInputType.url,
                          validator: (value) {
                            final uri = Uri.tryParse(value ?? '');
                            if (uri == null ||
                                uri.scheme != 'https' ||
                                uri.host.isEmpty ||
                                uri.userInfo.isNotEmpty) {
                              return l10n.businessProductDestinationInvalid;
                            }
                            if (widget.destinationExists?.call(value!) ??
                                false) {
                              return l10n.businessProductDestinationDuplicate;
                            }
                            return null;
                          },
                        ),
                        SizedBox(height: spacing.sp5),
                        ComposerImageAttachmentSection(
                          imagesState: ComposerImagesState(
                            images: _photos.map(_attachmentDraft).toList(),
                          ),
                          enabled: !_preparingImage && !_saving,
                          required: true,
                          maxImages: businessProductImageLimit,
                          useDetailsRoute: true,
                          keyPrefix: 'product',
                          imageUrlFor: (image) => _photos
                              .where((photo) => photo.id == image.id)
                              .firstOrNull
                              ?.image
                              .previewUrl,
                          onAddImages: () => _pickImage(ImageSource.gallery),
                          onTakePhoto: () => _pickImage(ImageSource.camera),
                          onAltTextChanged: (id, value) {
                            setState(() => _photo(id).alt = value);
                            _markDirty();
                          },
                          onRemove: _removeImage,
                          onReplace: (id, source) =>
                              _pickImage(source, replacing: id),
                          onReplaceUnavailable: (id, source) =>
                              _pickImage(source, replacing: id),
                          onEditImage: _editImage,
                          onReorder: (oldIndex, newIndex) {
                            setState(
                              () => _photos.insert(
                                newIndex,
                                _photos.removeAt(oldIndex),
                              ),
                            );
                            _markDirty();
                          },
                          validationErrorText:
                              _uploadError ??
                              (_showImageError && _photos.isEmpty
                                  ? l10n.businessProductImageRequired
                                  : null),
                        ),
                        SizedBox(height: spacing.sp5),
                        LayoutBuilder(
                          builder: (context, constraints) => _priceFields(
                            l10n,
                            spacing,
                            sideBySide: constraints.maxWidth >= 520,
                          ),
                        ),
                        if (_showPriceError) ...[
                          SizedBox(height: spacing.sp2),
                          Text(
                            l10n.businessProductPriceInvalid,
                            style: TextStyle(color: theme.colorScheme.error),
                          ),
                        ],
                        if (_saveError != null) ...[
                          SizedBox(height: spacing.sp2),
                          Text(
                            _saveError!,
                            style: TextStyle(color: theme.colorScheme.error),
                          ),
                        ],
                        SizedBox(
                          key: const Key('product-editor-bottom-safe-space'),
                          height:
                              spacing.sp9 +
                              MediaQuery.paddingOf(context).bottom,
                        ),
                      ],
                    ),
                  ),
                ),
              ),
              PositionedDirectional(
                start: spacing.sp4,
                end: spacing.sp4,
                bottom: 0,
                child: SafeArea(
                  top: false,
                  child: Padding(
                    padding: EdgeInsets.only(bottom: spacing.sp4),
                    child: ComposerBottomAction(
                      keyboardVisible: keyboardVisible,
                      doneKey: const Key('product-composer-done'),
                      actionKey: const ValueKey('product-submit'),
                      label: l10n.businessProductSave,
                      onPressed: _preparingImage || _saving ? null : _save,
                      style: ButtonStyle(
                        minimumSize: WidgetStatePropertyAll(
                          Size.fromHeight(spacing.sp7),
                        ),
                      ),
                      child: _saving
                          ? SizedBox.square(
                              dimension: spacing.sp5,
                              child: CircularProgressIndicator(
                                strokeWidth: 2,
                                semanticsLabel: l10n.businessSaving,
                              ),
                            )
                          : null,
                    ),
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _priceFields(
    AppLocalizations l10n,
    SpacingTheme spacing, {
    required bool sideBySide,
  }) {
    final amount = CraftskyTextInput(
      label: l10n.businessProductAmountLabel,
      textFieldKey: const ValueKey('product-amount'),
      controller: _amount,
      enabled: !_saving,
      keyboardType: const TextInputType.numberWithOptions(decimal: true),
      inputFormatters: [_productAmountFormatter],
    );
    final currency = CraftskySingleSelectInput<String>(
      label: l10n.businessProductCurrencyLabel,
      enabled: !_saving,
      value: _currency.isEmpty ? null : _currency,
      options: _currencyOptions,
      keyPrefix: 'product-currency',
      searchThreshold: 0,
      onChanged: (value) {
        setState(() => _currency = value ?? '');
        _markDirty();
      },
    );
    if (!sideBySide) {
      return Column(
        children: [
          amount,
          SizedBox(height: spacing.sp5),
          currency,
        ],
      );
    }
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Expanded(child: amount),
        SizedBox(width: spacing.sp4),
        Expanded(child: currency),
      ],
    );
  }

  Future<void> _cancel() async {
    if (_saving) return;
    if (_dirty) {
      final l10n = AppLocalizations.of(context);
      final discard = await showCraftskyConfirmDialog(
        context,
        title: l10n.editProfileDiscardTitle,
        message: l10n.editProfileDiscardMessage,
        confirmLabel: l10n.editProfileDiscardConfirm,
        cancelLabel: l10n.editProfileDiscardCancel,
      );
      if (!discard || !mounted) return;
      _dirty = false;
    }
    final onCancel = widget.onCancel;
    if (onCancel != null) {
      onCancel();
    } else if (Navigator.of(context).canPop()) {
      Navigator.of(context).pop();
    }
  }

  String _newPhotoId() =>
      _photoSequence++ == 0 ? 'image' : 'image-${_photoSequence - 1}';
  _ProductPhoto _photo(String id) =>
      _photos.firstWhere((photo) => photo.id == id);

  Future<void> _pickImage(ImageSource source, {String? replacing}) async {
    if (_saving ||
        _preparingImage ||
        (replacing == null && _photos.length >= businessProductImageLimit)) {
      return;
    }
    FocusManager.instance.primaryFocus?.unfocus();
    await SystemChannels.textInput.invokeMethod<void>('TextInput.hide');
    if (!mounted || !_isCurrent(_owner)) return;
    final l10n = AppLocalizations.of(context);
    final reporter = ref.read(errorReporterProvider);
    final picker =
        widget.pickImage ??
        (source, onPreviewReady) => ref
            .read(profileImagePickerProvider)
            .pickAndPrepare(
              source: source,
              onPreviewReady: onPreviewReady,
              isCurrent: () => _isCurrent(_owner),
            );
    setState(() {
      _preparingImage = true;
      _uploadError = null;
    });
    try {
      final limit = replacing == null
          ? businessProductImageLimit - _photos.length
          : 1;
      final List<PreparedProfileImage> results;
      if (source == ImageSource.gallery &&
          replacing == null &&
          widget.pickImage == null) {
        results =
            await (widget.pickImages?.call(limit) ??
                ref
                    .read(profileImagePickerProvider)
                    .pickGalleryAndPrepare(
                      limit: limit,
                      isCurrent: () => _isCurrent(_owner),
                    ));
      } else {
        final result = await picker(source, (_) {});
        results = [?result];
      }
      if (!_isCurrent(_owner) || results.isEmpty) return;
      if (results.length > limit) {
        throw ProfileImageSelectionLimitException(limit);
      }
      setState(() {
        for (final result in results) {
          final photo =
              (replacing == null
                    ? _ProductPhoto(
                        id: _newPhotoId(),
                        image: const MissingBusinessImageDraft(),
                        alt: '',
                      )
                    : _photo(replacing))
                ..pending = result
                ..preview = result.bytes;
          if (replacing == null) _photos.add(photo);
        }
        _showImageError = false;
        _dirty = true;
      });
    } on Object catch (error, stackTrace) {
      unawaited(
        reportImagePickFailure(
          reporter: reporter,
          error: error,
          stackTrace: stackTrace,
          operation: ImagePickOperation.businessProduct,
          accountDid: _owner?.session.account.did.value,
        ),
      );
      if (_isCurrent(_owner)) {
        setState(
          () => _uploadError = imagePickFailureMessage(
            error,
            l10n,
            fallback: l10n.businessProductsUploadError,
          ),
        );
      }
    } finally {
      if (mounted) setState(() => _preparingImage = false);
    }
  }

  Future<void> _editImage(String id) async {
    final photo = _photo(id);
    final image = photo.pending;
    if (image == null || _preparingImage || _saving) return;
    FocusManager.instance.primaryFocus?.unfocus();
    await SystemChannels.textInput.invokeMethod<void>('TextInput.hide');
    if (!mounted || !_isCurrent(_owner)) return;
    final ownership = _owner;
    final reporter = ref.read(errorReporterProvider);
    final media = ref.read(profileImageMediaServiceProvider);
    setState(() {
      _preparingImage = true;
      _uploadError = null;
    });
    bool current() =>
        mounted && _isCurrent(ownership) && identical(photo.pending, image);
    try {
      final request = BasicImageEditRequest(
        bytes: image.bytes,
        mimeType: image.mimeType,
        mode: BasicImageEditMode.composer,
      );
      final output =
          await (widget.editImage?.call(request) ??
              showBasicImageEditor(context, request, ownership: ownership));
      if (output == null || !current()) return;
      final prepared = await media
          .prepareImage(
            bytes: output.bytes,
            fileName: output.mimeType == 'image/png'
                ? 'edited.png'
                : 'edited.jpg',
            mimeType: output.mimeType,
          )
          .future;
      if (!current()) return;
      final check = media.validatePreparedUploadBytes(
        originalBytes: output.bytes.length,
        preparedBytes: prepared.bytes.length,
        width: prepared.width,
        height: prepared.height,
      );
      if (!check.canUpload) {
        throw const ProfileImagePickException(expectedRejection: true);
      }
      setState(() {
        photo
          ..pending = PreparedProfileImage(
            bytes: prepared.bytes,
            mimeType: prepared.mimeType,
            width: prepared.width,
            height: prepared.height,
          )
          ..preview = prepared.bytes;
        _markDirty();
      });
    } on Object catch (error, stackTrace) {
      unawaited(
        reportImagePickFailure(
          reporter: reporter,
          error: error,
          stackTrace: stackTrace,
          operation: ImagePickOperation.businessProduct,
          accountDid: ownership?.session.account.did.value,
        ),
      );
      if (current()) {
        setState(
          () => _uploadError = AppLocalizations.of(context).imageEditFailed,
        );
      }
    } finally {
      if (mounted) setState(() => _preparingImage = false);
    }
  }

  void _removeImage(String id) {
    setState(() {
      _photos.removeWhere((photo) => photo.id == id);
      _uploadError = null;
      _dirty = true;
    });
  }

  bool _isCurrent(ActiveAccountLease? ownership) =>
      ActiveAccountOperation.fromWidget(ref, ownership).isCurrent;

  BusinessImageDraft _draftImage(_ProductPhoto photo) {
    final pending = photo.pending;
    if (pending != null) {
      return UploadedBusinessImageDraft(
        cid: '',
        mime: pending.mimeType,
        size: pending.bytes.length,
        alt: photo.alt,
        aspectRatio: BusinessImageAspectRatio(
          width: pending.width,
          height: pending.height,
        ),
      );
    }
    return switch (photo.image) {
      final ExistingBusinessImageDraft image => image.withAlt(photo.alt),
      final UploadedBusinessImageDraft image => image.withAlt(photo.alt),
      _ => photo.image,
    };
  }

  Future<void> _save() async {
    if (!_isCurrent(_owner) || _saving || _preparingImage) return;
    final l10n = AppLocalizations.of(context);
    final amount = _canonicalAmount(_amount.text);
    final id = widget.initial?.id ?? const Uuid().v4();
    final title = _title.text;
    final destination = _destination.text;
    final currency = amount.isEmpty ? '' : _currency;
    final reporter = ref.read(errorReporterProvider);
    ProductDraft product() => ProductDraft(
      id: id,
      title: title,
      destination: destination,
      images: _photos.map(_draftImage).toList(),
      amount: amount,
      currency: currency,
    );
    final errors = product().validate();
    final formValid = _formKey.currentState?.validate() ?? false;
    setState(() {
      _showImageError = errors.contains(ProductDraftError.imageRequired);
      _showPriceError =
          errors.contains(ProductDraftError.priceIncomplete) ||
          errors.contains(ProductDraftError.priceInvalid);
      _saveError = null;
    });
    if (!formValid || errors.isNotEmpty) return;
    setState(() => _saving = true);
    try {
      for (final photo in _photos) {
        final pending = photo.pending;
        if (pending == null) continue;
        try {
          final uploaded =
              await (widget.uploadImage?.call(pending) ??
                  ref
                      .read(blobApiClientProvider)
                      .uploadImage(
                        bytes: pending.bytes,
                        mimeType: pending.mimeType,
                      ));
          if (!_isCurrent(_owner)) return;
          setState(() {
            photo
              ..image = UploadedBusinessImageDraft.fromUpload(
                uploaded,
                alt: photo.alt,
                previewBytes: pending.bytes,
                aspectRatio: BusinessImageAspectRatio(
                  width: pending.width,
                  height: pending.height,
                ),
              )
              ..pending = null
              ..preview = pending.bytes;
            _uploadError = null;
          });
        } on Object catch (error, stackTrace) {
          unawaited(
            reportImagePickFailure(
              reporter: reporter,
              error: error,
              stackTrace: stackTrace,
              operation: ImagePickOperation.businessProduct,
              accountDid: _owner?.session.account.did.value,
            ),
          );
          if (_isCurrent(_owner)) {
            setState(
              () => _uploadError = imagePickFailureMessage(
                error,
                l10n,
                fallback: l10n.businessProductsUploadError,
              ),
            );
          }
          return;
        }
      }
      final saved = product();
      final success = await (widget.persist?.call(saved) ?? Future.value(true));
      if (!_isCurrent(_owner)) return;
      if (!success) {
        setState(() => _saveError = l10n.businessProductsSaveError);
        return;
      }
      _dirty = false;
      widget.onSave(saved);
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  ComposerImageDraft _attachmentDraft(_ProductPhoto photo) {
    final image = photo.image;
    final pending = photo.pending;
    final ratio = pending != null
        ? CreatePostImageAspectRatio(
            width: pending.width,
            height: pending.height,
          )
        : switch (image) {
            ExistingBusinessImageDraft(:final aspectRatio) ||
            UploadedBusinessImageDraft(
              :final aspectRatio,
            ) when aspectRatio != null => CreatePostImageAspectRatio(
              width: aspectRatio.width,
              height: aspectRatio.height,
            ),
            _ => null,
          };
    final phase = pending != null
        ? ImageReady(
            bytes: pending.bytes,
            mimeType: pending.mimeType,
            width: pending.width,
            height: pending.height,
            sha256: '',
          )
        : ImageUploaded(
            UploadedDraftImage(
              cid: switch (image) {
                ExistingBusinessImageDraft(:final cid) ||
                UploadedBusinessImageDraft(:final cid) => cid,
                _ => '',
              },
              mime: switch (image) {
                ExistingBusinessImageDraft(:final mime) ||
                UploadedBusinessImageDraft(:final mime) => mime,
                _ => 'image/jpeg',
              },
              size: switch (image) {
                ExistingBusinessImageDraft(:final size) ||
                UploadedBusinessImageDraft(:final size) => size,
                _ => 0,
              },
              aspectRatio: ratio,
            ),
          );
    return ComposerImageDraft(
      id: photo.id,
      fileName: 'product-${photo.id}',
      mimeType: pending?.mimeType ?? 'image/jpeg',
      altText: photo.alt,
      phase: phase,
      previewBytes: photo.preview ?? image.previewBytes,
      previewAspectRatio: ratio,
    );
  }

  void _markDirty() {
    if (!_dirty) setState(() => _dirty = true);
  }
}

class _ProductPhoto {
  new({required this.id, required this.image, required this.alt});
  final String id;
  BusinessImageDraft image;
  String alt;
  PreparedProfileImage? pending;
  Uint8List? preview;
}

final _productAmountFormatter = TextInputFormatter.withFunction((old, next) {
  return RegExp(
        r'^(?:|0|[1-9][0-9]{0,11})(?:\.[0-9]{0,4})?$',
      ).hasMatch(next.text)
      ? next
      : old;
});

String _canonicalAmount(String value) {
  if (!value.contains('.')) return value;
  final canonical = value.replaceFirst(RegExp(r'0+$'), '');
  return canonical.endsWith('.')
      ? canonical.substring(0, canonical.length - 1)
      : canonical;
}

String _editableAmount(String value, String currency) {
  if (value.isEmpty || !businessProductCurrencies.contains(currency)) {
    return value;
  }
  final scale = businessCurrencyScale(currency);
  if (scale == 0) return value;
  final parts = value.split('.');
  if (parts.length > 2) return value;
  final fraction = parts.length == 1 ? '' : parts[1];
  if (fraction.length > scale) return value;
  return '${parts[0]}.${fraction.padRight(scale, '0')}';
}

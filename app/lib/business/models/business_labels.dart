import 'package:craftsky_app/business/models/business_formatters.dart';
import 'package:craftsky_app/business/models/business_profile.dart';
import 'package:craftsky_app/l10n/generated/app_localizations.dart';

abstract final class BusinessLabels {
  static const businessTypes = <BusinessOpenValue>[
    BusinessOpenValue(value: 'dyer', known: true),
    BusinessOpenValue(value: 'fiber-producer', known: true),
    BusinessOpenValue(value: 'fiber-processor', known: true),
    BusinessOpenValue(value: 'yarn-shop', known: true),
    BusinessOpenValue(value: 'fabric-shop', known: true),
    BusinessOpenValue(value: 'craft-supply-shop', known: true),
    BusinessOpenValue(value: 'pattern-designer', known: true),
    BusinessOpenValue(value: 'finished-goods-maker', known: true),
    BusinessOpenValue(value: 'tool-maker', known: true),
    BusinessOpenValue(value: 'teacher', known: true),
    BusinessOpenValue(value: 'craft-studio', known: true),
    BusinessOpenValue(value: 'repair-service', known: true),
    BusinessOpenValue(value: 'technical-editor', known: true),
    BusinessOpenValue(value: 'photographer', known: true),
    BusinessOpenValue(value: 'publisher', known: true),
    BusinessOpenValue(value: 'other-craft-business', known: true),
  ];

  static const offerings = <BusinessOpenValue>[
    BusinessOpenValue(value: 'yarn', known: true),
    BusinessOpenValue(value: 'fiber', known: true),
    BusinessOpenValue(value: 'fabric', known: true),
    BusinessOpenValue(value: 'patterns', known: true),
    BusinessOpenValue(value: 'kits', known: true),
    BusinessOpenValue(value: 'notions', known: true),
    BusinessOpenValue(value: 'tools', known: true),
    BusinessOpenValue(value: 'finished-goods', known: true),
    BusinessOpenValue(value: 'custom-work', known: true),
    BusinessOpenValue(value: 'repairs', known: true),
    BusinessOpenValue(value: 'classes', known: true),
    BusinessOpenValue(value: 'studio-hire', known: true),
    BusinessOpenValue(value: 'wholesale', known: true),
    BusinessOpenValue(value: 'digital-products', known: true),
    BusinessOpenValue(value: 'technical-editing', known: true),
    BusinessOpenValue(value: 'photography-services', known: true),
    BusinessOpenValue(value: 'fiber-processing', known: true),
  ];

  static const actions = <String>[
    'shop',
    'browse-patterns',
    'request-custom-order',
    'book-class',
    'book-appointment',
    'view-event-calendar',
    'email',
    'visit-website',
    'wholesale-enquiries',
  ];

  static String openValue(
    BusinessOpenValue value,
    AppLocalizations l10n,
  ) {
    if (!value.known) return l10n.businessUnknownValue(_fallback(value.value));
    return switch (value.value) {
      'dyer' => l10n.businessTypeDyer,
      'fiber-producer' => l10n.businessTypeFiberProducer,
      'fiber-processor' => l10n.businessTypeFiberProcessor,
      'yarn-shop' => l10n.businessTypeYarnShop,
      'fabric-shop' => l10n.businessTypeFabricShop,
      'craft-supply-shop' => l10n.businessTypeCraftSupplyShop,
      'pattern-designer' => l10n.businessTypePatternDesigner,
      'finished-goods-maker' => l10n.businessTypeFinishedGoodsMaker,
      'tool-maker' => l10n.businessTypeToolMaker,
      'teacher' => l10n.businessTypeTeacher,
      'craft-studio' => l10n.businessTypeCraftStudio,
      'repair-service' => l10n.businessTypeRepairService,
      'technical-editor' => l10n.businessTypeTechnicalEditor,
      'photographer' => l10n.businessTypePhotographer,
      'publisher' => l10n.businessTypePublisher,
      'other-craft-business' => l10n.businessTypeOtherCraftBusiness,
      'yarn' => l10n.businessOfferingYarn,
      'fiber' => l10n.businessOfferingFiber,
      'fabric' => l10n.businessOfferingFabric,
      'patterns' => l10n.businessOfferingPatterns,
      'kits' => l10n.businessOfferingKits,
      'notions' => l10n.businessOfferingNotions,
      'tools' => l10n.businessOfferingTools,
      'finished-goods' => l10n.businessOfferingFinishedGoods,
      'custom-work' => l10n.businessOfferingCustomWork,
      'repairs' => l10n.businessOfferingRepairs,
      'classes' => l10n.businessOfferingClasses,
      'studio-hire' => l10n.businessOfferingStudioHire,
      'wholesale' => l10n.businessOfferingWholesale,
      'digital-products' => l10n.businessOfferingDigitalProducts,
      'technical-editing' => l10n.businessOfferingTechnicalEditing,
      'photography-services' => l10n.businessOfferingPhotographyServices,
      'fiber-processing' => l10n.businessOfferingFiberProcessing,
      _ => l10n.businessUnknownValue(_fallback(value.value)),
    };
  }

  static String action(String type, AppLocalizations l10n) => switch (type) {
    'shop' => l10n.businessActionShop,
    'browse-patterns' => l10n.businessActionBrowsePatterns,
    'request-custom-order' => l10n.businessActionRequestCustomOrder,
    'book-class' => l10n.businessActionBookClass,
    'book-appointment' => l10n.businessActionBookAppointment,
    'view-event-calendar' => l10n.businessActionViewEventCalendar,
    'email' => l10n.businessActionEmail,
    'visit-website' => l10n.businessActionVisitWebsite,
    'wholesale-enquiries' => l10n.businessActionWholesaleEnquiries,
    _ => l10n.businessUnknownValue(_fallback(type)),
  };

  static String eventRole(
    BusinessOpenValue role,
    AppLocalizations l10n,
  ) => switch (role.value) {
    'organizer' when role.known => l10n.businessEventRoleOrganizer,
    'instructor' when role.known => l10n.businessEventRoleInstructor,
    'vendor' when role.known => l10n.businessEventRoleVendor,
    'exhibitor' when role.known => l10n.businessEventRoleExhibitor,
    'speaker' when role.known => l10n.businessEventRoleSpeaker,
    'demonstrator' when role.known => l10n.businessEventRoleDemonstrator,
    _ => l10n.businessUnknownValue(_fallback(role.value)),
  };

  static String eventMode(
    BusinessOpenValue mode,
    AppLocalizations l10n,
  ) => switch (mode.value) {
    'in-person' when mode.known => l10n.businessEventModeInPerson,
    'online' when mode.known => l10n.businessEventModeOnline,
    'hybrid' when mode.known => l10n.businessEventModeHybrid,
    _ => l10n.businessUnknownValue(_fallback(mode.value)),
  };

  static String eventStatus(
    BusinessOpenValue status,
    AppLocalizations l10n,
  ) => switch (status.value) {
    'scheduled' when status.known => l10n.businessEventStatusScheduled,
    'cancelled' when status.known => l10n.businessEventStatusCancelled,
    'postponed' when status.known => l10n.businessEventStatusPostponed,
    _ => l10n.businessUnknownValue(_fallback(status.value)),
  };

  static String location(
    BusinessLocation location,
    AppLocalizations l10n,
  ) => BusinessFormatters.location(location, l10n);

  static String _fallback(String value) {
    final safe = value
        .replaceAll(RegExp(r'[\x00-\x1F\x7F]+'), ' ')
        .replaceAll(RegExp(r'[-_\s]+'), ' ')
        .trim();
    final bounded = String.fromCharCodes(safe.runes.take(64));
    if (bounded.isEmpty) return '?';
    return '${bounded[0].toUpperCase()}${bounded.substring(1).toLowerCase()}';
  }
}

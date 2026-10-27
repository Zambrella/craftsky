// Policy pinned to Bluesky social-app 0d30bbead1df8409fac03370e4097bb8f33000fa.

const maxProfileLinks = 10;
const maxProfileLinkTitle = 30;

const profileLinkSupportLogos = <String, String>{
  'BackerKit': 'BackerKit',
  'Big Cartel': 'BigCartel',
  'Boosty': 'Boosty',
  'BOOTH': 'Booth',
  'Buy Me a Coffee': 'BuyMeACoffee',
  'Cash App': 'CashApp',
  'Chuffed': 'Chuffed',
  'Crowdfundr': 'Crowdfundr',
  'Donorbox': 'Donorbox',
  'Etsy': 'Etsy',
  'Fourthwall': 'Fourthwall',
  'Givebutter': 'Givebutter',
  'GiveSendGo': 'GiveSendGo',
  'GoFundMe': 'GoFundMe',
  'Gumroad': 'Gumroad',
  'Indiegogo': 'Indiegogo',
  'INPRNT': 'Inprint',
  'itch.io': 'ItchIo',
  'JustGiving': 'JustGiving',
  'Kickstarter': 'Kickstarter',
  'Ko-fi': 'KoFi',
  'Liberapay': 'Liberapay',
  'Open Collective': 'OpenCollective',
  'Patreon': 'Patreon',
  'PayPal': 'PayPal',
  'pixivFANBOX': 'PixivFanbox',
  'Redbubble': 'Redbubble',
  'Skeb': 'Skeb',
  'Stripe': 'Stripe',
  'Substack': 'Substack',
  'Throne': 'Throne',
  'Venmo': 'Venmo',
  'VGen': 'Vgen',
  'Zeffy': 'Zeffy',
};

const profileLinkShorteners = <String>{
  'aceimg.in',
  'acel.me',
  'adf.ly',
  'bit.ly',
  'bitly.com',
  'bl.ink',
  'buff.ly',
  'clck.ru',
  'cutt.ly',
  'fc-lc.xyz',
  'goo.gl',
  'goto.now',
  'is.gd',
  'link2.fm',
  'ln.run',
  'lnk.ua',
  'lnkd.in',
  'n9.cl',
  'onlylinks.cc',
  'ow.ly',
  'psee.io',
  'rb.gy',
  'rebrand.ly',
  'rg.by',
  's.id',
  'short.do',
  'short.gy',
  'shorten.as',
  'shorten.ee',
  'shorten.is',
  'shorten.so',
  'shorten.tv',
  'shorten.world',
  'shorter.me',
  'shortn.icu',
  'shorturl.at',
  'shorturl.lt',
  'shorturl.ly',
  'slicedrve.in',
  'slnt.to',
  'smarturl.lt',
  'smarturl.my',
  'sprl.me',
  'sw.run',
  't.co',
  't.ly',
  'tiny.cc',
  'tinyurl.com',
  'urlr.me',
  'v.gd',
  'vdey.in',
  'videy.at',
  'videy.lt',
  'videy.my',
  'x.gd',
  'zig.ht',
};

const profileLinkSupportProviders =
    <({String name, String domain, String? pathPrefix})>[
      (name: 'BackerKit', domain: 'backerkit.com', pathPrefix: null),
      (name: 'Big Cartel', domain: 'bigcartel.com', pathPrefix: null),
      (name: 'Boosty', domain: 'boosty.to', pathPrefix: null),
      (name: 'BOOTH', domain: 'booth.pm', pathPrefix: null),
      (name: 'Buy Me a Coffee', domain: 'buymeacoffee.com', pathPrefix: null),
      (name: 'Cash App', domain: 'cash.app', pathPrefix: null),
      (name: 'Chuffed', domain: 'chuffed.org', pathPrefix: '/campaign/'),
      (name: 'Chuffed', domain: 'chuffed.org', pathPrefix: '/project/'),
      (name: 'Crowdfundr', domain: 'crowdfundr.com', pathPrefix: null),
      (name: 'Donorbox', domain: 'donorbox.org', pathPrefix: null),
      (name: 'Etsy', domain: 'etsy.com', pathPrefix: null),
      (name: 'Fourthwall', domain: 'fourthwall.com', pathPrefix: null),
      (name: 'Givebutter', domain: 'givebutter.com', pathPrefix: null),
      (name: 'GiveSendGo', domain: 'givesendgo.com', pathPrefix: null),
      (name: 'GoFundMe', domain: 'gofundme.com', pathPrefix: null),
      (name: 'Gumroad', domain: 'gumroad.com', pathPrefix: null),
      (name: 'Indiegogo', domain: 'indiegogo.com', pathPrefix: null),
      (name: 'INPRNT', domain: 'inprnt.com', pathPrefix: '/gallery/'),
      (name: 'itch.io', domain: 'itch.io', pathPrefix: null),
      (name: 'JustGiving', domain: 'justgiving.com', pathPrefix: null),
      (name: 'Kickstarter', domain: 'kickstarter.com', pathPrefix: null),
      (name: 'Ko-fi', domain: 'ko-fi.com', pathPrefix: null),
      (name: 'Liberapay', domain: 'liberapay.com', pathPrefix: null),
      (name: 'Open Collective', domain: 'opencollective.com', pathPrefix: null),
      (name: 'Patreon', domain: 'patreon.com', pathPrefix: null),
      (name: 'PayPal', domain: 'paypal.com', pathPrefix: '/paypalme/'),
      (name: 'PayPal', domain: 'paypal.me', pathPrefix: null),
      (name: 'pixivFANBOX', domain: 'fanbox.cc', pathPrefix: null),
      (name: 'Redbubble', domain: 'redbubble.com', pathPrefix: null),
      (name: 'Skeb', domain: 'skeb.jp', pathPrefix: null),
      (name: 'Stripe', domain: 'buy.stripe.com', pathPrefix: null),
      (name: 'Stripe', domain: 'donate.stripe.com', pathPrefix: null),
      (name: 'Substack', domain: 'substack.com', pathPrefix: null),
      (name: 'Throne', domain: 'throne.com', pathPrefix: null),
      (name: 'Venmo', domain: 'venmo.com', pathPrefix: null),
      (name: 'VGen', domain: 'vgen.co', pathPrefix: null),
      (name: 'Zeffy', domain: 'zeffy.com', pathPrefix: null),
    ];

bool _matchesHost(String host, String domain) =>
    host == domain || host.endsWith('.$domain');

Uri? normalizeProfileLink(String value) {
  final raw = value.trim();
  final uri = Uri.tryParse(raw.contains('://') ? raw : 'https://$raw');
  if (uri == null ||
      uri.scheme != 'https' ||
      uri.host.isEmpty ||
      uri.userInfo.isNotEmpty ||
      RegExp(r'\s').hasMatch(raw) ||
      profileLinkShorteners.any(
        (domain) => _matchesHost(uri.host.toLowerCase(), domain),
      )) {
    return null;
  }
  return uri.replace(
    host: uri.host.toLowerCase(),
    path: uri.path.isEmpty ? '/' : uri.path,
  );
}

String? profileLinkSupportProvider(String value) {
  final uri = Uri.tryParse(value);
  if (uri == null || uri.scheme != 'https') return null;
  final host = uri.host.toLowerCase();
  final path = uri.path.toLowerCase().replaceFirst(RegExp(r'/+$'), '');
  for (final provider in profileLinkSupportProviders) {
    if (!_matchesHost(host, provider.domain)) continue;
    final prefix = provider.pathPrefix?.replaceFirst(RegExp(r'/+$'), '');
    if (prefix != null) {
      if (path.startsWith('$prefix/')) return provider.name;
    } else if ((host != provider.domain && host != 'www.${provider.domain}') ||
        path.isNotEmpty) {
      return provider.name;
    }
  }
  return null;
}

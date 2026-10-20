import Foundation
import PatrolImpl
import XCTest

// Patrol's standard iOS enterText action includes the entered value in NSLog.
// Keep this narrowly scoped route in the UI-test runner, never in Runner.app.
@objc(CraftSkySecureOAuthEntry)
final class SecureOAuthEntry: NSObject, PatrolServerExtension {
    var name: String { "CraftSky secure OAuth entry" }

    func register(on registrar: PatrolRouteRegistrar) {
        registrar.post("craftsky/enterOAuthPassword") { data in
            struct Request: Decodable { let password: String }
            let request = try JSONDecoder().decode(Request.self, from: data)
            let safari = XCUIApplication(bundleIdentifier: "com.apple.mobilesafari")
            let urlField = safari.textFields["URL"]
            let address = urlField.value as? String ?? ""
            let url = URL(string: address.contains("://") ? address : "https://" + address)
            let field = safari.secureTextFields.firstMatch
            guard url?.host == "bsky.social", field.exists, !request.password.isEmpty else {
                throw NSError(domain: "CraftSkyUITests", code: 1,
                              userInfo: [NSLocalizedDescriptionKey: "Expected the Bluesky OAuth password field in Safari."])
            }
            field.tap()
            // XCTest redacts typeText values for secure text fields.
            field.typeText(request.password)
            return Data("{}".utf8)
        }
    }
}

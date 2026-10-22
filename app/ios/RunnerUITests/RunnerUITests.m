@import XCTest;
@import patrol;
@import ObjectiveC.runtime;

@interface CraftSkyPatrolExtensions : NSObject
@end

@implementation CraftSkyPatrolExtensions
+ (void)load {
  PatrolRegisterServerExtensionClass(NSClassFromString(@"CraftSkySecureOAuthEntry"));
}
@end

PATROL_INTEGRATION_TEST_IOS_RUNNER(RunnerUITests)

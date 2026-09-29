package app

import "testing"

func TestProductionVideoIsAlwaysDisabled(t *testing.T) {
	if videoEnabledForLaunch(EnvProd, true) || videoEnabledForLaunch(EnvProd, false) {
		t.Fatal("production video gate became enabled")
	}
	if !videoEnabledForLaunch(EnvDev, true) {
		t.Fatal("development video fixture gate should remain available")
	}
}

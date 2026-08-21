package routes

import (
	"fmt"
	"strings"
	"testing"
)

func TestRouteConfigRedactsDevAuthSecret(t *testing.T) {
	config := Config{DevAuthSecret: "route-secret-must-not-leak"}
	for _, rendered := range []string{fmt.Sprint(config), fmt.Sprintf("%#v", config)} {
		if strings.Contains(rendered, "route-secret-must-not-leak") {
			t.Fatalf("route config exposed dev auth secret: %s", rendered)
		}
	}
}

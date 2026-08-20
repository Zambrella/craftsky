package app

import (
	"reflect"
	"testing"
)

func TestDependencyCleanupRunsReverseOrderExactlyOnce(t *testing.T) {
	var got []string
	cleanup := &dependencyCleanup{}
	cleanup.add(func() { got = append(got, "database") })
	cleanup.add(func() { got = append(got, "federated") })
	cleanup.add(func() { got = append(got, "observer") })

	cleanup.close()
	cleanup.close()

	want := []string{"observer", "federated", "database"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("cleanup order = %v, want %v", got, want)
	}
}

func TestDependencyCleanupIgnoresUnavailableSteps(t *testing.T) {
	cleanup := &dependencyCleanup{}
	cleanup.add(nil)
	cleanup.close()
}

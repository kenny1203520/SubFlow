package adapters

import "testing"

func TestUnsupportedDriversFail(t *testing.T) {
	for _, driver := range []string{"postgres", "mysql", "mongo"} {
		if _, err := New(driver, nil); err == nil {
			t.Fatalf("driver %q must fail until an adapter exists", driver)
		}
	}
}

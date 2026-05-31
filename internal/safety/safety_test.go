package safety

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"ibkr-stock-analysis/internal/account"
	"ibkr-stock-analysis/internal/market"
)

func TestMarketProviderInterfaceExposesOnlyReadOnlyMethods(t *testing.T) {
	providerType := reflect.TypeOf((*market.MarketDataProvider)(nil)).Elem()
	methods := make([]string, 0, providerType.NumMethod())
	for i := 0; i < providerType.NumMethod(); i++ {
		methods = append(methods, providerType.Method(i).Name)
	}

	forbidden := []string{
		token("Place", "Order"),
		token("Cancel", "Order"),
		"Order",
		"Trade",
		token("Position", "Size"),
		token("Account", "Allocation"),
	}
	for _, method := range methods {
		for _, token := range forbidden {
			if strings.Contains(method, token) {
				t.Fatalf("market provider exposes forbidden method %q", method)
			}
		}
	}
}

func TestAccountSnapshotProviderInterfaceExposesOnlySnapshotRead(t *testing.T) {
	providerType := reflect.TypeOf((*account.SnapshotProvider)(nil)).Elem()
	methods := make([]string, 0, providerType.NumMethod())
	for i := 0; i < providerType.NumMethod(); i++ {
		methods = append(methods, providerType.Method(i).Name)
	}
	if len(methods) != 1 || methods[0] != "Snapshot" {
		t.Fatalf("account snapshot provider methods = %#v, want only Snapshot", methods)
	}

	forbidden := []string{
		token("Place", "Order"),
		token("Cancel", "Order"),
		"Order",
		"Trade",
		"Allocate",
		token("Open", "Order"),
		"Stage",
		"Approve",
		token("Trans", "mit"),
	}
	for _, method := range methods {
		for _, token := range forbidden {
			if strings.Contains(method, token) {
				t.Fatalf("account provider exposes forbidden method %q", method)
			}
		}
	}
}

func TestOwnGoSourceDoesNotCallTradingMutationAPIs(t *testing.T) {
	root := projectRoot(t)
	forbidden := []string{
		token("Place", "Order", "("),
		token("Cancel", "Order", "("),
		token("Req", "Open", "Orders", "("),
		token("Req", "Account", "Updates", "("),
		token("Trans", "mit", ":"),
		token("Order", "ID"),
		token("Order", "Id"),
		token("Brack", "et"),
		"staged order",
		"stage order",
	}

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "frontend" || path == filepath.Join(root, "internal", "safety") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(data)
		for _, token := range forbidden {
			if strings.Contains(text, token) {
				t.Fatalf("%s contains forbidden API %s", path, token)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestIBKRAccountReadAPIsAppearOnlyInAccountBoundary(t *testing.T) {
	root := projectRoot(t)
	readAPIs := []string{
		token("Req", "Account", "Summary", "("),
		token("Cancel", "Account", "Summary", "("),
		token("Req", "Positions", "("),
		token("Cancel", "Positions", "("),
	}

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "frontend" || path == filepath.Join(root, "internal", "safety") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		text := string(data)
		for _, token := range readAPIs {
			if strings.Contains(text, token) && !strings.Contains(path, filepath.Join("internal", "account")) {
				t.Fatalf("%s contains account read API %s outside internal/account", path, token)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func token(parts ...string) string {
	return strings.Join(parts, "")
}

func projectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root")
		}
		dir = parent
	}
}

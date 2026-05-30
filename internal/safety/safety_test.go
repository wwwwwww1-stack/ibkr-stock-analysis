package safety

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"ibkr-stock-analysis/internal/market"
)

func TestMarketProviderInterfaceExposesOnlyReadOnlyMethods(t *testing.T) {
	providerType := reflect.TypeOf((*market.MarketDataProvider)(nil)).Elem()
	methods := make([]string, 0, providerType.NumMethod())
	for i := 0; i < providerType.NumMethod(); i++ {
		methods = append(methods, providerType.Method(i).Name)
	}

	forbidden := []string{"PlaceOrder", "CancelOrder", "Order", "Trade", "PositionSize", "AccountAllocation"}
	for _, method := range methods {
		for _, token := range forbidden {
			if strings.Contains(method, token) {
				t.Fatalf("market provider exposes forbidden method %q", method)
			}
		}
	}
}

func TestOwnGoSourceDoesNotCallTradingMutationAPIs(t *testing.T) {
	root := projectRoot(t)
	forbidden := []string{"PlaceOrder(", "CancelOrder(", "ReqAccountUpdates(", "ReqPositions(", "ReqOpenOrders("}

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

package deviceid

import (
	"testing"
)

// TestIdentifyDeviceBasic testa a função básica de identificação
func TestIdentifyDeviceBasic(t *testing.T) {
	testCases := []struct {
		name string
		mac string
		wantVendor string
	}{
		{
			name: "Apple MAC",
			mac: "A4:5E:60:12:34:56",
			wantVendor: "Apple",
		},
		{
			name: "VMware MAC",
			mac: "00:50:56:12:34:56",
			wantVendor: "VMware",
		},
		{
			name: "Invalid MAC",
			mac: "invalid",
			wantVendor: "Unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vendor, _ := IdentifyDevice(tc.mac)
			if vendor != tc.wantVendor {
				t.Errorf("IdentifyDevice(%s) = %s, want %s", tc.mac, vendor, tc.wantVendor)
			}
		})
	}
}

// TestIdentifyDeviceDetailed testa a função detalhada de identificação
func TestIdentifyDeviceDetailed(t *testing.T) {
	testCases := []struct {
		name string
		mac string
		wantVendor string
	}{
		{
			name: "Apple MAC detailed",
			mac: "A4:5E:60:12:34:56",
			wantVendor: "Apple",
		},
		{
			name: "Empty MAC",
			mac: "",
			wantVendor: "Unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info := IdentifyDeviceDetailed(tc.mac)
			if info.Name != tc.wantVendor {
				t.Errorf("IdentifyDeviceDetailed(%s).Name = %s, want %s", tc.mac, info.Name, tc.wantVendor)
			}
		})
	}
}

// TestGetDetailedVendorInfo testa a nova função que usa OUIja
func TestGetDetailedVendorInfo(t *testing.T) {
	// Testa com um MAC conhecido
	details, err := GetDetailedVendorInfo("A4:5E:60:12:34:56")

	// Se não conseguir conectar (offline), o teste deve passar
	if err != nil {
		t.Logf("Teste offline - não foi possível conectar à OUIja: %v", err)
		return
	}

	if details == nil {
		t.Error("GetDetailedVendorInfo retornou nil para MAC válido")
		return
	}

	if details.Vendor == "" {
		t.Error("GetDetailedVendorInfo não retornou vendor")
	}

	if details.OUI == "" {
		t.Error("GetDetailedVendorInfo não retornou OUI")
	}
}

// TestSearchVendorsByPattern testa a busca por padrão
func TestSearchVendorsByPattern(t *testing.T) {
	vendors, err := SearchVendorsByPattern("apple")

	// Se não conseguir conectar (offline), o teste deve passar
	if err != nil {
		t.Logf("Teste offline - não foi possível buscar vendors: %v", err)
		return
	}

	if len(vendors) == 0 {
		t.Error("SearchVendorsByPattern('apple') não retornou resultados")
	}

	// Verifica se pelo menos um dos resultados contém "Apple"
	found := false
	for _, vendor := range vendors {
		if vendor.Vendor == "Apple" {
			found = true
			break
		}
	}

	if !found {
		t.Error("SearchVendorsByPattern('apple') não encontrou 'Apple' nos resultados")
	}
}

// TestGetTopVendors testa a função de top vendors
func TestGetTopVendors(t *testing.T) {
	vendors := GetTopVendors(5)

	if len(vendors) == 0 {
		t.Error("GetTopVendors(5) não retornou nenhum vendor")
	}

	if len(vendors) > 5 {
		t.Errorf("GetTopVendors(5) retornou %d vendors, esperado máximo 5", len(vendors))
	}

	// Verifica se os vendors estão ordenados por count (decrescente)
	for i := 1; i < len(vendors); i++ {
		if vendors[i-1].Count < vendors[i].Count {
			t.Error("GetTopVendors não retornou vendors ordenados por count")
			break
		}
	}
}

// TestGetDatabaseStats testa as estatísticas da base
func TestGetDatabaseStats(t *testing.T) {
	stats, err := GetDatabaseStats()

	// Se não conseguir conectar (offline), o teste deve passar
	if err != nil {
		t.Logf("Teste offline - não foi possível obter estatísticas: %v", err)
		return
	}

	if stats == nil {
		t.Error("GetDatabaseStats retornou nil")
		return
	}

	if stats.TotalOUIs <= 0 {
		t.Error("GetDatabaseStats retornou TotalOUIs <= 0")
	}

	if stats.Source == "" {
		t.Error("GetDatabaseStats não retornou Source")
	}
}

// BenchmarkIdentifyDevice testa a performance da identificação
func BenchmarkIdentifyDevice(b *testing.B) {
	mac := "A4:5E:60:12:34:56"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IdentifyDevice(mac)
	}
}

// BenchmarkIdentifyDeviceDetailed testa a performance da identificação detalhada
func BenchmarkIdentifyDeviceDetailed(b *testing.B) {
	mac := "A4:5E:60:12:34:56"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IdentifyDeviceDetailed(mac)
	}
}

package main

import (
	"fmt"
	"log"

	"github.com/lucasrafaldini/wavetrack/internal/deviceid"
)

// testMACs contém alguns endereços MAC para demonstração
var testMACs = []string{
	"A4:5E:60:12:34:56", // Apple
	"00:50:56:12:34:56", // VMware
	"00:0C:29:12:34:56", // VMware
	"28:cf:e9:12:34:56", // Apple
	"B8:27:EB:12:34:56", // Raspberry Pi Foundation
	"00:15:5D:12:34:56", // Microsoft
	"52:54:00:12:34:56", // Red Hat (QEMU)
	"08:00:27:12:34:56", // Oracle VirtualBox
}

func main() {
	fmt.Println("🌐 Demonstração da Integração OUIja no WaveTrack")
	fmt.Println("=======================================================")

	// Testa identificação básica (método existente)
	fmt.Println("\n📋 Teste de Identificação Básica:")
	for _, mac := range testMACs {
		vendor, deviceType := deviceid.IdentifyDevice(mac)
		fmt.Printf("MAC: %s → Vendor: %s, Tipo: %s\n", mac, vendor, deviceType)
	}

	// Testa identificação detalhada
	fmt.Println("\n🔍 Teste de Identificação Detalhada:")
	for _, mac := range testMACs[:3] { // Testa apenas 3 para não poluir
		info := deviceid.IdentifyDeviceDetailed(mac)
		fmt.Printf("MAC: %s\n", mac)
		fmt.Printf("  Vendor: %s\n", info.Name)
		fmt.Printf("  Tipo: %s\n", info.DeviceType)
		fmt.Printf("  Ambíguo: %t\n", info.IsAmbiguous)
		if info.IsAmbiguous {
			fmt.Printf("  Tipos possíveis: %v\n", info.PossibleTypes)
		}
		fmt.Println()
	}

	// Testa informações detalhadas via OUIja
	fmt.Println("🌐 Teste de Informações Detalhadas (OUIja):")
	for _, mac := range testMACs[:2] { // Testa apenas 2
		details, err := deviceid.GetDetailedVendorInfo(mac)
		if err != nil {
			fmt.Printf("Erro ao buscar detalhes para %s: %v\n", mac, err)
			continue
		}

		fmt.Printf("MAC: %s\n", details.MAC)
		fmt.Printf("  Vendor: %s\n", details.Vendor)
		fmt.Printf("  OUI: %s\n", details.OUI)
		fmt.Printf("  Tipo: %s\n", details.DeviceType)
		fmt.Printf("  Ambíguo: %t\n", details.IsAmbiguous)
		fmt.Println()
	}

	// Testa busca por padrão
	fmt.Println("🔎 Teste de Busca por Padrão:")
	searchTerms := []string{"apple", "microsoft", "vmware"}
	for _, term := range searchTerms {
		vendors, err := deviceid.SearchVendorsByPattern(term)
		if err != nil {
			fmt.Printf("Erro ao buscar padrão '%s': %v\n", term, err)
			continue
		}

		fmt.Printf("Padrão '%s': %d resultados\n", term, len(vendors))
		for i, vendor := range vendors {
			if i >= 3 { // Mostra apenas os 3 primeiros
				fmt.Printf("  ... e mais %d\n", len(vendors)-3)
				break
			}
			fmt.Printf("  %s (%d OUIs)\n", vendor.Vendor, vendor.Count)
		}
		fmt.Println()
	}

	// Testa top vendors
	fmt.Println("🏆 Top 10 Fabricantes por OUIs:")
	topVendors := deviceid.GetTopVendors(10)
	for i, vendor := range topVendors {
		fmt.Printf("%2d. %s (%d OUIs)\n", i+1, vendor.Vendor, vendor.Count)
	}

	// Testa estatísticas da base
	fmt.Println("\n📊 Estatísticas da Base de Dados:")
	stats, err := deviceid.GetDatabaseStats()
	if err != nil {
		log.Printf("Erro ao obter estatísticas: %v", err)
		return
	}

	fmt.Printf("Total de OUIs: %d\n", stats.TotalOUIs)
	fmt.Printf("Fonte: %s\n", stats.Source)
	fmt.Printf("Última atualização: %s\n", stats.LastUpdate)

	fmt.Println("\n✅ Demonstração concluída!")
}

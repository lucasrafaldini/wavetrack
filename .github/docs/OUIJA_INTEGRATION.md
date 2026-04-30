# Integração com OUIja - Identificação de Fabricantes MAC

## Visão Geral

O WaveTrack agora utiliza a biblioteca [OUIja](https://github.com/lucasrafaldini/ouija) para identificação automática de fabricantes através de endereços MAC. Esta integração oferece uma base de dados oficial e atualizada do IEEE para identificação precisa de dispositivos.

## Características da Integração

### Método Principal: OUIja
- **Base de dados oficial**: Utiliza a base OUI mantida pela Wireshark/IEEE
- **Auto-atualização**: Cache inteligente com TTL configurável
- **Performance**: Busca em memória após carregamento inicial
- **Cobertura**: Base completa com milhares de fabricantes

### Fallback Inteligente
- **Base local**: Fallback para fabricantes mais comuns em caso de problemas de rede
- **Cache persistente**: Evita consultas repetidas para MACs já identificados
- **Tolerância a falhas**: Sistema continua funcionando mesmo offline

## Novos Recursos Disponíveis

### API REST Endpoints

#### 1. Detalhes do Fabricante
```bash
POST /api/vendor/details
Content-Type: application/json

{
 "mac_address": "00:50:56:12:34:56"
}
```

**Resposta:**
```json
{
 "mac": "00:50:56:12:34:56",
 "vendor": "VMware",
 "oui": "00:50:56",
 "device_type": "virtual_machine",
 "is_ambiguous": false,
 "possible_types": []
}
```

#### 2. Busca de Fabricantes por Padrão
```bash
POST /api/vendor/search
Content-Type: application/json

{
 "pattern": "apple"
}
```

**Resposta:**
```json
{
 "pattern": "apple",
 "vendors": [
 {
 "vendor": "Apple",
 "count": 127
 },
 {
 "vendor": "Apple Computer",
 "count": 45
 }
 ],
 "count": 2
}
```

#### 3. Top Fabricantes por OUIs
```bash
GET /api/vendor/top
```

**Resposta:**
```json
{
 "top_vendors": [
 {
 "vendor": "Intel",
 "count": 1247
 },
 {
 "vendor": "Apple",
 "count": 892
 }
 ],
 "limit": 20,
 "count": 20
}
```

#### 4. Estatísticas da Base de Dados
```bash
GET /api/vendor/stats
```

**Resposta:**
```json
{
 "total_ouis": 28945,
 "source": "Wireshark/IEEE Official Database",
 "last_update": "Auto-updated via OUIja"
}
```

## Funcionalidades Programáticas

### Uso Direto no Código

```go
import "github.com/lucasrafaldini/wavetrack/internal/deviceid"

// Identificação básica (compatível com código existente)
vendor, deviceType := deviceid.IdentifyDevice("A4:5E:60:12:34:56")
// Retorna: "Apple", "incerto" (com possíveis tipos)

// Identificação detalhada
info := deviceid.IdentifyDeviceDetailed("A4:5E:60:12:34:56")
fmt.Printf("Fabricante: %s", info.Name)
fmt.Printf("Tipo: %s", info.DeviceType)
fmt.Printf("Ambíguo: %t", info.IsAmbiguous)
fmt.Printf("Tipos possíveis: %v", info.PossibleTypes)

// Informações detalhadas via OUIja
details, err := deviceid.GetDetailedVendorInfo("A4:5E:60:12:34:56")
if err == nil {
 fmt.Printf("OUI: %s", details.OUI)
 fmt.Printf("MAC normalizado: %s", details.MAC)
}

// Busca por fabricante
vendors, err := deviceid.SearchVendorsByPattern("samsung")

// Top fabricantes
topVendors := deviceid.GetTopVendors(10)

// Estatísticas da base
stats, err := deviceid.GetDatabaseStats()
```

## Logs e Monitoramento

O sistema agora produz logs informativos mostrando a fonte da identificação:

```
 Sistema de identificação inicializado com OUIja (base IEEE oficial)
 OUIja: Biblioteca de identificação de fabricantes via MAC address
 Fallback: Base de dados local para casos offline

 OUIja: A4:5E:60:12:34:56 [Apple (incerto)] (base IEEE oficial)
 LOCAL: 00:03:93:12:34:56 [Apple (incerto)] (base OUI IEEE backup)
 FALLBACK: 28:cf:e9:12:34:56 [Apple (incerto)] (base interna)
```

## Performance e Cache

### Cache Inteligente
- **Cache local**: MACs identificados são cachados para evitar consultas repetidas
- **TTL configurável**: Base OUI atualizada automaticamente (padrão: 7 dias)
- **Fallback robusto**: Sistema funciona mesmo com problemas de rede

### Performance
- **Primeira consulta**: ~100-500ms (download inicial da base)
- **Consultas subsequentes**: ~1-5ms (busca em memória)
- **Uso de memória**: ~2-5MB (base OUI completa)

## Compatibilidade

### Retrocompatibilidade
- Todas as funções existentes continuam funcionando
- Interface da API mantida inalterada
- Logs e formatos de resposta preservados

### Melhorias
- Identificação mais precisa com base oficial IEEE
- Novos endpoints para funcionalidades avançadas
- Cache inteligente para melhor performance
- Sistema de fallback mais robusto

## Configuração

Não são necessárias configurações adicionais. A biblioteca OUIja funciona automaticamente com:

- **Cache automático**: `~/.cache/ouija/manuf` (Linux/macOS)
- **Download automático**: Base IEEE baixada na primeira execução
- **Atualizações automáticas**: TTL de 7 dias por padrão
- **Fallback offline**: Base local para casos sem internet

## Exemplo de Teste

Para testar a integração:

```bash
# Compilar o projeto
go build ./cmd/wavetrack

# Executar em modo de desenvolvimento
./wavetrack

# Testar endpoints via curl
curl -X POST http://localhost:8080/api/vendor/details \
 -H "Content-Type: application/json" \
 -d '{"mac_address": "A4:5E:60:12:34:56"}'

curl -X GET http://localhost:8080/api/vendor/top

curl -X GET http://localhost:8080/api/vendor/stats
```

## Benefícios da Integração

1. ** Base Oficial**: Utiliza dados mantidos oficialmente pela IEEE/Wireshark
2. ** Auto-atualização**: Mantém a base sempre atualizada automaticamente
3. ** Performance**: Cache inteligente para consultas rápidas
4. ** Robustez**: Sistema de fallback para garantir funcionamento offline
5. ** Precisão**: Identificação mais precisa com base completa de OUIs
6. ** Facilidade**: Integração transparente sem quebrar código existente

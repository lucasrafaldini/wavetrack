# WaveTrack

*Leia em [English](README.md).*

**Veja quem está no escritório pelos celulares no Wi-Fi.** O WaveTrack percebe quando um celular ou notebook cadastrado aparece na rede do escritório e transforma isso em um painel ao vivo, uma linha do tempo do dia e um histórico de presença. As pessoas se cadastram sozinhas com um QR code, nada é instalado no celular delas e nada é rastreado fora da rede do escritório.

![Painel do WaveTrack](docs/screenshots/now.png)

- **Agora:** quem está, desde quando e há quanto tempo hoje.
- **Linha do tempo:** cada permanência do dia como uma barra, inclusive o almoço.
- **Histórico:** horas por pessoa por dia, em 7, 14 ou 30 dias, exportável em CSV.
- **Pessoas:** cadastro por QR code, vários aparelhos por pessoa, anonimizar ou apagar a pedido.
- **Por perto:** aparelhos vistos recentemente que não pertencem a ninguém, prontos para atribuir.
- Um único binário em Go com o painel embutido, SQLite e um modo demo que não exige configuração.

## 1. Experimente em um minuto

O demo simula um escritório de doze pessoas com duas semanas de histórico, sem precisar de permissões de captura.

```bash
git clone https://github.com/lucasrafaldini/wavetrack.git
cd wavetrack
make demo
```

Abra o link que ele imprime. Fora do horário comercial, o relógio do demo vai para as 14h20 para o escritório sempre parecer movimentado.

## 2. Instalação

Precisa de Go 1.23+ e libpcap (`sudo apt install libpcap-dev` no Debian/Ubuntu; no macOS já vem com as Command Line Tools do Xcode).

```bash
make build    # gera ./wavetrack
```

## 3. Primeira execução

1. **Escolha a interface:** `./wavetrack interfaces` mostra as opções. O WaveTrack escolhe sozinho o Wi-Fi conectado e nunca adivinha uma porta cabeada; se o Wi-Fi e a rede cabeada forem a mesma rede, `--interface en0` também funciona.
2. **Confira as permissões:** `./wavetrack doctor` verifica tudo e diz exatamente o que fazer. Prefira conceder a permissão uma vez a rodar como root:
   - **macOS:** `brew install --cask wireshark-chmodbpf` e depois saia e entre na sessão.
   - **Linux:** `sudo setcap cap_net_raw,cap_net_admin=eip ./wavetrack`
3. **Inicie e entre:** `./wavetrack`. A primeira execução imprime, uma única vez, o token de administrador (só o hash dele é guardado). Perdeu? `./wavetrack token reset`.

![Tela de login](docs/screenshots/signin.png)

## 4. Cadastre as pessoas

**Opção A, convite por QR (recomendado).** Em **People**, clique em **Invite with QR code**. Cada convite vale para uma pessoa, por 24 horas.

![Convite por QR](docs/screenshots/invite.png)

A pessoa conecta no Wi-Fi do escritório, escaneia, digita o nome e concorda. O WaveTrack identifica o celular pela própria rede; nunca confia num endereço que o aparelho alega.

| Claro | Escuro |
| --- | --- |
| ![Página de cadastro](docs/screenshots/join.png) | ![Página de cadastro, escura](docs/screenshots/join-dark.png) |

**Opção B, manualmente.** **Add person** recebe nome, equipe e, se quiser, o MAC de um aparelho (no iPhone: *Ajustes › Wi-Fi › ⓘ › Endereço Wi-Fi*).

**Opção C, em Por perto.** **Nearby** lista aparelhos vistos que não são de ninguém; escolha uma pessoa no menu para atribuir.

![Por perto](docs/screenshots/nearby.png)

## 5. No dia a dia

| Agora | Linha do tempo | Histórico |
| --- | --- | --- |
| ![Agora](docs/screenshots/now-dark.png) | ![Linha do tempo](docs/screenshots/timeline.png) | ![Histórico](docs/screenshots/history.png) |

## 6. Privacidade

- **Só aparelhos cadastrados vão para o disco.** Os demais ficam um tempo na memória (para a tela Por perto) e nunca são gravados.
- **O consentimento é registrado** em quem se cadastra sozinho.
- **Duas formas de remover alguém:** *anonimizar* (troca o nome, esquece os aparelhos, mantém as horas) ou *apagar tudo* (pessoa, aparelhos e visitas), para pedidos sob a LGPD.

  ![Removendo uma pessoa](docs/screenshots/remove-person.png)

- **Retenção:** visitas mais antigas que `privacy.retention_days` (365 por padrão) são apagadas automaticamente.
- **Nada sai do prédio:** sem contas, nuvem ou telemetria.
- **Painel protegido:** token de administrador, cookies `HttpOnly` e `SameSite=Strict`, bloqueio de requisições de outros sites e CSP estrita.

## 7. Configuração, serviço e API

Tudo é opcional: copie [`config.example.yaml`](config.example.yaml) para `config.yaml`. Arquivos prontos para rodar como serviço estão em [`deploy/`](deploy) (systemd sem root no Linux, launchd no macOS). A API e a configuração estão detalhadas no [README em inglês](README.md#7-configuration).

## 8. Como funciona

Celulares e notebooks se anunciam o tempo todo na rede (ARP, DHCP, descoberta de vizinhos IPv6 e mDNS). O WaveTrack escuta esses pacotes, anota o endereço de hardware de quem enviou e transforma avistamentos em **visitas**: do primeiro avistamento até o último, depois de `away_after` sem nenhum. As decisões de projeto estão em [docs/architecture.md](docs/architecture.md).

## 9. Vindo da 2.x

```bash
./wavetrack import-v2 data/wavetrack.db --data data-v3
./wavetrack --data data-v3
```

Licença MIT · Lucas Rafaldini

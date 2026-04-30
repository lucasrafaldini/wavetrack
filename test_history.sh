#!/bin/bash

# Inicia o servidor em background
sudo ./wavetrack &
SERVER_PID=$!

# Aguarda servidor iniciar
echo "Aguardando servidor iniciar..."
sleep 3

# Testa o endpoint
echo -e "\n Testando endpoint de histórico:\n"
curl -s http://localhost:8080/api/history/7days | jq '.[0]' || curl -s http://localhost:8080/api/history/7days | head -50

# Mata o servidor
echo -e "\n\nFinalizando servidor..."
sudo kill $SERVER_PID 2>/dev/null

echo " Teste concluído"

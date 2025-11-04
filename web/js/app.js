let currentMAC = '';
let cachedDevices = [];
let sortField = 'name';
let sortDir = 'asc';
let onlyEmployees = false;
let currentTab = 'current';
let historyData = [];

// Carrega dados iniciais
loadData();

// Inicializa ícones de ordenação
setTimeout(() => {
    updateSortIcons();
}, 100);

// Atualiza automaticamente a cada 5 segundos (apenas aba atual)
setInterval(() => {
    if (currentTab === 'current') {
        loadData();
    }
}, 5000);

async function loadData() {
    try {
        await Promise.all([
            loadStats(),
            loadDevices()
        ]);
    } catch (error) {
        console.error('Erro ao carregar dados:', error);
    }
}

async function loadHistory() {
    try {
        const response = await fetch('/api/history/7days');
        historyData = await response.json();
        renderHistory();
    } catch (error) {
        console.error('Erro ao carregar histórico:', error);
        document.getElementById('historyContainer').innerHTML = `
            <div class="empty-state">
                <p style="color: #999;">Erro ao carregar histórico</p>
            </div>
        `;
    }
}

function switchTab(tabName) {
    currentTab = tabName;
    
    // Atualiza visual das tabs
    document.querySelectorAll('.tab').forEach(tab => tab.classList.remove('active'));
    document.querySelectorAll('.tab-content').forEach(content => content.classList.remove('active'));
    
    event.target.classList.add('active');
    document.getElementById(`tab-${tabName}`).classList.add('active');
    
    // Carrega dados da aba
    if (tabName === 'history' && historyData.length === 0) {
        loadHistory();
    } else if (tabName === 'employees') {
        loadEmployeesList();
    }
}

function renderHistory() {
    const container = document.getElementById('historyContainer');
    
    if (!historyData || historyData.length === 0) {
        container.innerHTML = `
            <div class="empty-state">
                <p style="color: #999;">Nenhum dado histórico disponível</p>
            </div>
        `;
        return;
    }

    container.innerHTML = historyData.map(day => `
        <div style="margin-bottom: 40px;">
            <div class="history-header">
                <div class="history-date">
                    📅 ${formatDate(day.date)}
                </div>
                <div class="history-stats">
                    <span>
                        👥 <strong>${day.total_employees}</strong> funcionário(s)
                    </span>
                    <span>
                        📱 <strong>${day.total_devices}</strong> dispositivo(s)
                    </span>
                    <span>
                        ⏱️ <strong>${formatDuration(day.total_hours * 3600)}</strong> total
                    </span>
                </div>
            </div>
            
            <table>
                <thead>
                    <tr>
                        <th>Funcionário</th>
                        <th>Dispositivo</th>
                        <th>Primeira Chegada</th>
                        <th>Última Saída</th>
                        <th>Tempo Online</th>
                        <th>Presença</th>
                    </tr>
                </thead>
                <tbody>
                    ${day.employees.map(emp => `
                        <tr>
                            <td class="employee-name">${emp.name}</td>
                            <td>
                                ${getDeviceTypeIcon(emp.device_type)} 
                                ${getDeviceTypeName(emp.device_type)}
                            </td>
                            <td>${emp.first_arrival || '—'}</td>
                            <td>${emp.last_departure || '—'}</td>
                            <td>
                                <strong style="color: #667eea;">${formatDuration(emp.duration_seconds)}</strong>
                            </td>
                            <td>
                                <span class="badge">${emp.presence_percentage}%</span>
                            </td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        </div>
    `).join('');
}

function formatDate(dateStr) {
    const date = new Date(dateStr);
    const today = new Date();
    const yesterday = new Date(today);
    yesterday.setDate(yesterday.getDate() - 1);
    
    const dateOnly = date.toDateString();
    const todayOnly = today.toDateString();
    const yesterdayOnly = yesterday.toDateString();
    
    if (dateOnly === todayOnly) return 'Hoje';
    if (dateOnly === yesterdayOnly) return 'Ontem';
    
    const weekdays = ['Domingo', 'Segunda', 'Terça', 'Quarta', 'Quinta', 'Sexta', 'Sábado'];
    return `${weekdays[date.getDay()]}, ${date.toLocaleDateString('pt-BR')}`;
}

async function loadStats() {
    const response = await fetch('/api/stats');
    const stats = await response.json();
    
    document.getElementById('totalDevices').textContent = stats.total_devices;
    document.getElementById('activeDevices').textContent = stats.active_devices;
    document.getElementById('totalEmployees').textContent = stats.total_employees;
    document.getElementById('presentEmployees').textContent = stats.present_employees;
}

async function loadDevices() {
    const response = await fetch('/api/devices');
    cachedDevices = await response.json();
    renderDevices();
}

function renderDevices() {
    const tbody = document.getElementById('devicesBody');
    const devices = [...cachedDevices]
        .filter(d => !onlyEmployees || !!d.employee_name)
        .sort((a, b) => compareDevices(a, b));

    if (devices.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="10" class="empty-state">
                    <div>Nenhum dispositivo para exibir...</div>
                </td>
            </tr>
        `;
        return;
    }

    tbody.innerHTML = devices.map(device => rowHTML(device)).join('');
}

function compareDevices(a, b) {
    let result = 0;
    if (sortField === 'status') {
        // Ordena: Ativo antes de Inativo
        result = (b.is_active ? 1 : 0) - (a.is_active ? 1 : 0);
    } else if (sortField === 'name') {
        const an = (a.employee_name || '').toLowerCase();
        const bn = (b.employee_name || '').toLowerCase();
        result = an.localeCompare(bn);
    } else if (sortField === 'arrival') {
        const at = a.first_seen_today ? new Date(a.first_seen_today).getTime() : Infinity;
        const bt = b.first_seen_today ? new Date(b.first_seen_today).getTime() : Infinity;
        result = at - bt;
    } else if (sortField === 'duration') {
        const ad = a.online_duration_seconds || 0;
        const bd = b.online_duration_seconds || 0;
        result = ad - bd;
    } else if (sortField === 'lastseen') {
        const at = a.last_seen ? new Date(a.last_seen).getTime() : 0;
        const bt = b.last_seen ? new Date(b.last_seen).getTime() : 0;
        result = at - bt;
    }
    return sortDir === 'asc' ? result : -result;
}

function rowHTML(device) {
    return `
        <tr>
            <td>
                <span class="status ${device.is_active ? 'active' : 'inactive'}">
                    ${device.is_active ? '🟢 Ativo' : '🔴 Inativo'}
                </span>
            </td>
            <td class="mac-address">${device.mac_address}</td>
            <td>
                ${renderDeviceType(device)}
            </td>
            <td style="font-size: 0.9em; color: #666;">
                ${device.vendor || 'Desconhecido'}
            </td>
            <td>
                ${device.employee_name 
                    ? `<span class="employee-name">${device.employee_name}</span>` 
                    : '<em style="color: #999;">Não cadastrado</em>'}
            </td>
            <td style="font-size: 0.9em; color: #444;">
                ${device.first_seen_today 
                    ? new Date(device.first_seen_today).toLocaleTimeString('pt-BR', {hour: '2-digit', minute: '2-digit'})
                    : '<span style="color:#999;">—</span>'}
            </td>
            <td>
                ${device.signal_strength === 0 && device.frequency === 0
                    ? '<span style="color: #999; font-style: italic;">N/A (modo não-monitor)</span>'
                    : `<div style="display:flex; flex-direction:column; gap:4px;">
                        ${device.signal_strength !== 0 ? `
                            <div class="signal">
                                <div class="signal-bar">
                                    <div class="signal-fill" style="width: ${getSignalPercent(device.signal_strength)}%"></div>
                                </div>
                                <span style="font-size: 0.85em; color: #666;">${device.signal_strength} dBm</span>
                            </div>
                        ` : ''}
                        ${device.channel !== 0 ? `
                            <span style="font-size: 0.8em; color: #888;">
                                📡 Canal ${device.channel} ${device.frequency ? `(${device.frequency} MHz)` : ''}
                            </span>
                        ` : ''}
                    </div>`
                }
            </td>
            <td style="font-size: 0.9em; color:#444;">
                ${formatDuration(device.online_duration_seconds)}
            </td>
            <td style="font-size: 0.9em; color: #666;">
                ${formatTime(device.last_seen)}
            </td>
            <td>
                ${!device.employee_name 
                    ? `<button class="btn btn-primary" onclick="openModal('${device.mac_address}')">Cadastrar</button>`
                    : `<button class="btn btn-primary" onclick="openEditModal('${device.mac_address}')" style="background:#ffc107;border:none;">✏️ Editar</button>`}
            </td>
        </tr>
    `;
}

function getDeviceTypeIcon(type) {
    // Tipos específicos
    if (type.includes('notebook') || type.includes('laptop') || type === 'laptop') return '💻';
    if (type.includes('iphone') || type.includes('samsung') || type.includes('xiaomi') || 
        type.includes('motorola') || type === 'smartphone') return '📱';
    if (type.includes('ipad') || type.includes('tab') || type === 'tablet') return '�';
    if (type === 'router' || type.includes('router')) return '🌐';
    if (type === 'iot' || type.includes('iot')) return '🔗';
    if (type === 'smarttv' || type.includes('smarttv')) return '📺';
    if (type === 'console' || type.includes('console')) return '🎮';
    if (type === 'incerto' || type === 'uncertain') return '❔';
    if (type === 'unknown' || type === 'desconhecido') return '❓';
    return '❓';
}

function getDeviceTypeName(type) {
    // Mapeia tipos customizados
    const customNames = {
        'notebook-windows': 'Notebook Windows',
        'notebook-mac': 'Notebook Mac',
        'notebook-linux': 'Notebook Linux',
        'iphone-17': 'iPhone 17',
        'iphone-16': 'iPhone 16',
        'iphone-15': 'iPhone 15',
        'iphone-14': 'iPhone 14',
        'iphone-13': 'iPhone 13',
        'iphone-12': 'iPhone 12',
        'samsung-s24': 'Samsung Galaxy S24',
        'samsung-s23': 'Samsung Galaxy S23',
        'samsung-a54': 'Samsung Galaxy A54',
        'xiaomi-13': 'Xiaomi 13',
        'xiaomi-redmi': 'Xiaomi Redmi Note',
        'motorola-edge': 'Motorola Edge',
        'motorola-moto-g': 'Moto G',
        'ipad-pro': 'iPad Pro',
        'ipad-air': 'iPad Air',
        'ipad': 'iPad',
        'samsung-tab-s9': 'Samsung Galaxy Tab S9',
        'samsung-tab-a': 'Samsung Galaxy Tab A'
    };
    
    if (customNames[type]) return customNames[type];
    
    // Fallback para tipos padrão
    const names = {
        'laptop': 'Laptop',
        'smartphone': 'Smartphone', 
        'tablet': 'Tablet',
        'router': 'Roteador',
        'iot': 'Dispositivo IoT',
        'smarttv': 'Smart TV',
        'console': 'Console',
        'incerto': 'Incerto',
        'unknown': 'Desconhecido'
    };
    return names[type] || type || 'Desconhecido';
}

function getSignalPercent(signal) {
    // Converte sinal dBm (-100 a 0) para percentual
    return Math.max(0, Math.min(100, (signal + 100)));
}

function formatTime(timestamp) {
    const date = new Date(timestamp);
    const now = new Date();
    const diff = Math.floor((now - date) / 1000);
    
    if (diff < 60) return 'Agora mesmo';
    if (diff < 3600) return `${Math.floor(diff / 60)} minutos atrás`;
    if (diff < 86400) return `${Math.floor(diff / 3600)} horas atrás`;
    
    return date.toLocaleString('pt-BR');
}

function formatDuration(seconds) {
    if (!seconds || seconds <= 0) return '—';
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    if (h > 0) return `${h}h ${m}m`;
    return `${m}m`;
}

function openModal(macAddress) {
    currentMAC = macAddress;
    document.getElementById('modalTitle').textContent = '👤 Cadastrar Funcionário';
    document.getElementById('modalMAC').value = macAddress;
    document.getElementById('employeeName').value = '';
    document.getElementById('employeeDepartment').value = '';
    document.getElementById('deviceType').value = '';
    document.getElementById('deviceVendor').value = '';
    document.getElementById('associateModal').classList.add('show');
}

async function openEditModal(macAddress) {
    currentMAC = macAddress;
    document.getElementById('modalTitle').textContent = '✏️ Editar Funcionário';
    document.getElementById('modalMAC').value = macAddress;
    
    // Busca dados do funcionário
    try {
        const response = await fetch('/api/employees');
        const employees = await response.json();
        const employee = employees.find(e => e.mac_address === macAddress);
        
        if (employee) {
            document.getElementById('employeeName').value = employee.name || '';
            document.getElementById('employeeDepartment').value = employee.department || '';
            document.getElementById('deviceType').value = employee.custom_device_type || '';
            document.getElementById('deviceVendor').value = employee.custom_vendor || '';
        }
    } catch (error) {
        console.error('Erro ao carregar dados do funcionário:', error);
    }
    
    document.getElementById('associateModal').classList.add('show');
}

function closeModal() {
    document.getElementById('associateModal').classList.remove('show');
}

// Fecha modal ao clicar fora
document.getElementById('associateModal').addEventListener('click', (e) => {
    if (e.target.id === 'associateModal') {
        closeModal();
    }
});

// Controles de UI: filtrar
document.getElementById('onlyEmployees').addEventListener('change', (e) => {
    onlyEmployees = e.target.checked;
    renderDevices();
});

// Função para ordenação por coluna
function sortTable(field) {
    // Se clicar na mesma coluna, inverte a direção
    if (sortField === field) {
        sortDir = sortDir === 'asc' ? 'desc' : 'asc';
    } else {
        // Nova coluna, começa ascendente
        sortField = field;
        sortDir = 'asc';
    }
    
    // Atualiza ícones
    updateSortIcons();
    
    // Re-renderiza
    renderDevices();
}

// Atualiza os ícones de ordenação
function updateSortIcons() {
    // Remove classe active de todos
    document.querySelectorAll('th.sortable').forEach(th => th.classList.remove('active'));
    
    // Reseta todos os ícones (deixa vazio para o CSS usar ::before)
    document.querySelectorAll('.sort-icon').forEach(icon => {
        icon.textContent = '';
    });
    
    // Atualiza o ícone da coluna ativa
    const icon = document.getElementById(`sort-${sortField}`);
    if (icon) {
        icon.closest('th').classList.add('active');
        icon.textContent = sortDir === 'asc' ? '↑' : '↓';
    }
}

// Download relatório do dia
async function downloadReport() {
    try {
        const response = await fetch('/api/report/today');
        const report = await response.json();

        // Formata como JSON bonito para download
        const blob = new Blob([JSON.stringify(report, null, 2)], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `wavetrack-report-${new Date().toISOString().split('T')[0]}.json`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    } catch (error) {
        console.error('Erro ao baixar relatório:', error);
        alert('Erro ao gerar relatório');
    }
}

// Limpa dispositivos inativos sem cadastro
async function cleanupInactive() {
    if (!confirm('⚠️ Tem certeza? Isso irá remover todos os dispositivos inativos sem cadastro de colaborador.\n\nEsta ação não pode ser desfeita!')) {
        return;
    }

    try {
        const response = await fetch('/api/cleanup/inactive', {
            method: 'POST'
        });

        if (!response.ok) {
            throw new Error('Erro na limpeza');
        }

        const result = await response.json();
        
        alert(`✅ Limpeza concluída!\n\n📱 Dispositivos removidos: ${result.removed_devices}\n📋 Eventos removidos: ${result.removed_events}\n\nDispositivos não cadastrados: ${result.total_before} → ${result.total_after}`);
        
        // Atualiza a interface
        loadData();
    } catch (error) {
        console.error('Erro na limpeza:', error);
        alert('❌ Erro ao executar limpeza. Tente novamente.');
    }
}

// ============================================
// GERENCIAMENTO DE COLABORADORES
// ============================================

let employeesList = [];

async function loadEmployeesList() {
    try {
        const response = await fetch('/api/employees');
        employeesList = await response.json();
        renderEmployeesList();
    } catch (error) {
        console.error('Erro ao carregar colaboradores:', error);
        document.getElementById('employeesBody').innerHTML = `
            <tr>
                <td colspan="7" class="empty-state">
                    <div>Erro ao carregar colaboradores</div>
                </td>
            </tr>
        `;
    }
}

function renderEmployeesList() {
    const tbody = document.getElementById('employeesBody');
    document.getElementById('employeeCount').textContent = employeesList.length;

    if (employeesList.length === 0) {
        tbody.innerHTML = `
            <tr>
                <td colspan="7" class="empty-state">
                    <div>Nenhum colaborador cadastrado</div>
                </td>
            </tr>
        `;
        return;
    }

    tbody.innerHTML = employeesList.map(emp => `
        <tr>
            <td class="employee-name">${emp.name}</td>
            <td class="mac-address">${emp.mac_address}</td>
            <td>${emp.department || '<em style="color:#999;">—</em>'}</td>
            <td>${emp.custom_device_type ? getDeviceTypeName(emp.custom_device_type) : '<em style="color:#999;">Auto</em>'}</td>
            <td>${emp.custom_vendor || '<em style="color:#999;">Auto</em>'}</td>
            <td style="font-size:0.85em;color:#666;">${formatDate2(emp.created_at)}</td>
            <td>
                <div class="btn-group">
                    <button class="btn btn-warning btn-sm" onclick="openEditEmployeeModal('${emp.mac_address}')" title="Editar">
                        ✏️
                    </button>
                    <button class="btn btn-danger btn-sm" onclick="confirmDeleteEmployee('${emp.mac_address}', '${emp.name}')" title="Excluir">
                        🗑️
                    </button>
                </div>
            </td>
        </tr>
    `).join('');
}

function formatDate2(dateStr) {
    if (!dateStr) return '—';
    const date = new Date(dateStr);
    return date.toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric' });
}

function openNewEmployeeModal() {
    currentMAC = '';
    document.getElementById('modalTitle').textContent = '➕ Novo Colaborador';
    document.getElementById('modalMAC').value = '';
    document.getElementById('modalMAC').removeAttribute('readonly');
    document.getElementById('employeeName').value = '';
    document.getElementById('employeeDepartment').value = '';
    document.getElementById('deviceType').value = '';
    document.getElementById('deviceVendor').value = '';
    document.getElementById('associateModal').classList.add('show');
}

async function openEditEmployeeModal(macAddress) {
    currentMAC = macAddress;
    document.getElementById('modalTitle').textContent = '✏️ Editar Colaborador';
    document.getElementById('modalMAC').value = macAddress;
    document.getElementById('modalMAC').removeAttribute('readonly'); // Permite editar MAC
    
    const employee = employeesList.find(e => e.mac_address === macAddress);
    if (employee) {
        document.getElementById('employeeName').value = employee.name || '';
        document.getElementById('employeeDepartment').value = employee.department || '';
        document.getElementById('deviceType').value = employee.custom_device_type || '';
        document.getElementById('deviceVendor').value = employee.custom_vendor || '';
    }
    
    document.getElementById('associateModal').classList.add('show');
}

let deleteCallback = null;

function confirmDeleteEmployee(macAddress, name) {
    const dialog = document.createElement('div');
    dialog.className = 'confirm-dialog show';
    dialog.innerHTML = `
        <div class="confirm-content">
            <h3>⚠️ Confirmar Exclusão</h3>
            <p>Tem certeza que deseja excluir o colaborador <strong>${name}</strong>?</p>
            <p style="font-size:0.85em;color:#999;">MAC: ${macAddress}</p>
            <div class="confirm-buttons">
                <button class="btn btn-danger" onclick="deleteEmployee('${macAddress}')">Sim, Excluir</button>
                <button class="btn btn-primary" onclick="closeConfirmDialog()">Cancelar</button>
            </div>
        </div>
    `;
    document.body.appendChild(dialog);
}

function closeConfirmDialog() {
    const dialog = document.querySelector('.confirm-dialog');
    if (dialog) {
        dialog.remove();
    }
}

async function deleteEmployee(macAddress) {
    try {
        const response = await fetch(`/api/employees/${macAddress}`, {
            method: 'DELETE'
        });

        if (response.ok) {
            closeConfirmDialog();
            loadEmployeesList();
            loadData(); // Atualiza dashboard principal
            alert('✓ Colaborador excluído com sucesso!');
        } else {
            alert('Erro ao excluir colaborador');
        }
    } catch (error) {
        console.error('Erro:', error);
        alert('Erro ao excluir colaborador');
    }
}

// Modifica o comportamento do formulário para suportar criação e edição
document.getElementById('associateForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    
    const macInput = document.getElementById('modalMAC');
    const newMAC = macInput.value.trim();
    
    if (!newMAC) {
        alert('Por favor, informe o MAC Address');
        return;
    }
    
    const data = {
        mac_address: newMAC,
        name: document.getElementById('employeeName').value,
        department: document.getElementById('employeeDepartment').value,
        custom_device_type: document.getElementById('deviceType').value,
        custom_vendor: document.getElementById('deviceVendor').value
    };

    // Se está editando e o MAC mudou, enviar o MAC antigo
    if (currentMAC && currentMAC !== newMAC) {
        data.old_mac_address = currentMAC;
    }

    try {
        const response = await fetch('/api/associate', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });

        if (response.ok) {
            closeModal();
            loadData();
            if (currentTab === 'employees') {
                loadEmployeesList();
            }
            alert('✓ Colaborador salvo com sucesso!');
        } else {
            alert('Erro ao salvar colaborador');
        }
    } catch (error) {
        console.error('Erro:', error);
        alert('Erro ao salvar colaborador');
    }
});

// ============================================
// QR CODE REGISTRATION
// ============================================

let currentQRUrl = '';

async function openQRCodeModal() {
    const modal = document.getElementById('qrModal');
    const loading = document.getElementById('qrLoading');
    const content = document.getElementById('qrContent');
    const error = document.getElementById('qrError');
    
    // Reset estado
    loading.style.display = 'block';
    content.style.display = 'none';
    error.style.display = 'none';
    
    modal.style.display = 'flex';
    
    try {
        const response = await fetch('/api/register/token', {
            method: 'POST'
        });
        
        if (!response.ok) {
            throw new Error('Erro ao gerar QR Code');
        }
        
        const data = await response.json();
        console.log('Dados recebidos:', data);
        
        // Exibe o QR Code (já vem com o prefixo data:image/png;base64,)
        document.getElementById('qrCodeImage').src = data.qr_code;
        currentQRUrl = data.url;
        
        console.log('QR Code URL:', currentQRUrl);
        
        // Calcula tempo de expiração
        const expiresAt = new Date(data.expires_at);
        const now = new Date();
        const hoursRemaining = Math.ceil((expiresAt - now) / (1000 * 60 * 60));
        document.getElementById('qrExpiration').textContent = `${hoursRemaining} hora(s)`;
        
        loading.style.display = 'none';
        content.style.display = 'block';
        
    } catch (err) {
        console.error('Erro ao gerar QR Code:', err);
        loading.style.display = 'none';
        error.style.display = 'block';
        document.getElementById('qrErrorMessage').textContent = 
            'Erro ao gerar QR Code. Tente novamente.';
    }
}

function closeQRModal() {
    document.getElementById('qrModal').style.display = 'none';
    currentQRUrl = '';
}

async function copyQRLink() {
    if (!currentQRUrl) {
        alert('❌ Nenhum link disponível');
        return;
    }
    
    try {
        await navigator.clipboard.writeText(currentQRUrl);
        alert('✓ Link copiado para a área de transferência!');
    } catch (err) {
        // Fallback para navegadores antigos
        const input = document.createElement('input');
        input.value = currentQRUrl;
        document.body.appendChild(input);
        input.select();
        document.execCommand('copy');
        document.body.removeChild(input);
        alert('✓ Link copiado para a área de transferência!');
    }
}

// Fecha modal ao clicar fora
window.onclick = function(event) {
    const qrModal = document.getElementById('qrModal');
    if (event.target == qrModal) {
        closeQRModal();
    }
}

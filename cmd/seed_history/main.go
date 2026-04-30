package main

import (
	"database/sql"
	"log"
	"math/rand"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Abre conexão com o banco
	db, err := sql.Open("sqlite3", "data/wavetrack.db")
	if err != nil {
		log.Fatalf("Erro ao abrir banco: %v", err)
	}
	defer db.Close()

	log.Println(" Iniciando seed de dados históricos...")

	// Lista de funcionários fake
	employees := []struct {
		Name string
		MAC string
		DeviceType string
		Vendor string
		Department string
	}{
		{"Lucas Rafaldini", "00:11:22:33:44:01", "iphone-16", "Apple", "TI"},
		{"Ana Silva", "00:11:22:33:44:02", "samsung-s24", "Samsung", "RH"},
		{"Carlos Mendes", "00:11:22:33:44:03", "notebook-mac", "Apple", "Vendas"},
		{"Maria Santos", "00:11:22:33:44:04", "iphone-15", "Apple", "Marketing"},
		{"João Oliveira", "00:11:22:33:44:05", "notebook-windows", "Dell", "TI"},
		{"Fernanda Costa", "00:11:22:33:44:06", "samsung-s23", "Samsung", "Financeiro"},
		{"Ricardo Lima", "00:11:22:33:44:07", "notebook-linux", "Lenovo", "TI"},
		{"Patricia Souza", "00:11:22:33:44:08", "iphone-14", "Apple", "RH"},
		{"Bruno Almeida", "00:11:22:33:44:09", "xiaomi-13", "Xiaomi", "Vendas"},
		{"Juliana Ferreira", "00:11:22:33:44:10", "motorola-edge", "Motorola", "Marketing"},
		{"Rafael Barbosa", "00:11:22:33:44:11", "ipad-pro", "Apple", "Design"},
		{"Camila Rocha", "00:11:22:33:44:12", "samsung-tab-s9", "Samsung", "Design"},
	}

	// Cadastra funcionários
	log.Println(" Cadastrando funcionários fake...")
	for _, emp := range employees {
		_, err := db.Exec(`
			INSERT OR REPLACE INTO employees (mac_address, name, department, custom_device_type, custom_vendor, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, datetime('now'), datetime('now'))
		`, emp.MAC, emp.Name, emp.Department, emp.DeviceType, emp.Vendor)

		if err != nil {
			log.Printf(" Erro ao inserir %s: %v", emp.Name, err)
		} else {
			log.Printf(" %s cadastrado", emp.Name)
		}
	}

	// Gera histórico dos últimos 7 dias
	log.Println("\n Gerando histórico dos últimos 7 dias...")

	rand.Seed(time.Now().UnixNano())

	for dayOffset := 0; dayOffset < 7; dayOffset++ {
		date := time.Now().AddDate(0, 0, -dayOffset)
		dayOfWeek := int(date.Weekday())

		log.Printf("\n Dia: %s", date.Format("02/01/2006"))

		// Define quantos funcionários estarão presentes (menos no fim de semana)
		numPresent := 5 + (dayOffset % 4) + (dayOfWeek % 3)
		if dayOfWeek == 0 || dayOfWeek == 6 { // Fim de semana
			numPresent = 2 + (dayOffset % 2)
		}

		// Embaralha funcionários e pega os primeiros N
		shuffled := make([]int, len(employees))
		for i := range shuffled {
			shuffled[i] = i
		}
		rand.Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		})

		for i := 0; i < numPresent && i < len(shuffled); i++ {
			emp := employees[shuffled[i]]

			// Horário de chegada (entre 7h e 10h)
			arrivalHour := 7 + rand.Intn(4)
			arrivalMinute := rand.Intn(60)
			arrivalTime := time.Date(date.Year(), date.Month(), date.Day(), arrivalHour, arrivalMinute, 0, 0, time.Local)

			// Duração (entre 4h e 10h)
			durationHours := 4 + rand.Intn(7)
			durationMinutes := rand.Intn(60)
			departureTime := arrivalTime.Add(time.Duration(durationHours)*time.Hour + time.Duration(durationMinutes)*time.Minute)

			// Insere device
			_, err := db.Exec(`
				INSERT OR REPLACE INTO devices
				(mac_address, type, vendor, signal_strength, frequency, channel, first_seen, last_seen, is_active)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, emp.MAC, emp.DeviceType, emp.Vendor, -50, 2437, 6, arrivalTime, departureTime, false)

			if err != nil {
				log.Printf(" Erro ao inserir device %s: %v", emp.Name, err)
				continue
			}

			// Evento de arrival
			_, err = db.Exec(`
				INSERT INTO events (mac_address, event_type, timestamp)
				VALUES (?, 'arrival', ?)
			`, emp.MAC, arrivalTime.Format("2006-01-02 15:04:05"))

			if err != nil {
				log.Printf(" Erro ao inserir arrival %s: %v", emp.Name, err)
			}

			// Evento de departure
			_, err = db.Exec(`
				INSERT INTO events (mac_address, event_type, timestamp)
				VALUES (?, 'departure', ?)
			`, emp.MAC, departureTime.Format("2006-01-02 15:04:05"))

			if err != nil {
				log.Printf(" Erro ao inserir departure %s: %v", emp.Name, err)
			}

			log.Printf(" %s: %s - %s (%dh%dm)",
				emp.Name,
				arrivalTime.Format("15:04"),
				departureTime.Format("15:04"),
				durationHours, durationMinutes)
		}
	}

	log.Println("\n Seed concluído com sucesso!")
	log.Println(" Execute o wavetrack e acesse a aba 'Histórico (7 dias)' para ver os dados")
}

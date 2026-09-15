package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	ProcFile = "/proc/continfo_pr2_so1_201800632"

	LoopInterval      = 20 * time.Second
	ValkeyAddr        = "localhost:6379"
	ValkeyPassword    = ""
	ValkeyDB          = 0
	MinLowContainers  = 3
	MinHighContainers = 2

	ScriptsDir = "./scripts"
)

// ProcData es el JSON que produce el módulo de kernel
type ProcData struct {
	RAMTotalMB uint64    `json:"ram_total_mb"`
	RAMFreeMB  uint64    `json:"ram_free_mb"`
	RAMUsedMB  uint64    `json:"ram_used_mb"`
	Processes  []Process `json:"processes"`
}

type Process struct {
	PID         int    `json:"pid"`
	Name        string `json:"name"`
	Cmdline     string `json:"cmdline"`
	VSZKB       uint64 `json:"vsz_kb"`
	RSSKB       uint64 `json:"rss_kb"`
	MemPercent  uint64 `json:"mem_percent"`
	CPUPercent  uint64 `json:"cpu_percent"`
	IsContainer int    `json:"is_container"`
}

// ContainerInfo combina datos del módulo de kernel + docker inspect
type ContainerInfo struct {
	ContainerID string
	Name        string
	PID         int
	VSZKB       uint64
	RSSKB       uint64
	MemPercent  uint64
	CPUPercent  uint64
	Type        string // "alto" | "bajo" | "grafana" | "valkey"
}

// LogEntry es lo que guardamos en Valkey para Grafana
type LogEntry struct {
	Timestamp        string          `json:"timestamp"`
	RAMTotalMB       uint64          `json:"ram_total_mb"`
	RAMFreeMB        uint64          `json:"ram_free_mb"`
	RAMUsedMB        uint64          `json:"ram_used_mb"`
	ContainersKilled []string        `json:"containers_killed"`
	TopRAM           []ContainerInfo `json:"top_ram"`
	TopCPU           []ContainerInfo `json:"top_cpu"`
}

// Valkey

var rdb *redis.Client
var ctx = context.Background()

func initValkey() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     ValkeyAddr,
		Password: ValkeyPassword,
		DB:       ValkeyDB,
	})

	// Verificar conexión
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("[Valkey] No se pudo conectar: %v", err)
	}
	log.Println("[Valkey] Conexión establecida.")
}

// runCmd ejecuta un comando del sistema y retorna su salida
func runCmd(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	return strings.TrimSpace(string(out)), err
}

// getRunningContainers devuelve lista de IDs de contenedores activos
func getRunningContainers() ([]string, error) {
	out, err := runCmd("sudo","docker", "ps", "-q")
	if err != nil {
		return nil, fmt.Errorf("docker ps falló: %w", err)
	}
	if out == "" {
		return []string{}, nil
	}
	return strings.Split(out, "\n"), nil
}

// getContainerLabel obtiene el label "tipo" de un contenedor
func getContainerLabel(containerID string) string {
	out, err := runCmd("sudo","docker", "inspect",
		"--format", "{{index .Config.Labels \"tipo\"}}", containerID)
	if err != nil {
		return "desconocido"
	}
	label := strings.TrimSpace(out)
	if label == "" {
		return "desconocido"
	}
	return label
}

// getContainerName obtiene el nombre de un contenedor
func getContainerName(containerID string) string {
	out, err := runCmd("docker", "inspect",
		"--format", "{{.Name}}", containerID)
	if err != nil {
		return containerID
	}
	return strings.TrimPrefix(strings.TrimSpace(out), "/")
}

// getContainerPID obtiene el PID del proceso principal de un contenedor
func getContainerPID(containerID string) int {
	out, err := runCmd("sudo","docker", "inspect",
		"--format", "{{.State.Pid}}", containerID)
	if err != nil {
		return 0
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(out))
	return pid
}

// killContainer detiene y elimina un contenedor
func killContainer(containerID string) error {
	log.Printf("[Docker] Eliminando contenedor %s ...", containerID)
	if _, err := runCmd("sudo","docker", "stop", containerID); err != nil {
		return fmt.Errorf("docker stop %s: %w", containerID, err)
	}
	if _, err := runCmd("sudo","docker", "rm", "-f", containerID); err != nil {
		// No es fatal si ya se eliminó solo (--rm flag)
		log.Printf("[Docker] rm ignorado para %s: %v", containerID, err)
	}
	return nil
}

// startGrafanaAndValkey levanta los contenedores de Grafana y Valkey
// via docker-compose y luego establece la conexión con Valkey.
func startGrafanaAndValkey() {
	log.Println("[Init] Levantando Grafana y Valkey...")

	bout, berr := runCmd("sudo","docker", "compose", "up", "-d")
	if berr != nil || bout == ""{
		log.Printf("[Init] Error iniciando docker-compose: %v", berr)
	}

	out, err := runCmd("docker", "ps", "-q", "-f", "name=grafana_so1")
	if err != nil || out == "" {
		_, err := runCmd("sudo","docker-compose", "-f",
			filepath.Join(".", "docker-compose.yml"), "up", "-d")
		if err != nil {
			log.Printf("[Init] Error iniciando docker-compose: %v", err)
		} else {
			log.Println("[Init] Grafana y Valkey arrancados.")
		}
	} else {
		log.Println("[Init] Grafana ya está corriendo.")
	}

	// Conectar al cliente Valkey una vez que el contenedor está arriba
	initValkey()
}

//cronjob

func installCronjob() {
	absScript, _ := filepath.Abs(filepath.Join(ScriptsDir, "containers.sh"))

	// Asegura permisos de ejecución
	os.Chmod(absScript, 0755)
	current, _ := runCmd("crontab", "-l")

	if strings.Contains(current, absScript) {
		log.Println("[Cron] Cronjob ya instalado.")
		return
	}

	newEntry := fmt.Sprintf("*/2 * * * * %s >> /tmp/containers_cron.log 2>&1\n", absScript)
	newCrontab := current + "\n" + newEntry

	// Instalar usando echo | crontab -
	cmd := exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(newCrontab)
	if err := cmd.Run(); err != nil {
		log.Printf("[Cron] Error instalando cronjob: %v", err)
	} else {
		log.Println("[Cron] Cronjob instalado — cada 2 minutos.")
	}
}

func removeCronjob() {
	absScript, _ := filepath.Abs(filepath.Join(ScriptsDir, "containers.sh"))
	current, err := runCmd("crontab", "-l")
	if err != nil {
		return
	}

	lines := strings.Split(current, "\n")
	filtered := []string{}
	for _, line := range lines {
		if !strings.Contains(line, absScript) {
			filtered = append(filtered, line)
		}
	}

	newCrontab := strings.Join(filtered, "\n")
	cmd := exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(newCrontab)
	if err := cmd.Run(); err != nil {
		log.Printf("[Cron] Error eliminando cronjob: %v", err)
	} else {
		log.Println("[Cron] Cronjob eliminado.")
	}
}

func loadKernelModule() {
	absScript, _ := filepath.Abs(filepath.Join(ScriptsDir, "load_module.sh"))
	os.Chmod(absScript, 0755)

	log.Println("[Kernel] Ejecutando script de carga del módulo...")
	cmd := exec.Command("bash", absScript)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("[Kernel] Advertencia al cargar módulo: %v", err)
	}
}

func readProcFile() (*ProcData, error) {
	data, err := os.ReadFile(ProcFile)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer %s: %w", ProcFile, err)
	}

	var proc ProcData
	if err := json.Unmarshal(data, &proc); err != nil {
		return nil, fmt.Errorf("error deserializando JSON: %w", err)
	}

	return &proc, nil
}

//eBPF container
func startEBPF(){
	log.Println("[Init] Levantando el módulo eBPF...")
	// Ve al directorio del módulo eBPF y compílalo
	ebpfDir := "./ebpf"
	cmd := exec.Command("make", "-C", ebpfDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("[eBPF] Error compilando módulo: %v", err)
		return
	}

	// Cargar el módulo eBPF (suponiendo que el binario se llama ebpf_module.o)
	cmd = exec.Command("sudo", "insmod", filepath.Join(ebpfDir, "ebpf_module.o"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("[eBPF] Error cargando módulo: %v", err)
		return
	}

	log.Println("[eBPF] Módulo eBPF cargado exitosamente.")
}

// buildContainerList cruza los datos de docker ps con los del módulo de kernel
func buildContainerList(proc *ProcData) ([]ContainerInfo, error) {
	ids, err := getRunningContainers()
	if err != nil {
		return nil, err
	}

	// Construye mapa PID → Process desde los datos del kernel
	pidMap := make(map[int]Process)
	for _, p := range proc.Processes {
		pidMap[p.PID] = p
	}

	var containers []ContainerInfo
	for _, id := range ids {
		if id == "" {
			continue
		}

		name := getContainerName(id)
		label := getContainerLabel(id)
		pid := getContainerPID(id)

		ci := ContainerInfo{
			ContainerID: id,
			Name:        name,
			PID:         pid,
			Type:        label,
		}

		// Enriquecer con datos del kernel si el PID coincide
		if p, ok := pidMap[pid]; ok {
			ci.VSZKB = p.VSZKB
			ci.RSSKB = p.RSSKB
			ci.MemPercent = p.MemPercent
			ci.CPUPercent = p.CPUPercent
		}

		containers = append(containers, ci)
	}

	return containers, nil
}

func manageContainers(containers []ContainerInfo) []string {
	var low, high, protected []ContainerInfo

	for _, c := range containers {
		name := strings.ToLower(c.Name)
		// Proteger Grafana y Valkey pase lo que pase
		if strings.Contains(name, "grafana") || strings.Contains(name, "valkey") {
			protected = append(protected, c)
			continue
		}

		switch c.Type {
		case "bajo":
			low = append(low, c)
		case "alto":
			high = append(high, c)
		default:
			// Sin label: lo tratamos como alto consumo
			high = append(high, c)
		}
	}

	log.Printf("[Manager] Contenedores: bajo=%d (min %d), alto=%d (min %d), protegidos=%d",
		len(low), MinLowContainers, len(high), MinHighContainers, len(protected))

	var killed []string

	// --- Ordenar contenedores de bajo consumo por RAM (mayor primero) para eliminar excedente ---
	sort.Slice(low, func(i, j int) bool {
		return low[i].MemPercent > low[j].MemPercent
	})

	// Eliminar excedente de bajo consumo (mantener solo MinLowContainers)
	for len(low) > MinLowContainers {
		victim := low[0]
		low = low[1:]
		if err := killContainer(victim.ContainerID); err != nil {
			log.Printf("[Manager] Error eliminando %s: %v", victim.ContainerID, err)
		} else {
			log.Printf("[Manager] Eliminado (bajo) → %s [%s]", victim.Name, victim.ContainerID)
			killed = append(killed, victim.ContainerID)
		}
	}

	sort.Slice(high, func(i, j int) bool {
		scoreI := high[i].CPUPercent*2 + high[i].MemPercent
		scoreJ := high[j].CPUPercent*2 + high[j].MemPercent
		return scoreI > scoreJ
	})

	for len(high) > MinHighContainers {
		victim := high[0]
		high = high[1:]
		if err := killContainer(victim.ContainerID); err != nil {
			log.Printf("[Manager] Error eliminando %s: %v", victim.ContainerID, err)
		} else {
			log.Printf("[Manager] Eliminado (alto) → %s [%s]", victim.Name, victim.ContainerID)
			killed = append(killed, victim.ContainerID)
		}
	}

	return killed
}

func top5ByRAM(containers []ContainerInfo) []ContainerInfo {
	sorted := make([]ContainerInfo, len(containers))
	copy(sorted, containers)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].RSSKB > sorted[j].RSSKB
	})
	if len(sorted) > 5 {
		return sorted[:5]
	}
	return sorted
}

func top5ByCPU(containers []ContainerInfo) []ContainerInfo {
	sorted := make([]ContainerInfo, len(containers))
	copy(sorted, containers)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].CPUPercent > sorted[j].CPUPercent
	})
	if len(sorted) > 5 {
		return sorted[:5]
	}
	return sorted
}

func saveToValkey(entry LogEntry) {
	ts := time.Now().UnixMilli()

	// Guardar métricas de RAM como hash para consulta en tiempo real
	rdb.HSet(ctx, "so1:ram:latest",
		"total_mb", entry.RAMTotalMB,
		"free_mb", entry.RAMFreeMB,
		"used_mb", entry.RAMUsedMB,
		"ts", entry.Timestamp,
	)

	// Serie temporal de RAM (sorted set, score = timestamp unix ms)
	rdb.ZAdd(ctx, "so1:ram:history", redis.Z{
		Score:  float64(ts),
		Member: fmt.Sprintf("%d:%d:%d", entry.RAMTotalMB, entry.RAMUsedMB, entry.RAMFreeMB),
	})

	// Contador de contenedores eliminados
	if len(entry.ContainersKilled) > 0 {
		rdb.ZAdd(ctx, "so1:killed:history", redis.Z{
			Score:  float64(ts),
			Member: fmt.Sprintf("%s:%d", entry.Timestamp, len(entry.ContainersKilled)),
		})
		rdb.IncrBy(ctx, "so1:killed:total",
			int64(len(entry.ContainersKilled)))
	}

	// Top 5 RAM
	if topRaw, err := json.Marshal(entry.TopRAM); err == nil {
		rdb.Set(ctx, "so1:top5:ram", topRaw, 0)
	}

	// Top 5 CPU
	if topRaw, err := json.Marshal(entry.TopCPU); err == nil {
		rdb.Set(ctx, "so1:top5:cpu", topRaw, 0)
	}

	// Log completo como entrada en lista (últimas 200)
	if raw, err := json.Marshal(entry); err == nil {
		rdb.LPush(ctx, "so1:logs", raw)
		rdb.LTrim(ctx, "so1:logs", 0, 199)
	}

	log.Printf("[Valkey] Datos guardados → RAM usada: %dMB, eliminados: %d",
		entry.RAMUsedMB, len(entry.ContainersKilled))
}

func processProcAndStore() {
	log.Println("[Loop] Leyendo /proc...")

	// 1. Leer /proc
	proc, err := readProcFile()
	if err != nil {
		log.Printf("[Loop] Error leyendo /proc: %v", err)
		return
	}

	log.Printf("[Loop] RAM → Total: %dMB | Usada: %dMB | Libre: %dMB",
		proc.RAMTotalMB, proc.RAMUsedMB, proc.RAMFreeMB)

	// 2. Construir lista de contenedores
	containers, err := buildContainerList(proc)
	if err != nil {
		log.Printf("[Loop] Error construyendo lista de contenedores: %v", err)
		return
	}

	log.Printf("[Loop] Contenedores detectados: %d", len(containers))

	killed := manageContainers(containers)

	containers, _ = buildContainerList(proc)

	topRAM := top5ByRAM(containers)
	topCPU := top5ByCPU(containers)

	// Loggear en Valkey
	entry := LogEntry{
		Timestamp:        time.Now().Format(time.RFC3339),
		RAMTotalMB:       proc.RAMTotalMB,
		RAMFreeMB:        proc.RAMFreeMB,
		RAMUsedMB:        proc.RAMUsedMB,
		ContainersKilled: killed,
		TopRAM:           topRAM,
		TopCPU:           topCPU,
	}
	saveToValkey(entry)

	log.Println("[Loop] Datos almacenados en Valkey. Esperando próximo ciclo...")
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Daeomn iniciando:::")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// 1. Llamar a función que cree el contenedor de Grafana + Valkey
	startGrafanaAndValkey()

	// 2. Código o función para crear el cronjob
	installCronjob()

	// 3. Código o función para instalar los módulos de kernel
	loadKernelModule()

	// 4. eBPF: Cargar el Dockerfile de EBPF, compilar y cargar; Y monitorear la información del eBPF


	// 5. Loop de lectura de los archivos de /proc
	ticker := time.NewTicker(LoopInterval)
	defer ticker.Stop()

	log.Printf("[Main] Loop activo — leyendo %s cada %v", ProcFile, LoopInterval)

	for {
		select {
		case <-ticker.C:
			processProcAndStore()

		case sig := <-stop:
			log.Printf("[Main] Señal recibida: %v — apagando daemon...", sig)
			removeCronjob()
			log.Println("[Main] Daemon finalizado.")
			os.Exit(0)
		}
	}
}

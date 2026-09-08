// handlers/health - Enhanced Health and System Status endpoint.
// Provides detailed telemetry in both JSON and a rich web dashboard UI.
package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/gauravjain0377/ticket-system/store"
)

// HealthHandler handles system health and telemetry monitoring.
type HealthHandler struct {
	db        *store.DB
	startTime time.Time
	version   string
}

// NewHealthHandler creates an initialized health handler.
func NewHealthHandler(db *store.DB) *HealthHandler {
	return &HealthHandler{
		db:        db,
		startTime: time.Now(),
		version:   "1.0.0",
	}
}

// DatabaseHealth reports connection and entity metrics.
type DatabaseHealth struct {
	Status       string  `json:"status"`
	Driver       string  `json:"driver"`
	LatencyMs    float64 `json:"latency_ms"`
	TotalUsers   int     `json:"total_users"`
	TotalTickets int     `json:"total_tickets"`
	Error        string  `json:"error,omitempty"`
}

// SystemHealth reports runtime and memory information.
type SystemHealth struct {
	GoVersion     string  `json:"go_version"`
	OS            string  `json:"os"`
	Arch          string  `json:"arch"`
	NumCPU        int     `json:"num_cpu"`
	NumGoroutines int     `json:"num_goroutines"`
	MemoryAllocMB float64 `json:"memory_alloc_mb"`
	MemorySysMB   float64 `json:"memory_sys_mb"`
	NumGC         uint32  `json:"num_gc"`
}

// EndpointInfo documents active public and protected endpoints.
type EndpointInfo struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Auth        string `json:"auth"`
	Description string `json:"description"`
}

// HealthResponse is the complete telemetry payload.
type HealthResponse struct {
	Status        string          `json:"status"`
	Service       string          `json:"service"`
	Version       string          `json:"version"`
	Timestamp     string          `json:"timestamp"`
	Uptime        string          `json:"uptime"`
	UptimeSeconds int64           `json:"uptime_seconds"`
	Database      DatabaseHealth  `json:"database"`
	System        SystemHealth    `json:"system"`
	Endpoints     []EndpointInfo  `json:"endpoints"`
}

// ServeHTTP inspects headers and query params to serve HTML or JSON.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := h.collectData()

	// Check if JSON format requested
	format := r.URL.Query().Get("format")
	accept := r.Header.Get("Accept")
	userAgent := strings.ToLower(r.Header.Get("User-Agent"))

	isJSON := format == "json" ||
		r.URL.Path == "/health/json" ||
		strings.Contains(accept, "application/json") ||
		strings.HasPrefix(userAgent, "curl") ||
		strings.HasPrefix(userAgent, "postman") ||
		strings.HasPrefix(userAgent, "httpie")

	// If browser explicitly accepts text/html, prefer HTML unless format=json is set
	if strings.Contains(accept, "text/html") && format != "json" && r.URL.Path != "/health/json" {
		isJSON = false
	}

	statusCode := http.StatusOK
	if data.Status != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	if isJSON {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(statusCode)
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(data)
		return
	}

	// Render HTML dashboard
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = dashboardTmpl.Execute(w, data)
}

func (h *HealthHandler) collectData() HealthResponse {
	uptimeDuration := time.Since(h.startTime)
	uptimeStr := formatDuration(uptimeDuration)

	// Runtime metrics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	system := SystemHealth{
		GoVersion:     runtime.Version(),
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		NumCPU:        runtime.NumCPU(),
		NumGoroutines: runtime.NumGoroutine(),
		MemoryAllocMB: toMB(m.Alloc),
		MemorySysMB:   toMB(m.Sys),
		NumGC:         m.NumGC,
	}

	// Database metrics
	dbHealth := DatabaseHealth{
		Status: "disconnected",
		Driver: "sqlite (modernc.org/sqlite, pure-Go)",
	}

	isHealthy := true
	if h.db != nil {
		startPing := time.Now()
		if err := h.db.Ping(); err != nil {
			dbHealth.Status = "error"
			dbHealth.Error = err.Error()
			isHealthy = false
		} else {
			dbHealth.LatencyMs = float64(time.Since(startPing).Microseconds()) / 1000.0
			dbHealth.Status = "connected"

			users, tickets, err := h.db.GetCounts()
			if err != nil {
				dbHealth.Error = err.Error()
			} else {
				dbHealth.TotalUsers = users
				dbHealth.TotalTickets = tickets
			}
		}
	} else {
		dbHealth.Error = "database connection uninitialized"
		isHealthy = false
	}

	status := "healthy"
	if !isHealthy {
		status = "degraded"
	}

	endpoints := []EndpointInfo{
		{Method: "GET", Path: "/health", Auth: "Public", Description: "System health check & interactive dashboard"},
		{Method: "GET", Path: "/health/json", Auth: "Public", Description: "Raw system health telemetry JSON"},
		{Method: "POST", Path: "/auth/register", Auth: "Public", Description: "Register new user account"},
		{Method: "POST", Path: "/auth/login", Auth: "Public", Description: "Authenticate and retrieve JWT token"},
		{Method: "GET", Path: "/tickets", Auth: "Bearer JWT", Description: "List all tickets for authenticated user"},
		{Method: "POST", Path: "/tickets", Auth: "Bearer JWT", Description: "Create a new support ticket"},
		{Method: "GET", Path: "/tickets/{id}", Auth: "Bearer JWT", Description: "Fetch ticket details by ID"},
		{Method: "PATCH", Path: "/tickets/{id}/status", Auth: "Bearer JWT", Description: "Update ticket status (open -> in_progress -> closed)"},
	}

	return HealthResponse{
		Status:        status,
		Service:       "EvaBharat Ticket System API",
		Version:       h.version,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Uptime:        uptimeStr,
		UptimeSeconds: int64(uptimeDuration.Seconds()),
		Database:      dbHealth,
		System:        system,
		Endpoints:     endpoints,
	}
}

func toMB(b uint64) float64 {
	return float64(b) / 1024 / 1024
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

var dashboardTmpl = template.Must(template.New("dashboard").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>System Status | {{.Service}}</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&family=JetBrains+Mono:wght@400;500;600&display=swap" rel="stylesheet">
    <script src="https://cdn.tailwindcss.com"></script>
    <script>
      tailwind.config = {
        theme: {
          extend: {
            fontFamily: {
              sans: ['Inter', 'sans-serif'],
              mono: ['JetBrains Mono', 'monospace'],
            },
          }
        }
      }
    </script>
    <style>
      @keyframes pulseGlow {
        0%, 100% { opacity: 1; transform: scale(1); }
        50% { opacity: 0.5; transform: scale(1.15); }
      }
      .pulse-indicator {
        animation: pulseGlow 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
      }
    </style>
</head>
<body class="bg-gray-950 text-gray-100 font-sans min-h-screen antialiased selection:bg-indigo-500 selection:text-white">

    <!-- Header Navigation -->
    <header class="border-b border-gray-800/80 bg-gray-900/60 backdrop-blur-md sticky top-0 z-40">
        <div class="max-w-6xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between">
            <div class="flex items-center gap-3">
                <span class="text-2xl">🎫</span>
                <div>
                    <h1 class="font-semibold text-base tracking-tight leading-none text-white">{{.Service}}</h1>
                    <span class="text-xs text-gray-400 font-mono">v{{.Version}} &bull; Live Status</span>
                </div>
            </div>

            <div class="flex items-center gap-2 sm:gap-3">
                <a href="/" class="text-xs sm:text-sm px-3 py-1.5 rounded-lg border border-gray-700 bg-gray-800/60 hover:bg-gray-800 text-gray-200 transition flex items-center gap-1.5">
                    <span>&larr;</span> Back to App
                </a>
                <a href="/health?format=json" target="_blank" class="text-xs sm:text-sm px-3 py-1.5 rounded-lg border border-indigo-500/40 bg-indigo-950/40 hover:bg-indigo-900/50 text-indigo-300 font-mono transition flex items-center gap-1.5">
                    <span>{ }</span> Raw JSON
                </a>
                <button onclick="window.location.reload()" class="text-xs sm:text-sm px-3 py-1.5 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white font-medium transition flex items-center gap-1.5 shadow-sm shadow-indigo-500/20">
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path></svg>
                    <span>Refresh</span>
                </button>
            </div>
        </div>
    </header>

    <main class="max-w-6xl mx-auto px-4 sm:px-6 py-8 space-y-8">

        <!-- Status Hero Banner -->
        <div class="relative overflow-hidden rounded-2xl border {{if eq .Status "healthy"}}border-emerald-500/30 bg-gradient-to-r from-emerald-950/40 via-gray-900 to-gray-900{{else}}border-amber-500/30 bg-gradient-to-r from-amber-950/40 via-gray-900 to-gray-900{{end}} p-6 sm:p-8">
            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                <div class="flex items-start sm:items-center gap-4">
                    <div class="relative flex h-5 w-5 mt-1 sm:mt-0">
                        {{if eq .Status "healthy"}}
                        <span class="pulse-indicator absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                        <span class="relative inline-flex rounded-full h-5 w-5 bg-emerald-500"></span>
                        {{else}}
                        <span class="pulse-indicator absolute inline-flex h-full w-full rounded-full bg-amber-400 opacity-75"></span>
                        <span class="relative inline-flex rounded-full h-5 w-5 bg-amber-500"></span>
                        {{end}}
                    </div>
                    <div>
                        <div class="flex items-center gap-2">
                            <h2 class="text-xl sm:text-2xl font-bold tracking-tight text-white">
                                {{if eq .Status "healthy"}}All Systems Operational{{else}}System Status Degraded{{end}}
                            </h2>
                            <span class="uppercase text-[11px] font-semibold px-2 py-0.5 rounded-full {{if eq .Status "healthy"}}bg-emerald-950 text-emerald-300 border border-emerald-800/80{{else}}bg-amber-950 text-amber-300 border border-amber-800/80{{end}}">
                                {{.Status}}
                            </span>
                        </div>
                        <p class="text-xs sm:text-sm text-gray-400 mt-1">
                            Verified at <span class="font-mono text-gray-300">{{.Timestamp}}</span> &bull; Up for <span class="text-white font-medium">{{.Uptime}}</span>
                        </p>
                    </div>
                </div>

                <div class="flex items-center gap-2 text-xs font-mono text-gray-400 bg-gray-950/60 px-3 py-2 rounded-lg border border-gray-800 self-start sm:self-auto">
                    <span class="w-2 h-2 rounded-full bg-emerald-500"></span>
                    <span>HTTP 200 OK</span>
                    <span class="text-gray-600">&bull;</span>
                    <span>{{.Database.LatencyMs}}ms DB ping</span>
                </div>
            </div>
        </div>

        <!-- Telemetry Cards Grid -->
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            <!-- Card 1: Database -->
            <div class="bg-gray-900/90 border border-gray-800 rounded-xl p-5 hover:border-gray-700 transition">
                <div class="flex items-center justify-between text-xs text-gray-400 mb-2">
                    <span class="font-medium uppercase tracking-wider">Database</span>
                    <span class="text-emerald-400 font-mono text-xs flex items-center gap-1">
                        <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                        {{.Database.Status}}
                    </span>
                </div>
                <div class="text-2xl font-bold text-white mb-1">{{.Database.LatencyMs}} <span class="text-xs font-normal text-gray-400 font-mono">ms latency</span></div>
                <div class="text-xs text-gray-400 space-y-0.5 font-mono">
                    <div class="truncate">Driver: modernc.org/sqlite</div>
                    <div>Pure Go (CGO-free)</div>
                </div>
            </div>

            <!-- Card 2: Entity Records -->
            <div class="bg-gray-900/90 border border-gray-800 rounded-xl p-5 hover:border-gray-700 transition">
                <div class="flex items-center justify-between text-xs text-gray-400 mb-2">
                    <span class="font-medium uppercase tracking-wider">Stored Data</span>
                    <span class="text-indigo-400 font-mono text-xs">Persistent DB</span>
                </div>
                <div class="flex items-baseline gap-4 mb-1">
                    <div>
                        <div class="text-2xl font-bold text-white">{{.Database.TotalTickets}}</div>
                        <div class="text-xs text-gray-400">Tickets</div>
                    </div>
                    <div class="text-gray-700 font-light text-2xl">/</div>
                    <div>
                        <div class="text-2xl font-bold text-indigo-400">{{.Database.TotalUsers}}</div>
                        <div class="text-xs text-gray-400">Users</div>
                    </div>
                </div>
                <div class="text-xs text-gray-400 font-mono">Single-file SQLite storage</div>
            </div>

            <!-- Card 3: Go Runtime & Concurrency -->
            <div class="bg-gray-900/90 border border-gray-800 rounded-xl p-5 hover:border-gray-700 transition">
                <div class="flex items-center justify-between text-xs text-gray-400 mb-2">
                    <span class="font-medium uppercase tracking-wider">Runtime & CPU</span>
                    <span class="text-cyan-400 font-mono text-xs">{{.System.GoVersion}}</span>
                </div>
                <div class="text-2xl font-bold text-white mb-1">{{.System.NumGoroutines}} <span class="text-xs font-normal text-gray-400 font-mono">goroutines</span></div>
                <div class="text-xs text-gray-400 space-y-0.5 font-mono">
                    <div>CPU Cores: {{.System.NumCPU}}</div>
                    <div>Platform: {{.System.OS}}/{{.System.Arch}}</div>
                </div>
            </div>

            <!-- Card 4: Memory & GC -->
            <div class="bg-gray-900/90 border border-gray-800 rounded-xl p-5 hover:border-gray-700 transition">
                <div class="flex items-center justify-between text-xs text-gray-400 mb-2">
                    <span class="font-medium uppercase tracking-wider">Memory Allocation</span>
                    <span class="text-purple-400 font-mono text-xs">GC: {{.System.NumGC}}</span>
                </div>
                <div class="text-2xl font-bold text-white mb-1">{{printf "%.2f" .System.MemoryAllocMB}} <span class="text-xs font-normal text-gray-400 font-mono">MB heap</span></div>
                <div class="text-xs text-gray-400 space-y-0.5 font-mono">
                    <div>System Mem: {{printf "%.2f" .System.MemorySysMB}} MB</div>
                    <div>High performance runtime</div>
                </div>
            </div>
        </div>

        <!-- API Endpoints Catalogue -->
        <div class="bg-gray-900/80 border border-gray-800 rounded-xl overflow-hidden shadow-sm">
            <div class="px-6 py-4 border-b border-gray-800 flex items-center justify-between">
                <div>
                    <h3 class="font-semibold text-white">Registered API Endpoints</h3>
                    <p class="text-xs text-gray-400 mt-0.5">Live routing table handled by Gorilla Mux</p>
                </div>
                <span class="text-xs font-mono bg-gray-800 text-gray-300 px-2.5 py-1 rounded-md">{{len .Endpoints}} Routes</span>
            </div>
            <div class="overflow-x-auto">
                <table class="w-full text-left text-xs sm:text-sm">
                    <thead class="bg-gray-950/60 text-gray-400 text-xs uppercase tracking-wider font-mono border-b border-gray-800">
                        <tr>
                            <th class="px-6 py-3">Method</th>
                            <th class="px-6 py-3">Endpoint Path</th>
                            <th class="px-6 py-3">Authorization</th>
                            <th class="px-6 py-3">Description</th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-gray-800 font-mono text-xs">
                        {{range .Endpoints}}
                        <tr class="hover:bg-gray-800/40 transition">
                            <td class="px-6 py-3.5">
                                {{if eq .Method "GET"}}<span class="px-2 py-0.5 rounded bg-blue-950 text-blue-400 border border-blue-800/60 font-semibold">{{.Method}}</span>{{end}}
                                {{if eq .Method "POST"}}<span class="px-2 py-0.5 rounded bg-emerald-950 text-emerald-400 border border-emerald-800/60 font-semibold">{{.Method}}</span>{{end}}
                                {{if eq .Method "PATCH"}}<span class="px-2 py-0.5 rounded bg-amber-950 text-amber-400 border border-amber-800/60 font-semibold">{{.Method}}</span>{{end}}
                            </td>
                            <td class="px-6 py-3.5 font-medium text-white">{{.Path}}</td>
                            <td class="px-6 py-3.5">
                                {{if eq .Auth "Public"}}
                                <span class="text-gray-400 bg-gray-800/80 px-2 py-0.5 rounded text-[11px]">{{.Auth}}</span>
                                {{else}}
                                <span class="text-indigo-300 bg-indigo-950/80 border border-indigo-800/60 px-2 py-0.5 rounded text-[11px]">{{.Auth}}</span>
                                {{end}}
                            </td>
                            <td class="px-6 py-3.5 text-gray-300 font-sans text-xs sm:text-sm">{{.Description}}</td>
                        </tr>
                        {{end}}
                    </tbody>
                </table>
            </div>
        </div>

        <!-- Live JSON Viewer / Quick Tool -->
        <div class="bg-gray-900/80 border border-gray-800 rounded-xl overflow-hidden shadow-sm">
            <div class="px-6 py-4 border-b border-gray-800 flex items-center justify-between">
                <div class="flex items-center gap-2">
                    <span class="text-sm font-mono text-indigo-400">{ }</span>
                    <h3 class="font-semibold text-white">Live Diagnostic Telemetry (JSON)</h3>
                </div>
                <button id="copy-btn" onclick="copyJsonPayload()" class="text-xs px-3 py-1 rounded bg-gray-800 hover:bg-gray-700 text-gray-200 border border-gray-700 transition flex items-center gap-1.5">
                    <span id="copy-icon">📋</span>
                    <span id="copy-text">Copy JSON</span>
                </button>
            </div>
            <div class="p-4 bg-gray-950/90 overflow-x-auto">
                <pre id="json-block" class="font-mono text-xs text-indigo-200/90 leading-relaxed"></pre>
            </div>
        </div>

    </main>

    <!-- Footer -->
    <footer class="border-t border-gray-800/80 mt-12 py-6 text-center text-xs text-gray-500 font-mono">
        <p>{{.Service}} &bull; Monitored with pure Go runtime metrics &bull; SQLite Database</p>
    </footer>

    <script>
        // Fetch current live JSON for display and copying
        const healthData = {
            status: "{{.Status}}",
            service: "{{.Service}}",
            version: "{{.Version}}",
            timestamp: "{{.Timestamp}}",
            uptime: "{{.Uptime}}",
            uptime_seconds: {{.UptimeSeconds}},
            database: {
                status: "{{.Database.Status}}",
                driver: "{{.Database.Driver}}",
                latency_ms: {{.Database.LatencyMs}},
                total_users: {{.Database.TotalUsers}},
                total_tickets: {{.Database.TotalTickets}}
            },
            system: {
                go_version: "{{.System.GoVersion}}",
                os: "{{.System.OS}}",
                arch: "{{.System.Arch}}",
                num_cpu: {{.System.NumCPU}},
                num_goroutines: {{.System.NumGoroutines}},
                memory_alloc_mb: {{.System.MemoryAllocMB}},
                memory_sys_mb: {{.System.MemorySysMB}},
                num_gc: {{.System.NumGC}}
            }
        };

        const jsonFormatted = JSON.stringify(healthData, null, 2);
        document.getElementById('json-block').textContent = jsonFormatted;

        function copyJsonPayload() {
            navigator.clipboard.writeText(jsonFormatted).then(() => {
                const text = document.getElementById('copy-text');
                const icon = document.getElementById('copy-icon');
                text.textContent = 'Copied!';
                icon.textContent = '✅';
                setTimeout(() => {
                    text.textContent = 'Copy JSON';
                    icon.textContent = '📋';
                }, 2000);
            });
        }
    </script>
</body>
</html>
`))

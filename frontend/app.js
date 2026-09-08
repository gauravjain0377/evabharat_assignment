// Ticket System - Frontend Logic
// Handles auth, ticket CRUD, and UI updates.

const API = window.location.origin;

// --- State ---
let token = localStorage.getItem("token") || "";
let username = localStorage.getItem("username") || "";

// --- DOM Elements ---
const $ = (id) => document.getElementById(id);

// --- Init ---
window.onload = () => {
    if (token) {
        showDashboard();
    }
    setupEventListeners();
};

function setupEventListeners() {
    $("login-form").onsubmit = handleLogin;
    $("register-form").onsubmit = handleRegister;
    $("show-register").onclick = (e) => { e.preventDefault(); toggleAuthForm("register"); };
    $("show-login").onclick = (e) => { e.preventDefault(); toggleAuthForm("login"); };
    $("logout-btn").onclick = handleLogout;
    $("create-ticket-btn").onclick = () => $("create-modal").classList.remove("hidden");
    $("create-ticket-form").onsubmit = handleCreateTicket;
}

// --- Auth ---
function toggleAuthForm(form) {
    $("login-form").classList.toggle("hidden", form !== "login");
    $("register-form").classList.toggle("hidden", form !== "register");
    $("auth-error").textContent = "";
}

async function handleRegister(e) {
    e.preventDefault();
    const body = {
        username: $("reg-username").value.trim(),
        email: $("reg-email").value.trim(),
        password: $("reg-password").value,
    };

    const res = await api("POST", "/auth/register", body);
    if (res.error) {
        $("auth-error").textContent = res.error;
        return;
    }
    toast("Account created! Please sign in.", "success");
    toggleAuthForm("login");
}

async function handleLogin(e) {
    e.preventDefault();
    const body = {
        username: $("login-username").value.trim(),
        password: $("login-password").value,
    };

    const res = await api("POST", "/auth/login", body);
    if (res.error) {
        $("auth-error").textContent = res.error;
        return;
    }

    token = res.token;
    username = body.username;
    localStorage.setItem("token", token);
    localStorage.setItem("username", username);
    showDashboard();
}

function handleLogout() {
    token = "";
    username = "";
    localStorage.removeItem("token");
    localStorage.removeItem("username");
    $("auth-screen").classList.remove("hidden");
    $("dashboard-screen").classList.add("hidden");
    $("login-form").reset();
    toggleAuthForm("login");
}

// --- Dashboard ---
function showDashboard() {
    $("auth-screen").classList.add("hidden");
    $("dashboard-screen").classList.remove("hidden");
    $("nav-username").textContent = username;
    loadTickets();
}

async function loadTickets() {
    const tickets = await api("GET", "/tickets");
    if (tickets.error) {
        toast(tickets.error, "error");
        if (tickets.status === 401) handleLogout();
        return;
    }

    // Update stats
    $("stat-total").textContent = tickets.length;
    $("stat-open").textContent = tickets.filter(t => t.status === "open").length;
    $("stat-progress").textContent = tickets.filter(t => t.status === "in_progress").length;
    $("stat-closed").textContent = tickets.filter(t => t.status === "closed").length;

    // Render tickets
    const list = $("tickets-list");
    const empty = $("empty-state");

    if (tickets.length === 0) {
        list.innerHTML = "";
        empty.classList.remove("hidden");
        return;
    }

    empty.classList.add("hidden");
    list.innerHTML = tickets.map(t => `
        <div onclick="viewTicket(${t.id})" class="flex items-center justify-between p-4 bg-gray-900 border border-gray-800 rounded-lg cursor-pointer hover:border-indigo-500 transition">
            <div class="min-w-0 flex-1">
                <div class="font-medium text-sm truncate">${esc(t.title)}</div>
                <div class="text-xs text-gray-500 mt-1">#${t.id} · ${timeAgo(t.created_at)}</div>
            </div>
            <span class="status-badge status-${t.status} ml-3">${t.status.replace("_", " ")}</span>
        </div>
    `).join("");
}

// --- Create Ticket ---
async function handleCreateTicket(e) {
    e.preventDefault();
    const body = {
        title: $("ticket-title").value.trim(),
        description: $("ticket-description").value.trim(),
    };

    if (!body.title) return;

    const res = await api("POST", "/tickets", body);
    if (res.error) {
        toast(res.error, "error");
        return;
    }

    toast("Ticket created!", "success");
    closeCreateModal();
    $("create-ticket-form").reset();
    loadTickets();
}

function closeCreateModal() {
    $("create-modal").classList.add("hidden");
}

// --- View Ticket ---
async function viewTicket(id) {
    const ticket = await api("GET", `/tickets/${id}`);
    if (ticket.error) {
        toast(ticket.error, "error");
        return;
    }

    let actions = "";
    if (ticket.status === "open") {
        actions = `<button onclick="updateStatus(${id}, 'in_progress')" class="px-3 py-1.5 bg-yellow-600 hover:bg-yellow-500 rounded-lg text-xs font-medium transition">Move to In Progress</button>`;
    } else if (ticket.status === "in_progress") {
        actions = `<button onclick="updateStatus(${id}, 'closed')" class="px-3 py-1.5 bg-green-600 hover:bg-green-500 rounded-lg text-xs font-medium transition">Close Ticket</button>`;
    }

    $("detail-body").innerHTML = `
        <div class="space-y-3">
            <div>
                <div class="text-[10px] uppercase tracking-wider text-gray-500 mb-1">Title</div>
                <div class="text-sm">${esc(ticket.title)}</div>
            </div>
            <div>
                <div class="text-[10px] uppercase tracking-wider text-gray-500 mb-1">Description</div>
                <div class="text-sm text-gray-300">${esc(ticket.description) || "No description"}</div>
            </div>
            <div class="flex gap-6">
                <div>
                    <div class="text-[10px] uppercase tracking-wider text-gray-500 mb-1">Status</div>
                    <span class="status-badge status-${ticket.status}">${ticket.status.replace("_", " ")}</span>
                </div>
                <div>
                    <div class="text-[10px] uppercase tracking-wider text-gray-500 mb-1">Created</div>
                    <div class="text-sm">${new Date(ticket.created_at).toLocaleDateString()}</div>
                </div>
            </div>
            ${actions ? `<div class="pt-3 border-t border-gray-800">${actions}</div>` : ""}
        </div>
    `;
    $("detail-modal").classList.remove("hidden");
}

function closeDetailModal() {
    $("detail-modal").classList.add("hidden");
}

// --- Update Status ---
async function updateStatus(id, status) {
    const res = await api("PATCH", `/tickets/${id}/status`, { status });
    if (res.error) {
        toast(res.error, "error");
        return;
    }
    toast("Status updated!", "success");
    closeDetailModal();
    loadTickets();
}

// --- API Helper ---
async function api(method, path, body) {
    const opts = {
        method,
        headers: { "Content-Type": "application/json" },
    };
    if (token) opts.headers["Authorization"] = `Bearer ${token}`;
    if (body) opts.body = JSON.stringify(body);

    try {
        const res = await fetch(API + path, opts);
        const data = await res.json();
        if (!res.ok) return { error: data.error || "Something went wrong", status: res.status };
        return data;
    } catch (err) {
        return { error: "Network error" };
    }
}

// --- Toast ---
function toast(message, type) {
    const el = $("toast");
    el.textContent = message;
    el.className = `fixed bottom-6 right-6 px-4 py-3 rounded-lg text-sm font-medium z-50 shadow-lg ${type === "success" ? "bg-green-600" : "bg-red-600"} text-white`;
    setTimeout(() => el.className += " hidden", 3000);
}

// --- Helpers ---
function esc(str) {
    const div = document.createElement("div");
    div.textContent = str || "";
    return div.innerHTML;
}

function timeAgo(date) {
    const seconds = Math.floor((Date.now() - new Date(date)) / 1000);
    if (seconds < 60) return "just now";
    if (seconds < 3600) return Math.floor(seconds / 60) + "m ago";
    if (seconds < 86400) return Math.floor(seconds / 3600) + "h ago";
    return Math.floor(seconds / 86400) + "d ago";
}

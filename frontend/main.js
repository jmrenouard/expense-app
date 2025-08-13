// main.js implements a minimal client-side interface to the expense application.

const API_BASE = '/api';
const appDiv = document.getElementById('app');

// Entry point: render either login or dashboard depending on token presence.
function init() {
    const token = localStorage.getItem('token');
    if (!token) {
        renderLogin();
    } else {
        renderDashboard();
    }
}

// Render login form
function renderLogin() {
    appDiv.innerHTML = `
        <div class="bg-white shadow-md rounded px-8 pt-6 pb-8 mb-4">
            <h2 class="text-2xl font-bold mb-4">Connexion</h2>
            <div class="mb-4">
                <label class="block text-gray-700 text-sm font-bold mb-2" for="email">Email</label>
                <input id="email" type="email" class="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700" required />
            </div>
            <div class="mb-4">
                <label class="block text-gray-700 text-sm font-bold mb-2" for="password">Mot de passe</label>
                <input id="password" type="password" class="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700" required />
            </div>
            <button id="loginBtn" class="bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded">Se connecter</button>
            <p id="loginError" class="text-red-500 mt-2"></p>
        </div>
    `;
    document.getElementById('loginBtn').addEventListener('click', handleLogin);
}

// Handle login request
async function handleLogin() {
    const email = document.getElementById('email').value;
    const password = document.getElementById('password').value;
    try {
        const res = await fetch(`${API_BASE}/auth/login`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password })
        });
        if (!res.ok) {
            const err = await res.json();
            throw new Error(err.error || 'Erreur de connexion');
        }
        const data = await res.json();
        localStorage.setItem('token', data.token);
        renderDashboard();
    } catch (e) {
        document.getElementById('loginError').textContent = e.message;
    }
}

// Render dashboard with report list and creation form
function renderDashboard() {
    appDiv.innerHTML = `
        <div class="mb-6">
            <h2 class="text-2xl font-bold">Mes notes de frais</h2>
            <div id="reportsList" class="mt-4"></div>
        </div>
        <div class="bg-white shadow-md rounded px-8 pt-6 pb-8 mb-4">
            <h3 class="text-xl font-bold mb-2">Créer une nouvelle note de frais</h3>
            <input id="reportTitle" type="text" placeholder="Titre" class="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700 mb-3" />
            <button id="createReportBtn" class="bg-green-500 hover:bg-green-700 text-white font-bold py-2 px-4 rounded">Créer</button>
            <p id="reportError" class="text-red-500 mt-2"></p>
        </div>
    `;
    document.getElementById('createReportBtn').addEventListener('click', createReport);
    listReports();
}

// Fetch and render reports
async function listReports() {
    const token = localStorage.getItem('token');
    const res = await fetch(`${API_BASE}/reports`, {
        headers: { 'Authorization': `Bearer ${token}` }
    });
    if (!res.ok) {
        return;
    }
    const reports = await res.json();
    const container = document.getElementById('reportsList');
    if (reports.length === 0) {
        container.innerHTML = '<p class="text-gray-600">Aucune note de frais pour le moment.</p>';
        return;
    }
    container.innerHTML = '';
    reports.forEach(r => {
        const reportDiv = document.createElement('div');
        reportDiv.className = 'bg-white shadow rounded p-4 mb-4';
        // Build items HTML
        let itemsHtml = '';
        if (r.items && r.items.length > 0) {
            r.items.forEach(item => {
                itemsHtml += `<div class="border-t pt-2"><p>${item.description} – ${item.amount_ttc.toFixed(2)} €</p></div>`;
            });
        }
        reportDiv.innerHTML = `
            <h4 class="font-bold text-lg">${r.title}</h4>
            <p>Status : <span class="font-semibold">${r.status}</span></p>
            <div id="items-${r.id}" class="mt-2">${itemsHtml}</div>
            ${r.status === 'draft' ? `<button class="mt-2 bg-blue-500 hover:bg-blue-700 text-white py-1 px-2 rounded" onclick="showAddItemForm(${r.id})">Ajouter une dépense</button>
            <button class="ml-2 mt-2 bg-purple-500 hover:bg-purple-700 text-white py-1 px-2 rounded" onclick="submitReport(${r.id})">Soumettre</button>` : ''}
        `;
        container.appendChild(reportDiv);
    });
}

// Fetch items for a report and render them
async function listItems(reportId) {
    // We'll call export JSON of items? Not necessary. Instead, call API to fetch via SQL.
    // For simplicity, we use /reports? We do not have items list endpoint. We'll call export JSON and filter.
    const token = localStorage.getItem('token');
    const res = await fetch(`${API_BASE}/reports`, {
        headers: { 'Authorization': `Bearer ${token}` }
    });
    if (!res.ok) return;
    const reports = await res.json();
    const report = reports.find(r => r.id === reportId);
    const itemsContainer = document.getElementById(`items-${reportId}`);
    if (!report || !report.items) {
        itemsContainer.innerHTML = '';
        return;
    }
    report.items.forEach(item => {
        const div = document.createElement('div');
        div.className = 'border-t pt-2';
        div.innerHTML = `<p>${item.description} – ${item.amount_ttc} €</p>`;
        itemsContainer.appendChild(div);
    });
}

// Create a new report
async function createReport() {
    const title = document.getElementById('reportTitle').value;
    const token = localStorage.getItem('token');
    try {
        const res = await fetch(`${API_BASE}/reports`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
            body: JSON.stringify({ title })
        });
        if (!res.ok) {
            const err = await res.json();
            throw new Error(err.error || 'Erreur');
        }
        document.getElementById('reportTitle').value = '';
        listReports();
    } catch (e) {
        document.getElementById('reportError').textContent = e.message;
    }
}

// Display form to add item to report
function showAddItemForm(reportId) {
    const itemsContainer = document.getElementById(`items-${reportId}`);
    const formDiv = document.createElement('div');
    formDiv.className = 'mt-2 p-2 border rounded';
    formDiv.innerHTML = `
        <input type="text" id="desc-${reportId}" placeholder="Description" class="border p-1 mr-2" />
        <input type="date" id="date-${reportId}" class="border p-1 mr-2" />
        <input type="number" step="0.01" id="ht-${reportId}" placeholder="Montant HT" class="border p-1 mr-2" />
        <input type="number" step="0.01" id="vat-${reportId}" placeholder="TVA (ex: 0.2)" class="border p-1 mr-2" />
        <button class="bg-green-500 hover:bg-green-700 text-white py-1 px-2 rounded" onclick="addItem(${reportId})">Ajouter</button>
    `;
    itemsContainer.appendChild(formDiv);
}

// Add item to report
async function addItem(reportId) {
    const desc = document.getElementById(`desc-${reportId}`).value;
    const date = document.getElementById(`date-${reportId}`).value;
    const ht = parseFloat(document.getElementById(`ht-${reportId}`).value);
    const vat = parseFloat(document.getElementById(`vat-${reportId}`).value);
    const token = localStorage.getItem('token');
    try {
        const res = await fetch(`${API_BASE}/reports/${reportId}/items`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
            body: JSON.stringify({ description: desc, expense_date: date, amount_ht: ht, vat_rate: vat })
        });
        if (!res.ok) {
            const err = await res.json();
            throw new Error(err.error || 'Erreur');
        }
        listReports();
    } catch (e) {
        alert(e.message);
    }
}

// Submit report
async function submitReport(reportId) {
    const token = localStorage.getItem('token');
    try {
        const res = await fetch(`${API_BASE}/reports/${reportId}/submit`, {
            method: 'POST',
            headers: { 'Authorization': `Bearer ${token}` }
        });
        if (!res.ok) {
            const err = await res.json();
            throw new Error(err.error || 'Erreur');
        }
        listReports();
    } catch (e) {
        alert(e.message);
    }
}

// On load
window.addEventListener('load', init);
const API_BASE = '/api';
const appDiv = document.getElementById('app');
let currentLang = 'fr';

const translations = {
    fr: {
        login: "Connexion",
        email: "Email",
        password: "Mot de passe",
        loginButton: "Se connecter",
        loginError: "Erreur de connexion",
        myExpenseReports: "Mes notes de frais",
        noExpenseReports: "Aucune note de frais pour le moment.",
        createExpenseReport: "Créer une nouvelle note de frais",
        title: "Titre",
        createButton: "Créer",
        status: "Status",
        addExpense: "Ajouter une dépense",
        submit: "Soumettre",
        description: "Description",
        amountHT: "Montant HT",
        addButton: "Ajouter",
        error: "Erreur"
    },
    en: {
        login: "Login",
        email: "Email",
        password: "Password",
        loginButton: "Login",
        loginError: "Login error",
        myExpenseReports: "My Expense Reports",
        noExpenseReports: "No expense reports yet.",
        createExpenseReport: "Create a new expense report",
        title: "Title",
        createButton: "Create",
        status: "Status",
        addExpense: "Add an expense",
        submit: "Submit",
        description: "Description",
        amountHT: "Amount (tax excl.)",
        addButton: "Add",
        error: "Error"
    }
};

function getMsg(key) {
    return translations[currentLang][key] || key;
}

function setLang(lang) {
    currentLang = lang;
    init();
}

// Entry point: render either login or dashboard depending on token presence.
function init() {
    document.getElementById('langSelector').value = currentLang;
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
            <h2 class="text-2xl font-bold mb-4">${getMsg('login')}</h2>
            <div class="mb-4">
                <label class="block text-gray-700 text-sm font-bold mb-2" for="email">${getMsg('email')}</label>
                <input id="email" type="email" class="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700" required />
            </div>
            <div class="mb-4">
                <label class="block text-gray-700 text-sm font-bold mb-2" for="password">${getMsg('password')}</label>
                <input id="password" type="password" class="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700" required />
            </div>
            <button id="loginBtn" class="bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded">${getMsg('loginButton')}</button>
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
            headers: { 'Content-Type': 'application/json', 'Accept-Language': currentLang },
            body: JSON.stringify({ email, password })
        });
        if (!res.ok) {
            const err = await res.json();
            throw new Error(err.error || getMsg('loginError'));
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
            <h2 class="text-2xl font-bold">${getMsg('myExpenseReports')}</h2>
            <div id="reportsList" class="mt-4"></div>
        </div>
        <div class="bg-white shadow-md rounded px-8 pt-6 pb-8 mb-4">
            <h3 class="text-xl font-bold mb-2">${getMsg('createExpenseReport')}</h3>
            <input id="reportTitle" type="text" placeholder="${getMsg('title')}" class="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700 mb-3" />
            <button id="createReportBtn" class="bg-green-500 hover:bg-green-700 text-white font-bold py-2 px-4 rounded">${getMsg('createButton')}</button>
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
        headers: { 'Authorization': `Bearer ${token}`, 'Accept-Language': currentLang }
    });
    if (!res.ok) {
        return;
    }
    const reports = await res.json();
    const container = document.getElementById('reportsList');
    if (reports.length === 0) {
        container.innerHTML = `<p class="text-gray-600">${getMsg('noExpenseReports')}</p>`;
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
            <p>${getMsg('status')} : <span class="font-semibold">${r.status}</span></p>
            <div id="items-${r.id}" class="mt-2">${itemsHtml}</div>
            ${r.status === 'draft' ? `<button class="mt-2 bg-blue-500 hover:bg-blue-700 text-white py-1 px-2 rounded" onclick="showAddItemForm(${r.id})">${getMsg('addExpense')}</button>
            <button class="ml-2 mt-2 bg-purple-500 hover:bg-purple-700 text-white py-1 px-2 rounded" onclick="submitReport(${r.id})">${getMsg('submit')}</button>` : ''}
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
        headers: { 'Authorization': `Bearer ${token}`, 'Accept-Language': currentLang }
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
            headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}`, 'Accept-Language': currentLang },
            body: JSON.stringify({ title })
        });
        if (!res.ok) {
            const err = await res.json();
            throw new Error(err.error || getMsg('error'));
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
        <input type="text" id="desc-${reportId}" placeholder="${getMsg('description')}" class="border p-1 mr-2" />
        <input type="date" id="date-${reportId}" class="border p-1 mr-2" />
        <input type="number" step="0.01" id="ht-${reportId}" placeholder="${getMsg('amountHT')}" class="border p-1 mr-2" />
        <button class="bg-green-500 hover:bg-green-700 text-white py-1 px-2 rounded" onclick="addItem(${reportId})">${getMsg('addButton')}</button>
    `;
    itemsContainer.appendChild(formDiv);
}

// Add item to report
async function addItem(reportId) {
    const desc = document.getElementById(`desc-${reportId}`).value;
    const date = document.getElementById(`date-${reportId}`).value;
    const ht = parseFloat(document.getElementById(`ht-${reportId}`).value);
    const token = localStorage.getItem('token');
    try {
        const res = await fetch(`${API_BASE}/reports/${reportId}/items`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}`, 'Accept-Language': currentLang },
            body: JSON.stringify({ description: desc, expense_date: date, amount_ht: ht })
        });
        if (!res.ok) {
            const err = await res.json();
            throw new Error(err.error || getMsg('error'));
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
            headers: { 'Authorization': `Bearer ${token}`, 'Accept-Language': currentLang }
        });
        if (!res.ok) {
            const err = await res.json();
            throw new Error(err.error || getMsg('error'));
        }
        listReports();
    } catch (e) {
        alert(e.message);
    }
}

// On load
window.addEventListener('load', init);
document.getElementById('langSelector').addEventListener('change', (e) => setLang(e.target.value));
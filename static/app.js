let weightChart = null;
let statusWeightChart = null;
let milkTransferChart = null;
let allLogs = [];

document.addEventListener("DOMContentLoaded", () => {
    loadDashboard();
    loadVitamins();

    document.getElementById("refreshButton").addEventListener("click", loadDashboard);
    document.getElementById("searchInput").addEventListener("input", event => {
        renderLogsTable(event.target.value);
    });
    document.getElementById("openFormButton").addEventListener("click", openForm);
    document.getElementById("closeFormButton").addEventListener("click", closeForm);
    document.getElementById("cancelFormButton").addEventListener("click", closeForm);
    document.getElementById("formOverlay").addEventListener("click", event => {
        if (event.target === document.getElementById("formOverlay")) closeForm();
    });
    document.getElementById("entryForm").addEventListener("submit", submitEntry);

    document.getElementById("closeEditButton").addEventListener("click", closeEditForm);
    document.getElementById("cancelEditButton").addEventListener("click", closeEditForm);
    document.getElementById("editOverlay").addEventListener("click", event => {
        if (event.target === document.getElementById("editOverlay")) closeEditForm();
    });
    document.getElementById("editForm").addEventListener("submit", submitEdit);

    const today = new Date().toISOString().split("T")[0];
    document.getElementById("logDate").value = today;

    document.getElementById("dVitaminCheck").addEventListener("change", e => {
        const date = new Date().toISOString().slice(0, 10);
        saveVitamin("d-vitamin", e.target.checked, date);
        if (e.target.checked) e.target.disabled = true;
    });

    document.getElementById("kVitaminCheck").addEventListener("change", e => {
        const today = new Date();
        const date = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, "0")}`;
        saveVitamin("k-vitamin", e.target.checked, date);
        if (e.target.checked) e.target.disabled = true;
    });

    document.querySelectorAll(".quick-tag").forEach(cb => {
        cb.addEventListener("change", syncQuickTags);
    });
});

function syncQuickTags() {
    const checked = [...document.querySelectorAll(".quick-tag:checked")].map(cb => cb.value);
    document.getElementById("dailySummary").value = checked.join(", ");
}

async function loadVitamins() {
    const today = new Date();
    const todayStr = today.toISOString().slice(0, 10);
    const monthStr = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, "0")}`;

    const data = await fetch("/api/vitamins").then(r => r.json()).catch(() => ({}));

    const dEl = document.getElementById("dVitaminCheck");
    const d = data["d-vitamin"];
    const dCheckedToday = d && d.date === todayStr && d.checked === true;
    dEl.checked = dCheckedToday;
    dEl.disabled = dCheckedToday;

    const kEl = document.getElementById("kVitaminCheck");
    const k = data["k-vitamin"];
    const kCheckedThisMonth = k && k.date === monthStr && k.checked === true;
    kEl.checked = kCheckedThisMonth;
    kEl.disabled = kCheckedThisMonth;
}

async function saveVitamin(key, checked, date) {
    await fetch(`/api/vitamins/${key}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ checked, date }),
    });
}

function openForm() {
    document.getElementById("formOverlay").classList.remove("hidden");
    document.body.style.overflow = "hidden";
    document.getElementById("formError").classList.add("hidden");
    document.getElementById("formSuccess").classList.add("hidden");

    const now = new Date();
    const hh = String(now.getHours()).padStart(2, "0");
    const mm = String(now.getMinutes()).padStart(2, "0");
    document.getElementById("logTime").value = `${hh}:${mm}`;
}

function closeForm() {
    document.getElementById("formOverlay").classList.add("hidden");
    document.body.style.overflow = "";
    document.getElementById("entryForm").reset();
    document.getElementById("formError").classList.add("hidden");
    document.getElementById("formSuccess").classList.add("hidden");
    const today = new Date().toISOString().split("T")[0];
    document.getElementById("logDate").value = today;
}

async function submitEntry(event) {
    event.preventDefault();

    const errorEl = document.getElementById("formError");
    const successEl = document.getElementById("formSuccess");
    errorEl.classList.add("hidden");
    successEl.classList.add("hidden");

    const form = event.target;
    const submitBtn = form.querySelector("button[type=submit]");
    submitBtn.disabled = true;
    submitBtn.textContent = "Saving...";

    const getValue = id => {
        const val = document.getElementById(id).value.trim();
        return val === "" ? null : val;
    };
    const getInt = id => {
        const val = getValue(id);
        return val === null ? null : parseInt(val, 10);
    };

    const payload = {
        logDate:            getValue("logDate"),
        logTime:            getValue("logTime"),
        dailySummary:       getValue("dailySummary") || "",
        statusWeightG:      getInt("statusWeightG"),
        preFeedWeightG:     getInt("preFeedWeightG"),
        postFeedWeightG:    getInt("postFeedWeightG"),
        milkTransferG:      getInt("milkTransferG"),
        expressedLeftMl:    getInt("expressedLeftMl"),
        measurementWeightG: getInt("measurementWeightG"),
    };

    try {
        const response = await fetch("/api/logs", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(payload),
        });
        if (!response.ok) {
            const data = await response.json().catch(() => ({}));
            throw new Error(data.error || `Server error: ${response.status}`);
        }
        successEl.textContent = "Entry saved!";
        successEl.classList.remove("hidden");
        setTimeout(() => { closeForm(); loadDashboard(); }, 1200);
    } catch (err) {
        errorEl.textContent = err.message;
        errorEl.classList.remove("hidden");
    } finally {
        submitBtn.disabled = false;
        submitBtn.textContent = "Save Entry";
    }
}

function openEditForm(item) {
    document.getElementById("editOverlay").classList.remove("hidden");
    document.body.style.overflow = "hidden";
    document.getElementById("editError").classList.add("hidden");
    document.getElementById("editSuccess").classList.add("hidden");

    document.getElementById("editId").value = item.id;
    document.getElementById("editLogDate").value = item.logDate ? item.logDate.substring(0, 10) : "";

    let timeVal = "";
    if (item.logTime) {
        if (item.logTime.includes("T")) {
            timeVal = item.logTime.substring(11, 16);
        } else {
            timeVal = item.logTime.substring(0, 5);
        }
    }
    document.getElementById("editLogTime").value = timeVal;
    document.getElementById("editDailySummary").value = item.dailySummary || "";
}

function closeEditForm() {
    document.getElementById("editOverlay").classList.add("hidden");
    document.body.style.overflow = "";
    document.getElementById("editForm").reset();
    document.getElementById("editError").classList.add("hidden");
    document.getElementById("editSuccess").classList.add("hidden");
}

async function submitEdit(event) {
    event.preventDefault();

    const errorEl = document.getElementById("editError");
    const successEl = document.getElementById("editSuccess");
    errorEl.classList.add("hidden");
    successEl.classList.add("hidden");

    const submitBtn = event.target.querySelector("button[type=submit]");
    submitBtn.disabled = true;
    submitBtn.textContent = "Saving...";

    const id = document.getElementById("editId").value;
    const logDate = document.getElementById("editLogDate").value.trim();
    const logTimeRaw = document.getElementById("editLogTime").value.trim();
    const dailySummary = document.getElementById("editDailySummary").value.trim();

    const payload = {
        logDate,
        logTime: logTimeRaw === "" ? null : logTimeRaw,
        dailySummary,
    };

    try {
        const res = await fetch(`/api/logs/${id}`, {
            method: "PUT",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(payload),
        });
        if (!res.ok) {
            const data = await res.json().catch(() => ({}));
            throw new Error(data.error || `Server error: ${res.status}`);
        }
        successEl.textContent = "Entry updated!";
        successEl.classList.remove("hidden");
        setTimeout(() => { closeEditForm(); loadDashboard(); }, 1200);
    } catch (err) {
        errorEl.textContent = err.message;
        errorEl.classList.remove("hidden");
    } finally {
        submitBtn.disabled = false;
        submitBtn.textContent = "Save Changes";
    }
}

async function loadDashboard() {
    await Promise.all([loadSummary(), loadWeightChart(), loadStatusWeightChart(), loadMilkTransferChart(), loadLogs()]);
}

async function loadSummary() {
    const data = await fetch("/api/summary").then(r => r.json());
    document.getElementById("totalLogs").textContent = data.totalLogs;
    document.getElementById("weightEntries").textContent = data.weightEntries;
    document.getElementById("firstWeight").textContent = formatGram(data.firstWeight);
    document.getElementById("latestWeight").textContent = formatGram(data.latestWeight);
    document.getElementById("weightGain").textContent = formatGram(data.weightGain);
    document.getElementById("milkEntries").textContent = data.milkEntries;
    document.getElementById("averageMilk").textContent = formatGram(data.averageMilkG);
}

async function loadWeightChart() {
    const data = await fetch("/api/weights").then(r => r.json());
    const ctx = document.getElementById("weightChart");
    if (weightChart) weightChart.destroy();
    weightChart = new Chart(ctx, {
        type: "line",
        data: {
            labels: data.map(i => i.date.substring(0, 10)),
            datasets: [{
                label: "Weight (g)",
                data: data.map(i => i.weight),
                borderColor: "#7b174e",
                backgroundColor: "rgba(235, 63, 126, 0.88)",
                borderWidth: 3,
                pointRadius: 4,
                pointHoverRadius: 7,
                tension: 0.3,
                fill: true
            }]
        },
        options: {
            responsive: true,
            plugins: { tooltip: { callbacks: { label: ctx => `${ctx.parsed.y} g` } } },
            scales: {
                y: { title: { display: true, text: "Weight in grams" } },
                x: { title: { display: true, text: "Date" } }
            }
        }
    });
}

async function loadStatusWeightChart() {
    const data = await fetch("/api/status-weights").then(r => r.json());
    const ctx = document.getElementById("statusWeightChart");
    if (statusWeightChart) statusWeightChart.destroy();
    statusWeightChart = new Chart(ctx, {
        type: "line",
        data: {
            labels: data.map(i => i.date.substring(0, 10)),
            datasets: [{
                label: "Status weight (g)",
                data: data.map(i => i.weight),
                borderColor: "#f472b6",
                backgroundColor: "rgba(244, 114, 182, 0.15)",
                borderWidth: 3,
                pointRadius: 4,
                pointHoverRadius: 7,
                tension: 0.3,
                fill: true
            }]
        },
        options: {
            responsive: true,
            plugins: { tooltip: { callbacks: { label: ctx => `${ctx.parsed.y} g` } } },
            scales: {
                y: { title: { display: true, text: "Weight in grams" } },
                x: { title: { display: true, text: "Date" } }
            }
        }
    });
}

async function loadMilkTransferChart() {
    const data = await fetch("/api/milk-transfer").then(r => r.json());
    const ctx = document.getElementById("milkTransferChart");
    if (milkTransferChart) milkTransferChart.destroy();
    milkTransferChart = new Chart(ctx, {
        type: "bar",
        data: {
            labels: data.map(i => i.date.substring(0, 10)),
            datasets: [{
                label: "Milk transfer (g)",
                data: data.map(i => i.milkTransferG),
                backgroundColor: "#eb6da2",
                borderColor: "#ac3980",
                borderWidth: 1
            }]
        },
        options: {
            responsive: true,
            plugins: { tooltip: { callbacks: { label: ctx => `${ctx.parsed.y} g` } } },
            scales: {
                y: { beginAtZero: true, title: { display: true, text: "Milk transfer in grams" } },
                x: { title: { display: true, text: "Date" } }
            }
        }
    });
}

async function loadLogs() {
    allLogs = await fetch("/api/logs").then(r => r.json());
    renderLogsTable("");
}

function renderLogsTable(searchText) {
    const tbody = document.getElementById("logsTableBody");
    tbody.innerHTML = "";
    const q = searchText.toLowerCase().trim();

    const filtered = allLogs.filter(item => {
        if (!q) return true;
        return [item.logDate, item.logTime, item.dailySummary, item.measurementWeightG, item.milkTransferG]
            .join(" ").toLowerCase().includes(q);
    });

    filtered.forEach(item => {
        const row = document.createElement("tr");

        const actionsCell = document.createElement("td");

        const editBtn = document.createElement("button");
        editBtn.textContent = "✏️";
        editBtn.title = "Edit entry";
        editBtn.className = "row-btn";
        editBtn.onclick = () => openEditForm(item);

        const delBtn = document.createElement("button");
        delBtn.textContent = "🗑️";
        delBtn.title = "Delete entry";
        delBtn.className = "row-btn";
        delBtn.onclick = () => deleteEntry(item);

        actionsCell.append(editBtn, delBtn);

        const dateCell = document.createElement("td");
        dateCell.textContent = item.logDate
            ? item.logDate.substring(0, 10) + (item.logTime ? " " + item.logTime.substring(11, 16) : "")
            : "-";

        const summaryCell = document.createElement("td");
        summaryCell.textContent = item.dailySummary || "";

        const weightCell = document.createElement("td");
        weightCell.textContent = formatGram(item.measurementWeightG);

        const milkCell = document.createElement("td");
        milkCell.textContent = formatGram(item.milkTransferG);

        row.append(actionsCell, dateCell, summaryCell, weightCell, milkCell);
        tbody.appendChild(row);
    });
}

async function deleteEntry(item) {
    if (!confirm("Delete entry from " + item.logDate + "?")) return;
    const res = await fetch(`/api/logs/${item.id}`, { method: "DELETE" });
    if (res.ok) loadDashboard();
    else alert("Failed to delete.");
}

function formatGram(value) {
    if (value === null || value === undefined) return "-";
    return `${value} g`;
}


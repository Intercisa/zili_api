let weightChart = null;
let statusWeightChart = null;
let milkConsumedChart = null;
let allLogs = [];

document.addEventListener("DOMContentLoaded", () => {
    const today = new Date().toISOString().split("T")[0];
    const weekAgo = new Date(Date.now() - 6 * 24 * 60 * 60 * 1000).toISOString().split("T")[0];
    document.getElementById("milkFromDate").value = weekAgo;
    document.getElementById("milkToDate").value = today;
    document.getElementById("milkDateApply").addEventListener("click", loadMilkConsumedChart);

    loadDashboard();
    loadVitamins();
    loadGrowth();
    loadBirthDate();

    document.querySelectorAll(".tab-btn").forEach(btn => {
        btn.addEventListener("click", () => {
            const target = btn.dataset.tab;
            document.querySelectorAll(".tab-btn").forEach(b => b.classList.remove("active"));
            document.querySelectorAll(".tab-panel").forEach(p => p.classList.add("hidden"));
            btn.classList.add("active");
            document.getElementById(target).classList.remove("hidden");
            if (target === "statusWeightTab" && !statusWeightChart) loadStatusWeightChart();
        });
    });

    document.getElementById("refreshButton").addEventListener("click", () => {
        loadDashboard();
        loadGrowth();
    });
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

    document.getElementById("openGrowthFormButton").addEventListener("click", openGrowthForm);
    document.getElementById("closeGrowthButton").addEventListener("click", closeGrowthForm);
    document.getElementById("cancelGrowthButton").addEventListener("click", closeGrowthForm);
    document.getElementById("growthOverlay").addEventListener("click", event => {
        if (event.target === document.getElementById("growthOverlay")) closeGrowthForm();
    });
    document.getElementById("growthForm").addEventListener("submit", submitGrowth);

    document.getElementById("logDate").value = today;
    document.getElementById("growthDate").value = today;

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

function openGrowthForm() {
    document.getElementById("growthOverlay").classList.remove("hidden");
    document.body.style.overflow = "hidden";
    document.getElementById("growthError").classList.add("hidden");
    document.getElementById("growthSuccess").classList.add("hidden");
}

function closeGrowthForm() {
    document.getElementById("growthOverlay").classList.add("hidden");
    document.body.style.overflow = "";
    document.getElementById("growthForm").reset();
    document.getElementById("growthError").classList.add("hidden");
    document.getElementById("growthSuccess").classList.add("hidden");
    const today = new Date().toISOString().split("T")[0];
    document.getElementById("growthDate").value = today;
}

async function submitGrowth(event) {
    event.preventDefault();
    const errorEl = document.getElementById("growthError");
    const successEl = document.getElementById("growthSuccess");
    errorEl.classList.add("hidden");
    successEl.classList.add("hidden");

    const submitBtn = event.target.querySelector("button[type=submit]");
    submitBtn.disabled = true;
    submitBtn.textContent = "Saving...";

    const getFloat = id => {
        const val = document.getElementById(id).value.trim();
        return val === "" ? null : parseFloat(val);
    };
    const getInt = id => {
        const val = document.getElementById(id).value.trim();
        return val === "" ? null : parseInt(val, 10);
    };

    const payload = {
        logDate:  document.getElementById("growthDate").value,
        weightG:  getInt("growthWeight"),
        heightCm: getFloat("growthHeight"),
        headCm:   getFloat("growthHead"),
    };

    try {
        const res = await fetch("/api/growth", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify(payload),
        });
        if (!res.ok) {
            const data = await res.json().catch(() => ({}));
            throw new Error(data.error || `Server error: ${res.status}`);
        }
        successEl.textContent = "Growth measurement saved!";
        successEl.classList.remove("hidden");
        setTimeout(() => { closeGrowthForm(); loadGrowth(); }, 1200);
    } catch (err) {
        errorEl.textContent = err.message;
        errorEl.classList.remove("hidden");
    } finally {
        submitBtn.disabled = false;
        submitBtn.textContent = "Save Measurement";
    }
}

async function loadGrowth() {
    const data = await fetch("/api/growth").then(r => r.json()).catch(() => []);
    if (!data || data.length === 0) return;
    renderGrowthLatest(data[data.length - 1]);
    renderGrowthTimeline(data);
}

function renderGrowthLatest(latest) {
    const container = document.getElementById("growthLatest");
    const weight = latest.weightG ? (latest.weightG / 1000).toFixed(2) + " kg" : "—";
    const height = latest.heightCm ? latest.heightCm + " cm" : "—";
    const head   = latest.headCm   ? latest.headCm   + " cm" : "—";
    const date   = latest.date ? latest.date.substring(0, 10) : "";

    container.innerHTML = `
        <div class="growth-latest-header">Latest measurement <span class="growth-latest-date">${date}</span></div>
        <div class="growth-latest-stats">
            <div class="growth-latest-stat">
                <div class="growth-latest-icon">⚖️</div>
                <div class="growth-latest-value">${weight}</div>
                <div class="growth-latest-label">Weight</div>
            </div>
            <div class="growth-latest-stat">
                <div class="growth-latest-icon">📏</div>
                <div class="growth-latest-value">${height}</div>
                <div class="growth-latest-label">Height</div>
            </div>
            <div class="growth-latest-stat">
                <div class="growth-latest-icon">🎀</div>
                <div class="growth-latest-value">${head}</div>
                <div class="growth-latest-label">Head</div>
            </div>
        </div>
    `;
}

function renderGrowthTimeline(data) {
    const container = document.getElementById("growthTimeline");
    container.innerHTML = "";

    const byMonth = {};
    data.forEach(item => {
        const month = item.date.substring(0, 7);
        if (!byMonth[month]) byMonth[month] = [];
        byMonth[month].push(item);
    });

    const sorted = Object.keys(byMonth).sort().reverse();

    sorted.forEach((month, idx) => {
        const entries = byMonth[month];
        const last = entries[entries.length - 1];
        const prev = sorted[idx + 1] ? byMonth[sorted[idx + 1]][byMonth[sorted[idx + 1]].length - 1] : null;

        const monthLabel = new Date(month + "-01").toLocaleString("default", { month: "long", year: "numeric" });

        const weightVal = last.weightG  ? (last.weightG / 1000).toFixed(2) + " kg" : "—";
        const heightVal = last.heightCm ? last.heightCm + " cm" : "—";
        const headVal   = last.headCm   ? last.headCm   + " cm" : "—";

        const weightDiff = prev && prev.weightG  && last.weightG  ? "+" + ((last.weightG  - prev.weightG)  / 1000).toFixed(2) + " kg" : null;
        const heightDiff = prev && prev.heightCm && last.heightCm ? "+" + (last.heightCm - prev.heightCm).toFixed(1) + " cm" : null;
        const headDiff   = prev && prev.headCm   && last.headCm   ? "+" + (last.headCm   - prev.headCm).toFixed(1)   + " cm" : null;

        const card = document.createElement("div");
        card.className = "growth-month-card";
        card.innerHTML = `
            <div class="growth-month-label">${monthLabel}</div>
            <div class="growth-month-stats">
                <div class="growth-month-stat">
                    <span class="growth-month-icon">⚖️</span>
                    <span class="growth-month-value">${weightVal}</span>
                    ${weightDiff ? `<span class="growth-month-diff">${weightDiff}</span>` : ""}
                </div>
                <div class="growth-month-stat">
                    <span class="growth-month-icon">📏</span>
                    <span class="growth-month-value">${heightVal}</span>
                    ${heightDiff ? `<span class="growth-month-diff">${heightDiff}</span>` : ""}
                </div>
                <div class="growth-month-stat">
                    <span class="growth-month-icon">🎀</span>
                    <span class="growth-month-value">${headVal}</span>
                    ${headDiff ? `<span class="growth-month-diff">${headDiff}</span>` : ""}
                </div>
            </div>
        `;
        container.appendChild(card);
    });
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
    const getFloat = id => {
        const val = getValue(id);
        return val === null ? null : parseFloat(val);
    };

    const payload = {
        logDate:            getValue("logDate"),
        logTime:            getValue("logTime"),
        dailySummary:       getValue("dailySummary") || "",
        statusWeightG:      getInt("statusWeightG"),
        preFeedWeightG:     getInt("preFeedWeightG"),
        postFeedWeightG:    getInt("postFeedWeightG"),
        milkTransferG:      getInt("milkTransferG"),
        heightCm:           getFloat("heightCm"),
        headCm:             getFloat("headCm"),
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
        timeVal = item.logTime.includes("T") ? item.logTime.substring(11, 16) : item.logTime.substring(0, 5);
    }
    document.getElementById("editLogTime").value = timeVal;
    document.getElementById("editDailySummary").value = item.dailySummary || "";
    document.getElementById("editHeightCm").value = item.heightCm || "";
    document.getElementById("editHeadCm").value = item.headCm || "";
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
    const heightCmRaw = document.getElementById("editHeightCm").value.trim();
    const headCmRaw = document.getElementById("editHeadCm").value.trim();

    const payload = {
        logDate,
        logTime:     logTimeRaw  === "" ? null : logTimeRaw,
        dailySummary,
        heightCm:    heightCmRaw === "" ? null : parseFloat(heightCmRaw),
        headCm:      headCmRaw   === "" ? null : parseFloat(headCmRaw),
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
    await Promise.all([loadSummary(), loadWeightChart(), loadMilkConsumedChart(), loadLogs()]);
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

async function loadMilkConsumedChart() {
    const from = document.getElementById("milkFromDate").value;
    const to = document.getElementById("milkToDate").value;
    const data = await fetch(`/api/milk-consumed?from=${from}&to=${to}`).then(r => r.json()).catch(() => []);
    const ctx = document.getElementById("milkConsumedChart");
    if (milkConsumedChart) milkConsumedChart.destroy();
    milkConsumedChart = new Chart(ctx, {
        type: "bar",
        data: {
            labels: data.map(i => i.date.substring(0, 10)),
            datasets: [{
                label: "Milk consumed (g)",
                data: data.map(i => i.milkConsumedG),
                backgroundColor: "#f9a8d4",
                borderColor: "#db2777",
                borderWidth: 1
            }]
        },
        options: {
            responsive: true,
            plugins: { tooltip: { callbacks: { label: ctx => `${ctx.parsed.y} g` } } },
            scales: {
                y: { beginAtZero: true, title: { display: true, text: "Milk consumed in grams" } },
                x: { title: { display: true, text: "Date" } }
            }
        }
    });
}

async function loadLogs() {
    allLogs = await fetch("/api/logs").then(r => r.json());
    renderLogsTable("");
    updateAwakeStatus();
}

function updateAwakeStatus() {
    const now = new Date();
    const today = now.toISOString().split("T")[0];
    const yesterday = new Date(now - 86400000).toISOString().split("T")[0];

    const sleepTags = ["elaludt", "cicin elaludt"];

    // Convert a log entry to an absolute Date for cross-day comparison
    function logToDate(l) {
        const t = l.logTime.includes("T") ? l.logTime.substring(11, 16) : l.logTime.substring(0, 5);
        const [h, m] = t.split(":").map(Number);
        const d = new Date(l.logDate.substring(0, 10));
        d.setHours(h, m, 0, 0);
        return d;
    }

    const relevantLogs = allLogs
        .filter(l => l.logDate && l.logTime && (l.logDate.substring(0, 10) === today || l.logDate.substring(0, 10) === yesterday))
        .sort((a, b) => logToDate(a) - logToDate(b));

    const lastSleep  = [...relevantLogs].reverse().find(l => l.dailySummary && sleepTags.some(t => l.dailySummary.includes(t)));
    const lastEbredt = [...relevantLogs].reverse().find(l => l.dailySummary && l.dailySummary.includes("ébredt"));

    const el = document.getElementById("awakeStatus");

    const isSleeping = lastSleep && (!lastEbredt || logToDate(lastSleep) > logToDate(lastEbredt));

    if (isSleeping) {
        const diffMs = now - logToDate(lastSleep);
        const diffH = Math.floor(diffMs / 3600000);
        const diffM = Math.floor((diffMs % 3600000) / 60000);
        el.textContent = `alszik ${diffH > 0 ? `${diffH}h ${diffM}m` : `${diffM}m`}`;
        return;
    }

    if (!lastEbredt) { el.textContent = "alszik"; return; }

    const wakeDate = logToDate(lastEbredt);
    const diffMs = now - wakeDate;
    if (diffMs < 0) { el.textContent = "-"; return; }
    const diffH = Math.floor(diffMs / 3600000);
    const diffM = Math.floor((diffMs % 3600000) / 60000);
    el.textContent = diffH > 0 ? `${diffH}h ${diffM}m` : `${diffM}m`;
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

async function loadBirthDate() {
    const res = await fetch("/api/settings/birth-date");
    if (res.status === 404) {
        document.getElementById("ageStatus").classList.add("hidden");
        document.getElementById("birthDateForm").classList.remove("hidden");
        return;
    }
    const data = await res.json();
    document.getElementById("birthDateForm").classList.add("hidden");
    document.getElementById("ageStatus").classList.remove("hidden");
    updateAgeDisplay(data.value);
}

async function saveBirthDate() {
    const value = document.getElementById("birthDateInput").value;
    if (!value) return;
    await fetch("/api/settings/birth-date", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ value }),
    });
    document.getElementById("birthDateForm").classList.add("hidden");
    document.getElementById("ageStatus").classList.remove("hidden");
    updateAgeDisplay(value);
}

function updateAgeDisplay(birthDateStr) {
    const birth = new Date(birthDateStr);
    const now = new Date();
    const totalDays = Math.floor((now - birth) / 86400000);
    const totalWeeks = Math.floor(totalDays / 7);

    const years = now.getFullYear() - birth.getFullYear();
    const monthDiff = now.getMonth() - birth.getMonth() + years * 12;
    const months = now.getDate() >= birth.getDate() ? monthDiff : monthDiff - 1;

    let text;
    if (months < 6) {
        text = `${totalWeeks} hetes`;
    } else if (months < 12) {
        text = `${months} hónapos (${totalWeeks} hetes)`;
    } else {
        const y = Math.floor(months / 12);
        text = `${y} éves (${totalWeeks} hetes)`;
    }
    document.getElementById("ageStatus").textContent = text;
}


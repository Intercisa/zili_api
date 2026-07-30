let weightChart = null;
let statusWeightChart = null;
let milkConsumedChart = null;
let sleepAwakeChart = null;
let logsOffset = 0;
let logsAllLoaded = false;
let logsLoading = false;

document.addEventListener("DOMContentLoaded", () => {
    const today = new Date().toISOString().split("T")[0];
    const weekAgo = new Date(Date.now() - 6 * 24 * 60 * 60 * 1000).toISOString().split("T")[0];
    const monthAgo = new Date(Date.now() - 29 * 24 * 60 * 60 * 1000).toISOString().split("T")[0];
    document.getElementById("milkFromDate").value = weekAgo;
    document.getElementById("milkToDate").value = today;
    document.getElementById("milkDateApply").addEventListener("click", loadMilkConsumedChart);
    document.getElementById("weightFromDate").value = monthAgo;
    document.getElementById("weightToDate").value = today;
    document.getElementById("weightDateApply").addEventListener("click", loadWeightChart);
    document.getElementById("statusWeightFromDate").value = monthAgo;
    document.getElementById("statusWeightToDate").value = today;
    document.getElementById("statusWeightDateApply").addEventListener("click", loadStatusWeightChart);
    document.getElementById("sleepFromDate").value = weekAgo;
    document.getElementById("sleepToDate").value = today;
    document.getElementById("sleepDateApply").addEventListener("click", loadSleepAwakeChart);

    loadDashboard();
    loadVitamins();
    loadGrowth();
    loadBirthDate();

    const tableWrapper = document.querySelector(".table-wrapper");
    tableWrapper.addEventListener("scroll", () => {
        if (tableWrapper.scrollTop + tableWrapper.clientHeight >= tableWrapper.scrollHeight - 50) {
            loadMoreLogs();
        }
    });

    document.querySelectorAll(".tab-btn").forEach(btn => {
        btn.addEventListener("click", () => {
            const target = btn.dataset.tab;
            document.querySelectorAll(".tab-btn").forEach(b => b.classList.remove("active"));
            document.querySelectorAll(".tab-panel").forEach(p => p.classList.add("hidden"));
            btn.classList.add("active");
            document.getElementById(target).classList.remove("hidden");
            if (target === "statusWeightTab" && !statusWeightChart) loadStatusWeightChart();
            if (target === "sleepAwakeTab" && !sleepAwakeChart) loadSleepAwakeChart();
        });
    });

    document.getElementById("refreshButton").addEventListener("click", () => { loadDashboard(); loadGrowth(); });
    document.getElementById("searchInput").addEventListener("input", async event => {
        const q = event.target.value.trim();
        if (!q) { loadLogs(); return; }
        const data = await fetch(`/api/logs?search=${encodeURIComponent(q)}&offset=0`).then(r => r.json()).catch(() => []);
        document.getElementById("logsTableBody").innerHTML = "";
        appendLogsToTable(data);
        logsAllLoaded = true;
    });
    document.getElementById("openFormButton").addEventListener("click", openForm);
    document.getElementById("closeFormButton").addEventListener("click", closeForm);
    document.getElementById("cancelFormButton").addEventListener("click", closeForm);
    document.getElementById("formOverlay").addEventListener("click", e => { if (e.target === document.getElementById("formOverlay")) closeForm(); });
    document.getElementById("entryForm").addEventListener("submit", submitEntry);
    document.getElementById("closeEditButton").addEventListener("click", closeEditForm);
    document.getElementById("cancelEditButton").addEventListener("click", closeEditForm);
    document.getElementById("editOverlay").addEventListener("click", e => { if (e.target === document.getElementById("editOverlay")) closeEditForm(); });
    document.getElementById("editForm").addEventListener("submit", submitEdit);
    document.getElementById("openGrowthFormButton").addEventListener("click", openGrowthForm);
    document.getElementById("closeGrowthButton").addEventListener("click", closeGrowthForm);
    document.getElementById("cancelGrowthButton").addEventListener("click", closeGrowthForm);
    document.getElementById("growthOverlay").addEventListener("click", e => { if (e.target === document.getElementById("growthOverlay")) closeGrowthForm(); });
    document.getElementById("growthForm").addEventListener("submit", submitGrowth);
    document.getElementById("logDate").value = today;
    document.getElementById("growthDate").value = today;
    document.getElementById("dVitaminCheck").addEventListener("change", e => {
        const date = new Date().toISOString().slice(0, 10);
        saveVitamin("d-vitamin", e.target.checked, date);
        if (e.target.checked) e.target.disabled = true;
    });
    document.getElementById("kVitaminCheck").addEventListener("change", e => {
        const t = new Date();
        const date = `${t.getFullYear()}-${String(t.getMonth() + 1).padStart(2, "0")}`;
        saveVitamin("k-vitamin", e.target.checked, date);
        if (e.target.checked) e.target.disabled = true;
    });
    document.querySelectorAll(".quick-tag").forEach(cb => cb.addEventListener("change", syncQuickTags));
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
    dEl.checked = d && d.date === todayStr && d.checked === true;
    dEl.disabled = dEl.checked;
    const kEl = document.getElementById("kVitaminCheck");
    const k = data["k-vitamin"];
    kEl.checked = k && k.date === monthStr && k.checked === true;
    kEl.disabled = kEl.checked;
}

async function saveVitamin(key, checked, date) {
    await fetch(`/api/vitamins/${key}`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ checked, date }) });
}

function openForm() {
    document.getElementById("formOverlay").classList.remove("hidden");
    document.body.style.overflow = "hidden";
    document.getElementById("formError").classList.add("hidden");
    document.getElementById("formSuccess").classList.add("hidden");
    const now = new Date();
    document.getElementById("logTime").value = `${String(now.getHours()).padStart(2,"0")}:${String(now.getMinutes()).padStart(2,"0")}`;
}

function closeForm() {
    document.getElementById("formOverlay").classList.add("hidden");
    document.body.style.overflow = "";
    document.getElementById("entryForm").reset();
    document.getElementById("formError").classList.add("hidden");
    document.getElementById("formSuccess").classList.add("hidden");
    document.getElementById("logDate").value = new Date().toISOString().split("T")[0];
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
    document.getElementById("growthDate").value = new Date().toISOString().split("T")[0];
}

async function submitGrowth(event) {
    event.preventDefault();
    const errorEl = document.getElementById("growthError");
    const successEl = document.getElementById("growthSuccess");
    errorEl.classList.add("hidden"); successEl.classList.add("hidden");
    const submitBtn = event.target.querySelector("button[type=submit]");
    submitBtn.disabled = true; submitBtn.textContent = "Saving...";
    const getFloat = id => { const v = document.getElementById(id).value.trim(); return v === "" ? null : parseFloat(v); };
    const getInt   = id => { const v = document.getElementById(id).value.trim(); return v === "" ? null : parseInt(v, 10); };
    const payload = { logDate: document.getElementById("growthDate").value, weightG: getInt("growthWeight"), heightCm: getFloat("growthHeight"), headCm: getFloat("growthHead") };
    try {
        const res = await fetch("/api/growth", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) });
        if (!res.ok) { const d = await res.json().catch(() => ({})); throw new Error(d.error || `Server error: ${res.status}`); }
        successEl.textContent = "Growth measurement saved!"; successEl.classList.remove("hidden");
        setTimeout(() => { closeGrowthForm(); loadGrowth(); }, 1200);
    } catch (err) { errorEl.textContent = err.message; errorEl.classList.remove("hidden"); }
    finally { submitBtn.disabled = false; submitBtn.textContent = "Save Measurement"; }
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
            <div class="growth-latest-stat"><div class="growth-latest-icon">⚖️</div><div class="growth-latest-value">${weight}</div><div class="growth-latest-label">Weight</div></div>
            <div class="growth-latest-stat"><div class="growth-latest-icon">📏</div><div class="growth-latest-value">${height}</div><div class="growth-latest-label">Height</div></div>
            <div class="growth-latest-stat"><div class="growth-latest-icon">🎀</div><div class="growth-latest-value">${head}</div><div class="growth-latest-label">Head</div></div>
        </div>`;
}

function renderGrowthTimeline(data) {
    const container = document.getElementById("growthTimeline");
    container.innerHTML = "";
    const byMonth = {};
    data.forEach(item => { const m = item.date.substring(0, 7); if (!byMonth[m]) byMonth[m] = []; byMonth[m].push(item); });
    Object.keys(byMonth).sort().reverse().forEach((month, idx, arr) => {
        const last = byMonth[month][byMonth[month].length - 1];
        const prev = arr[idx + 1] ? byMonth[arr[idx + 1]][byMonth[arr[idx + 1]].length - 1] : null;
        const monthLabel = new Date(month + "-01").toLocaleString("default", { month: "long", year: "numeric" });
        const wv = last.weightG  ? (last.weightG / 1000).toFixed(2) + " kg" : "—";
        const hv = last.heightCm ? last.heightCm + " cm" : "—";
        const cv = last.headCm   ? last.headCm   + " cm" : "—";
        const wd = prev && prev.weightG  && last.weightG  ? "+" + ((last.weightG  - prev.weightG)  / 1000).toFixed(2) + " kg" : null;
        const hd = prev && prev.heightCm && last.heightCm ? "+" + (last.heightCm - prev.heightCm).toFixed(1) + " cm" : null;
        const cd = prev && prev.headCm   && last.headCm   ? "+" + (last.headCm   - prev.headCm).toFixed(1)   + " cm" : null;
        const card = document.createElement("div");
        card.className = "growth-month-card";
        card.innerHTML = `<div class="growth-month-label">${monthLabel}</div><div class="growth-month-stats">
            <div class="growth-month-stat"><span class="growth-month-icon">⚖️</span><span class="growth-month-value">${wv}</span>${wd ? `<span class="growth-month-diff">${wd}</span>` : ""}</div>
            <div class="growth-month-stat"><span class="growth-month-icon">📏</span><span class="growth-month-value">${hv}</span>${hd ? `<span class="growth-month-diff">${hd}</span>` : ""}</div>
            <div class="growth-month-stat"><span class="growth-month-icon">🎀</span><span class="growth-month-value">${cv}</span>${cd ? `<span class="growth-month-diff">${cd}</span>` : ""}</div>
        </div>`;
        container.appendChild(card);
    });
}

async function submitEntry(event) {
    event.preventDefault();
    const errorEl = document.getElementById("formError"); const successEl = document.getElementById("formSuccess");
    errorEl.classList.add("hidden"); successEl.classList.add("hidden");
    const submitBtn = event.target.querySelector("button[type=submit]");
    submitBtn.disabled = true; submitBtn.textContent = "Saving...";
    const getValue = id => { const v = document.getElementById(id).value.trim(); return v === "" ? null : v; };
    const getInt   = id => { const v = getValue(id); return v === null ? null : parseInt(v, 10); };
    const getFloat = id => { const v = getValue(id); return v === null ? null : parseFloat(v); };
    const payload = {
        logDate: getValue("logDate"), logTime: getValue("logTime"), dailySummary: getValue("dailySummary") || "",
        statusWeightG: getInt("statusWeightG"), preFeedWeightG: getInt("preFeedWeightG"), postFeedWeightG: getInt("postFeedWeightG"),
        milkTransferG: getInt("milkTransferG"), heightCm: getFloat("heightCm"), headCm: getFloat("headCm"), measurementWeightG: getInt("measurementWeightG"),
    };
    try {
        const response = await fetch("/api/logs", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) });
        if (!response.ok) { const d = await response.json().catch(() => ({})); throw new Error(d.error || `Server error: ${response.status}`); }
        successEl.textContent = "Entry saved!"; successEl.classList.remove("hidden");
        setTimeout(() => { closeForm(); loadDashboard(); }, 1200);
    } catch (err) { errorEl.textContent = err.message; errorEl.classList.remove("hidden"); }
    finally { submitBtn.disabled = false; submitBtn.textContent = "Save Entry"; }
}

function openEditForm(item) {
    document.getElementById("editOverlay").classList.remove("hidden");
    document.body.style.overflow = "hidden";
    document.getElementById("editError").classList.add("hidden");
    document.getElementById("editSuccess").classList.add("hidden");
    document.getElementById("editId").value = item.id;
    document.getElementById("editLogDate").value = item.logDate ? item.logDate.substring(0, 10) : "";
    let timeVal = "";
    if (item.logTime) timeVal = item.logTime.includes("T") ? item.logTime.substring(11, 16) : item.logTime.substring(0, 5);
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
    const errorEl = document.getElementById("editError"); const successEl = document.getElementById("editSuccess");
    errorEl.classList.add("hidden"); successEl.classList.add("hidden");
    const submitBtn = event.target.querySelector("button[type=submit]");
    submitBtn.disabled = true; submitBtn.textContent = "Saving...";
    const id = document.getElementById("editId").value;
    const logDate = document.getElementById("editLogDate").value.trim();
    const logTimeRaw = document.getElementById("editLogTime").value.trim();
    const dailySummary = document.getElementById("editDailySummary").value.trim();
    const heightCmRaw = document.getElementById("editHeightCm").value.trim();
    const headCmRaw = document.getElementById("editHeadCm").value.trim();
    const payload = { logDate, logTime: logTimeRaw === "" ? null : logTimeRaw, dailySummary, heightCm: heightCmRaw === "" ? null : parseFloat(heightCmRaw), headCm: headCmRaw === "" ? null : parseFloat(headCmRaw) };
    try {
        const res = await fetch(`/api/logs/${id}`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) });
        if (!res.ok) { const d = await res.json().catch(() => ({})); throw new Error(d.error || `Server error: ${res.status}`); }
        successEl.textContent = "Entry updated!"; successEl.classList.remove("hidden");
        setTimeout(() => { closeEditForm(); loadDashboard(); }, 1200);
    } catch (err) { errorEl.textContent = err.message; errorEl.classList.remove("hidden"); }
    finally { submitBtn.disabled = false; submitBtn.textContent = "Save Changes"; }
}
async function loadDashboard() {
    await Promise.all([loadSummary(), loadWeightChart(), loadMilkConsumedChart(), loadLogs()]);
}

async function loadSummary() {
    const data = await fetch("/api/summary").then(r => r.json());
    document.getElementById("totalLogs").textContent = data.totalLogs;
    document.getElementById("firstWeight").textContent = formatGram(data.firstWeight);
    document.getElementById("latestWeight").textContent = formatGram(data.latestWeight);
    document.getElementById("weightGain").textContent = formatGram(data.weightGain);
    document.getElementById("averageMilk").textContent = formatGram(data.averageMilkG);
}

async function loadLogs() {
    logsOffset = 0;
    logsAllLoaded = false;
    document.getElementById("logsTableBody").innerHTML = "";
    await loadMoreLogs();
    updateCurrentStatus();
}

async function loadMoreLogs() {
    if (logsLoading || logsAllLoaded) return;
    logsLoading = true;
    const batch = await fetch(`/api/logs?offset=${logsOffset}`).then(r => r.json()).catch(() => []);
    if (batch.length < 100) logsAllLoaded = true;
    logsOffset += batch.length;
    appendLogsToTable(batch);
    logsLoading = false;
}

async function updateCurrentStatus() {
    const data = await fetch("/api/current-status").then(r => r.json()).catch(() => null);
    if (!data) return;
    const label = data.state === "sleep" ? "alszik" : "ébren van";
    const tile = document.getElementById("awakeStatus").closest(".summary-card");
    document.getElementById("awakeStatus").innerHTML = `<span style="font-size:0.85rem;font-weight:600;display:block">${label}</span>${data.duration}`;
    tile.style.background = data.state === "sleep" ? "#dbeafe" : "#fce7f3";
    tile.style.borderColor = data.state === "sleep" ? "#93c5fd" : "#f9a8d4";
    const fmt = min => { const h = Math.floor(min / 60); const m = min % 60; return h > 0 ? `${h}h ${m}m` : `${m}m`; };
    document.getElementById("todaySleep").textContent = ` ${fmt(data.sleepMin)}`;
    document.getElementById("todayAwake").textContent = ` ${fmt(data.awakeMin)}`;
}

async function loadWeightChart() {
    const from = document.getElementById("weightFromDate").value;
    const to = document.getElementById("weightToDate").value;
    const data = await fetch(`/api/weights?from=${from}&to=${to}`).then(r => r.json());
    const ctx = document.getElementById("weightChart");
    if (weightChart) weightChart.destroy();
    weightChart = new Chart(ctx, {
        type: "line",
        data: { labels: data.map(i => i.date.substring(0, 10)), datasets: [{ label: "Weight (g)", data: data.map(i => i.weight), borderColor: "#7b174e", backgroundColor: "rgba(235, 63, 126, 0.88)", borderWidth: 3, pointRadius: 4, pointHoverRadius: 7, tension: 0.3, fill: true }] },
        options: { responsive: true, plugins: { datalabels: { display: false }, tooltip: { callbacks: { label: ctx => `${ctx.parsed.y} g` } } }, scales: { y: { title: { display: true, text: "Weight in grams" } }, x: { title: { display: true, text: "Date" } } } }
    });
}

async function loadStatusWeightChart() {
    const from = document.getElementById("statusWeightFromDate").value;
    const to = document.getElementById("statusWeightToDate").value;
    const data = await fetch(`/api/status-weights?from=${from}&to=${to}`).then(r => r.json());
    const ctx = document.getElementById("statusWeightChart");
    if (statusWeightChart) statusWeightChart.destroy();
    statusWeightChart = new Chart(ctx, {
        type: "line",
        data: { labels: data.map(i => i.date.substring(0, 10)), datasets: [{ label: "Status weight (g)", data: data.map(i => i.weight), borderColor: "#f472b6", backgroundColor: "rgba(244, 114, 182, 0.15)", borderWidth: 3, pointRadius: 4, pointHoverRadius: 7, tension: 0.3, fill: true }] },
        options: { responsive: true, plugins: { datalabels: { display: false }, tooltip: { callbacks: { label: ctx => `${ctx.parsed.y} g` } } }, scales: { y: { title: { display: true, text: "Weight in grams" } }, x: { title: { display: true, text: "Date" } } } }
    });
}

async function loadSleepAwakeChart() {
    const from = document.getElementById("sleepFromDate").value;
    const to = document.getElementById("sleepToDate").value;
    const data = await fetch(`/api/sleep-awake?from=${from}&to=${to}`).then(r => r.json()).catch(() => []);
    const ctx = document.getElementById("sleepAwakeChart");
    if (sleepAwakeChart) sleepAwakeChart.destroy();
    sleepAwakeChart = new Chart(ctx, {
        type: "bar",
        data: {
            labels: data.map(i => i.Date),
            datasets: [
                { label: "Aludt (óra)", data: data.map(i => +(i.SleepMin / 60).toFixed(2)), backgroundColor: "#f9a8d4", borderColor: "#db2777", borderWidth: 1 },
                { label: "Ébren (óra)", data: data.map(i => +(i.AwakeMin / 60).toFixed(2)), backgroundColor: "#c4b5fd", borderColor: "#7c3aed", borderWidth: 1 }
            ]
        },
        options: {
            responsive: true,
            plugins: { datalabels: { display: false }, tooltip: { callbacks: { label: ctx => { const h = Math.floor(ctx.parsed.y); const m = Math.round((ctx.parsed.y - h) * 60); return `${ctx.dataset.label}: ${h > 0 ? h + "h " : ""}${m}m`; } } } },
            scales: { x: { stacked: true, title: { display: true, text: "Date" } }, y: { stacked: true, title: { display: true, text: "Hours" } } }
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
        data: { labels: data.map(i => i.date.substring(0, 10)), datasets: [{ label: "Milk consumed (g)", data: data.map(i => i.milkConsumedG), backgroundColor: "#f9a8d4", borderColor: "#db2777", borderWidth: 1 }] },
        options: {
            responsive: true,
            plugins: { datalabels: { anchor: "center", align: "center", color: "#9d174d", font: { weight: "700", size: 14 }, formatter: v => `${v} g` }, tooltip: { callbacks: { label: ctx => `${ctx.parsed.y} g` } } },
            scales: { y: { beginAtZero: true, title: { display: true, text: "Milk consumed in grams" } }, x: { title: { display: true, text: "Date" } } }
        }
    });
}

function appendLogsToTable(items) {
    const tbody = document.getElementById("logsTableBody");
    items.forEach(item => {
        const row = document.createElement("tr");
        const actionsCell = document.createElement("td");
        const editBtn = document.createElement("button");
        editBtn.textContent = "✏️"; editBtn.title = "Edit entry"; editBtn.className = "row-btn";
        editBtn.onclick = () => openEditForm(item);
        const delBtn = document.createElement("button");
        delBtn.textContent = "🗑️"; delBtn.title = "Delete entry"; delBtn.className = "row-btn";
        delBtn.onclick = () => deleteEntry(item);
        actionsCell.append(editBtn, delBtn);
        const dateCell = document.createElement("td");
        dateCell.textContent = item.logDate ? item.logDate.substring(0, 10) + (item.logTime ? " " + item.logTime.substring(0, 5) : "") : "-";
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
    await fetch("/api/settings/birth-date", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ value }) });
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


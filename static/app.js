let weightChart = null;
let statusWeightChart = null;
let milkConsumedChart = null;
let sleepAwakeChart = null;
let logsOffset = 0;
let logsAllLoaded = false;
let logsLoading = false;

const PENDING_FEED_KEY = "zili_pending_feed";

async function savePendingFeed(data) {
    const payload = {
        logDate: data.logDate, logTime: data.logTime,
        preFeedWeightG: data.preFeedWeightG, postFeedWeightG: data.postFeedWeightG,
        fedBreast: data.fedBreast, fedBottle: data.fedBottle,
        statusWeightG: data.statusWeightG ? parseInt(data.statusWeightG) : null,
        measurementWeightG: data.measurementWeightG ? parseInt(data.measurementWeightG) : null,
        dailySummary: data.dailySummary || "",
        pending: true,
    };
    const res = await fetch("/api/logs", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) });
    const { id } = await res.json();
    localStorage.setItem(PENDING_FEED_KEY, JSON.stringify({ ...data, id }));
    renderPendingBanner();
}

async function loadPendingFeed() {
    const res = await fetch("/api/pending-feed").catch(() => null);
    if (!res || !res.ok) return null;
    const data = await res.json();
    if (!data) return null;
    let logTime = data.logTime;
    if (logTime && logTime.length > 5) logTime = logTime.substring(0, 5);
    const logDate = data.logDate ? data.logDate.substring(0, 10) : null;
    const local = (() => { try { return JSON.parse(localStorage.getItem(PENDING_FEED_KEY)); } catch { return null; } })();
    const quickTags = (local && local.id === data.id) ? (local.quickTags || []) : [];
    return {
        id: data.id, logDate, logTime,
        preFeedWeightG: data.preFeedWeightG, postFeedWeightG: data.postFeedWeightG,
        fedBreast: data.fedBreast, fedBottle: data.fedBottle,
        statusWeightG: data.statusWeightG, measurementWeightG: data.measurementWeightG,
        dailySummary: data.dailySummary, quickTags,
    };
}

async function clearPendingFeed(id) {
    if (id) await fetch(`/api/pending-feed/${id}`, { method: "DELETE" });
    localStorage.removeItem(PENDING_FEED_KEY);
    document.getElementById("pendingFeedBanner").classList.add("hidden");
}

async function renderPendingBanner() {
    const pending = await loadPendingFeed();
    const banner = document.getElementById("pendingFeedBanner");
    if (!pending) { banner.classList.add("hidden"); return; }
    const what = pending.preFeedWeightG !== null ? `pre: ${pending.preFeedWeightG}g — missing post` : `post: ${pending.postFeedWeightG}g — missing pre`;
    document.getElementById("pendingFeedText").textContent = `🍼 Unsent feeding (${what})`;
    banner.classList.remove("hidden");
}

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
    document.getElementById("whoToggleBtn").addEventListener("click", () => {
        if (!weightChart) return;
        const btn = document.getElementById("whoToggleBtn");
        const visible = weightChart.data.datasets[1].hidden;
        [1, 2, 3].forEach(i => { if (weightChart.data.datasets[i]) weightChart.data.datasets[i].hidden = !visible; });
        weightChart.update();
        btn.textContent = visible ? "WHO: ON" : "WHO: OFF";
    });
    document.getElementById("statusWeightFromDate").value = monthAgo;
    document.getElementById("statusWeightToDate").value = today;
    document.getElementById("statusWeightDateApply").addEventListener("click", loadStatusWeightChart);
    document.getElementById("sleepFromDate").value = weekAgo;
    document.getElementById("sleepToDate").value = today;
    document.getElementById("sleepDateApply").addEventListener("click", loadSleepAwakeChart);

    loadDashboard();
    loadVitamins();
    loadGrowth();
    loadEvents();
    loadBirthDate();
    loadWordOfTheDay();
    renderPendingBanner();
    loadDiaperAlert();

    document.getElementById("pendingFeedContinue").addEventListener("click", () => { openForm(true); });
    document.getElementById("pendingFeedDiscard").addEventListener("click", async () => {
        const pending = await loadPendingFeed();
        clearPendingFeed(pending?.id);
    });

    const tableWrapper = document.querySelector(".table-wrapper");
    tableWrapper.addEventListener("scroll", () => {
        if (tableWrapper.scrollTop + tableWrapper.clientHeight >= tableWrapper.scrollHeight - 50) {
            loadMoreLogs();
        }
    });

    window.addEventListener("scroll", () => {
        const btn = document.getElementById("scrollTopBtn");
        const floatAdd = document.getElementById("floatAddBtn");
        const openFormBtn = document.getElementById("openFormButton");
        const modalOpen = !!document.getElementById("floatAddBtn").dataset.formOpen;
        if (!modalOpen && openFormBtn.getBoundingClientRect().bottom < 0) btn.classList.remove("hidden");
        else btn.classList.add("hidden");
        const formBtnVisible = openFormBtn.getBoundingClientRect().bottom > 0;
        if (formBtnVisible || floatAdd.dataset.formOpen) floatAdd.classList.add("hidden");
        else floatAdd.classList.remove("hidden");
    });

    document.querySelectorAll(".tab-btn").forEach(btn => {
        btn.addEventListener("click", () => {
            const target = btn.dataset.tab;
            const section = btn.closest(".chart-card");
            section.querySelectorAll(".tab-btn").forEach(b => b.classList.remove("active"));
            section.querySelectorAll(".tab-panel").forEach(p => p.classList.add("hidden"));
            btn.classList.add("active");
            document.getElementById(target).classList.remove("hidden");
            if (target === "statusWeightTab" && !statusWeightChart) loadStatusWeightChart();
            if (target === "sleepAwakeTab" && !sleepAwakeChart) loadSleepAwakeChart();
        });
    });


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
    document.getElementById("openEventFormButton").addEventListener("click", () => openEventForm());
    document.getElementById("closeEventButton").addEventListener("click", closeEventForm);
    document.getElementById("cancelEventButton").addEventListener("click", closeEventForm);
    document.getElementById("eventOverlay").addEventListener("click", e => { if (e.target === document.getElementById("eventOverlay")) closeEventForm(); });
    document.getElementById("eventForm").addEventListener("submit", submitEvent);
    document.getElementById("addWordButton").addEventListener("click", openWordForm);
    document.getElementById("closeWordButton").addEventListener("click", closeWordForm);
    document.getElementById("cancelWordButton").addEventListener("click", closeWordForm);
    document.getElementById("wordOverlay").addEventListener("click", e => { if (e.target === document.getElementById("wordOverlay")) closeWordForm(); });
    document.getElementById("toggleWordsListButton").addEventListener("click", toggleWordsList);
    document.getElementById("wordForm").addEventListener("submit", async e => {
        e.preventDefault();
        const word = document.getElementById("wordInput").value.trim();
        const notedDate = document.getElementById("wordDateInput").value;
        const notes = document.getElementById("wordNotes").value.trim() || null;
        if (!word) return;
        await fetch("/api/words", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ word, notedDate, notes }) });
        document.getElementById("wordForm").reset();
        document.getElementById("wordDateInput").value = new Date().toISOString().split("T")[0];
        loadWordOfTheDay();
    });
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
    document.querySelectorAll(".cat-btn").forEach(btn => {
        btn.addEventListener("click", () => {
            document.querySelectorAll(".cat-btn").forEach(b => b.classList.remove("active"));
            btn.classList.add("active");
            document.querySelectorAll(".cat-panel").forEach(p => p.classList.add("hidden"));
            const cat = btn.dataset.cat;
            if (cat === "measurements") document.getElementById("catMeasurements").classList.remove("hidden");
            if (cat === "milk") document.getElementById("catMilk").classList.remove("hidden");
            if (cat === "words") document.getElementById("catWords").classList.remove("hidden");
        });
    });
    document.getElementById("fedBreast").addEventListener("change", syncQuickTags);
    document.getElementById("fedBottle").addEventListener("change", syncQuickTags);
    document.querySelectorAll(".quick-tag").forEach(cb => cb.addEventListener("change", syncQuickTags));
    document.getElementById("measurementWeightG").addEventListener("input", syncQuickTags);
    document.getElementById("statusWeightG").addEventListener("input", syncQuickTags);
    document.getElementById("preFeedWeightG").addEventListener("input", syncQuickTags);
    document.getElementById("postFeedWeightG").addEventListener("input", syncQuickTags);

    const statusTile = document.getElementById("awakeStatus").closest(".summary-card");
    attachLongPress(statusTile, async () => {
        const now = new Date();
        const logDate = now.toISOString().split("T")[0];
        const logTime = `${String(now.getHours()).padStart(2,"0")}:${String(now.getMinutes()).padStart(2,"0")}`;
        const isSleep = statusTile.dataset.currentState === "sleep";
        const sleepEvent = isSleep ? "woke_up" : "fell_asleep";
        const dailySummary = isSleep ? "ébredt" : "elaludt";
        await fetch("/api/logs", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ logDate, logTime, dailySummary, sleepEvent })
        });
        await updateCurrentStatus();
        loadLogs();
    });
});

function syncQuickTags() {
    const parts = [];
    const fedBreast = document.getElementById("fedBreast").checked;
    const fedBottle = document.getElementById("fedBottle").checked;
    if (fedBreast && fedBottle) parts.push("cici, cumisüveg");
    else if (fedBreast) parts.push("cici");
    else if (fedBottle) parts.push("cumisüveg");
    const pre = document.getElementById("preFeedWeightG").value.trim();
    const post = document.getElementById("postFeedWeightG").value.trim();
    if (pre || post) parts.push(`pre: ${pre || "?"}g, post: ${post || "?"}g`);
    const checked = [...document.querySelectorAll(".quick-tag:checked")].map(cb => cb.value);
    parts.push(...checked);
    const measW = document.getElementById("measurementWeightG").value.trim();
    const statW = document.getElementById("statusWeightG").value.trim();
    if (measW) parts.push(`pucér popós súly: ${measW}g`);
    if (statW) parts.push(`státusz súly: ${statW}g`);
    document.getElementById("dailySummary").value = parts.join(", ");
}

function syncEditQuickTags() {
    const parts = [];
    const fedBreast = document.getElementById("editFedBreast").checked;
    const fedBottle = document.getElementById("editFedBottle").checked;
    if (fedBreast && fedBottle) parts.push("cici, cumisüveg");
    else if (fedBreast) parts.push("cici");
    else if (fedBottle) parts.push("cumisüveg");
    const pre = document.getElementById("editPreFeedWeightG").value.trim();
    const post = document.getElementById("editPostFeedWeightG").value.trim();
    if (pre || post) parts.push(`pre: ${pre || "?"}g, post: ${post || "?"}g`);
    const checked = [...document.querySelectorAll(".edit-quick-tag:checked")].map(cb => cb.value);
    parts.push(...checked);
    const measW = document.getElementById("editMeasurementWeightG").value.trim();
    const statW = document.getElementById("editStatusWeightG").value.trim();
    if (measW) parts.push(`pucér popós súly: ${measW}g`);
    if (statW) parts.push(`státusz súly: ${statW}g`);
    document.getElementById("editDailySummary").value = parts.join(", ");
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

function lockScroll() {
    document.body.dataset.scrollY = window.scrollY;
    document.body.style.overflow = 'hidden';
}

function unlockScroll() {
    document.body.style.overflow = '';
    window.scrollTo(0, parseInt(document.body.dataset.scrollY || '0'));
    delete document.body.dataset.scrollY;
    window.dispatchEvent(new Event('scroll'));
}

async function openForm(restorePending = false) {
    document.getElementById("formOverlay").classList.remove("hidden");
    document.getElementById("floatAddBtn").classList.add("hidden");
    document.getElementById("scrollTopBtn").classList.add("hidden");
    document.getElementById("floatAddBtn").dataset.formOpen = "1";
    lockScroll();
    document.getElementById("formError").classList.add("hidden");
    document.getElementById("formSuccess").classList.add("hidden");
    document.querySelectorAll(".cat-btn").forEach(b => b.classList.remove("active"));
    document.querySelector(".cat-btn[data-cat='milk']").classList.add("active");
    document.querySelectorAll(".cat-panel").forEach(p => p.classList.add("hidden"));
    document.getElementById("catMilk").classList.remove("hidden");
    const pending = restorePending ? await loadPendingFeed() : null;
    if (pending) {
        document.getElementById("logDate").value = pending.logDate || new Date().toISOString().split("T")[0];
        document.getElementById("logTime").value = pending.logTime || "";
        document.getElementById("preFeedWeightG").value = pending.preFeedWeightG ?? "";
        document.getElementById("postFeedWeightG").value = pending.postFeedWeightG ?? "";
        document.getElementById("fedBreast").checked = pending.fedBreast || false;
        document.getElementById("fedBottle").checked = pending.fedBottle || false;
        document.getElementById("statusWeightG").value = pending.statusWeightG ?? "";
        document.getElementById("measurementWeightG").value = pending.measurementWeightG ?? "";
        document.getElementById("dailySummary").value = pending.dailySummary || "";
        if (pending.quickTags) {
            document.querySelectorAll(".quick-tag").forEach(cb => {
                cb.checked = pending.quickTags.includes(cb.value);
            });
        }
    } else {
        const now = new Date();
        document.getElementById("logTime").value = `${String(now.getHours()).padStart(2,"0")}:${String(now.getMinutes()).padStart(2,"0")}`;
    }
    syncQuickTags();
}

function closeForm() {
    document.getElementById("formOverlay").classList.add("hidden");
    delete document.getElementById("floatAddBtn").dataset.formOpen;
    unlockScroll();
    document.getElementById("entryForm").reset();
    document.getElementById("dailySummary").value = "";
    document.getElementById("formError").classList.add("hidden");
    document.getElementById("formSuccess").classList.add("hidden");
    document.getElementById("logDate").value = new Date().toISOString().split("T")[0];
    document.querySelectorAll(".cat-btn").forEach(b => b.classList.remove("active"));
    document.querySelector(".cat-btn[data-cat='milk']").classList.add("active");
    document.querySelectorAll(".cat-panel").forEach(p => p.classList.add("hidden"));
    document.getElementById("catMilk").classList.remove("hidden");
}

function openGrowthForm() {
    document.getElementById("growthOverlay").classList.remove("hidden");
    lockScroll();
    document.getElementById("growthError").classList.add("hidden");
    document.getElementById("growthSuccess").classList.add("hidden");
}

function closeGrowthForm() {
    document.getElementById("growthOverlay").classList.add("hidden");
    unlockScroll();
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
    const getValue = id => { const v = document.getElementById(id).value.trim(); return v === "" ? null : v; };
    const getInt   = id => { const v = getValue(id); return v === null ? null : parseInt(v, 10); };

    // Handle words category separately
    const isWordsCat = !document.getElementById("catWords").classList.contains("hidden");
    if (isWordsCat) {
        const word = document.getElementById("entryWordInput").value.trim();
        if (!word) { errorEl.textContent = "Please enter a word or sound."; errorEl.classList.remove("hidden"); return; }
        const notedDate = getValue("logDate") || new Date().toISOString().split("T")[0];
        const notes = document.getElementById("entryWordNotes").value.trim() || null;
        const submitBtn = event.target.querySelector("button[type=submit]");
        submitBtn.disabled = true; submitBtn.textContent = "Saving...";
        try {
            const res = await fetch("/api/words", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ word, notedDate, notes }) });
            if (!res.ok) throw new Error("Failed to save word");
            successEl.textContent = "Word saved!"; successEl.classList.remove("hidden");
            setTimeout(() => { closeForm(); loadWordOfTheDay(); }, 1200);
        } catch (err) { errorEl.textContent = err.message; errorEl.classList.remove("hidden"); }
        finally { submitBtn.disabled = false; submitBtn.textContent = "Save Entry"; }
        return;
    }

    const pre = getInt("preFeedWeightG");
    const post = getInt("postFeedWeightG");
    const isMilkCat = !document.getElementById("catMilk").classList.contains("hidden");
    const fedBreast = document.getElementById("fedBreast").checked;
    const fedBottle = document.getElementById("fedBottle").checked;
    if (isMilkCat && (pre !== null || post !== null) && !fedBreast && !fedBottle) {
        errorEl.textContent = "Please select at least one feeding source (breast or bottle).";
        errorEl.classList.remove("hidden");
        return;
    }
    if (isMilkCat && (pre !== null || post !== null) && !(pre !== null && post !== null)) {
        const pending = {
            logDate: getValue("logDate"), logTime: getValue("logTime"),
            preFeedWeightG: pre, postFeedWeightG: post, fedBreast, fedBottle,
            statusWeightG: getValue("statusWeightG"),
            measurementWeightG: getValue("measurementWeightG"),
            dailySummary: getValue("dailySummary"),
            quickTags: [...document.querySelectorAll(".quick-tag:checked")].map(cb => cb.value),
        };
        savePendingFeed(pending);
        successEl.textContent = "Saved as pending — add the missing weight to complete.";
        successEl.classList.remove("hidden");
        setTimeout(() => closeForm(), 1500);
        return;
    }
    const submitBtn = event.target.querySelector("button[type=submit]");
    submitBtn.disabled = true; submitBtn.textContent = "Saving...";
    const milkTransfer = (pre !== null && post !== null && post > pre) ? post - pre : null;
    const payload = {
        logDate: getValue("logDate"), logTime: getValue("logTime"), dailySummary: getValue("dailySummary") || "",
        statusWeightG: getInt("statusWeightG"), preFeedWeightG: pre, postFeedWeightG: post,
        milkTransferG: milkTransfer, heightCm: null, headCm: null, measurementWeightG: getInt("measurementWeightG"),
        fedBreast, fedBottle,
        diaper: (() => {
            const kakis = [...document.querySelectorAll(".quick-tag:checked")].some(cb => cb.value === "kakis pelus");
            const pisis = [...document.querySelectorAll(".quick-tag:checked")].some(cb => cb.value === "pisis pelus");
            if (kakis && pisis) return "both";
            if (kakis) return "dirty";
            if (pisis) return "wet";
            return null;
        })(),
        sleepEvent: (() => {
            const checked = [...document.querySelectorAll(".quick-tag:checked")].map(cb => cb.value);
            if (checked.includes("ébredt")) return "woke_up";
            if (checked.includes("elaludt") || checked.includes("cicin elaludt")) return "fell_asleep";
            return null;
        })(),
        bathed: [...document.querySelectorAll(".quick-tag:checked")].some(cb => cb.value === "fürcsi"),
        milestone: [...document.querySelectorAll(".quick-tag:checked")].some(cb => cb.value === "🎉"),
    };
    try {
        const pendingEntry = await loadPendingFeed();
        const response = await fetch("/api/logs", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) });
        if (!response.ok) { const d = await response.json().catch(() => ({})); throw new Error(d.error || `Server error: ${response.status}`); }
        if (pendingEntry?.id) await fetch(`/api/pending-feed/${pendingEntry.id}`, { method: "DELETE" });
        localStorage.removeItem(PENDING_FEED_KEY);
        successEl.textContent = "Entry saved!"; successEl.classList.remove("hidden");
        setTimeout(() => { closeForm(); loadDashboard(); loadDiaperAlert(); renderPendingBanner(); }, 1200);
    } catch (err) { errorEl.textContent = err.message; errorEl.classList.remove("hidden"); }
    finally { submitBtn.disabled = false; submitBtn.textContent = "Save Entry"; }
}

function openEditForm(item) {
    document.getElementById("editOverlay").classList.remove("hidden");
    lockScroll();
    document.getElementById("editError").classList.add("hidden");
    document.getElementById("editSuccess").classList.add("hidden");
    document.getElementById("editId").value = item.id;
    document.getElementById("editLogDate").value = item.logDate ? item.logDate.substring(0, 10) : "";
    let timeVal = "";
    if (item.logTime) timeVal = item.logTime.includes("T") ? item.logTime.substring(11, 16) : item.logTime.substring(0, 5);
    document.getElementById("editLogTime").value = timeVal;
    document.getElementById("editHeightCm").value = item.heightCm || "";
    document.getElementById("editHeadCm").value = item.headCm || "";
    document.getElementById("editPreFeedWeightG").value = item.preFeedWeightG ?? "";
    document.getElementById("editPostFeedWeightG").value = item.postFeedWeightG ?? "";
    document.getElementById("editStatusWeightG").value = item.statusWeightG ?? "";
    document.getElementById("editMeasurementWeightG").value = item.measurementWeightG ?? "";
    document.getElementById("editFedBreast").checked = item.fedBreast || false;
    document.getElementById("editFedBottle").checked = item.fedBottle || false;
    const diaper = item.diaper || "";
    const bathed = item.bathed || false;
    const milestone = item.milestone || false;
    const sleepEvent = item.sleepEvent || "";
    document.querySelectorAll(".edit-quick-tag").forEach(cb => {
        if (cb.value === "kakis pelus") cb.checked = diaper === "dirty" || diaper === "both";
        else if (cb.value === "pisis pelus") cb.checked = diaper === "wet" || diaper === "both";
        else if (cb.value === "ébredt") cb.checked = sleepEvent === "woke_up";
        else if (cb.value === "elaludt" || cb.value === "cicin elaludt") cb.checked = sleepEvent === "fell_asleep" && cb.value === "elaludt";
        else if (cb.value === "fürcsi") cb.checked = bathed;
        else if (cb.value === "🎉") cb.checked = milestone;
        else cb.checked = false;
        cb.onchange = syncEditQuickTags;
    });
    document.getElementById("editDailySummary").value = item.dailySummary || "";
    document.getElementById("editFedBreast").onchange = syncEditQuickTags;
    document.getElementById("editFedBottle").onchange = syncEditQuickTags;
    document.getElementById("editPreFeedWeightG").oninput = syncEditQuickTags;
    document.getElementById("editPostFeedWeightG").oninput = syncEditQuickTags;
    document.getElementById("editStatusWeightG").oninput = syncEditQuickTags;
    document.getElementById("editMeasurementWeightG").oninput = syncEditQuickTags;
}

function closeEditForm() {
    document.getElementById("editOverlay").classList.add("hidden");
    unlockScroll();
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
    const getVal = id => { const v = document.getElementById(id).value.trim(); return v === "" ? null : v; };
    const getInt = id => { const v = getVal(id); return v === null ? null : parseInt(v, 10); };
    const getFloat = id => { const v = getVal(id); return v === null ? null : parseFloat(v); };
    const pre = getInt("editPreFeedWeightG");
    const post = getInt("editPostFeedWeightG");
    const milkTransfer = (pre !== null && post !== null && post > pre) ? post - pre : null;
    const kakis = [...document.querySelectorAll(".edit-quick-tag:checked")].some(cb => cb.value === "kakis pelus");
    const pisis = [...document.querySelectorAll(".edit-quick-tag:checked")].some(cb => cb.value === "pisis pelus");
    const checked = [...document.querySelectorAll(".edit-quick-tag:checked")].map(cb => cb.value);
    const payload = {
        logDate: getVal("editLogDate"),
        logTime: getVal("editLogTime"),
        dailySummary: getVal("editDailySummary") || "",
        heightCm: getFloat("editHeightCm"),
        headCm: getFloat("editHeadCm"),
        preFeedWeightG: pre,
        postFeedWeightG: post,
        milkTransferG: milkTransfer,
        statusWeightG: getInt("editStatusWeightG"),
        measurementWeightG: getInt("editMeasurementWeightG"),
        fedBreast: document.getElementById("editFedBreast").checked,
        fedBottle: document.getElementById("editFedBottle").checked,
        diaper: kakis && pisis ? "both" : kakis ? "dirty" : pisis ? "wet" : null,
        sleepEvent: checked.includes("ébredt") ? "woke_up" : (checked.includes("elaludt") || checked.includes("cicin elaludt")) ? "fell_asleep" : null,
        bathed: checked.includes("fürcsi"),
        milestone: checked.includes("🎉"),
    };
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
    tile.dataset.currentState = data.state;
    const fmt = min => { const h = Math.floor(min / 60); const m = min % 60; return h > 0 ? `${h}h ${m}m` : `${m}m`; };
    document.getElementById("todaySleep").textContent = ` ${fmt(data.sleepMin)}`;
    document.getElementById("todayAwake").textContent = ` ${fmt(data.awakeMin)}`;
}

function attachLongPress(el, onTrigger, duration = 800) {
    let bar = el.querySelector(".long-press-bar");
    if (!bar) {
        bar = document.createElement("div");
        bar.className = "long-press-bar";
        el.appendChild(bar);
    }
    el.classList.add("long-press-target");
    let timer = null;
    function start(e) {
        e.preventDefault();
        bar.classList.add("active");
        bar.style.transition = "none";
        bar.style.width = "0%";
        requestAnimationFrame(() => {
            requestAnimationFrame(() => {
                bar.style.transition = `width ${duration}ms linear`;
                bar.style.width = "100%";
            });
        });
        timer = setTimeout(() => { cancel(false); onTrigger(); }, duration);
    }
    function cancel(retract = true) {
        clearTimeout(timer);
        if (retract) {
            bar.style.transition = "width 0.15s ease";
            bar.style.width = "0%";
            setTimeout(() => bar.classList.remove("active"), 150);
        } else {
            bar.classList.remove("active");
            bar.style.width = "0%";
        }
    }
    el.addEventListener("mousedown", start);
    el.addEventListener("touchstart", start, { passive: false });
    el.addEventListener("mouseup", () => cancel());
    el.addEventListener("mouseleave", () => cancel());
    el.addEventListener("touchend", () => cancel());
    el.addEventListener("touchcancel", () => cancel());
}

// WHO girls weight-for-age (grams): index = month 0..60, [median, -2SD, +2SD]
const WHO_GIRLS_WEIGHT = [
    [3232,2400,4300],[4176,3200,5400],[5128,3900,6600],[5990,4600,7600],[6694,5100,8400],
    [7268,5600,9000],[7775,6000,9500],[8218,6300,10000],[8613,6600,10400],[8986,6900,10900],
    [9325,7100,11200],[9637,7300,11500],[9924,7500,11800],[10189,7700,12100],[10436,7900,12400],
    [10668,8100,12700],[10887,8300,12900],[11095,8400,13200],[11294,8600,13400],[11486,8700,13600],
    [11671,8900,13900],[11851,9000,14100],[12027,9100,14300],[12199,9300,14500],[12368,9400,14700],
    [12534,9500,14900],[12697,9700,15100],[12858,9800,15300],[13017,9900,15500],[13174,10000,15700],
    [13329,10100,15900],[13482,10200,16100],[13634,10300,16300],[13784,10400,16500],[13932,10500,16700],
    [14079,10600,16900],[14224,10700,17100],[14368,10800,17300],[14510,10900,17500],[14651,11000,17700],
    [14790,11100,17900],[14928,11200,18100],[15065,11300,18300],[15200,11400,18500],[15334,11500,18700],
    [15467,11600,18900],[15599,11700,19100],[15730,11800,19300],[15860,11900,19500],[15989,12000,19700],
    [16117,12100,19900],[16244,12200,20100],[16370,12300,20300],[16495,12400,20500],[16619,12500,20700],
    [16742,12600,20900],[16864,12700,21100],[16985,12800,21300],[17105,12900,21500],[17224,13000,21700],
    [17342,13100,21900]
];

async function loadWeightChart() {
    const from = document.getElementById("weightFromDate").value;
    const to = document.getElementById("weightToDate").value;
    const [data, birthRes] = await Promise.all([
        fetch(`/api/weights?from=${from}&to=${to}`).then(r => r.json()).catch(() => []),
        fetch("/api/settings/birth-date").catch(() => null)
    ]);
    const safeData = data ?? [];
    const ctx = document.getElementById("weightChart");
    if (weightChart) weightChart.destroy();

    const datasets = [{ label: "Weight (g)", data: safeData.map(i => i.weight), borderColor: "#7b174e", backgroundColor: "rgba(235, 63, 126, 0.88)", borderWidth: 3, pointRadius: 4, pointHoverRadius: 7, tension: 0.3, fill: true }];

    if (birthRes && birthRes.ok) {
        const { value: birthDateStr } = await birthRes.json();
        const birth = new Date(birthDateStr);
        const labels = safeData.map(i => i.date.substring(0, 10));
        const toMonth = dateStr => Math.floor((new Date(dateStr) - birth) / (30.4375 * 86400000));
        const whoMedian = [], minus2 = [], plus2 = [];
        labels.forEach(d => {
            const m = Math.max(0, Math.min(60, toMonth(d)));
            whoMedian.push(WHO_GIRLS_WEIGHT[m][0]);
            minus2.push(WHO_GIRLS_WEIGHT[m][1]);
            plus2.push(WHO_GIRLS_WEIGHT[m][2]);
        });
        datasets.push({ label: "WHO median", data: whoMedian, borderColor: "#16a34a", borderWidth: 2, borderDash: [6,3], pointRadius: 0, fill: false, tension: 0.3 });
        datasets.push({ label: "WHO -2SD", data: minus2, borderColor: "#f59e0b", borderWidth: 1, borderDash: [3,3], pointRadius: 0, fill: false, tension: 0.3 });
        datasets.push({ label: "WHO +2SD", data: plus2, borderColor: "#f59e0b", borderWidth: 1, borderDash: [3,3], pointRadius: 0, fill: false, tension: 0.3 });
    }

    weightChart = new Chart(ctx, {
        type: "line",
        data: { labels: safeData.map(i => i.date.substring(0, 10)), datasets },
        options: { responsive: true, plugins: { datalabels: { display: false }, tooltip: { callbacks: { label: ctx => `${ctx.parsed.y} g` } } }, scales: { y: { title: { display: true, text: "Weight in grams" } }, x: { title: { display: true, text: "Date" } } } }
    });
    setupChartScroll("weightChart", "weightFromDate", "weightToDate", loadWeightChart);
}

async function loadStatusWeightChart() {
    const from = document.getElementById("statusWeightFromDate").value;
    const to = document.getElementById("statusWeightToDate").value;
    const data = await fetch(`/api/status-weights?from=${from}&to=${to}`).then(r => r.json()).catch(() => []) ?? [];
    const ctx = document.getElementById("statusWeightChart");
    if (statusWeightChart) statusWeightChart.destroy();
    statusWeightChart = new Chart(ctx, {
        type: "line",
        data: { labels: data.map(i => i.date.substring(0, 10)), datasets: [{ label: "Status weight (g)", data: data.map(i => i.weight), borderColor: "#f472b6", backgroundColor: "rgba(244, 114, 182, 0.15)", borderWidth: 3, pointRadius: 4, pointHoverRadius: 7, tension: 0.3, fill: true }] },
        options: { responsive: true, plugins: { datalabels: { display: false }, tooltip: { callbacks: { label: ctx => `${ctx.parsed.y} g` } } }, scales: { y: { title: { display: true, text: "Weight in grams" } }, x: { title: { display: true, text: "Date" } } } }
    });
    setupChartScroll("statusWeightChart", "statusWeightFromDate", "statusWeightToDate", loadStatusWeightChart);
}

async function loadSleepAwakeChart() {
    const from = document.getElementById("sleepFromDate").value;
    const to = document.getElementById("sleepToDate").value;
    const data = await fetch(`/api/sleep-awake?from=${from}&to=${to}`).then(r => r.json()).catch(() => []) ?? [];
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
    setupChartScroll("sleepAwakeChart", "sleepFromDate", "sleepToDate", loadSleepAwakeChart);
}

async function loadMilkConsumedChart() {
    const from = document.getElementById("milkFromDate").value;
    const to = document.getElementById("milkToDate").value;
    const data = await fetch(`/api/milk-consumed?from=${from}&to=${to}`).then(r => r.json()).catch(() => []) ?? [];
    const ctx = document.getElementById("milkConsumedChart");
    if (milkConsumedChart) milkConsumedChart.destroy();
    milkConsumedChart = new Chart(ctx, {
        type: "bar",
        data: { labels: data.map(i => i.date.substring(0, 10)), datasets: [{ label: "Milk consumed (g)", data: data.map(i => i.milkConsumedG), backgroundColor: "#f9a8d4", borderColor: "#db2777", borderWidth: 1 }] },
        options: {
            responsive: true,
            plugins: { datalabels: { anchor: "center", align: "center", color: "#9d174d", font: { weight: "700", size: 14 }, rotation: () => window.innerWidth < 600 ? -90 : 0, formatter: v => `${v} g` }, tooltip: { callbacks: { label: ctx => `${ctx.parsed.y} g` } } },
            scales: { y: { beginAtZero: true, title: { display: true, text: "Milk consumed in grams" } }, x: { title: { display: true, text: "Date" } } }
        }
    });
    setupChartScroll("milkConsumedChart", "milkFromDate", "milkToDate", loadMilkConsumedChart);
}

function setupChartScroll(canvasId, fromId, toId, reloadFn) {
    const el = document.getElementById(canvasId);
    if (el._scrollAbort) el._scrollAbort.abort();
    const controller = new AbortController();
    el._scrollAbort = controller;
    const signal = controller.signal;

    let touchStartX = null;
    let scrollCooldown = false;

    function shift(forward) {
        if (scrollCooldown) return;
        scrollCooldown = true;
        setTimeout(() => scrollCooldown = false, 300);

        const fmt = d => d.toISOString().split("T")[0];
        const todayStr = fmt(new Date());
        const from = new Date(document.getElementById(fromId).value + "T12:00:00");
        const to = new Date(document.getElementById(toId).value + "T12:00:00");
        const rangeDays = Math.round((to - from) / 86400000);
        const step = Math.max(1, Math.round(rangeDays / 3));
        const delta = forward ? step : -step;

        let newTo = new Date(to); newTo.setDate(newTo.getDate() + delta);
        let newFrom = new Date(newTo); newFrom.setDate(newTo.getDate() - rangeDays);

        if (forward && fmt(newTo) > todayStr) {
            newTo = new Date(todayStr + "T12:00:00");
            newFrom = new Date(newTo); newFrom.setDate(newTo.getDate() - rangeDays);
        }

        if (!forward && fmt(newFrom) < "2020-01-01") {
            newFrom = new Date("2020-01-01T12:00:00");
            newTo = new Date(newFrom); newTo.setDate(newFrom.getDate() + rangeDays);
        }

        const currentFrom = document.getElementById(fromId).value;
        const currentTo = document.getElementById(toId).value;
        if (fmt(newFrom) === currentFrom && fmt(newTo) === currentTo) return;
        document.getElementById(fromId).value = fmt(newFrom);
        document.getElementById(toId).value = fmt(newTo);
        reloadFn();
    }

    el.addEventListener("wheel", e => {
        e.preventDefault();
        shift(e.deltaX > 0 || e.deltaY > 0);
    }, { passive: false, signal });

    el.addEventListener("touchstart", e => { touchStartX = e.touches[0].clientX; }, { passive: true, signal });
    el.addEventListener("touchend", e => {
        if (touchStartX === null) return;
        const diff = touchStartX - e.changedTouches[0].clientX;
        if (Math.abs(diff) > 40) shift(diff > 0);
        touchStartX = null;
    }, { passive: true, signal });
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
        dateCell.textContent = item.logDate ? item.logDate.substring(0, 10) + (item.logTime ? " " + (item.logTime.includes("T") ? item.logTime.substring(11, 16) : item.logTime.substring(0, 5)) : "") : "-";
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

const EVENT_CATEGORIES = {
    orvos:       { icon: "🩺", label: "Orvos",        color: "#dbeafe", border: "#93c5fd" },
    oltas:       { icon: "💉", label: "Oltás",         color: "#fce7f3", border: "#f9a8d4" },
    gyogyszer:   { icon: "💊", label: "Gyógyszer",     color: "#fef3c7", border: "#fcd34d" },
    meres:       { icon: "⚖️", label: "Mérés",         color: "#f0fdf4", border: "#86efac" },
    merfoldko:   { icon: "🎉", label: "Mérföldkő",    color: "#fdf4ff", border: "#e879f9" },
    nevnap:      { icon: "🌸", label: "Névnap",        color: "#fff1f2", border: "#fda4af" },
    szuletesnap: { icon: "🎂", label: "Születésnap",   color: "#fff7ed", border: "#fdba74" },
    lista:       { icon: "🛒", label: "Lista",         color: "#f0fdf4", border: "#6ee7b7" },
    egyeb:       { icon: "📅", label: "Egyéb",         color: "#f8fafc", border: "#cbd5e1" },
};

let allEvents = [];
let activeEventFilter = null;

function openEventForm(event = null) {
    document.getElementById("eventOverlay").classList.remove("hidden");
    lockScroll();
    document.getElementById("eventError").classList.add("hidden");
    document.getElementById("eventSuccess").classList.add("hidden");
    document.getElementById("eventForm").reset();
    const now = new Date();
    if (event) {
        document.getElementById("eventId").value = event.id;
        document.getElementById("eventDate").value = event.eventDate;
        document.getElementById("eventTime").value = event.eventTime || "";
        document.getElementById("eventDuration").value = event.durationMin;
        document.getElementById("eventTitle").value = event.title;
        document.getElementById("eventNotes").value = event.notes || "";
        document.getElementById("eventRecurring").value = event.recurring || "none";
        document.getElementById("eventAllDay").checked = event.allDay || false;
        const radio = document.querySelector(`input[name="eventCategory"][value="${event.category}"]`);
        if (radio) radio.checked = true;
        document.querySelector(".modal-header h2").textContent = "📅 Edit Event";
    } else {
        document.getElementById("eventId").value = "";
        document.getElementById("eventDate").value = now.toISOString().split("T")[0];
        document.getElementById("eventTime").value = `${String(now.getHours()).padStart(2,"0")}:${String(now.getMinutes()).padStart(2,"0")}`;
        document.getElementById("eventDuration").value = 60;
        document.getElementById("eventRecurring").value = "none";
        document.getElementById("eventAllDay").checked = false;
        document.querySelector(".modal-header h2").textContent = "📅 Add Event";
    }
    document.querySelectorAll("input[name='eventCategory']").forEach(r => {
        r.addEventListener("change", () => {
            const autoRecurring = ["nevnap", "szuletesnap"].includes(r.value);
            document.getElementById("eventRecurring").value = autoRecurring ? "yearly" : "none";
        });
    });
    document.getElementById("eventAllDay").addEventListener("change", e => {
        document.getElementById("eventTime").disabled = e.target.checked;
        document.getElementById("eventDuration").disabled = e.target.checked;
        if (e.target.checked) document.getElementById("eventTime").value = "";
    });
}

function closeEventForm() {
    document.getElementById("eventOverlay").classList.add("hidden");
    unlockScroll();
}

async function submitEvent(e) {
    e.preventDefault();
    const errorEl = document.getElementById("eventError");
    const successEl = document.getElementById("eventSuccess");
    errorEl.classList.add("hidden"); successEl.classList.add("hidden");
    const submitBtn = e.target.querySelector("button[type=submit]");
    submitBtn.disabled = true; submitBtn.textContent = "Saving...";
    const id = document.getElementById("eventId").value;
    const payload = {
        title: document.getElementById("eventTitle").value.trim(),
        category: document.querySelector("input[name='eventCategory']:checked")?.value || "egyeb",
        eventDate: document.getElementById("eventDate").value,
        eventTime: document.getElementById("eventTime").value || null,
        durationMin: parseInt(document.getElementById("eventDuration").value) || 60,
        notes: document.getElementById("eventNotes").value.trim() || null,
        recurring: document.getElementById("eventRecurring").value,
        allDay: document.getElementById("eventAllDay").checked,
    };
    try {
        const url = id ? `/api/events/${id}` : "/api/events";
        const method = id ? "PUT" : "POST";
        const res = await fetch(url, { method, headers: { "Content-Type": "application/json" }, body: JSON.stringify(payload) });
        if (!res.ok) { const d = await res.json().catch(() => ({})); throw new Error(d.error || `Server error: ${res.status}`); }
        successEl.textContent = "Saved!"; successEl.classList.remove("hidden");
        setTimeout(() => { closeEventForm(); loadEvents(); }, 1000);
    } catch (err) { errorEl.textContent = err.message; errorEl.classList.remove("hidden"); }
    finally { submitBtn.disabled = false; submitBtn.textContent = "Save Event"; }
}

async function loadEvents() {
    allEvents = await fetch("/api/events").then(r => r.json()).catch(() => []);
    renderEvents();
}

function renderEvents() {
    const now = new Date();
    const expanded = [];
    const baseEvents = activeEventFilter && activeEventFilter !== "__archived" ? allEvents.filter(e => e.category === activeEventFilter) : allEvents;
    const oneYearOut = new Date(now); oneYearOut.setFullYear(oneYearOut.getFullYear() + 1);
    baseEvents.forEach(e => {
        if (!e.recurring || e.recurring === "none") { expanded.push(e); return; }
        const origin = new Date(e.eventDate);
        let cur = new Date(origin);
        while (cur > now) {
            if (e.recurring === "daily") cur.setDate(cur.getDate() - 1);
            else if (e.recurring === "weekly") cur.setDate(cur.getDate() - 7);
            else if (e.recurring === "monthly") cur.setMonth(cur.getMonth() - 1);
            else if (e.recurring === "yearly") cur.setFullYear(cur.getFullYear() - 1);
        }
        while (cur <= oneYearOut) {
            const dateStr = cur.toISOString().split("T")[0];
            expanded.push({ ...e, eventDate: dateStr });
            if (e.recurring === "daily") cur.setDate(cur.getDate() + 1);
            else if (e.recurring === "weekly") cur.setDate(cur.getDate() + 7);
            else if (e.recurring === "monthly") cur.setMonth(cur.getMonth() + 1);
            else if (e.recurring === "yearly") cur.setFullYear(cur.getFullYear() + 1);
        }
    });
    const byId = new Map();
    expanded.forEach(e => {
        const start = new Date(`${e.eventDate}T${e.eventTime || "00:00"}`);
        const existing = byId.get(e.id);
        if (!existing) { byId.set(e.id, e); return; }
        const existingStart = new Date(`${existing.eventDate}T${existing.eventTime || "00:00"}`);
        const eIsFuture = start >= now;
        const exIsFuture = existingStart >= now;
        if (eIsFuture && !exIsFuture) { byId.set(e.id, e); return; }
        if (!eIsFuture && exIsFuture) return;
        if (eIsFuture && exIsFuture && start < existingStart) { byId.set(e.id, e); return; }
        if (!eIsFuture && !exIsFuture && start > existingStart) { byId.set(e.id, e); }
    });
    const events = [...byId.values()].sort((a, b) => a.eventDate.localeCompare(b.eventDate));
    const ongoing = events.filter(e => {
        const start = new Date(`${e.eventDate}T${e.eventTime || "00:00"}`);
        const end = new Date(start.getTime() + (e.allDay ? 86400000 : e.durationMin * 60000));
        return start <= now && now <= end;
    });
    const upcoming = events.filter(e => new Date(`${e.eventDate}T${e.eventTime || "00:00"}`) > now);
    const ongoingIds = new Set(ongoing.map(e => e.id));
    const upcomingIds = new Set(upcoming.map(e => e.id));
    const past = events.filter(e => {
        const start = new Date(`${e.eventDate}T${e.eventTime || "00:00"}`);
        const end = new Date(start.getTime() + (e.allDay ? 86400000 : e.durationMin * 60000));
        return end < now && !ongoingIds.has(e.id) && !upcomingIds.has(e.id);
    });
    document.getElementById("eventsOngoing").innerHTML = ongoing.length ? `<div class="events-section-label">🔴 Folyamatban</div>` + ongoing.map(e => eventCard(e, "ongoing")).join("") : "";
    document.getElementById("eventsUpcoming").innerHTML = upcoming.length ? `<div class="events-section-label">⏳ Közelgő</div>` + upcoming.slice(0, 1).map(e => eventCard(e, "upcoming")).join("") : "";
    const prominentKeys = new Set([
        ...ongoing.map(e => `${e.id}-${e.eventDate}`),
        ...upcoming.slice(0, 1).map(e => `${e.id}-${e.eventDate}`),
    ]);
    const rest = events.filter(e => !prominentKeys.has(`${e.id}-${e.eventDate}`));
    const restPast = rest.filter(e => past.find(p => p.id === e.id && p.eventDate === e.eventDate));
    const restFuture = rest.filter(e => !restPast.find(p => p.id === e.id && p.eventDate === e.eventDate));
    renderEventFilters();
    if (activeEventFilter === "__archived") {
        document.getElementById("eventsPast").innerHTML = "";
        document.getElementById("eventsAll").innerHTML = past.length
            ? `<div class="events-section-label">🗄️ Archív</div>` + [...past].reverse().map(e => eventCard(e, "past")).join("")
            : `<div class="events-section-label">Nincs archívált esemény</div>`;
    } else {
        document.getElementById("eventsPast").innerHTML = past.length
            ? `<div class="events-section-label">✅ Legutóbbi</div>` + past.slice(-2).map(e => eventCard(e, "past")).join("")
            : "";
        document.getElementById("eventsAll").innerHTML = restFuture.length
            ? `<div class="events-section-label">📋 Többi</div>` + [...restFuture].reverse().map(e => eventCard(e, "all")).join("")
            : "";
    }
}

function eventCard(e, context) {
    const cat = EVENT_CATEGORIES[e.category] || EVENT_CATEGORIES.egyeb;
    const timeStr = e.allDay ? "Egész napos" : (e.eventTime ? e.eventTime.substring(0,5) : "");
    const recurStr = e.recurring && e.recurring !== "none" ? ` · 🔄 ${e.recurring}` : "";
    const prominent = context === "ongoing" || context === "upcoming";
    const isPast = context === "past" || (context === "all" && new Date(`${e.eventDate}T${e.eventTime || "00:00"}`) < new Date());
    const itemsHtml = e.category === "lista" && e.items && e.items.length > 0
        ? `<ul class="checklist-items">${e.items.map(item =>
            `<li class="checklist-item ${item.checked ? "checked" : ""}">
                <label><input type="checkbox" ${item.checked ? "checked" : ""} onchange="toggleEventItem(${item.id}, this.checked, ${e.id})" /><span>${item.text}</span></label>
                <button class="row-btn" onclick="deleteEventItem(${item.id}, ${e.id})">🗑️</button>
            </li>`).join("")}</ul>` : "";
    const addItemHtml = e.category === "lista"
        ? `<div class="checklist-add-item">
            <input class="checklist-new-input" type="text" placeholder="Add item..." id="newEventItem-${e.id}" onkeydown="if(event.key==='Enter'){addEventItem(${e.id});}" />
            <button class="btn-secondary" onclick="addEventItem(${e.id})">Add</button>
           </div>` : "";
    return `<div class="event-card ${prominent ? "event-card--prominent" : ""} ${isPast ? "event-card--past" : ""}" style="background:${cat.color};border-color:${cat.border}">
        <div class="event-card-icon">${cat.icon}</div>
        <div class="event-card-body">
            <div class="event-card-title">${e.title}</div>
            <div class="event-card-meta">${e.eventDate}${timeStr ? " · " + timeStr : ""}${e.allDay ? "" : " · " + e.durationMin + " min"} · ${cat.label}${recurStr}</div>
            ${e.notes ? `<div class="event-card-notes">${e.notes}</div>` : ""}
            ${itemsHtml}
            ${addItemHtml}
        </div>
        <div class="event-card-actions">
            <button class="row-btn" onclick="openEventForm(${JSON.stringify(e).split('"').join('&quot;')})">✏️</button>
            <button class="row-btn" onclick="deleteEvent(${e.id})">🗑️</button>
        </div>
    </div>`;
}

function renderEventFilters() {
    const bar = document.getElementById("eventFilterBar");
    const cats = [...new Set(allEvents.map(e => e.category))];
    bar.innerHTML = `<button class="event-filter-btn ${!activeEventFilter ? "active" : ""}" onclick="setEventFilter(null)">Összes</button>` +
        cats.map(c => {
            const cat = EVENT_CATEGORIES[c] || EVENT_CATEGORIES.egyeb;
            return `<button class="event-filter-btn ${activeEventFilter === c ? "active" : ""}" onclick="setEventFilter('${c}')">${cat.icon} ${cat.label}</button>`;
        }).join("") +
        `<button class="event-filter-btn ${activeEventFilter === "__archived" ? "active" : ""}" onclick="setEventFilter('__archived')">🗄️ Archív</button>`;
}

function setEventFilter(cat) {
    activeEventFilter = cat;
    renderEvents();
}

async function deleteEvent(id) {
    if (!confirm("Delete this event?")) return;
    await fetch(`/api/events/${id}`, { method: "DELETE" });
    allEvents = allEvents.filter(e => e.id !== id);
    renderEvents();
}

async function addEventItem(eventId) {
    const input = document.getElementById(`newEventItem-${eventId}`);
    const text = input.value.trim();
    if (!text) return;
    const item = await fetch(`/api/events/${eventId}/items`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ text }),
    }).then(r => r.json());
    const ev = allEvents.find(e => e.id === eventId);
    if (ev) { if (!ev.items) ev.items = []; ev.items.push(item); }
    renderEvents();
    const newInput = document.getElementById(`newEventItem-${eventId}`);
    if (newInput) newInput.focus();
}

async function toggleEventItem(itemId, checked, eventId) {
    await fetch(`/api/checklist-items/${itemId}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ checked }),
    });
    const ev = allEvents.find(e => e.id === eventId);
    if (ev && ev.items) {
        const item = ev.items.find(i => i.id === itemId);
        if (item) item.checked = checked;
    }
    renderEvents();
}

async function deleteEventItem(itemId, eventId) {
    await fetch(`/api/checklist-items/${itemId}`, { method: "DELETE" });
    const ev = allEvents.find(e => e.id === eventId);
    if (ev && ev.items) ev.items = ev.items.filter(i => i.id !== itemId);
    renderEvents();
}

async function loadWordOfTheDay() {
    const res = await fetch("/api/words/random").catch(() => null);
    if (!res || !res.ok) return;
    const w = await res.json();
    if (!w.word) return;
    document.getElementById("wordText").textContent = w.word;
    document.getElementById("wordOfTheDay").classList.remove("hidden");
}

async function loadWordsList() {
    const res = await fetch("/api/words").catch(() => null);
    if (!res || !res.ok) return;
    const words = await res.json();
    const container = document.getElementById("wordsList");
    if (!words.length) { container.innerHTML = "<p style='color:#888;font-size:0.85rem'>No words yet.</p>"; return; }
    container.innerHTML = words.map(w =>
        `<div class="word-item">
            <span class="word-item-text">${w.word}</span>
            <span class="word-item-date">${w.notedDate}</span>
            ${w.notes ? `<span class="word-item-notes">${w.notes}</span>` : ""}
            <button onclick="deleteWord(${w.id})" class="btn-word-delete">&times;</button>
        </div>`
    ).join("");
}

async function deleteWord(id) {
    await fetch(`/api/words/${id}`, { method: "DELETE" });
    loadWordsList();
    loadWordOfTheDay();
}

async function toggleWordsList() {
    const list = document.getElementById("wordsList");
    const btn = document.getElementById("toggleWordsListButton");
    const isHidden = list.classList.contains("hidden");
    if (isHidden) {
        await loadWordsList();
        list.classList.remove("hidden");
        btn.textContent = "🙈 Hide words";
    } else {
        list.classList.add("hidden");
        btn.textContent = "📋 Show all words";
    }
}

function openWordForm() {
    document.getElementById("wordOverlay").classList.remove("hidden");
    document.getElementById("wordDateInput").value = new Date().toISOString().split("T")[0];
}

function closeWordForm() {
    document.getElementById("wordOverlay").classList.add("hidden");
    document.getElementById("wordForm").reset();
    document.getElementById("wordsList").classList.add("hidden");
    document.getElementById("toggleWordsListButton").textContent = "📋 Show all words";
}

async function loadDiaperAlert() {
    const data = await fetch("/api/diaper-alert").then(r => r.json()).catch(() => null);
    if (!data) return;
    const banner = document.getElementById("diaperAlertBanner");
    const { consecutiveDaysWithoutDirty } = data;

    if (consecutiveDaysWithoutDirty >= 1) {
        banner.className = "diaper-alert-banner alert";
        banner.textContent = `🚨 kakis pelenka nélküli napok száma: ${consecutiveDaysWithoutDirty} !`;
    } else {
        banner.className = "diaper-alert-banner hidden";
    }
}


async function subscribeToPush() {
    if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
        alert('Push not supported on this browser/device');
        return false;
    }
    const permission = await Notification.requestPermission();
    if (permission !== 'granted') {
        alert('Permission denied: ' + permission);
        return false;
    }
    const reg = await navigator.serviceWorker.ready;
    const existing = await reg.pushManager.getSubscription();
    if (existing) return true;
    const res = await fetch('/api/push-vapid-key').catch(() => null);
    if (!res || !res.ok) {
        alert('Failed to get VAPID key');
        return false;
    }
    const { publicKey } = await res.json();
    const sub = await reg.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: publicKey,
    }).catch(e => { alert('Subscribe error: ' + e.message); return null; });
    if (!sub) return false;
    await fetch('/api/push-subscribe', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(sub),
    });
    return true;
}

async function initPushNotifications() {
    if (!('serviceWorker' in navigator) || !('PushManager' in window)) return;
    const reg = await navigator.serviceWorker.ready;
    const existing = await reg.pushManager.getSubscription();
    const isIOS = /iphone|ipad|ipod/i.test(navigator.userAgent);
    if (isIOS) {
        // iOS requires user gesture; show button if not yet subscribed
        if (!existing) document.getElementById('pushEnableBtn').classList.remove('hidden');
        return;
    }
    if (existing) {
        // Re-send to server in case it was lost (e.g. after server restart/DB wipe)
        await fetch('/api/push-subscribe', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(existing),
        });
        return;
    }
    await subscribeToPush();
}

document.addEventListener('DOMContentLoaded', () => {
    initPushNotifications();
});

document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'visible' && !document.querySelector('.overlay:not(.hidden)')) location.reload();
});


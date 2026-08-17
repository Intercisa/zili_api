let currentState = 'awake';

function showToast(msg) {
    const t = document.getElementById('toast');
    t.textContent = msg;
    t.classList.add('show');
    setTimeout(() => t.classList.remove('show'), 2200);
}

function flash(btn) {
    btn.classList.remove('flash');
    void btn.offsetWidth;
    btn.classList.add('flash');
}

function attachLongPress(el, onTrigger, duration = 700) {
    const bar = el.querySelector('.long-press-bar');
    let timer = null;
    function start(e) {
        e.preventDefault();
        bar.style.transition = 'none';
        bar.style.width = '0%';
        bar.classList.add('active');
        requestAnimationFrame(() => requestAnimationFrame(() => {
            bar.style.transition = `width ${duration}ms linear`;
            bar.style.width = '100%';
        }));
        timer = setTimeout(() => { cancel(false); onTrigger(); }, duration);
    }
    function cancel(retract = true) {
        clearTimeout(timer);
        if (retract) {
            bar.style.transition = 'width 0.15s ease';
            bar.style.width = '0%';
            setTimeout(() => bar.classList.remove('active'), 150);
        } else {
            bar.classList.remove('active');
            bar.style.width = '0%';
        }
    }
    el.addEventListener('mousedown', start);
    el.addEventListener('touchstart', start, { passive: false });
    el.addEventListener('mouseup', () => cancel());
    el.addEventListener('mouseleave', () => cancel());
    el.addEventListener('touchend', () => cancel());
    el.addEventListener('touchcancel', () => cancel());
}

async function logDiaper(diaper) {
    const now = new Date();
    const logDate = `${now.getFullYear()}-${String(now.getMonth()+1).padStart(2,'0')}-${String(now.getDate()).padStart(2,'0')}`;
    const logTime = `${String(now.getHours()).padStart(2,'0')}:${String(now.getMinutes()).padStart(2,'0')}`;
    const summary = diaper === 'wet' ? 'pisis pelus' : diaper === 'dirty' ? 'kakis pelus' : 'kakis pelus, pisis pelus';
    const label = diaper === 'wet' ? '💧 pisis pelus' : diaper === 'dirty' ? '💩 kakis pelus' : '🌊💩 mindkettő';
    const res = await fetch('/api/logs', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ logDate, logTime, dailySummary: summary, diaper }),
    }).catch(() => null);
    if (!res || !res.ok) { showToast('❌ hiba történt'); return; }
    showToast(label + ' mentve ✓');
}

async function toggleSleep() {
    const now = new Date();
    const logDate = `${now.getFullYear()}-${String(now.getMonth()+1).padStart(2,'0')}-${String(now.getDate()).padStart(2,'0')}`;
    const logTime = `${String(now.getHours()).padStart(2,'0')}:${String(now.getMinutes()).padStart(2,'0')}`;
    const isSleep = currentState === 'sleep';
    const sleepEvent = isSleep ? 'woke_up' : 'fell_asleep';
    const dailySummary = isSleep ? 'ébredt' : 'elaludt';
    const res = await fetch('/api/logs', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ logDate, logTime, dailySummary, sleepEvent }),
    }).catch(() => null);
    if (!res || !res.ok) { showToast('❌ hiba történt'); return; }
    showToast(isSleep ? '🌸 ébredt!' : '😴 elaludt!');
    await updateStatus();
}

async function updateStatus() {
    const data = await fetch('/api/current-status').then(r => r.json()).catch(() => null);
    if (!data) return;

    currentState = data.state;
    const sleeping = data.state === 'sleep';

    document.body.classList.toggle('sleeping', sleeping);
    document.getElementById('statusEmoji').textContent = sleeping ? '😴' : '🌸';
    document.getElementById('statusLabel').textContent = sleeping ? 'alszik' : 'ébren van';
    document.getElementById('statusDuration').textContent = data.duration || '—';
    document.getElementById('sleepBtnEmoji').textContent = sleeping ? '🌸' : '😴';
    document.getElementById('sleepBtnLabel').textContent = sleeping ? 'ébredt' : 'elaludt';
}

attachLongPress(document.getElementById('btnPisis'), async () => {
    flash(document.getElementById('btnPisis'));
    await logDiaper('wet');
});
attachLongPress(document.getElementById('btnKakis'), async () => {
    flash(document.getElementById('btnKakis'));
    await logDiaper('dirty');
});
attachLongPress(document.getElementById('btnBoth'), async () => {
    flash(document.getElementById('btnBoth'));
    await logDiaper('both');
});
attachLongPress(document.getElementById('btnSleep'), async () => {
    flash(document.getElementById('btnSleep'));
    await toggleSleep();
});

updateStatus();
setInterval(updateStatus, 10000);

document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'visible') updateStatus();
});


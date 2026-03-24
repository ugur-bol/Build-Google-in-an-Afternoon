// =========================================
//  Dashboard Client
// =========================================

const API = '';

// ---- State Polling ----
async function pollState() {
    try {
        const resp = await fetch(`${API}/api/state`);
        const data = await resp.json();
        updateMetrics(data);
        updateCrawlList(data.jobs || []);
        updatePauseControls(data.status);
    } catch (e) {
        console.error('state poll error:', e);
    }
}

function updateMetrics(d) {
    setText('m-status', d.status);
    setText('m-processed', d.processed);
    setText('m-queued', d.queued);
    setText('m-workers', d.active_workers);
    setText('m-failed', d.failed);
    setText('m-skipped', d.skipped_visited);
    setText('m-maxq', d.max_queue_depth);
    setText('m-throttled', d.throttled ? 'Yes' : 'No');
    setText('m-words', d.indexed_words || 0);
    setText('m-postings', d.total_postings || 0);

    const statusEl = document.getElementById('m-status');
    statusEl.className = 'metric-value status-' + d.status;
}

// ---- Crawl History ----
function updateCrawlList(jobs) {
    const container = document.getElementById('crawl-list');
    if (!jobs || jobs.length === 0) {
        container.innerHTML = '<p class="empty-msg">No crawls yet. Start one from the panel →</p>';
        return;
    }

    // Reverse so newest is on top
    const sorted = [...jobs].reverse();
    container.innerHTML = sorted.map(j => {
        const time = new Date(j.started_at).toLocaleTimeString();
        const statusClass = 'crawl-status-' + j.status;
        const origin = escapeHtml(j.origin);
        const shortOrigin = origin.replace(/^https?:\/\//, '').replace(/\/$/, '');
        return `
            <div class="crawl-card">
                <div class="crawl-origin" title="${origin}">${shortOrigin}</div>
                <div class="crawl-meta">
                    <span class="crawl-status ${statusClass}">${j.status}</span>
                    <span>depth: ${j.max_depth}</span>
                    <span>pages: ${j.pages}</span>
                    <span>${time}</span>
                </div>
            </div>
        `;
    }).join('');
}

// ---- Pause/Resume Controls ----
function updatePauseControls(status) {
    const controls = document.getElementById('pause-controls');
    const btnPause = document.getElementById('btn-pause');
    const btnResume = document.getElementById('btn-resume');

    if (status === 'running') {
        controls.style.display = 'flex';
        btnPause.style.display = '';
        btnResume.style.display = 'none';
    } else if (status === 'paused') {
        controls.style.display = 'flex';
        btnPause.style.display = 'none';
        btnResume.style.display = '';
    } else {
        controls.style.display = 'none';
    }
}

async function pauseCrawl() {
    try {
        await fetch(`${API}/api/pause`, { method: 'POST' });
    } catch (e) {
        console.error('pause error:', e);
    }
    pollState();
}

async function resumeCrawl() {
    try {
        await fetch(`${API}/api/resume`, { method: 'POST' });
    } catch (e) {
        console.error('resume error:', e);
    }
    pollState();
}

// ---- Health Check ----
async function checkHealth() {
    const badge = document.getElementById('health-badge');
    try {
        const resp = await fetch(`${API}/health`);
        if (resp.ok) {
            badge.textContent = 'online';
            badge.className = 'badge badge-ok';
        } else {
            badge.textContent = 'error';
            badge.className = 'badge badge-error';
        }
    } catch {
        badge.textContent = 'offline';
        badge.className = 'badge badge-error';
    }
}

// ---- Index Form ----
document.getElementById('index-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const origin = document.getElementById('origin').value.trim();
    const maxDepth = parseInt(document.getElementById('maxDepth').value, 10);

    const resultBox = document.getElementById('index-result');
    resultBox.classList.remove('hidden');
    resultBox.textContent = 'Starting crawl…';

    try {
        const resp = await fetch(`${API}/index`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ origin, maxDepth })
        });
        const data = await resp.json();
        resultBox.textContent = `✓ ${data.message} — Origin: ${data.origin}, Depth: ${data.maxDepth}`;
        pollState();
    } catch (err) {
        resultBox.textContent = '✗ Error: ' + err.message;
    }
});

// ---- Search Form ----
document.getElementById('search-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const query = document.getElementById('query').value.trim();
    if (!query) return;

    const resultBox = document.getElementById('search-result');
    const metaEl = document.getElementById('search-meta');
    const listEl = document.getElementById('search-results-list');

    resultBox.classList.remove('hidden');
    metaEl.textContent = 'Searching…';
    listEl.innerHTML = '';

    try {
        const resp = await fetch(`${API}/search?query=${encodeURIComponent(query)}&sortBy=relevance`);
        const data = await resp.json();

        metaEl.textContent = `Found ${data.count} result(s) for "${data.query}"`;

        if (data.results.length === 0) {
            listEl.innerHTML = '<p style="color: var(--text-muted)">No results found.</p>';
            return;
        }

        listEl.innerHTML = data.results.map(r => `
            <div class="result-item">
                <a href="${escapeHtml(r.relevant_url)}" target="_blank">${escapeHtml(r.relevant_url)}</a>
                <div class="meta">
                    word: <strong>${escapeHtml(r.word)}</strong> ·
                    origin: ${escapeHtml(r.origin_url)} ·
                    depth: ${r.depth} ·
                    freq: ${r.frequency} ·
                    score: <span class="score">${r.relevance_score}</span>
                </div>
            </div>
        `).join('');

    } catch (err) {
        metaEl.textContent = '✗ Error: ' + err.message;
    }
});

// ---- Helpers ----
function setText(id, value) {
    const el = document.getElementById(id);
    if (el) el.textContent = value;
}

function escapeHtml(str) {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
}

// ---- Init ----
checkHealth();
pollState();
setInterval(pollState, 2000);
setInterval(checkHealth, 10000);

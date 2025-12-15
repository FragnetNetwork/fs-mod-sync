import './style.css';
import {
    GetConfig,
    SaveConfig,
    ValidateURL,
    GetDefaultModsDir,
    BrowseDirectory,
    GetSyncStatus,
    StartSync,
    CancelSync,
    Quit,
    GetVersion,
    GetGitHubRepo
} from '../wailsjs/go/main/App';
import { EventsOn, ClipboardGetText, ClipboardSetText, BrowserOpenURL } from '../wailsjs/runtime/runtime';

let state = {
    serverURL: '',
    modsDir: '',
    gameVersion: '',
    mods: [],
    syncStatus: null
};

let contextTarget = null;
const contextMenu = document.getElementById('context-menu');
const ctxCopy = document.getElementById('ctx-copy');
const ctxPaste = document.getElementById('ctx-paste');

document.addEventListener('contextmenu', (e) => {
    e.preventDefault();

    contextTarget = e.target;

    contextMenu.style.left = e.clientX + 'px';
    contextMenu.style.top = e.clientY + 'px';
    contextMenu.classList.add('show');
});

document.addEventListener('click', () => {
    contextMenu.classList.remove('show');
});

ctxCopy.addEventListener('click', async () => {
    const selection = window.getSelection().toString();
    if (selection) {
        await ClipboardSetText(selection);
    } else if (contextTarget && (contextTarget.tagName === 'INPUT' || contextTarget.tagName === 'TEXTAREA')) {
        const input = contextTarget;
        const selectedText = input.value.substring(input.selectionStart, input.selectionEnd);
        if (selectedText) {
            await ClipboardSetText(selectedText);
        }
    }
    contextMenu.classList.remove('show');
});

ctxPaste.addEventListener('click', async () => {
    if (contextTarget && (contextTarget.tagName === 'INPUT' || contextTarget.tagName === 'TEXTAREA')) {
        const text = await ClipboardGetText();
        if (text) {
            const input = contextTarget;
            const start = input.selectionStart;
            const end = input.selectionEnd;
            input.value = input.value.substring(0, start) + text + input.value.substring(end);
            input.selectionStart = input.selectionEnd = start + text.length;
            input.focus();
        }
    }
    contextMenu.classList.remove('show');
});

const screens = {
    setup: document.getElementById('setup-screen'),
    directory: document.getElementById('directory-screen'),
    summary: document.getElementById('summary-screen'),
    progress: document.getElementById('progress-screen'),
    complete: document.getElementById('complete-screen')
};

const elements = {
    serverUrl: document.getElementById('server-url'),
    validateBtn: document.getElementById('validate-btn'),
    validationError: document.getElementById('validation-error'),
    gameVersion: document.getElementById('game-version'),
    modCount: document.getElementById('mod-count'),
    modsDir: document.getElementById('mods-dir'),
    browseBtn: document.getElementById('browse-btn'),
    scanBtn: document.getElementById('scan-btn'),
    totalMods: document.getElementById('total-mods'),
    modsToSync: document.getElementById('mods-to-sync'),
    totalSize: document.getElementById('total-size'),
    modsList: document.getElementById('mods-list'),
    backBtn: document.getElementById('back-btn'),
    syncBtn: document.getElementById('sync-btn'),
    progressStatus: document.getElementById('progress-status'),
    progressCount: document.getElementById('progress-count'),
    progressFill: document.getElementById('progress-fill'),
    currentFile: document.getElementById('current-file'),
    downloadSpeed: document.getElementById('download-speed'),
    cancelBtn: document.getElementById('cancel-btn'),
    completeMessage: document.getElementById('complete-message'),
    closeBtn: document.getElementById('close-btn'),
    summaryCloseBtn: document.getElementById('summary-close-btn')
};

function showScreen(name) {
    Object.values(screens).forEach(s => s.classList.remove('active'));
    screens[name].classList.add('active');
}

function showError(message) {
    elements.validationError.textContent = message;
    elements.validationError.classList.remove('hidden');
}

function hideError() {
    elements.validationError.classList.add('hidden');
}

function setLoading(button, loading) {
    if (loading) {
        button.disabled = true;
        button.dataset.originalText = button.textContent;
        button.innerHTML = '<span class="spinner"></span>Loading...';
    } else {
        button.disabled = false;
        button.textContent = button.dataset.originalText || button.textContent;
    }
}

async function loadConfig() {
    try {
        const config = await GetConfig();
        if (config.serverUrl) {
            elements.serverUrl.value = config.serverUrl;
            state.serverURL = config.serverUrl;
        }
        if (config.modsDirectory) {
            state.modsDir = config.modsDirectory;
        }
        if (config.gameVersion) {
            state.gameVersion = config.gameVersion;
        }
    } catch (err) {
        console.error('Failed to load config:', err);
    }
}

async function validateURL() {
    const url = elements.serverUrl.value.trim();
    if (!url) {
        showError('Please enter a URL');
        return;
    }

    hideError();
    setLoading(elements.validateBtn, true);

    try {
        const result = await ValidateURL(url);

        if (!result.valid) {
            showError(result.error || 'Invalid URL');
            return;
        }

        state.serverURL = url;
        state.gameVersion = result.gameVersion;

        const gameDisplayName = result.gameVersion === 'FS25' ? 'Farming Simulator 2025' : 'Farming Simulator 2022';
        elements.gameVersion.textContent = gameDisplayName;
        elements.modCount.textContent = result.modCount;

        if (!state.modsDir) {
            state.modsDir = await GetDefaultModsDir(result.gameVersion);
        }
        elements.modsDir.value = state.modsDir;

        await SaveConfig({
            serverUrl: state.serverURL,
            modsDirectory: state.modsDir,
            gameVersion: state.gameVersion
        });

        showScreen('directory');
    } catch (err) {
        showError('Connection failed: ' + err);
    } finally {
        setLoading(elements.validateBtn, false);
    }
}

async function browseDirectory() {
    try {
        const dir = await BrowseDirectory();
        if (dir) {
            state.modsDir = dir;
            elements.modsDir.value = dir;
        }
    } catch (err) {
        console.error('Failed to browse:', err);
    }
}

async function scanMods() {
    if (!state.modsDir) {
        alert('Please select a mods directory');
        return;
    }

    setLoading(elements.scanBtn, true);

    try {
        const result = await GetSyncStatus(state.serverURL, state.modsDir);

        const status = result.status || { totalMods: 0, modsToSync: 0, totalSize: '0 B' };
        const mods = result.mods || [];

        state.syncStatus = status;
        state.mods = mods;

        elements.totalMods.textContent = status.totalMods || 0;
        elements.modsToSync.textContent = status.modsToSync || 0;
        elements.totalSize.textContent = status.totalSize || '0 B';

        renderModsList(mods);

        await SaveConfig({
            serverUrl: state.serverURL,
            modsDirectory: state.modsDir,
            gameVersion: state.gameVersion
        });

        showScreen('summary');
    } catch (err) {
        alert('Failed to scan mods: ' + err);
    } finally {
        setLoading(elements.scanBtn, false);
    }
}

function renderModsList(mods) {
    const modsToDownload = mods.filter(m => m.needsUpdate && !m.isDLC);
    const downloadableMods = mods.filter(m => !m.isDLC && m.url);

    if (modsToDownload.length === 0) {
        let html = '<p class="empty-message">All mods are up to date!</p>';
        if (downloadableMods.length > 0) {
            html += '<div class="mods-detail"><p style="margin-top:10px;color:#666;font-size:13px;">Detected versions:</p>';
            html += downloadableMods.map(mod => `
                <div class="mod-item" style="padding:8px;">
                    <span class="mod-name" style="font-size:13px;">${escapeHtml(mod.filename)}</span>
                    <span class="mod-version" style="font-size:12px;color:#888;">Local: ${mod.localVersion || 'N/A'} | Server: ${mod.version}</span>
                </div>
            `).join('');
            html += '</div>';
        }
        elements.modsList.innerHTML = html;
        elements.syncBtn.disabled = true;
        elements.syncBtn.style.display = 'none';
        return;
    }

    elements.syncBtn.disabled = false;
    elements.syncBtn.style.display = '';
    elements.modsList.innerHTML = modsToDownload.map(mod => `
        <div class="mod-item">
            <div>
                <div class="mod-name">${escapeHtml(mod.name)}</div>
                <div class="mod-version">
                    ${mod.localVersion ? `${mod.localVersion} → ${mod.version}` : `New: ${mod.version}`}
                </div>
            </div>
            <div class="mod-size">${mod.size || '-'}</div>
        </div>
    `).join('');
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

async function startSync() {
    showScreen('progress');
    elements.progressFill.style.width = '0%';
    elements.progressStatus.textContent = 'Starting...';
    elements.progressCount.textContent = '0 / 0';
    elements.currentFile.textContent = '-';
    elements.downloadSpeed.textContent = '-';

    try {
        await StartSync(state.serverURL, state.modsDir);
    } catch (err) {
        alert('Sync failed: ' + err);
        showScreen('summary');
    }
}

async function cancelSync() {
    try {
        await CancelSync();
    } catch (err) {
        console.error('Failed to cancel:', err);
    }
}

function goBack() {
    showScreen('directory');
}

elements.validateBtn.addEventListener('click', validateURL);
elements.browseBtn.addEventListener('click', browseDirectory);
elements.scanBtn.addEventListener('click', scanMods);
elements.backBtn.addEventListener('click', goBack);
elements.syncBtn.addEventListener('click', startSync);
elements.cancelBtn.addEventListener('click', cancelSync);
elements.closeBtn.addEventListener('click', () => Quit());
elements.summaryCloseBtn.addEventListener('click', () => Quit());

elements.serverUrl.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') {
        validateURL();
    }
});

EventsOn('download:progress', (event) => {
    elements.progressStatus.textContent = 'Downloading...';
    elements.progressCount.textContent = `${event.downloaded} / ${event.total}`;
    elements.currentFile.textContent = event.filename;
    elements.downloadSpeed.textContent = event.speed;
    elements.progressFill.style.width = `${event.progress * 100}%`;
});

EventsOn('download:complete', (filename) => {
    console.log('Downloaded:', filename);
});

EventsOn('download:error', (event) => {
    console.error('Download error:', event.filename, event.error);
});

EventsOn('sync:complete', () => {
    elements.completeMessage.textContent = 'All mods have been downloaded successfully.';
    showScreen('complete');
});

EventsOn('sync:cancelled', () => {
    elements.completeMessage.textContent = 'Sync was cancelled.';
    showScreen('complete');
});

async function initFooter() {
    try {
        const version = await GetVersion();
        const repoUrl = await GetGitHubRepo();

        document.getElementById('version-info').textContent = version;

        const githubLink = document.getElementById('github-link');
        githubLink.addEventListener('click', (e) => {
            e.preventDefault();
            BrowserOpenURL(repoUrl);
        });
    } catch (err) {
        console.error('Failed to init footer:', err);
        document.getElementById('version-info').textContent = 'dev';
    }
}

loadConfig();
initFooter();

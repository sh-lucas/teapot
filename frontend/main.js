
// State
const state = {
  host: localStorage.getItem('teapot_host') || '',
  secret: localStorage.getItem('teapot_secret') || '',
  files: JSON.parse(localStorage.getItem('teapot_files') || '[]'),
  currentFile: null,
  lines: 70,
  skip: 0,
  isTailing: true, // "Watching" state
  isLoading: false,
  pollInterval: null
};

// DOM Elements
const hostInput = document.getElementById('host-input');
const secretInput = document.getElementById('secret-input');
const saveCredsBtn = document.getElementById('save-creds-btn');
const clearCredsBtn = document.getElementById('clear-creds-btn');
const newFileInput = document.getElementById('new-file-input');
const addFileBtn = document.getElementById('add-file-btn');
const fileList = document.getElementById('file-list');
const currentFileName = document.getElementById('current-file-name');
const logContent = document.getElementById('log-content');
const logContainer = document.getElementById('log-container');
const loadingIndicator = document.getElementById('loading-indicator');
const scrollDownBtn = document.getElementById('scroll-down-btn');
const linesInput = document.getElementById('lines-input');
const refreshBtn = document.getElementById('refresh-btn');

// Initialization
function init() {
  hostInput.value = state.host;
  secretInput.value = state.secret;
  renderFileList();
  startPolling();
}

// Event Listeners
saveCredsBtn.addEventListener('click', () => {
  state.host = hostInput.value.replace(/\/$/, '');
  state.secret = secretInput.value;
  localStorage.setItem('teapot_host', state.host);
  localStorage.setItem('teapot_secret', state.secret);
  alert('Credentials saved!');
  fetchLogs(true);
});

clearCredsBtn.addEventListener('click', () => {
  state.host = '';
  state.secret = '';
  hostInput.value = '';
  secretInput.value = '';
  localStorage.removeItem('teapot_host');
  localStorage.removeItem('teapot_secret');
});

addFileBtn.addEventListener('click', addFile);
newFileInput.addEventListener('keypress', (e) => {
  if (e.key === 'Enter') addFile();
});

refreshBtn.addEventListener('click', () => fetchLogs(true));
linesInput.addEventListener('change', (e) => {
  state.lines = parseInt(e.target.value) || 70;
  if (state.currentFile) fetchLogs(true);
});

logContainer.addEventListener('scroll', handleScroll);
scrollDownBtn.addEventListener('click', () => {
  scrollToBottom();
  state.isTailing = true;
  updateUIState();
  fetchLogs(false); // Immediate refresh
});

// Functions
function addFile() {
  const fileName = newFileInput.value.trim();
  if (fileName && !state.files.includes(fileName)) {
    state.files.push(fileName);
    localStorage.setItem('teapot_files', JSON.stringify(state.files));
    newFileInput.value = '';
    renderFileList();
    selectFile(fileName);
  }
}

function renderFileList() {
  fileList.innerHTML = '';
  state.files.forEach(file => {
    const li = document.createElement('li');
    li.textContent = file;
    if (file === state.currentFile) {
      li.classList.add('active');
    }
    li.addEventListener('click', () => selectFile(file));
    fileList.appendChild(li);
  });
}

function selectFile(file) {
  state.currentFile = file;
  currentFileName.textContent = file;
  state.skip = 0;
  state.isTailing = true;
  renderFileList();
  fetchLogs(true);
}

function handleScroll() {
  const { scrollTop, scrollHeight, clientHeight } = logContainer;

  // Detect scroll to top (Infinite Scroll Up)
  if (scrollTop === 0 && !state.isLoading) {
    loadMoreLogs();
  }

  // Detect scroll to bottom (Toggle Tailing)
  // Allow a small buffer (e.g., 20px)
  const isAtBottom = scrollHeight - scrollTop - clientHeight <= 20;

  if (isAtBottom) {
    state.isTailing = true;
  } else {
    state.isTailing = false;
  }
  updateUIState();
}

function updateUIState() {
  if (state.isTailing) {
    scrollDownBtn.classList.add('hidden');
  } else {
    scrollDownBtn.classList.remove('hidden');
  }
}

async function loadMoreLogs() {
  if (!state.currentFile || state.isLoading) return;

  state.isLoading = true;
  loadingIndicator.classList.remove('hidden');

  // Simulate small delay for UX
  await new Promise(r => setTimeout(r, 500));

  const oldHeight = logContainer.scrollHeight;
  const nextSkip = state.skip + state.lines;

  try {
    // Fetch older logs: n=lines, skip=nextSkip
    const logs = await fetchLogData(state.lines, nextSkip);
    if (logs) {
      if (logs.trim().length > 0) {
        logContent.textContent = logs + '\n' + logContent.textContent;
        state.skip = nextSkip;

        // Restore scroll position
        // New scroll top = New Height - Old Height
        // This keeps the view stable
        logContainer.scrollTop = logContainer.scrollHeight - oldHeight;
      }
    }
  } catch (err) {
    console.error("Error loading more logs:", err);
  } finally {
    state.isLoading = false;
    loadingIndicator.classList.add('hidden');
  }
}

async function fetchLogs(reset = false) {
  if (!state.currentFile) return;
  if (!state.host || !state.secret) {
    logContent.textContent = 'Please configure Host and Secret first.';
    return;
  }

  if (reset) {
    state.skip = 0;
    state.isTailing = true;
    logContent.textContent = 'Loading...';
  }

  try {
    // If tailing, we want the LATEST logs, so skip=0
    // If NOT tailing (looking up), we generally don't auto-refresh the view
    // But if this is a manual refresh (reset=true), we fetch latest.

    const logs = await fetchLogData(state.lines, 0);
    if (logs !== null) {
      logContent.textContent = logs || '(No logs found)';
      if (state.isTailing) {
        scrollToBottom();
      }
    }
  } catch (error) {
    logContent.textContent = error.message;
  }
}

async function fetchLogData(n, skip) {
  const url = `${state.host}/logs/${state.currentFile}?n=${n}&skip=${skip}`;
  const secret = state.secret.trim();
  const authHeader = secret.startsWith('Bearer ') ? secret : `Bearer ${secret}`;

  const response = await fetch(url, {
    headers: {
      'Authorization': authHeader
    }
  });

  if (!response.ok) {
    if (response.status === 401) throw new Error('Unauthorized (Check Secret)');
    if (response.status === 404) return '';
    throw new Error(`Error: ${response.statusText} `);
  }

  return await response.text();
}

function startPolling() {
  if (state.pollInterval) clearInterval(state.pollInterval);
  state.pollInterval = setInterval(async () => {
    // Only poll if we are in "Watching" state (isTailing)
    if (state.currentFile && state.isTailing && !state.isLoading) {
      try {
        const logs = await fetchLogData(state.lines, 0);
        if (logs !== null) {
          if (logContent.textContent !== logs) {
            logContent.textContent = logs || '(No logs found)';
            scrollToBottom();
          }
        }
      } catch (e) {
        console.error("Polling error:", e);
      }
    }
  }, 3000);
}

function scrollToBottom() {
  logContainer.scrollTop = logContainer.scrollHeight;
}

// Start
init();


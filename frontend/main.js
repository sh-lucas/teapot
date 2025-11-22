// State
const state = {
  host: localStorage.getItem('teapot_host') || '',
  secret: localStorage.getItem('teapot_secret') || '',
  files: JSON.parse(localStorage.getItem('teapot_files') || '[]'),
  currentFile: null,
  lines: 50
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
const linesInput = document.getElementById('lines-input');
const refreshBtn = document.getElementById('refresh-btn');

// Initialization
function init() {
  hostInput.value = state.host;
  secretInput.value = state.secret;
  renderFileList();
}

// Event Listeners
saveCredsBtn.addEventListener('click', () => {
  state.host = hostInput.value.replace(/\/$/, ''); // Remove trailing slash
  state.secret = secretInput.value;
  localStorage.setItem('teapot_host', state.host);
  localStorage.setItem('teapot_secret', state.secret);
  alert('Credentials saved!');
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

refreshBtn.addEventListener('click', fetchLogs);
linesInput.addEventListener('change', (e) => {
  state.lines = parseInt(e.target.value) || 50;
  if (state.currentFile) fetchLogs();
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

    // Add delete button (optional, but good for UX)
    // For simplicity, right click to delete? Or just keep it simple as requested.

    fileList.appendChild(li);
  });
}

function selectFile(file) {
  state.currentFile = file;
  currentFileName.textContent = file;
  renderFileList(); // Update active state
  fetchLogs();
}

async function fetchLogs() {
  if (!state.currentFile) return;
  if (!state.host || !state.secret) {
    logContent.textContent = 'Please configure Host and Secret first.';
    return;
  }

  logContent.textContent = 'Loading...';

  try {
    const url = `${state.host}/logs/${state.currentFile}?n=${state.lines}`;
    const response = await fetch(url, {
      headers: {
        'Authorization': `Bearer ${state.secret}`
      }
    });

    if (!response.ok) {
      if (response.status === 401) throw new Error('Unauthorized (Check Secret)');
      if (response.status === 404) throw new Error('File not found');
      throw new Error(`Error: ${response.statusText}`);
    }

    const text = await response.text();
    logContent.textContent = text || '(No logs found)';
  } catch (error) {
    logContent.textContent = error.message;
  }
}

// Start
init();

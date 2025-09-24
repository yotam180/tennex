import { app, BrowserWindow, ipcMain, shell } from 'electron';
import path from 'path';
import { initializeDatabase } from './database/index.js';
import { SyncService } from './sync/syncService.js';
import { registerIpcHandlers } from './ipc/handlers.js';

let mainWindow: BrowserWindow | null = null;
let syncService: SyncService | null = null;

const isDev = process.env.NODE_ENV === 'development';

function createWindow() {
  mainWindow = new BrowserWindow({
    width: 1200,
    height: 800,
    minWidth: 800,
    minHeight: 600,
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
      preload: path.join(__dirname, 'preload.js'),
    },
    titleBarStyle: 'hiddenInset',
    show: false, // Don't show until ready
  });

  // Load the app
  if (isDev) {
    mainWindow.loadURL('http://localhost:5173');
    mainWindow.webContents.openDevTools();
  } else {
    mainWindow.loadFile(path.join(__dirname, '../renderer/index.html'));
  }

  // Show window when ready to prevent visual flash
  mainWindow.once('ready-to-show', () => {
    mainWindow?.show();
  });

  // Handle window closed
  mainWindow.on('closed', () => {
    mainWindow = null;
  });

  // Handle external links
  mainWindow.webContents.setWindowOpenHandler(({ url }) => {
    shell.openExternal(url);
    return { action: 'deny' };
  });
}

async function initializeApp() {
  try {
    // Initialize database
    const db = initializeDatabase();
    console.log('Database initialized');

    // Initialize sync service (will be configured later via IPC)
    // syncService = new SyncService({
    //   backendUrl: 'http://localhost:8082',
    //   authToken: '', // Will be set after authentication
    //   syncIntervalMs: 5000,
    // });

    // Register IPC handlers
    registerIpcHandlers(db);
    console.log('IPC handlers registered');

  } catch (error) {
    console.error('Failed to initialize app:', error);
    app.quit();
  }
}

// App event handlers
app.whenReady().then(async () => {
  await initializeApp();
  createWindow();

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow();
    }
  });
});

app.on('window-all-closed', () => {
  if (syncService) {
    syncService.stop();
  }
  
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('before-quit', () => {
  if (syncService) {
    syncService.stop();
  }
});

// Export for IPC handlers
export { syncService };
export function setSyncService(service: SyncService) {
  syncService = service;
}

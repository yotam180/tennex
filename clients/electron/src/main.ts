import path from 'node:path';
import started from 'electron-squirrel-startup';
import { app, ipcMain, BrowserWindow } from 'electron';

import * as dbOps from './db/operations';
import { getDatabaseService } from './db/database';

// Handle creating/removing shortcuts on Windows when installing/uninstalling.
if (started) {
  app.quit();
}

const createWindow = () => {
  // Create the browser window.
  const mainWindow = new BrowserWindow({
    width: 1600,
    height: 1200,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
    },
  });

  // and load the index.html of the app.
  if (MAIN_WINDOW_VITE_DEV_SERVER_URL) {
    mainWindow.loadURL(MAIN_WINDOW_VITE_DEV_SERVER_URL);
  } else {
    mainWindow.loadFile(path.join(__dirname, `../renderer/${MAIN_WINDOW_VITE_NAME}/index.html`));
  }
};

// This method will be called when Electron has finished
// initialization and is ready to create browser windows.
// Some APIs can only be used after this event occurs.
app.on('ready', async () => {
  // Initialize database
  try {
    const dbService = getDatabaseService();
    await dbService.initialize();
    console.log('✅ Database initialized successfully');
  } catch (error) {
    console.error('❌ Failed to initialize database:', error);
  }

  // Set up IPC handlers
  setupIpcHandlers();

  // Create window
  createWindow();
});

// Quit when all windows are closed, except on macOS. There, it's common
// for applications and their menu bar to stay active until the user quits
// explicitly with Cmd + Q.
app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('activate', () => {
  // On OS X it's common to re-create a window in the app when the
  // dock icon is clicked and there are no other windows open.
  if (BrowserWindow.getAllWindows().length === 0) {
    createWindow();
  }
});

// In this file you can include the rest of your app's specific main process
// code. You can also put them in separate files and import them here.

// =============================================================================
// IPC HANDLERS FOR DATABASE OPERATIONS
// =============================================================================

function setupIpcHandlers() {
  // Sync State
  ipcMain.handle('db:getSyncState', async (_, integrationId: number) => {
    try {
      return await dbOps.getSyncState(integrationId);
    } catch (error) {
      console.error('Error getting sync state:', error);
      throw error;
    }
  });

  ipcMain.handle('db:upsertSyncState', async (_, data: any) => {
    try {
      return await dbOps.upsertSyncState(data);
    } catch (error) {
      console.error('Error upserting sync state:', error);
      throw error;
    }
  });

  // Conversations
  ipcMain.handle('db:upsertConversations', async (_, conversations: any[]) => {
    try {
      return await dbOps.upsertConversations(conversations);
    } catch (error) {
      console.error('Error upserting conversations:', error);
      throw error;
    }
  });

  ipcMain.handle('db:getConversations', async (_, integrationId: number, limit?: number) => {
    try {
      return await dbOps.getConversations(integrationId, limit);
    } catch (error) {
      console.error('Error getting conversations:', error);
      throw error;
    }
  });

  // Messages
  ipcMain.handle('db:upsertMessages', async (_, messages: any[]) => {
    try {
      return await dbOps.upsertMessages(messages);
    } catch (error) {
      console.error('Error upserting messages:', error);
      throw error;
    }
  });

  ipcMain.handle('db:getMessages', async (_, conversationId: string, limit?: number) => {
    try {
      return await dbOps.getMessages(conversationId, limit);
    } catch (error) {
      console.error('Error getting messages:', error);
      throw error;
    }
  });

  ipcMain.handle(
    'db:getMessagesByIntegration',
    async (_, integrationId: number, limit?: number) => {
      try {
        return await dbOps.getMessagesByIntegration(integrationId, limit);
      } catch (error) {
        console.error('Error getting messages by integration:', error);
        throw error;
      }
    }
  );

  // Contacts
  ipcMain.handle('db:upsertContacts', async (_, contacts: any[]) => {
    try {
      return await dbOps.upsertContacts(contacts);
    } catch (error) {
      console.error('Error upserting contacts:', error);
      throw error;
    }
  });

  ipcMain.handle('db:getContacts', async (_, integrationId: number, limit?: number) => {
    try {
      return await dbOps.getContacts(integrationId, limit);
    } catch (error) {
      console.error('Error getting contacts:', error);
      throw error;
    }
  });

  // Database Stats
  ipcMain.handle('db:getStats', async () => {
    try {
      return await dbOps.getDatabaseStats();
    } catch (error) {
      console.error('Error getting database stats:', error);
      throw error;
    }
  });

  console.log('✅ IPC handlers registered');
}

// Clean up database on quit
app.on('before-quit', () => {
  const dbService = getDatabaseService();
  dbService.close();
});

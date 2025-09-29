import { useState } from 'react';
import {
  Box,
  Card,
  Grid,
  Button,
  Typography,
  CardContent,
  CircularProgress,
  Alert,
  Paper,
  LinearProgress,
  Divider,
} from '@mui/material';
import QRCode from 'react-qr-code';

import { DashboardContent } from 'src/layouts/dashboard';
import axios from 'src/lib/axios';
import { Iconify } from 'src/components/iconify';

// ----------------------------------------------------------------------

interface QRResponse {
  qr_code: string;
  session_id: string;
  expires_at: string;
  instructions: string;
}

interface SyncProgress {
  stage: 'conversations' | 'messages' | 'contacts' | 'complete';
  current: number;
  total: number;
  message: string;
}

export function WhatsAppSyncView() {
  const [loading, setLoading] = useState(false);
  const [qrData, setQrData] = useState<QRResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Sync state
  const [syncing, setSyncing] = useState(false);
  const [syncProgress, setSyncProgress] = useState<SyncProgress | null>(null);
  const [syncSuccess, setSyncSuccess] = useState(false);

  const handleConnectWhatsApp = async () => {
    setLoading(true);
    setError(null);
    setQrData(null);

    try {
      console.log('🚀 Initiating WhatsApp connection...');

      // Call bridge API to get QR code
      const response = await axios.post('http://localhost:6003/whatsapp/connect');
      const data = response.data as QRResponse;

      console.log('✅ QR code received:', data);
      setQrData(data);
    } catch (err: any) {
      console.error('❌ Failed to connect WhatsApp:', err);

      if (err.response?.data?.code === 'already_connected') {
        setError('WhatsApp account is already connected to this profile.');
      } else if (err.response?.data?.code === 'qr_timeout') {
        setError('QR code generation timed out. Please try again.');
      } else {
        setError(err.response?.data?.message || 'Failed to generate QR code. Please try again.');
      }
    } finally {
      setLoading(false);
    }
  };

  const handleSyncAllData = async () => {
    setSyncing(true);
    setError(null);
    setSyncSuccess(false);

    try {
      // TODO: Get actual integration_id from user's WhatsApp integration
      const integrationId = 1; // Hardcoded for now

      console.log('🔄 Starting data sync for integration:', integrationId);

      // Step 1: Sync conversations
      setSyncProgress({
        stage: 'conversations',
        current: 0,
        total: 0,
        message: 'Fetching conversations...',
      });

      let conversationSeq = 0;
      let hasMore = true;
      let totalConversations = 0;

      while (hasMore) {
        const response = await axios.get(`/sync/conversations/${integrationId}`, {
          params: { since_seq: conversationSeq, limit: 100 },
        });

        const data = response.data;
        totalConversations += data.conversations.length;

        // TODO: Store conversations in local SQLite database
        console.log(`📥 Received ${data.conversations.length} conversations`);

        conversationSeq = data.latest_seq;
        hasMore = data.has_more;

        setSyncProgress({
          stage: 'conversations',
          current: totalConversations,
          total: totalConversations + (hasMore ? 1 : 0),
          message: `Synced ${totalConversations} conversations`,
        });
      }

      console.log(`✅ Synced ${totalConversations} conversations`);

      // Step 2: Sync messages
      setSyncProgress({
        stage: 'messages',
        current: 0,
        total: 0,
        message: 'Fetching messages...',
      });

      let messageSeq = 0;
      hasMore = true;
      let totalMessages = 0;

      while (hasMore) {
        const response = await axios.get(`/sync/messages/${integrationId}`, {
          params: { since_seq: messageSeq, limit: 1500 },
        });

        const data = response.data;
        totalMessages += data.messages.length;

        // TODO: Store messages in local SQLite database
        console.log(`📥 Received ${data.messages.length} messages`);

        messageSeq = data.latest_seq;
        hasMore = data.has_more;

        setSyncProgress({
          stage: 'messages',
          current: totalMessages,
          total: totalMessages + (hasMore ? 1000 : 0),
          message: `Synced ${totalMessages} messages`,
        });
      }

      console.log(`✅ Synced ${totalMessages} messages`);

      // Step 3: Sync contacts
      setSyncProgress({
        stage: 'contacts',
        current: 0,
        total: 0,
        message: 'Fetching contacts...',
      });

      let contactSeq = 0;
      hasMore = true;
      let totalContacts = 0;

      while (hasMore) {
        const response = await axios.get(`/sync/contacts/${integrationId}`, {
          params: { since_seq: contactSeq, limit: 500 },
        });

        const data = response.data;
        totalContacts += data.contacts.length;

        // TODO: Store contacts in local SQLite database
        console.log(`📥 Received ${data.contacts.length} contacts`);

        contactSeq = data.latest_seq;
        hasMore = data.has_more;

        setSyncProgress({
          stage: 'contacts',
          current: totalContacts,
          total: totalContacts + (hasMore ? 100 : 0),
          message: `Synced ${totalContacts} contacts`,
        });
      }

      console.log(`✅ Synced ${totalContacts} contacts`);

      // Complete!
      setSyncProgress({
        stage: 'complete',
        current: 100,
        total: 100,
        message: `Sync complete! ${totalConversations} conversations, ${totalMessages} messages, ${totalContacts} contacts`,
      });

      setSyncSuccess(true);
    } catch (err: any) {
      console.error('❌ Sync failed:', err);
      setError(err.response?.data?.error || err.message || 'Sync failed. Please try again.');
    } finally {
      setSyncing(false);
    }
  };

  return (
    <DashboardContent maxWidth="xl">
      <Typography variant="h4" sx={{ mb: { xs: 3, md: 5 } }}>
        WhatsApp Sync
      </Typography>

      <Grid container spacing={3}>
        {/* Data Sync Section */}
        <Grid size={{ xs: 12 }}>
          <Card>
            <CardContent>
              <Box
                sx={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  mb: 2,
                }}
              >
                <Box>
                  <Typography variant="h6" gutterBottom>
                    Sync WhatsApp Data
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    Fetch all conversations, messages, and contacts from the backend
                  </Typography>
                </Box>
                <Button
                  variant="contained"
                  color="primary"
                  onClick={handleSyncAllData}
                  disabled={syncing}
                  startIcon={
                    syncing ? (
                      <CircularProgress size={20} color="inherit" />
                    ) : (
                      <Iconify icon="solar:refresh-circle-outline" />
                    )
                  }
                  sx={{ minWidth: 160 }}
                >
                  {syncing ? 'Syncing...' : 'Sync All Data'}
                </Button>
              </Box>

              {error && !syncing && (
                <Alert severity="error" sx={{ mb: 2 }}>
                  {error}
                </Alert>
              )}

              {syncSuccess && !syncing && (
                <Alert severity="success" sx={{ mb: 2 }}>
                  {syncProgress?.message}
                </Alert>
              )}

              {syncProgress && syncing && (
                <Box sx={{ mt: 2 }}>
                  <Box
                    sx={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'space-between',
                      mb: 1,
                    }}
                  >
                    <Typography variant="body2" color="text.secondary">
                      {syncProgress.message}
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      {syncProgress.stage.charAt(0).toUpperCase() + syncProgress.stage.slice(1)}
                    </Typography>
                  </Box>
                  <LinearProgress
                    variant={syncProgress.total > 0 ? 'determinate' : 'indeterminate'}
                    value={
                      syncProgress.total > 0 ? (syncProgress.current / syncProgress.total) * 100 : 0
                    }
                    sx={{ height: 8, borderRadius: 1 }}
                  />
                </Box>
              )}
            </CardContent>
          </Card>
        </Grid>

        <Grid size={{ xs: 12 }}>
          <Divider sx={{ my: 2 }} />
          <Typography variant="h6" sx={{ mb: 2 }}>
            WhatsApp Connection
          </Typography>
        </Grid>

        {/* Connection Section */}
        <Grid size={{ xs: 12, md: qrData ? 6 : 12 }}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Connect Your WhatsApp
              </Typography>

              {error && !syncProgress && (
                <Alert severity="error" sx={{ mb: 2 }}>
                  {error}
                </Alert>
              )}

              {!qrData && (
                <>
                  <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
                    Click the button below to generate a QR code. Then scan it with your WhatsApp
                    mobile app to connect.
                  </Typography>

                  <Button
                    variant="contained"
                    color="primary"
                    onClick={handleConnectWhatsApp}
                    disabled={loading}
                    sx={{ minWidth: 160 }}
                  >
                    {loading ? (
                      <>
                        <CircularProgress size={20} color="inherit" sx={{ mr: 1 }} />
                        Connecting...
                      </>
                    ) : (
                      'Connect WhatsApp'
                    )}
                  </Button>
                </>
              )}

              {qrData && (
                <Box>
                  <Typography variant="body2" color="success.main" sx={{ mb: 2 }}>
                    ✅ QR Code generated! Scan with your WhatsApp app.
                  </Typography>
                  <Button
                    variant="outlined"
                    onClick={handleConnectWhatsApp}
                    disabled={loading}
                    size="small"
                  >
                    Generate New QR
                  </Button>
                </Box>
              )}
            </CardContent>
          </Card>
        </Grid>

        {/* QR Code Display */}
        {qrData && (
          <Grid size={{ xs: 12, md: 6 }}>
            <Card>
              <CardContent sx={{ textAlign: 'center' }}>
                <Typography variant="h6" gutterBottom>
                  Scan QR Code
                </Typography>

                <Paper
                  elevation={1}
                  sx={{
                    p: 2,
                    display: 'inline-block',
                    bgcolor: 'white',
                    mb: 2,
                  }}
                >
                  <QRCode value={qrData.qr_code} size={200} level="M" />
                </Paper>

                <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                  {qrData.instructions ||
                    'Open WhatsApp → Menu → Linked Devices → Link a Device → Scan this code'}
                </Typography>

                <Typography variant="caption" color="text.disabled">
                  QR code expires at {new Date(qrData.expires_at).toLocaleTimeString()}
                </Typography>
              </CardContent>
            </Card>
          </Grid>
        )}

        {/* Instructions */}
        <Grid size={{ xs: 12 }}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                How to Connect
              </Typography>
              <Box component="ol" sx={{ pl: 2 }}>
                <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                  Click "Connect WhatsApp" to generate a QR code
                </Typography>
                <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                  Open WhatsApp on your phone
                </Typography>
                <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                  Tap Menu (⋮) → Linked Devices → Link a Device
                </Typography>
                <Typography component="li" variant="body2" sx={{ mb: 1 }}>
                  Scan the QR code displayed above
                </Typography>
                <Typography component="li" variant="body2">
                  Wait for the connection and sync to complete
                </Typography>
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </DashboardContent>
  );
}

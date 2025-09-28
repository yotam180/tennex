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
  Paper
} from '@mui/material';
import QRCode from 'react-qr-code';

import { DashboardContent } from 'src/layouts/dashboard';
import axios from 'src/lib/axios';

// ----------------------------------------------------------------------

interface QRResponse {
  qr_code: string;
  session_id: string;
  expires_at: string;
  instructions: string;
}

export function WhatsAppSyncView() {
  const [loading, setLoading] = useState(false);
  const [qrData, setQrData] = useState<QRResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

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

  return (
    <DashboardContent maxWidth="xl">
      <Typography variant="h4" sx={{ mb: { xs: 3, md: 5 } }}>
        WhatsApp Connection
      </Typography>

      <Grid container spacing={3}>
        {/* Connection Section */}
        <Grid size={{ xs: 12, md: qrData ? 6 : 12 }}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Connect Your WhatsApp
              </Typography>
              
              {error && (
                <Alert severity="error" sx={{ mb: 2 }}>
                  {error}
                </Alert>
              )}
              
              {!qrData && (
                <>
                  <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
                    Click the button below to generate a QR code. Then scan it with your WhatsApp mobile app to connect.
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
                    mb: 2 
                  }}
                >
                  <QRCode 
                    value={qrData.qr_code} 
                    size={200}
                    level="M"
                  />
                </Paper>
                
                <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                  {qrData.instructions || 'Open WhatsApp → Menu → Linked Devices → Link a Device → Scan this code'}
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

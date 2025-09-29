import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';

import Alert from '@mui/material/Alert';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import Card from '@mui/material/Card';
import CardContent from '@mui/material/CardContent';
import CardHeader from '@mui/material/CardHeader';
import Chip from '@mui/material/Chip';
import Skeleton from '@mui/material/Skeleton';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';

import { Iconify } from 'src/components/iconify';

import axiosInstance, { endpoints } from 'src/lib/axios';
import { JWT_STORAGE_KEY } from 'src/auth/context/jwt/constant';

// ----------------------------------------------------------------------

interface WhatsAppInfo {
  connected: boolean;
  status: string;
  wa_jid?: string;
  display_name?: string;
  last_seen?: string;
}

interface SettingsResponse {
  user_id: string;
  whatsapp: WhatsAppInfo;
}

// ----------------------------------------------------------------------

async function fetchSettings(): Promise<SettingsResponse> {
  // Get the auth token from localStorage using the correct key
  const token = localStorage.getItem(JWT_STORAGE_KEY);
  
  if (!token) {
    throw new Error('No authentication token found');
  }
  
  const config = {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  };

  const response = await axiosInstance.get(endpoints.settings, config);
  return response.data;
}

// ----------------------------------------------------------------------

export function AccountIntegrations() {
  const [connectingWhatsApp, setConnectingWhatsApp] = useState(false);

  const {
    data: settings,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: ['settings'],
    queryFn: fetchSettings,
    refetchOnWindowFocus: false,
  });

  const handleConnectWhatsApp = async () => {
    setConnectingWhatsApp(true);
    try {
      // Navigate to WhatsApp connection page
      window.location.href = '/dashboard/whatsapp';
    } catch (err) {
      console.error('Failed to navigate to WhatsApp connection:', err);
    } finally {
      setConnectingWhatsApp(false);
    }
  };

  const handleDisconnectWhatsApp = async () => {
    // TODO: Implement disconnect functionality
    console.log('Disconnect WhatsApp - TODO');
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'connected':
        return 'success';
      case 'connecting':
        return 'warning';
      case 'error':
        return 'error';
      default:
        return 'default';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'connected':
        return 'eva:checkmark-circle-2-fill';
      case 'connecting':
        return 'eva:clock-fill';
      case 'error':
        return 'eva:alert-triangle-fill';
      default:
        return 'eva:close-circle-fill';
    }
  };

  if (isLoading) {
    return (
      <Card>
        <CardHeader title="Integrations" />
        <CardContent>
          <Stack spacing={3}>
            <Skeleton variant="rectangular" height={120} />
          </Stack>
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card>
        <CardHeader title="Integrations" />
        <CardContent>
          <Alert severity="error">
            Failed to load integration settings. Please try refreshing the page.
          </Alert>
        </CardContent>
      </Card>
    );
  }

  const whatsappInfo = settings?.whatsapp;

  return (
    <Card>
      <CardHeader title="Integrations" subheader="Manage your external service connections" />
      
      <CardContent>
        <Stack spacing={3}>
          {/* WhatsApp Integration */}
          <Card variant="outlined">
            <CardContent>
              <Stack direction="row" alignItems="center" spacing={2}>
                <Box
                  sx={{
                    width: 48,
                    height: 48,
                    bgcolor: '#25D366',
                    borderRadius: '12px',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                  }}
                >
                  <Iconify icon="logos:whatsapp-icon" width={32} sx={{ color: 'white' }} />
                </Box>
                
                <Box sx={{ flexGrow: 1 }}>
                  <Stack direction="row" alignItems="center" spacing={1} sx={{ mb: 0.5 }}>
                    <Typography variant="h6">WhatsApp Business</Typography>
                    <Chip
                      size="small"
                      label={whatsappInfo?.status || 'disconnected'}
                      color={getStatusColor(whatsappInfo?.status || 'disconnected')}
                      icon={<Iconify icon={getStatusIcon(whatsappInfo?.status || 'disconnected')} />}
                    />
                  </Stack>
                  
                  <Typography variant="body2" color="text.secondary">
                    {whatsappInfo?.connected
                      ? `Connected as ${whatsappInfo.wa_jid || 'Unknown number'}`
                      : 'Connect your WhatsApp account to send and receive messages'
                    }
                  </Typography>
                  
                  {whatsappInfo?.last_seen && (
                    <Typography variant="caption" color="text.secondary">
                      Last seen: {new Date(whatsappInfo.last_seen).toLocaleString()}
                    </Typography>
                  )}
                </Box>
                
                <Box>
                  {whatsappInfo?.connected ? (
                    <Stack spacing={1}>
                      <Button
                        variant="outlined"
                        color="error"
                        size="small"
                        onClick={handleDisconnectWhatsApp}
                        startIcon={<Iconify icon="eva:close-outline" />}
                      >
                        Disconnect
                      </Button>
                      <Button
                        variant="outlined"
                        size="small"
                        onClick={handleConnectWhatsApp}
                        startIcon={<Iconify icon="eva:refresh-outline" />}
                      >
                        Reconnect
                      </Button>
                    </Stack>
                  ) : (
                    <Button
                      variant="contained"
                      onClick={handleConnectWhatsApp}
                      startIcon={<Iconify icon="eva:plus-outline" />}
                      disabled={connectingWhatsApp}
                    >
                      {connectingWhatsApp ? 'Connecting...' : 'Connect'}
                    </Button>
                  )}
                </Box>
              </Stack>
            </CardContent>
          </Card>

          {/* Future integrations placeholder */}
          <Card variant="outlined" sx={{ opacity: 0.5 }}>
            <CardContent>
              <Stack direction="row" alignItems="center" spacing={2}>
                <Box
                  sx={{
                    width: 48,
                    height: 48,
                    bgcolor: 'grey.300',
                    borderRadius: '12px',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                  }}
                >
                  <Iconify icon="eva:email-outline" width={32} />
                </Box>
                
                <Box sx={{ flexGrow: 1 }}>
                  <Typography variant="h6" color="text.secondary">
                    Email Integration
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    Coming soon - Connect your email accounts
                  </Typography>
                </Box>
                
                <Button disabled size="small">
                  Coming Soon
                </Button>
              </Stack>
            </CardContent>
          </Card>
        </Stack>
      </CardContent>
    </Card>
  );
}

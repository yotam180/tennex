import Box from '@mui/material/Box';
import Card from '@mui/material/Card';
import Grid from '@mui/material/Grid2';
import Button from '@mui/material/Button';
import Avatar from '@mui/material/Avatar';
import Typography from '@mui/material/Typography';
import CardContent from '@mui/material/CardContent';

import { useAuthContext } from 'src/auth/hooks';

import { Iconify } from 'src/components/iconify';

// ----------------------------------------------------------------------

export default function TennexDashboard() {
  const { user } = useAuthContext();

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" sx={{ mb: 3 }}>
        Welcome to Tennex! 🎉
      </Typography>

      <Grid container spacing={3}>
        {/* User Info Card */}
        <Grid size={{ xs: 12, md: 6 }}>
          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
                <Avatar
                  sx={{ 
                    width: 56, 
                    height: 56, 
                    mr: 2,
                    bgcolor: 'primary.main'
                  }}
                >
                  {user?.displayName?.charAt(0) || user?.email?.charAt(0) || 'U'}
                </Avatar>
                <Box>
                  <Typography variant="h6">
                    {user?.displayName || 'User'}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    {user?.email || 'No email'}
                  </Typography>
                </Box>
              </Box>
              
              <Typography variant="body2" color="text.secondary">
                JWT Token Active ✅
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        {/* Quick Actions */}
        <Grid size={{ xs: 12, md: 6 }}>
          <Card>
            <CardContent>
              <Typography variant="h6" sx={{ mb: 2 }}>
                Quick Actions
              </Typography>
              
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                <Button
                  variant="outlined"
                  startIcon={<Iconify icon="solar:chat-circle-outline" />}
                  sx={{ justifyContent: 'flex-start' }}
                >
                  Start WhatsApp Conversation
                </Button>
                
                <Button
                  variant="outlined"
                  startIcon={<Iconify icon="solar:qr-code-outline" />}
                  sx={{ justifyContent: 'flex-start' }}
                >
                  Scan QR Code
                </Button>
                
                <Button
                  variant="outlined"
                  startIcon={<Iconify icon="solar:settings-outline" />}
                  sx={{ justifyContent: 'flex-start' }}
                >
                  Account Settings
                </Button>
              </Box>
            </CardContent>
          </Card>
        </Grid>

        {/* Status Card */}
        <Grid size={{ xs: 12 }}>
          <Card>
            <CardContent>
              <Typography variant="h6" sx={{ mb: 2 }}>
                System Status
              </Typography>
              
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <Iconify icon="solar:check-circle-bold" color="success.main" />
                  <Typography variant="body2">
                    Authentication: Connected
                  </Typography>
                </Box>
                
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <Iconify icon="solar:wifi-router-outline" color="warning.main" />
                  <Typography variant="body2">
                    Backend Connection: Checking...
                  </Typography>
                </Box>
                
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <Iconify icon="solar:smartphone-outline" color="info.main" />
                  <Typography variant="body2">
                    WhatsApp Bridge: Ready
                  </Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
}

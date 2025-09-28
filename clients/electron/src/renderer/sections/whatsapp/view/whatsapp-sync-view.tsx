import { Box, Card, Grid, Button, Typography, CardContent } from '@mui/material';

import { DashboardContent } from 'src/layouts/dashboard';

// ----------------------------------------------------------------------

export function WhatsAppSyncView() {
  return (
    <DashboardContent maxWidth="xl">
      <Typography variant="h4" sx={{ mb: { xs: 3, md: 5 } }}>
        WhatsApp Synchronization
      </Typography>

      <Grid container spacing={3}>
        <Grid size={{ xs: 12, md: 6 }}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Connection Status
              </Typography>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                Connect your WhatsApp account to start syncing messages and contacts.
              </Typography>
              <Box sx={{ display: 'flex', gap: 2 }}>
                <Button variant="contained" color="primary">
                  Connect WhatsApp
                </Button>
                <Button variant="outlined" color="secondary">
                  Scan QR Code
                </Button>
              </Box>
            </CardContent>
          </Card>
        </Grid>

        <Grid size={{ xs: 12, md: 6 }}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Sync Settings
              </Typography>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                Configure your synchronization preferences and data management options.
              </Typography>
              <Box sx={{ display: 'flex', gap: 2 }}>
                <Button variant="outlined" color="primary">
                  Configure Settings
                </Button>
                <Button variant="outlined" color="info">
                  View Logs
                </Button>
              </Box>
            </CardContent>
          </Card>
        </Grid>

        <Grid size={{ xs: 12 }}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Recent Activity
              </Typography>
              <Typography variant="body2" color="text.secondary">
                No recent sync activity. Connect your WhatsApp account to see synchronization status and history.
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </DashboardContent>
  );
}

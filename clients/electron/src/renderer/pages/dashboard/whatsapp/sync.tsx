import { Helmet } from 'react-helmet-async';

import { CONFIG } from 'src/global-config';

import { WhatsAppSyncView } from 'src/sections/whatsapp/view';

// ----------------------------------------------------------------------

const metadata = { title: `WhatsApp Sync - ${CONFIG.appName}` };

export default function WhatsAppSyncPage() {
  return (
    <>
      <Helmet>
        <title> {metadata.title}</title>
      </Helmet>

      <WhatsAppSyncView />
    </>
  );
}

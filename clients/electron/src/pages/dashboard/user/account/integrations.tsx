import { Helmet } from 'react-helmet-async';

import { CONFIG } from 'src/global-config';

import { AccountIntegrationsView } from 'src/sections/account/view';

// ----------------------------------------------------------------------
// Account integrations page
const metadata = { title: `Account integrations | Dashboard - ${CONFIG.appName}` };

export default function Page() {
  return (
    <>
      <Helmet>
        <title> {metadata.title}</title>
      </Helmet>

      <AccountIntegrationsView />
    </>
  );
}

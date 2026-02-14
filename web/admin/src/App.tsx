import router from '@/router';
import { theme } from '@/themes';
import { ThemeProvider } from '@ctzhian/ui';
import { useLocation, useRoutes } from 'react-router-dom';

import '@ctzhian/tiptap/dist/index.css';

function App() {
  const location = useLocation();
  const { pathname } = location;
  const routerView = useRoutes(router);
  const loginPage = pathname.includes('/login');
  const onlyAllowShareApi = loginPage;

  const token = localStorage.getItem('panda_wiki_token') || '';

  if (!token && !onlyAllowShareApi) {
    window.location.href = window.__BASENAME__ + '/login';
    return null;
  }

  return (
    <ThemeProvider theme={theme} defaultMode='light' storageManager={null}>
      {routerView}
    </ThemeProvider>
  );
}

export default App;

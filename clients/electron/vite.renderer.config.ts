import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react-swc';
import { pluginExposeRenderer } from './vite.base.config';

// https://vitejs.dev/config
export default defineConfig((env) => {
  /** @type {import('vite').ConfigEnv<'renderer'>} */
  const forgeEnv = env as any;
  const { forgeConfigSelf } = forgeEnv;
  const name = forgeConfigSelf.name ?? '';

  return {
    root: '.',
    mode: forgeEnv.mode,
    base: './',
    build: {
      outDir: `.vite/renderer/${name}`,
    },
    plugins: [
      react(),
      pluginExposeRenderer(name)
    ],
    resolve: {
      preserveSymlinks: true,
      alias: {
        'src': '/src/renderer',
      },
    },
    clearScreen: false,
  };
});

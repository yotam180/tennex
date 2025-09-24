import { defineConfig } from 'vite';
import { builtins, external } from './vite.base.config';

// https://vitejs.dev/config
export default defineConfig({
  build: {
    sourcemap: true,
    outDir: '.vite/build',
    lib: {
      entry: 'src/main.ts',
      formats: ['cjs'],
      fileName: () => 'main.js',
    },
    rollupOptions: {
      external,
      output: {
        entryFileNames: '[name].js',
        format: 'cjs',
      },
    },
  },
  resolve: {
    browserField: false,
    mainFields: ['module', 'jsnext:main', 'jsnext'],
  },
});

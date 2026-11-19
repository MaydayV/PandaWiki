import react from '@vitejs/plugin-react';
import fs from 'fs';
import path from 'path';
import { visualizer } from 'rollup-plugin-visualizer';
import { defineConfig, loadEnv, Plugin } from 'vite';
import { execSync } from 'child_process';

const FLY_VERSION_RULE = {
  major: 2,
  feature: 6,
  featureStartCommit: '9f001318',
};

function readUpstreamVersion() {
  try {
    const pkgPath = path.resolve(__dirname, 'package.json');
    const pkg = JSON.parse(fs.readFileSync(pkgPath, 'utf-8')) as {
      version?: string;
    };
    return pkg.version || '2.11.1';
  } catch {
    return '2.11.1';
  }
}

function readFeatureCommitCount() {
  try {
    const output = execSync(
      `git rev-list --count ${FLY_VERSION_RULE.featureStartCommit}..HEAD`,
      {
        cwd: __dirname,
        stdio: ['ignore', 'pipe', 'ignore'],
      },
    )
      .toString()
      .trim();
    const count = Number(output);
    return Number.isFinite(count) && count > 0 ? count : 1;
  } catch {
    return 1;
  }
}

function buildFlyVersion() {
  const upstreamVersion = readUpstreamVersion();
  const commit = readFeatureCommitCount();
  return `FV${FLY_VERSION_RULE.major}.${FLY_VERSION_RULE.feature}.${commit}.${upstreamVersion.replace(/\./g, '')}`;
}

// 创建路由生成插件
function generateRoutesPlugin(): Plugin {
  return {
    name: 'generate-routes',
    buildStart() {
      // 构建开始时生成路由
      try {
        execSync('node scripts/generate-routes.js', { stdio: 'inherit' });
      } catch (error) {
        console.error('生成路由失败:', error);
      }
    },
    handleHotUpdate({ file, server }) {
      // 开发模式下监听路由文件变化
      const routerPath = path.resolve(__dirname, 'src/router.tsx');
      if (file === routerPath) {
        console.log('🔄 检测到路由文件变化，正在更新路由列表...');
        try {
          execSync('node scripts/generate-routes.js', { stdio: 'inherit' });
          // 触发 HMR 更新 index.html
          server.ws.send({
            type: 'update',
            updates: [
              {
                type: 'js-update',
                path: '/index.html',
                acceptedPath: '/index.html',
                timestamp: Date.now(),
              },
            ],
          });
        } catch (error) {
          console.error('❌ 更新路由列表失败:', error);
        }
      }
    },
  };
}

export default defineConfig(({ command, mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  const shouldAnalyze =
    process.argv.includes('--analyze') || env.ANALYZE === 'true';
  const flyVersion = buildFlyVersion();

  return {
    define: {
      'import.meta.env.VITE_APP_VERSION': JSON.stringify(flyVersion),
    },
    build: {
      assetsDir: 'panda-wiki-admin-assets',
      rollupOptions: {
        output: {
          // 使用 rolldown 原生 codeSplitting 分组(manualChunks 仿真会把匹配模块的
          // 全部依赖闭包递归吸入同一 chunk,导致 @mui/@ctzhian/ui 等被卷入
          // vendor-modelkit 并进入首屏预加载)。各分组与原 manualChunks 覆盖的
          // 包保持一致;modelkit/emoji 两组的依赖(@mui、@ctzhian/ui、react)由
          // 通用分组接管,因此关闭 includeDependenciesRecursively。
          codeSplitting: {
            groups: [
              {
                name: 'vendor-react',
                test:
                  /node_modules[\\/](?:react\/|react-dom\/|react-router\/|react-redux\/|@reduxjs\/toolkit\/)/,
              },
              { name: 'vendor-mui', test: /node_modules[\\/]@mui\// },
              {
                name: 'vendor-editor',
                test:
                  /node_modules[\\/](?:highlight\.js\/|lowlight\/|katex\/|prosemirror-state\/)/,
              },
              {
                name: 'vendor-markdown',
                test:
                  /node_modules[\\/](?:react-markdown\/|remark-gfm\/|remark-math\/|remark-breaks\/|rehype-katex\/|rehype-raw\/|rehype-sanitize\/)/,
              },
              {
                name: 'vendor-yjs',
                test: /node_modules[\\/](?:yjs\/|y-websocket\/)/,
              },
              {
                name: 'vendor-echarts',
                test: /node_modules[\\/](?:echarts\/|zrender\/)/,
              },
              {
                name: 'vendor-modelkit',
                test: /node_modules[\\/]@ctzhian\/modelkit\//,
                includeDependenciesRecursively: false,
              },
              {
                name: 'vendor-emoji',
                test: /node_modules[\\/](?:emoji-mart\/|@emoji-mart\/)/,
                includeDependenciesRecursively: false,
              },
              {
                name: 'vendor-cytoscape',
                test:
                  /node_modules[\\/](?:cytoscape\/|cytoscape-cose-bilkent\/|cytoscape-fcose\/)/,
              },
              { name: 'vendor-elkjs', test: /node_modules[\\/]elkjs\// },
            ],
          },
        },
      },
    },
    server: {
      hmr: true,
      proxy: {
        '/api': {
          target: env.TARGET,
          secure: false,
          changeOrigin: true,
        },
        '/static-file': {
          target: env.STATIC_FILE_TARGET,
          secure: false,
          changeOrigin: true,
        },
        '/share': {
          target: env.SHARE_TARGET,
          secure: false,
          changeOrigin: true,
        },
      },
      host: '0.0.0.0',
    },
    esbuild: {
      // 保留函数和类名，避免第三方库依赖 constructor.name 的逻辑在压缩后失效
      keepNames: true,
    },
    plugins: [
      react(),
      generateRoutesPlugin(),
      ...(command === 'build' && shouldAnalyze
        ? [
            visualizer({
              open: true,
              gzipSize: true,
              brotliSize: true,
              filename: 'dist/stats.html',
            }),
          ]
        : []),
    ],
    resolve: {
      alias: {
        '@': path.resolve(__dirname, 'src'),
      },
    },
  };
});

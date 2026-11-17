import prettierConfig from 'eslint-config-prettier';
import nextCoreWebVitals from 'eslint-config-next/core-web-vitals';
import nextTypescript from 'eslint-config-next/typescript';

const eslintConfig = [
  // 构建产物与生成文件不参与 lint（否则 12 万+ 误报）
  {
    ignores: [
      'dist/**',
      '.next/**',
      'out/**',
      'build/**',
      'next-env.d.ts',
      // 第三方压缩产物（cap wasm loader）
      'public/cap@0.0.6/**',
    ],
  },
  ...nextCoreWebVitals,
  ...nextTypescript,
  prettierConfig,
  {
    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/ban-ts-comment': 'off',
      '@typescript-eslint/no-unused-vars': 'warn',
      'react/no-unescaped-entities': 'off',
    },
  },
];

export default eslintConfig;

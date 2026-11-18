'use client';

import mermaid from 'mermaid';
import React from 'react';
import { sanitizeMermaidSvg } from './sanitizeMermaidSvg';

const MERMAID_CONFIG = {
  startOnLoad: false,
  theme: 'default' as const,
  securityLevel: 'strict' as const,
  fontFamily: 'inherit',
  suppressErrorRendering: true,
};

// 流式输出期间每个 token 都会重新渲染 markdown，未闭合的 mermaid 代码块
// 内容持续变化；若每个 token 都调度一次 mermaid.render 会造成 CPU 峰值。
// 对同一代码块做 trailing 去抖：内容停止变化一段时间后才真正渲染。
const MERMAID_RENDER_DEBOUNCE_MS = 500;

// ==================== 全局状态 ====================
let isMermaidInitialized = false;

/**
 * 初始化 Mermaid
 */
export const initializeMermaid = (): boolean => {
  if (!isMermaidInitialized) {
    try {
      mermaid.initialize(MERMAID_CONFIG);
      isMermaidInitialized = true;
      return true;
    } catch (error) {
      console.error('Mermaid initialization error:', error);
      return false;
    }
  }
  return true;
};

/**
 * 创建 Mermaid 渲染器
 */
export const createMermaidRenderer = (
  mermaidSuccessIdRef: React.RefObject<Map<number, string>>,
) => {
  // 同一 mermaid 块的待执行渲染任务（去抖用），key 为块序号
  const pendingTimers = new Map<number, ReturnType<typeof setTimeout>>();
  // 已成功渲染过的代码内容，用于跳过内容未变化的重复渲染
  const renderedCodes = new Map<number, string>();

  const render = (code: string, mermaidCount: number, className: string) => {
    initializeMermaid();
    void (async () => {
      try {
        const id = `mermaid-${Date.now()}-${Math.random()
          .toString(36)
          .slice(2, 9)}`;
        const renderResult = await mermaid.render(id, code);
        const safeSvg = sanitizeMermaidSvg(renderResult.svg);
        if (!safeSvg) return;
        mermaidSuccessIdRef.current?.set(mermaidCount, safeSvg);
        renderedCodes.set(mermaidCount, code);
        const mermaidContainer = document.querySelector(`.${className}`);
        if (mermaidContainer) {
          mermaidContainer.innerHTML = safeSvg;
        }
      } catch {}
    })();
  };

  type MermaidRenderFn = ((code: string, mermaidCount: number) => string) & {
    cancelAll: () => void;
  };

  const renderFn: MermaidRenderFn = (code, mermaidCount) => {
    const svg = mermaidSuccessIdRef.current?.get(mermaidCount) || '';
    const className = `mermaid-container-${mermaidCount}`;

    // 已用相同代码渲染成功且 svg 已缓存：新 HTML 已内嵌 svg，无需重复渲染
    if (svg && renderedCodes.get(mermaidCount) === code) {
      return `<pre><div class="mermaid-container ${className}">${svg}</div></pre>`;
    }

    // 取消同块上一次尚未执行的任务，只保留最后一次（trailing 去抖）
    const prevTimer = pendingTimers.get(mermaidCount);
    if (prevTimer !== undefined) {
      clearTimeout(prevTimer);
    }
    const timer = setTimeout(() => {
      pendingTimers.delete(mermaidCount);
      render(code, mermaidCount, className);
    }, MERMAID_RENDER_DEBOUNCE_MS);
    pendingTimers.set(mermaidCount, timer);

    return `<pre><div class="mermaid-container ${className}">${svg}</div></pre>`;
  };

  /**
   * 取消所有尚未执行的渲染任务（组件卸载时调用）
   */
  renderFn.cancelAll = () => {
    pendingTimers.forEach(timer => clearTimeout(timer));
    pendingTimers.clear();
  };

  return renderFn;
};

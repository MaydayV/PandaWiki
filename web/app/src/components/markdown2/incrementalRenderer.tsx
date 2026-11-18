/**
 * 增量渲染器 - 按顶层块比较新旧 HTML，仅替换从第一个差异块开始的部分
 *
 * 背景：流式输出时新 token 通常落在最后一个未闭合块（如 `<p>`）内部，
 * 旧的"字符串前缀比较"（newHtml.startsWith(oldHtml)）几乎必然失配，
 * 导致每个 token 都触发 `container.innerHTML = newHtml` 全量重建 DOM，
 * 造成图片闪跳、mermaid 重排、滚动抖动。
 *
 * 现方案：markdown-it 输出是一串顶层块级元素（块之间的换行为空白文本节点）。
 * 流式时除最后一块外，前面所有块的序列化结果保持不变，因此可以：
 *  1. 保留前面完全一致的块（其中 React 挂载的图片、已渲染的 mermaid 不受影响）；
 *  2. 从第一个差异块起移除旧节点，插入新后缀。
 *
 * 安全性：新后缀以字符串形式经 insertAdjacentHTML 插入，与 innerHTML
 * 具有相同的安全语义（解析产生的 script 不会执行）；XSS 清洗仍由上游
 * （markdown-it 渲染规则、sanitizeMermaidSvg、事件属性过滤）负责，本层不引入新内容。
 */

const escapeTextForHtml = (text: string): string =>
  text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');

/**
 * 将节点序列化为可与"新鲜解析结果"互相比较的 HTML 字符串。
 * 两侧（live DOM 与新解析的临时容器）都经过同一序列化函数，保证可比性。
 * 返回 null 表示该节点类型无法安全比较（调用方应退回全量重建）。
 */
function serializeNode(node: Node): string | null {
  if (node.nodeType === Node.TEXT_NODE) {
    // textContent 是解码后的文本，重新转义以参与比较/插入
    return escapeTextForHtml(node.textContent || '');
  }
  if (node.nodeType === Node.COMMENT_NODE) {
    return `<!--${node.nodeValue || ''}-->`;
  }
  if (node.nodeType === Node.ELEMENT_NODE) {
    return (node as Element).outerHTML;
  }
  return null;
}

/**
 * 主要的增量渲染函数
 */
export function incrementalRender(
  container: HTMLElement,
  newHtml: string,
): void {
  if (!container) return;

  try {
    // 用与容器相同的上下文（div）解析新 HTML，保证解析产物与 live DOM 可比
    const parser = document.createElement('div');
    parser.innerHTML = newHtml;

    const newNodes = Array.from(parser.childNodes);
    const newSerialized = newNodes.map(serializeNode);
    const oldSerialized = Array.from(container.childNodes).map(serializeNode);

    // 含无法安全比较的节点类型时退回全量重建（与旧行为一致）
    if (
      newSerialized.some(html => html === null) ||
      oldSerialized.some(html => html === null)
    ) {
      container.innerHTML = newHtml;
      return;
    }

    // 找到最长的相同顶层块前缀
    const minLength = Math.min(oldSerialized.length, newSerialized.length);
    let commonCount = 0;
    while (
      commonCount < minLength &&
      oldSerialized[commonCount] === newSerialized[commonCount]
    ) {
      commonCount++;
    }

    // 完全一致，无需更新
    if (
      commonCount === oldSerialized.length &&
      commonCount === newSerialized.length
    ) {
      return;
    }

    // 从第一个差异块起移除旧节点（保留的前缀块中已挂载的 React 图片组件、mermaid 均不受影响）
    while (container.childNodes.length > commonCount) {
      container.removeChild(container.lastChild as ChildNode);
    }

    // 以字符串形式插入新后缀（insertAdjacentHTML 与 innerHTML 一样不会执行脚本）
    const suffixHtml = newSerialized.slice(commonCount).join('');
    if (suffixHtml) {
      container.insertAdjacentHTML('beforeend', suffixHtml);
    }
  } catch (error) {
    console.error('增量渲染错误:', error);
    // 降级到完整渲染
    container.innerHTML = newHtml;
  }
}

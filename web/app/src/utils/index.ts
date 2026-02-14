import { ITreeItem } from '@/assets/type';
import { message } from '@ctzhian/ui';
import { ResolvingMetadata } from 'next';
import { MESSAGES } from '@/i18n/messages';
import { DEFAULT_LOCALE, resolveLanguage } from '@/i18n/locale';
export { getBasePath } from './getBasePath';
export { getImagePath } from './getImagePath';

export function addOpacityToColor(color: string, opacity: number) {
  let red, green, blue;

  if (color.startsWith('#')) {
    red = parseInt(color.slice(1, 3), 16);
    green = parseInt(color.slice(3, 5), 16);
    blue = parseInt(color.slice(5, 7), 16);
  } else if (color.startsWith('rgb')) {
    const matches = color.match(
      /^rgba?\((\d+),\s*(\d+),\s*(\d+)/,
    ) as RegExpMatchArray;
    red = parseInt(matches[1], 10);
    green = parseInt(matches[2], 10);
    blue = parseInt(matches[3], 10);
  } else {
    return '';
  }

  const alpha = opacity;

  return `rgba(${red}, ${green}, ${blue}, ${alpha})`;
}

/**
 * 复制文本到剪贴板
 * 优先使用现代 Clipboard API（需要安全上下文：HTTPS 或 localhost）
 * 降级使用 document.execCommand（兼容非 HTTPS 环境）
 */
export const copyText = (text: string, callback?: () => void) => {
  const locale =
    typeof document !== 'undefined'
      ? resolveLanguage(document.documentElement.lang || DEFAULT_LOCALE)
      : DEFAULT_LOCALE;
  const messages = MESSAGES[locale] || MESSAGES[DEFAULT_LOCALE];

  const fallbackCopy = () => {
    const textArea = document.createElement('textarea');
    textArea.value = text;
    textArea.style.position = 'fixed';
    textArea.style.opacity = '0';
    textArea.style.left = '-9999px';
    textArea.style.top = '-9999px';
    document.body.appendChild(textArea);
    textArea.focus();
    textArea.select();

    try {
      const successful = document.execCommand('copy');
      if (successful) {
        message.success(messages['common.copySuccess']);
        callback?.();
      } else {
        message.error(messages['common.copyFailed']);
      }
    } catch (err) {
      console.error('复制失败:', err);
      message.error(messages['common.copyFailed']);
    } finally {
      document.body.removeChild(textArea);
    }
  };

  if (navigator.clipboard && window.isSecureContext) {
    navigator.clipboard
      .writeText(text)
      .then(() => {
        message.success(messages['common.copySuccess']);
        callback?.();
      })
      .catch(err => {
        console.error('Clipboard API 失败，尝试降级方案:', err);
        fallbackCopy();
      });
  } else {
    fallbackCopy();
  }
};

export const formatMeta = async (
  {
    title,
    description,
    keywords,
  }: { title?: string; description?: string; keywords?: string | string[] },
  parent: ResolvingMetadata,
) => {
  const keywordsIsEmpty =
    !keywords || (Array.isArray(keywords) && !keywords.length);
  const {
    title: parentTitle,
    description: parentDescription,
    keywords: parentKeywords,
  } = await parent;

  return {
    title: title ? `${parentTitle?.absolute} - ${title}` : parentTitle,
    description: description || parentDescription,
    keywords: keywordsIsEmpty ? parentKeywords : keywords,
  };
};

export const parsePathname = (
  pathname: string,
): { page: string; id: string; hash: string; search: string } => {
  const [filterSearch, search] = pathname.split('?');
  const [path, hash] = filterSearch.split('#');
  const [page, id] = path.split('/').filter(Boolean);
  return {
    page,
    id,
    hash,
    search,
  };
};

/**
 * 过滤树形数据，只保留匹配搜索关键词的节点及其父节点
 */
export const filterTreeBySearch = (
  tree: ITreeItem[],
  searchTerm: string,
): ITreeItem[] => {
  if (!searchTerm.trim()) {
    return tree;
  }

  const filtered: ITreeItem[] = [];

  const filterNode = (node: ITreeItem): ITreeItem | null => {
    const nameMatches = node.name
      .toLowerCase()
      .includes(searchTerm.toLowerCase());

    // 递归过滤子节点
    const filteredChildren: ITreeItem[] = [];
    if (node.children) {
      for (const child of node.children) {
        const filteredChild = filterNode(child);
        if (filteredChild) {
          filteredChildren.push(filteredChild);
        }
      }
    }

    // 如果当前节点匹配或有匹配的子节点，则保留
    if (nameMatches || filteredChildren.length > 0) {
      return {
        ...node,
        children:
          filteredChildren.length > 0 ? filteredChildren : node.children,
        defaultExpand: true, // 搜索时展开所有匹配的节点
        expanded: true,
      };
    }

    return null;
  };

  for (const node of tree) {
    const filteredNode = filterNode(node);
    if (filteredNode) {
      filtered.push(filteredNode);
    }
  }

  return filtered;
};

export const deepSearchFirstNode = (
  tree: ITreeItem[],
): ITreeItem | undefined => {
  for (const node of tree) {
    if (node.type === 2) {
      return node;
    }
    if (node.children) {
      const result = deepSearchFirstNode(node.children);
      if (result) {
        return result;
      }
    }
  }
};

/**
 * 将树形结构扁平化为文档列表（只包含 type === 2 的文档节点）
 * 按照树的前序遍历顺序排列
 */
const flattenDocuments = (tree: ITreeItem[]): ITreeItem[] => {
  const documents: ITreeItem[] = [];

  const traverse = (nodes: ITreeItem[]) => {
    for (const node of nodes) {
      // 只添加文档节点（type === 2），不添加文件夹（type === 1）
      if (node.type === 2) {
        documents.push(node);
      }
      // 递归遍历子节点
      if (node.children && node.children.length > 0) {
        traverse(node.children);
      }
    }
  };

  traverse(tree);
  return documents;
};

/**
 * 根据当前文档 ID 查找上一篇和下一篇文档
 * @param tree 目录树结构
 * @param currentId 当前文档 ID
 * @returns 返回 { prev: 上一篇文档, next: 下一篇文档 }，如果不存在则返回 undefined
 */
export const findAdjacentDocuments = (
  tree: ITreeItem[],
  currentId: string,
): { prev?: ITreeItem; next?: ITreeItem } | undefined => {
  if (!tree || tree.length === 0 || !currentId) {
    return undefined;
  }

  // 扁平化树结构，只保留文档节点
  const documents = flattenDocuments(tree);

  // 找到当前文档的索引
  const currentIndex = documents.findIndex(doc => doc.id === currentId);

  // 如果找不到当前文档，返回 undefined
  if (currentIndex === -1) {
    return undefined;
  }

  // 获取上一篇和下一篇
  const prev = currentIndex > 0 ? documents[currentIndex - 1] : undefined;
  const next =
    currentIndex < documents.length - 1
      ? documents[currentIndex + 1]
      : undefined;

  return { prev, next };
};

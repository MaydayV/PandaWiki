'use client';

import React, { useState, useEffect } from 'react';
import { createRoot } from 'react-dom/client';
import { styled, SvgIcon, SvgIconProps } from '@mui/material';
import { useI18n } from '@/i18n/useI18n';

// ==================== 图片数据缓存工具函数 ====================
// 下载图片并转换为 blob URL
const fetchImageAsBlob = async (
  src: string,
  imageBlobCache: Map<string, string>,
): Promise<string> => {
  // 检查缓存
  if (imageBlobCache.has(src)) {
    return imageBlobCache.get(src)!;
  }

  try {
    const response = await fetch(src, {
      method: 'GET',
      mode: 'cors',
      credentials: 'omit',
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch image: ${response.status}`);
    }

    const blob = await response.blob();
    const blobUrl = URL.createObjectURL(blob);

    // 缓存 blob URL
    imageBlobCache.set(src, blobUrl);

    return blobUrl;
  } catch (error) {
    console.error('Error fetching image as blob:', error);
    throw error;
  }
};

// 清理图片 blob 缓存
export const clearImageBlobCache = (imageBlobCache: Map<string, string>) => {
  imageBlobCache.forEach(url => {
    URL.revokeObjectURL(url);
  });
  imageBlobCache.clear();
};

const StyledErrorContainer = styled('div')(({ theme }) => ({
  display: 'flex',
  flexDirection: 'column',
  padding: theme.spacing(1, 6),
  alignItems: 'center',
  justifyContent: 'center',
  gap: '8px',
  borderStyle: 'none',
  borderRadius: '10px',
  marginLeft: '5px',
  cursor: 'pointer',
  boxSizing: 'content-box' as const,
  backgroundColor: 'var(--mui-palette-background-default)',
  border: `1px dashed var(--mui-palette-divider)`,
  color: 'var(--mui-palette-text-tertiary)',
  fontSize: '14px',
}));

const StyledErrorText = styled('div')(() => ({
  fontSize: '12px',
  marginBottom: 10,
}));

export const ImageErrorIcon = (props: SvgIconProps) => {
  return (
    <SvgIcon
      viewBox='0 0 1024 1024'
      version='1.1'
      xmlns='http://www.w3.org/2000/svg'
      {...props}
    >
      <path
        d='M520 672L256 413.44l-109.76 93.76V246.72h261.12a41.6 41.6 0 1 0 0-82.88H104.64A41.6 41.6 0 0 0 64 205.44V800a41.6 41.6 0 0 0 41.6 41.6h267.84a40.96 40.96 0 0 0 32-67.52h21.76z'
        p-id='4874'
      ></path>
      <path
        d='M952 211.52a41.92 41.92 0 0 0-28.48-15.68l-310.08-32a41.6 41.6 0 0 0-8.32 82.88l267.2 27.52-55.04 411.84-113.28-160-99.2 17.92 42.56 96-123.84 123.52-32-1.6a41.6 41.6 0 1 0-4.16 82.88l352 17.92h1.92a41.28 41.28 0 0 0 41.28-35.84L960 242.88a42.56 42.56 0 0 0-8-31.36z'
        p-id='4875'
      ></path>
      <path
        d='M695.36 397.44m-64 0a64 64 0 1 0 128 0 64 64 0 1 0-128 0Z'
        p-id='4876'
      ></path>
    </SvgIcon>
  );
};

// 错误展示组件
const ImageErrorDisplay: React.FC<{ message: string }> = ({ message }) => (
  <StyledErrorContainer>
    <ImageErrorIcon
      sx={{ color: 'var(--mui-palette-text-tertiary)', fontSize: 140 }}
    />
    <StyledErrorText>{message}</StyledErrorText>
  </StyledErrorContainer>
);

// ==================== 类型定义 ====================
interface ImageComponentProps {
  src: string;
  alt: string;
  attrs: [string, string][];
  imageIndex: number;
  onLoad: (index: number, html: string) => void;
  onError: (index: number, html: string) => void;
  imageBlobCache: Map<string, string>;
}

// ==================== 图片组件 ====================
const ImageComponent: React.FC<ImageComponentProps> = ({
  src,
  alt,
  attrs,
  imageIndex,
  onLoad,
  onError,
  imageBlobCache,
}) => {
  const [status, setStatus] = useState<'loading' | 'success' | 'error'>(
    'loading',
  );
  const [blobUrl, setBlobUrl] = useState<string>('');
  const { t } = useI18n();

  // 基础样式对象
  const baseStyleObj = {
    borderStyle: 'none' as const,
    borderRadius: '10px',
    marginLeft: '5px',
    boxShadow: '0px 0px 3px 1px rgba(0,0,5,0.15)',
    cursor: 'pointer',
    maxWidth: '60%',
    boxSizing: 'content-box' as const,
    backgroundColor: 'var(--color-canvas-default)',
  };

  // 获取图片 blob URL
  useEffect(() => {
    let mounted = true;
    fetchImageAsBlob(src, imageBlobCache)
      .then(url => {
        if (mounted) {
          setBlobUrl(url);
        }
      })
      .catch(err => {
        console.error('Failed to fetch image blob:', err);
        if (mounted) {
          // 如果获取 blob 失败，回退到使用原始 URL
          setBlobUrl(src);
        }
      });

    return () => {
      mounted = false;
    };
  }, [src, imageBlobCache]);

  // 解析自定义样式
  const parseStyleString = (styleStr: string) => {
    if (!styleStr) return {};
    const styleObj: Record<string, string> = {};
    const declarations = styleStr.split(';').filter(Boolean);
    declarations.forEach(decl => {
      const [prop, value] = decl.split(':').map(s => s.trim());
      if (prop && value) {
        const camelProp = prop.replace(/-([a-z])/g, (_, letter) =>
          letter.toUpperCase(),
        );
        styleObj[camelProp] = value;
      }
    });
    return styleObj;
  };

  // 获取其他属性
  const getOtherProps = () => {
    const props: Record<string, string> = {};
    const customStyle = attrs.find(([name]) => name === 'style')?.[1] || '';

    attrs.forEach(([name, value]) => {
      if (!['src', 'alt', 'style'].includes(name)) {
        props[name] = value;
      }
    });

    return {
      ...props,
      style: {
        ...baseStyleObj,
        ...parseStyleString(customStyle),
      } as React.CSSProperties,
    };
  };

  const handleLoad = () => {
    setStatus('success');
    const classname = `.image-container-${imageIndex}`;
    const containerDom = document.querySelector(classname);
    if (containerDom) {
      onLoad(imageIndex, containerDom.outerHTML);
    }
  };

  const handleError = () => {
    setStatus('error');
    // 通知父组件错误状态
    const classname = `.image-container-${imageIndex}`;
    const containerDom = document.querySelector(classname);
    if (containerDom) {
      requestAnimationFrame(() => {
        onError(imageIndex, containerDom.outerHTML);
      });
    }
  };

  return (
    <>
      {status === 'error' ? (
        <ImageErrorDisplay message={t('image.loadFailed')} />
      ) : blobUrl ? (
        /* eslint-disable-next-line @next/next/no-img-element */
        <img
          src={blobUrl}
          alt={alt || 'markdown-img'}
          referrerPolicy='no-referrer'
          onLoad={handleLoad}
          onError={handleError}
          data-original-src={src}
          className='markdown-image'
          {...getOtherProps()}
        />
      ) : (
        // 加载中显示占位符
        <div
          style={{
            ...baseStyleObj,
            minHeight: '100px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            color: '#999',
          }}
        >
          {t('image.loading')}
        </div>
      )}
    </>
  );
};

// ==================== 图片渲染器 ====================
export interface ImageRendererOptions {
  onImageLoad: (index: number, html: string) => void;
  onImageError: (index: number, html: string) => void;
  imageRenderCache: Map<number, string>;
  imageBlobCache: Map<string, string>;
}

export const createImageRenderer = (options: ImageRendererOptions) => {
  const { onImageLoad, onImageError, imageRenderCache, imageBlobCache } =
    options;
  return (
    src: string,
    alt: string,
    attrs: [string, string][] = [],
    imageIndex: number,
  ) => {
    // 检查缓存
    const cached = imageRenderCache.get(imageIndex);
    if (cached) {
      return cached;
    }

    // 直接返回占位符，让 React 组件在 DOM 中渲染
    const placeholderHtml = `<div class="image-container image-container-${imageIndex}"></div>`;

    // 使用 requestAnimationFrame 确保在下一帧渲染时执行
    requestAnimationFrame(() => {
      const placeholder = document.querySelector(
        `.image-container-${imageIndex}`,
      );
      if (placeholder) {
        const root = createRoot(placeholder);
        root.render(
          <ImageComponent
            src={src}
            alt={alt}
            attrs={attrs}
            imageIndex={imageIndex}
            onLoad={onImageLoad}
            onError={onImageError}
            imageBlobCache={imageBlobCache}
          />,
        );
      } else {
        console.warn(`Placeholder with index ${imageIndex} not found`);
      }
    });

    return placeholderHtml;
  };
};

'use client';
import React, { useMemo } from 'react';
import { styled, Box, useTheme } from '@mui/material';

export type WatermarkProps = {
  content?: string | string[];
  fontSize?: number;
  color?: string;
  opacity?: number; // 0~1
  mode?: 'visible' | 'invisible';
  rotate?: number; // deg
  gapX?: number; // 水印水平间距
  gapY?: number; // 水印垂直间距
  zIndex?: number;
  fullPage?: boolean; // 是否铺满全页面
  fontFamily?: string;
  fontWeight?: number | string;
  lineHeight?: number; // 行高倍数，仅对多行文本生效
  offsetLeft?: number; // 背景平铺的起始左偏移
  offsetTop?: number; // 背景平铺的起始上偏移
  tileWidth?: number; // 单元格宽度（不传则自动根据文本与 gap 计算）
  tileHeight?: number; // 单元格高度
  pointerEvents?: 'auto' | 'none';
  children?: React.ReactNode;
};

type GenerateOptions = {
  contentLines: string[];
  fontSize: number;
  fontFamily: string;
  fontWeight: number | string;
  color: string;
  opacity: number;
  rotate: number;
  gapX: number;
  gapY: number;
  lineHeight: number;
  tileWidth?: number;
  tileHeight?: number;
};

function generateWatermarkDataUrl(options: GenerateOptions): string | null {
  if (typeof window === 'undefined') return null;
  const {
    contentLines,
    fontSize,
    fontFamily,
    fontWeight,
    color,
    opacity,
    rotate,
    gapX,
    gapY,
    lineHeight,
    tileWidth,
    tileHeight,
  } = options;

  const canvas = document.createElement('canvas');
  const ctx = canvas.getContext('2d');
  if (!ctx) return null;

  // 移除设备像素比处理，直接使用逻辑尺寸确保文字大小一致
  const font = `${typeof fontWeight === 'number' ? fontWeight : fontWeight} ${fontSize}px ${fontFamily}`;
  ctx.font = font;

  // 估算文本最大宽度
  let maxTextWidth = 0;
  for (const line of contentLines) {
    const metrics = ctx.measureText(line);
    maxTextWidth = Math.max(maxTextWidth, metrics.width);
  }

  const textBlockHeight = contentLines.length * fontSize * lineHeight;

  // 旋转后需要更大的画布，简单起见：给一定余量
  const estimatedWidth = Math.ceil(maxTextWidth + gapX);
  const estimatedHeight = Math.ceil(textBlockHeight + gapY);

  const logicalWidth = tileWidth ?? estimatedWidth;
  const logicalHeight = tileHeight ?? estimatedHeight;

  // 直接使用逻辑尺寸，不乘以设备像素比
  canvas.width = Math.max(1, logicalWidth);
  canvas.height = Math.max(1, logicalHeight);

  ctx.clearRect(0, 0, logicalWidth, logicalHeight);
  ctx.globalAlpha = opacity;
  ctx.fillStyle = color;
  ctx.textAlign = 'center';
  ctx.textBaseline = 'middle';
  ctx.font = font;

  // 将原点移到单元格中心旋转后绘制，使平铺更自然
  const centerX = logicalWidth / 2;
  const centerY = logicalHeight / 2;
  ctx.translate(centerX, centerY);
  ctx.rotate((Math.PI / 180) * rotate);

  const totalTextHeight = contentLines.length * fontSize * lineHeight;
  const startY = -totalTextHeight / 2 + fontSize * 0.5;
  for (let i = 0; i < contentLines.length; i += 1) {
    const line = contentLines[i];
    const y = startY + i * fontSize * lineHeight;
    ctx.fillText(line, 0, y);
  }

  return canvas.toDataURL('image/png');
}

const StyledWatermarkOverlay = styled(Box)(() => ({
  position: 'absolute',
  left: 0,
  top: 0,
  right: 0,
  bottom: 0,
  width: '100%',
  height: '100%',
  pointerEvents: 'none',
}));

const StyledFullPageOverlay = styled(Box)(() => ({
  position: 'fixed',
  left: 0,
  top: 0,
  width: '100%',
  height: '100%',
  pointerEvents: 'none',
}));

const StyledContainer = styled(Box)(() => ({
  position: 'relative',
  height: '100%',
  width: '100%',
}));

export default function Watermark(props: WatermarkProps) {
  const theme = useTheme();
  const {
    content,
    fontSize = 14,
    color,
    opacity,
    rotate = -22,
    gapX = 120,
    gapY = 120,
    zIndex = 9999,
    fullPage = true,
    fontFamily = 'sans-serif',
    fontWeight = 'normal',
    lineHeight = 1.2,
    offsetLeft = 0,
    offsetTop = 0,
    tileWidth,
    tileHeight,
    pointerEvents = 'none',
    children,
  } = props;

  // 解析 rgb/rgba/hex 颜色为 {r,g,b,a}
  const parseColorToRgba = (
    input: string,
  ): { r: number; g: number; b: number; a: number } | null => {
    if (!input) return null;
    const hexMatch = input.trim().match(/^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/);
    if (hexMatch) {
      let hex = hexMatch[1];
      if (hex.length === 3) {
        hex = hex
          .split('')
          .map(c => c + c)
          .join('');
      }
      const r = parseInt(hex.slice(0, 2), 16);
      const g = parseInt(hex.slice(2, 4), 16);
      const b = parseInt(hex.slice(4, 6), 16);
      return { r, g, b, a: 1 };
    }
    const rgbaMatch = input.trim().match(/^rgba?\(([^)]+)\)$/i);
    if (rgbaMatch) {
      const parts = rgbaMatch[1].split(',').map(v => v.trim());
      const r = parseInt(parts[0], 10);
      const g = parseInt(parts[1], 10);
      const b = parseInt(parts[2], 10);
      const a = parts[3] !== undefined ? parseFloat(parts[3]) : 1;
      if ([r, g, b].some(v => Number.isNaN(v))) return null;
      return { r, g, b, a: Number.isNaN(a) ? 1 : a };
    }
    return null;
  };

  const invertRgb = (r: number, g: number, b: number) => ({
    r: 255 - r,
    g: 255 - g,
    b: 255 - b,
  });

  const resolvedColor = useMemo(() => {
    if (color) return color;
    if (typeof window === 'undefined') return theme.palette.text.disabled;
    const rootEl = document.getElementById('app-theme-root');
    if (!rootEl) return theme.palette.text.disabled;
    const bg = getComputedStyle(rootEl).backgroundColor;
    const rgba = parseColorToRgba(bg);
    if (!rgba) return theme.palette.text.disabled;
    const inv = invertRgb(rgba.r, rgba.g, rgba.b);
    return `rgba(${inv.r}, ${inv.g}, ${inv.b}, 1)`;
  }, [color, theme.palette.text.disabled]);

  const contentLines = useMemo(
    () => (Array.isArray(content) ? content : content ? [content] : []),
    [content],
  );
  // dataUrl 是纯派生值（离屏 canvas 生成，无外部副作用），直接用 useMemo 计算，
  // 避免 effect 内同步 setState 触发级联渲染
  const dataUrl = useMemo(
    () =>
      generateWatermarkDataUrl({
        contentLines,
        fontSize,
        fontFamily,
        fontWeight,
        color: resolvedColor,
        opacity: opacity ?? 0.1,
        rotate,
        gapX,
        gapY,
        lineHeight,
        tileWidth,
        tileHeight,
      }),
    [
      contentLines,
      fontSize,
      fontFamily,
      fontWeight,
      resolvedColor,
      opacity,
      rotate,
      gapX,
      gapY,
      lineHeight,
      tileWidth,
      tileHeight,
    ],
  );

  const backgroundStyles = useMemo(
    () => ({
      backgroundImage: dataUrl ? `url(${dataUrl})` : undefined,
      backgroundRepeat: 'repeat' as const,
      backgroundPosition: `${offsetLeft}px ${offsetTop}px`,
      zIndex,
      pointerEvents,
    }),
    [dataUrl, offsetLeft, offsetTop, zIndex, pointerEvents],
  );

  if (fullPage && !children) {
    return <StyledFullPageOverlay sx={backgroundStyles} />;
  }

  return (
    <StyledContainer>
      {children}
      <StyledWatermarkOverlay sx={backgroundStyles} />
    </StyledContainer>
  );
}

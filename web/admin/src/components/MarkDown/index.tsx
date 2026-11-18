import { addOpacityToColor, copyText } from '@/utils';
import { Box, IconButton, useTheme } from '@mui/material';
import 'katex/dist/katex.min.css';
import React, { useState } from 'react';
import ReactMarkdown from 'react-markdown';
// 使用 light 构建,避免捆绑 highlight.js 全量语言集,仅注册下方常用语言
import { Light as SyntaxHighlighter } from 'react-syntax-highlighter';
import bash from 'react-syntax-highlighter/dist/esm/languages/hljs/bash';
import c from 'react-syntax-highlighter/dist/esm/languages/hljs/c';
import cpp from 'react-syntax-highlighter/dist/esm/languages/hljs/cpp';
import csharp from 'react-syntax-highlighter/dist/esm/languages/hljs/csharp';
import css from 'react-syntax-highlighter/dist/esm/languages/hljs/css';
import dart from 'react-syntax-highlighter/dist/esm/languages/hljs/dart';
import diff from 'react-syntax-highlighter/dist/esm/languages/hljs/diff';
import dockerfile from 'react-syntax-highlighter/dist/esm/languages/hljs/dockerfile';
import go from 'react-syntax-highlighter/dist/esm/languages/hljs/go';
import ini from 'react-syntax-highlighter/dist/esm/languages/hljs/ini';
import java from 'react-syntax-highlighter/dist/esm/languages/hljs/java';
import javascript from 'react-syntax-highlighter/dist/esm/languages/hljs/javascript';
import json from 'react-syntax-highlighter/dist/esm/languages/hljs/json';
import kotlin from 'react-syntax-highlighter/dist/esm/languages/hljs/kotlin';
import less from 'react-syntax-highlighter/dist/esm/languages/hljs/less';
import lua from 'react-syntax-highlighter/dist/esm/languages/hljs/lua';
import makefile from 'react-syntax-highlighter/dist/esm/languages/hljs/makefile';
import markdown from 'react-syntax-highlighter/dist/esm/languages/hljs/markdown';
import nginx from 'react-syntax-highlighter/dist/esm/languages/hljs/nginx';
import objectivec from 'react-syntax-highlighter/dist/esm/languages/hljs/objectivec';
import perl from 'react-syntax-highlighter/dist/esm/languages/hljs/perl';
import php from 'react-syntax-highlighter/dist/esm/languages/hljs/php';
import properties from 'react-syntax-highlighter/dist/esm/languages/hljs/properties';
import protobuf from 'react-syntax-highlighter/dist/esm/languages/hljs/protobuf';
import python from 'react-syntax-highlighter/dist/esm/languages/hljs/python';
import ruby from 'react-syntax-highlighter/dist/esm/languages/hljs/ruby';
import rust from 'react-syntax-highlighter/dist/esm/languages/hljs/rust';
import scala from 'react-syntax-highlighter/dist/esm/languages/hljs/scala';
import scss from 'react-syntax-highlighter/dist/esm/languages/hljs/scss';
import shell from 'react-syntax-highlighter/dist/esm/languages/hljs/shell';
import sql from 'react-syntax-highlighter/dist/esm/languages/hljs/sql';
import swift from 'react-syntax-highlighter/dist/esm/languages/hljs/swift';
import typescript from 'react-syntax-highlighter/dist/esm/languages/hljs/typescript';
import xml from 'react-syntax-highlighter/dist/esm/languages/hljs/xml';
import yaml from 'react-syntax-highlighter/dist/esm/languages/hljs/yaml';
import { anOldHope } from 'react-syntax-highlighter/dist/esm/styles/hljs';
import rehypeKatex from 'rehype-katex';
import rehypeRaw from 'rehype-raw';
import rehypeSanitize, { defaultSchema } from 'rehype-sanitize';
import remarkBreaks from 'remark-breaks';
import remarkGfm from 'remark-gfm';
import remarkMath from 'remark-math';
import { IconXiajiantou } from '@panda-wiki/icons';
import { getBasePath } from '@/utils/getBasePath';

interface MarkDownProps {
  loading?: boolean;
  content: string;
}

// 代码块语言来自 markdown 围栏(如 ```js),此处注册常用语言集;
// hljs 注册语言时会一并注册其别名(如 js/jsx、ts/tsx、py、sh、yml、html 等)。
// 未注册的语言会走 hljs 自动检测,不影响渲染。
const hljsLanguages = {
  bash,
  c,
  cpp,
  csharp,
  css,
  dart,
  diff,
  dockerfile,
  go,
  ini,
  java,
  javascript,
  json,
  kotlin,
  less,
  lua,
  makefile,
  markdown,
  nginx,
  objectivec,
  perl,
  php,
  properties,
  protobuf,
  python,
  ruby,
  rust,
  scala,
  scss,
  shell,
  sql,
  swift,
  typescript,
  xml,
  yaml,
};

Object.entries(hljsLanguages).forEach(([lang, language]) => {
  SyntaxHighlighter.registerLanguage(lang, language);
});

const MarkDown = ({ loading = false, content }: MarkDownProps) => {
  const theme = useTheme();
  const [showThink, setShowThink] = useState(false);

  let answer = content;
  if (!answer.includes('\n\n</think>')) {
    const idx = answer.indexOf('\n</think>');
    if (idx !== -1) {
      answer = content.slice(0, idx) + '\n\n</think>' + content.slice(idx + 9);
    }
  }

  if (content.length === 0) return null;

  return (
    <Box
      className='markdown-body'
      id='markdown-body'
      sx={{
        fontSize: '14px',
        background: 'transparent',
        '#chat-thinking': {
          display: 'flex',
          alignItems: 'flex-end',
          gap: '16px',
          fontSize: '12px',
          color: 'text.tertiary',
          marginBottom: '40px',
          lineHeight: '20px',
          backgroundColor: 'background.paper3',
          padding: '16px',
          cursor: 'pointer',
          borderRadius: '10px',
          div: {
            transition: 'height 0.3s',
            overflow: 'hidden',
            height: showThink ? 'auto' : '60px',
          },
        },
        // LaTeX公式样式
        '.katex': {
          display: 'inline-block',
          fontSize: '24px',
          py: 2,
          color: 'text.primary',
        },
        '.katex-display': {
          textAlign: 'center',
          margin: '1em 0',
          '& > .katex': {
            display: 'inline-block',
            fontSize: '20px',
            py: 2,
            color: 'text.primary',
          },
        },
      }}
    >
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkBreaks, remarkMath]}
        rehypePlugins={[
          rehypeRaw,
          [
            rehypeSanitize,
            {
              tagNames: [
                ...(defaultSchema.tagNames! as string[]),
                'think',
                'error',
              ],
            },
          ],
          rehypeKatex,
        ]}
        components={{
          // eslint-disable-next-line @typescript-eslint/ban-ts-comment
          // @ts-ignore
          think: (props: React.HTMLAttributes<HTMLElement>) => {
            return (
              <div id='chat-thinking'>
                <div
                  className={!showThink ? 'three-ellipsis' : ''}
                  {...props}
                ></div>
                {!loading && (
                  <IconButton
                    size='small'
                    onClick={() => setShowThink(!showThink)}
                    sx={{
                      bgcolor: 'background.paper',
                      ':hover': {
                        bgcolor: addOpacityToColor(
                          theme.palette.primary.main,
                          0.1,
                        ),
                        color: theme.palette.primary.main,
                      },
                    }}
                  >
                    <IconXiajiantou
                      sx={{
                        fontSize: 18,
                        flexShrink: 0,
                        transform: showThink
                          ? 'rotate(-180deg)'
                          : 'rotate(0deg)',
                      }}
                    />
                  </IconButton>
                )}
              </div>
            );
          },
          error: (props: React.HTMLAttributes<HTMLElement>) => {
            return <div className='chat-error' {...props}></div>;
          },
          h1: (props: React.HTMLAttributes<HTMLHeadingElement>) => (
            <h2 {...props} />
          ),
          a: ({
            children,
            style,
            ...rest
          }: React.HTMLAttributes<HTMLAnchorElement>) => (
            <a
              {...rest}
              target='_blank'
              rel='noopener noreferrer'
              style={{
                color: theme.palette.primary.main,
                textDecoration: 'underline',
                ...style,
              }}
            >
              {children}
            </a>
          ),
          img: (props: React.ImgHTMLAttributes<HTMLImageElement>) => {
            const { style, alt, src, ...rest } = props;
            return (
              <img
                alt={alt || 'markdown-img'}
                {...rest}
                src={getBasePath(src || '')}
                style={{
                  ...style,
                  borderRadius: '10px',
                  marginLeft: '5px',
                  boxShadow: '0px 0px 3px 1px rgba(0,0,5,0.15)',
                  cursor: 'pointer',
                }}
                referrerPolicy='no-referrer'
              />
            );
          },
          code({
            children,
            className,
            ...rest
          }: React.HTMLAttributes<HTMLElement>) {
            const match = /language-(\w+)/.exec(className || '');
            return match ? (
              <SyntaxHighlighter
                showLineNumbers
                {...rest}
                language={match[1] || 'bash'}
                style={anOldHope}
                onClick={() => {
                  copyText(String(children).replace(/\n$/, ''));
                }}
              >
                {String(children).replace(/\n$/, '')}
              </SyntaxHighlighter>
            ) : (
              <code
                {...rest}
                className={className}
                onClick={() => {
                  copyText(String(children));
                }}
              >
                {children}
              </code>
            );
          },
        }}
      >
        {answer}
      </ReactMarkdown>
    </Box>
  );
};

export default MarkDown;

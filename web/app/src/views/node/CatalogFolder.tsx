import { ITreeItem } from '@/assets/type';
import { IconXiajiantou, IconWenjianjia, IconWenjian } from '@panda-wiki/icons';
import { useStore } from '@/provider';
import { useBasePath } from '@/hooks';
import { addOpacityToColor } from '@/utils';
import { Ellipsis } from '@ctzhian/ui';
import { Box, Stack, useTheme, IconButton } from '@mui/material';
import Link from 'next/link';
import { useParams } from 'next/navigation';

interface CatalogFolderProps {
  item: ITreeItem;
  depth?: number;
}

const CatalogFolder = ({ item, depth = 1 }: CatalogFolderProps) => {
  const theme = useTheme();
  const { themeMode = 'light', setTree } = useStore();
  const params = useParams() || {};
  const activeId = params.id as string;
  const basePath = useBasePath();

  return (
    <Stack key={item.id} gap={0.5}>
      <Stack
        direction='row'
        alignItems='center'
        justifyContent='space-between'
        gap={0.5}
        sx={{
          position: 'relative',
          lineHeight: '40px',
          cursor: 'pointer',
          borderRadius: '10px',
          color: activeId === item.id ? 'primary.main' : 'text.primary',
          bgcolor:
            activeId === item.id
              ? addOpacityToColor(theme.palette.primary.main, 0.08)
              : 'transparent',
          transition: 'all 0.2s ease-in-out',
          '&:hover': {
            color: activeId === item.id ? 'primary.main' : 'text.primary',
            bgcolor:
              activeId === item.id
                ? addOpacityToColor(theme.palette.primary.main, 0.08)
                : themeMode === 'dark'
                  ? '#394052'
                  : 'background.paper3',
          },
        }}
        id={`catalog-item-${item.id}`}
      >
        {item.type === 2 ? (
          <Box sx={{ flex: 1 }}>
            <Link href={`${basePath}/node/${item.id}`} prefetch={false}>
              <Box sx={{ pl: depth * 2, pr: 1 }}>
                <Stack direction='row' alignItems='center' gap={1}>
                  {item.emoji ? (
                    <Box sx={{ flexShrink: 0, fontSize: 14 }}>{item.emoji}</Box>
                  ) : (
                    <IconWenjian sx={{ flexShrink: 0, fontSize: 12 }} />
                  )}
                  <Ellipsis sx={{ flex: 1, width: 0, pr: 1 }}>
                    {item.name}
                  </Ellipsis>
                </Stack>
              </Box>
            </Link>
          </Box>
        ) : (
          <Stack
            direction='row'
            alignItems='center'
            justifyContent={'space-between'}
            sx={{ flex: 1, minWidth: 0, pl: depth * 2, pr: 1 }}
          >
            <Link
              href={`${basePath}/node/${item.id}`}
              prefetch={false}
              style={{ flex: 1, minWidth: 0, display: 'block' }}
            >
              <Stack
                direction='row'
                alignItems='center'
                gap={1}
                sx={{ flex: 1, minWidth: 0 }}
              >
                {item.emoji ? (
                  <Box sx={{ flexShrink: 0, fontSize: 12 }}>{item.emoji}</Box>
                ) : item.type === 1 ? (
                  <IconWenjianjia sx={{ flexShrink: 0, fontSize: 12 }} />
                ) : (
                  <IconWenjian sx={{ flexShrink: 0, fontSize: 12 }} />
                )}
                <Ellipsis sx={{ flex: 1, width: 0, pr: 1 }}>
                  {item.name}
                </Ellipsis>
              </Stack>
            </Link>

            <IconButton
              size='small'
              sx={{
                flexShrink: 0,
                position: 'relative',
                zIndex: 1,
                '&:hover': {
                  color: 'primary.main',
                },
              }}
              onClick={e => {
                e.stopPropagation();
                if (item.type === 1) {
                  setTree?.(tree =>
                    (tree || []).map(node => toggleNodeExpanded(node, item.id)),
                  );
                  return;
                }
              }}
            >
              <IconXiajiantou
                sx={{
                  flexShrink: 0,
                  fontSize: 16,
                  transform: item.expanded ? 'none' : 'rotate(-90deg)',
                  transition: 'transform 0.2s',
                }}
              />
            </IconButton>
          </Stack>
        )}
      </Stack>
      {item.children && item.children.length > 0 && item.expanded && (
        <Stack gap={0.5}>
          {item.children.map(child => (
            <CatalogFolder key={child.id} depth={depth + 1} item={child} />
          ))}
        </Stack>
      )}
    </Stack>
  );
};

const toggleNodeExpanded = (node: ITreeItem, id: string): ITreeItem => {
  if (node.id === id) {
    return { ...node, expanded: !node.expanded };
  }
  if (!node.children?.length) return node;
  return {
    ...node,
    children: node.children.map(child => toggleNodeExpanded(child, id)),
  };
};

export default CatalogFolder;

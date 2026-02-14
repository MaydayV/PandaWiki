import { ITreeItem } from '@/api';
import Cascader from '@/components/Cascader';
import { setProperty } from '@/components/TreeDragSortable/utilities';
import { V1NodeDetailResp } from '@/request/types';
import { useAppSelector } from '@/store';
import { addOpacityToColor } from '@/utils';
import { Ellipsis } from '@ctzhian/ui';
import { alpha, Box, IconButton, Stack, useTheme } from '@mui/material';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useLocation, useNavigate, useParams } from 'react-router-dom';
import DocAddByCustomText from '../../component/DocAddByCustomText';
import KBSwitch from './KBSwitch';
import {
  IconIcon_tool_close,
  IconWenjianjia,
  IconWenjian,
  IconMulushouqi,
  IconXiajiantou,
} from '@panda-wiki/icons';

interface CatalogProps {
  curNode: V1NodeDetailResp;
  setCatalogOpen: (open: boolean) => void;
  catalogData: ITreeItem[];
  onRefresh: () => Promise<ITreeItem[]>;
}

const Catalog = ({
  curNode,
  setCatalogOpen,
  catalogData: externalData,
  onRefresh,
}: CatalogProps) => {
  const theme = useTheme();
  const navigate = useNavigate();
  const { id = '' } = useParams();
  const { pathname } = useLocation();
  const { kb_id = '' } = useAppSelector(state => state.config);

  const isHistory = useMemo(() => {
    return pathname.includes('/doc/editor/history');
  }, [pathname]);

  const [data, setData] = useState<ITreeItem[]>([]);
  const [expandedFolders, setExpandedFolders] = useState<Set<string>>(
    new Set(),
  );

  const [opraParentId, setOpraParentId] = useState<string>('');
  const [docFileKey, setDocFileKey] = useState<1 | 2>(1);
  const [customDocOpen, setCustomDocOpen] = useState(false);

  const ImportContentWays = {
    docFile: {
      label: '创建文件夹',
      onClick: (parentId: string) => {
        setOpraParentId(parentId);
        setDocFileKey(1);
        setCustomDocOpen(true);
      },
    },
    customDoc: {
      label: '创建文档',
      onClick: (parentId: string) => {
        setOpraParentId(parentId);
        setDocFileKey(2);
        setCustomDocOpen(true);
      },
    },
  };

  const getCatalogData = useCallback(() => {
    onRefresh().then(tree => {
      setData(tree);
      // 计算当前文档的所有父级文件夹，并默认展开
      try {
        const currentId = id as string;
        if (!currentId) {
          setExpandedFolders(new Set());
          return;
        }

        // 从树形结构中构建 id 到节点的映射
        const buildMap = (items: ITreeItem[], map: Map<string, ITreeItem>) => {
          items.forEach(item => {
            map.set(item.id, item);
            if (item.children && item.children.length > 0) {
              buildMap(item.children, map);
            }
          });
        };

        const map = new Map<string, ITreeItem>();
        buildMap(tree, map);

        const expanded = new Set<string>();
        let cur = map.get(currentId);
        while (cur && cur.parentId) {
          const parent = map.get(cur.parentId);
          if (!parent) break;
          if (parent.type === 1 && parent.id) {
            expanded.add(parent.id);
          }
          cur = parent;
        }
        setExpandedFolders(expanded);
      } catch (e) {
        setExpandedFolders(new Set());
      }
    });
  }, [onRefresh, id]);

  // 同步外部数据到内部状态
  useEffect(() => {
    if (externalData.length > 0) {
      setData(externalData);
    }
  }, [externalData]);

  const toggleFolder = (folderId: string) => {
    setExpandedFolders(prev => {
      const newSet = new Set(prev);
      if (newSet.has(folderId)) {
        newSet.delete(folderId);
      } else {
        newSet.add(folderId);
      }
      return newSet;
    });
  };

  const renderAdd = (parentId: string) => {
    return (
      <Box sx={{ flexShrink: 0 }} onClick={e => e.stopPropagation()}>
        <Cascader
          anchorOrigin={{
            vertical: 'bottom',
            horizontal: 'left',
          }}
          transformOrigin={{
            vertical: 'top',
            horizontal: 'left',
          }}
          context={
            <IconButton>
              <IconIcon_tool_close
                className='catalog-folder-add-icon'
                sx={{
                  fontSize: 16,
                  color: 'action.selected',
                  transform: 'rotate(45deg)',
                }}
              />
            </IconButton>
          }
          list={Object.entries(ImportContentWays).map(([key, value]) => ({
            key,
            label: (
              <Box key={key}>
                <Stack
                  direction={'row'}
                  alignItems={'center'}
                  gap={1}
                  sx={{
                    fontSize: 14,
                    px: 2,
                    lineHeight: '40px',
                    height: 40,
                    width: 180,
                    borderRadius: '5px',
                    cursor: 'pointer',
                    ':hover': {
                      bgcolor: addOpacityToColor(
                        theme.palette.primary.main,
                        0.1,
                      ),
                    },
                  }}
                  onClick={() => value.onClick(parentId)}
                >
                  {value.label}
                </Stack>
                {key === 'OfflineFile' && (
                  <Box
                    sx={{
                      borderTop: '1px solid',
                      borderColor: theme.palette.divider,
                      my: 0.5,
                    }}
                  />
                )}
              </Box>
            ),
          }))}
        />
      </Box>
    );
  };

  const renderTree = (items: ITreeItem[], pl = 2.5, depth = 1) => {
    const sortedItems = [...items].sort(
      (a, b) => (a.order ?? 0) - (b.order ?? 0),
    );
    return sortedItems.map(item => (
      <Stack
        key={item.id}
        sx={{
          position: 'relative',
        }}
      >
        <Stack
          direction={'row'}
          alignItems={'center'}
          gap={1}
          sx={{
            pl: pl * depth,
            py: 0.75,
            borderRadius: 1,
            cursor: 'pointer',
            fontWeight: item.type === 1 ? 'bold' : 'normal',
            color: id === item.id ? 'primary.main' : 'text.primary',
            bgcolor:
              id === item.id
                ? alpha(theme.palette.primary.main, 0.1)
                : 'transparent',
            '&:hover .catalog-folder-add-icon': {
              color: 'text.tertiary',
            },
            '&:hover': {
              bgcolor:
                id === item.id
                  ? alpha(theme.palette.primary.main, 0.1)
                  : 'action.hover',
            },
          }}
          onClick={async () => {
            if (item.type === 1) {
              toggleFolder(item.id);
            } else {
              // if (edited) await save(true);
              if (isHistory) {
                navigate(`/doc/editor/history/${item.id}`);
              } else {
                navigate(`/doc/editor/${item.id}`);
              }
            }
          }}
        >
          {item.type === 1 && (
            <Box
              sx={{
                position: 'absolute',
                left: -18 + (pl || 0) * 8 * depth,
                top: 13,
              }}
            >
              <IconXiajiantou
                sx={{
                  fontSize: 16,
                  color: 'text.disabled',
                  flexShrink: 0,
                  transform: expandedFolders.has(item.id)
                    ? 'none'
                    : 'rotate(-90deg)',
                  transition: 'transform 0.2s',
                }}
              />
            </Box>
          )}
          {item.emoji ? (
            <Box sx={{ fontSize: 14, flexShrink: 0 }}>{item.emoji}</Box>
          ) : item.type === 1 ? (
            <IconWenjianjia
              sx={{ color: '#2f80f7', flexShrink: 0, fontSize: 14 }}
            />
          ) : (
            <IconWenjian
              sx={{ color: '#2f80f7', flexShrink: 0, fontSize: 14 }}
            />
          )}
          <Ellipsis>{item.name}</Ellipsis>
          {item.content_type === 'md' && (
            <Box
              sx={{
                flexShrink: 0,
                fontSize: 10,
                // color: 'text.primary',
                color: 'white',
                background: 'linear-gradient(135deg, #a8edea 0%, #fed6e3 100%)',
                px: 1,
                mr: 1,
                fontWeight: '500',
                borderRadius: '4px',
                height: '14px',
                lineHeight: '14px',
                display: 'inline-block',
              }}
            >
              MD
            </Box>
          )}
          {item.type === 1 && renderAdd(item.id)}
        </Stack>
        {item.children &&
          item.children.length > 0 &&
          expandedFolders.has(item.id) && (
            <Stack>{renderTree(item.children, 2.5, depth + 1)}</Stack>
          )}
      </Stack>
    ));
  };

  useEffect(() => {
    if (curNode.id) {
      setData(prev => {
        let next = setProperty(prev, curNode.id!, 'name', val =>
          curNode.name !== undefined ? curNode.name : (val as string),
        ) as ITreeItem[];
        next = setProperty(next, curNode.id!, 'emoji', val =>
          curNode.meta?.emoji !== undefined
            ? curNode.meta.emoji
            : (val as string | undefined),
        ) as ITreeItem[];
        return [...next];
      });
    }
  }, [curNode]);

  useEffect(() => {
    getCatalogData();
  }, [kb_id]);

  return (
    <Stack
      sx={{
        bgcolor: 'background.paper3',
        height: '100%',
        color: 'text.primary',
      }}
    >
      <Stack
        direction='row'
        justifyContent='space-between'
        sx={{ p: 2 }}
        gap={1}
      >
        <KBSwitch />
        <Stack
          alignItems='center'
          justifyContent='space-between'
          onClick={() => setCatalogOpen(false)}
          sx={{
            cursor: 'pointer',
            color: 'text.tertiary',
            ':hover': {
              color: 'text.primary',
            },
          }}
        >
          <IconMulushouqi
            sx={{
              fontSize: 24,
            }}
          />
        </Stack>
      </Stack>
      <Stack
        direction={'row'}
        alignItems={'center'}
        justifyContent={'space-between'}
        sx={{ pr: 1 }}
      >
        <Box
          sx={{
            px: 2,
            fontSize: 14,
            fontWeight: 'bold',
            color: 'text.tertiary',
          }}
        >
          目录
        </Box>
        {renderAdd('')}
      </Stack>
      <Stack
        sx={{
          my: 1,
          px: 1,
          fontSize: 14,
          overflowY: 'auto',
          overflowX: 'hidden',
        }}
      >
        {renderTree(data)}
      </Stack>
      <DocAddByCustomText
        type={docFileKey}
        autoJump={false}
        open={customDocOpen}
        parentId={opraParentId}
        onCreated={node => {
          onRefresh();
          if (opraParentId) {
            setExpandedFolders(prev => {
              const ns = new Set(prev);
              if (opraParentId) ns.add(opraParentId);
              return ns;
            });
          }
        }}
        onClose={() => setCustomDocOpen(false)}
      />
    </Stack>
  );
};

export default Catalog;

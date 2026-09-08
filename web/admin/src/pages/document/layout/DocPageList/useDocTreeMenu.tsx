import {
  TreeMenuItem,
  TreeMenuOptions,
} from '@/components/Drag/DragTree/TreeMenu';
import { ConstsNodeRagInfoStatus } from '@/request/types';
import { useCallback } from 'react';
import type { DocTreeMenuHandlers } from './types';

export function useDocTreeMenu(
  handlers: DocTreeMenuHandlers,
): (opra: TreeMenuOptions) => TreeMenuItem[] {
  const {
    handleDelete,
    handlePublish,
    handleRestudy,
    handleProperties,
    handleFrontDoc,
  } = handlers;

  return useCallback(
    (opra: TreeMenuOptions): TreeMenuItem[] => {
      const { item, createItem, renameItem, isEditing } = opra;
      const menuItems: TreeMenuItem[] = [];

      if (item.type === 1) {
        menuItems.push(
          { label: '创建文件夹', key: 'folder', onClick: () => createItem(1) },
          {
            label: '创建文档',
            key: 'doc',
            children: [
              {
                label: '创建富文本',
                key: 'rich_text',
                onClick: () => createItem(2, 'html'),
              },
              {
                label: '创建 Markdown',
                key: 'md',
                onClick: () => createItem(2, 'md'),
              },
            ],
          },
        );
      }

      if (item.type === 2) {
        if (item.status === 1 || item.status === 0) {
          menuItems.push({
            label: item.status === 1 ? '发布更新' : '发布文档',
            key: 'update_publish',
            onClick: () => handlePublish(item),
          });
        }
        if (item.status !== 0) {
          menuItems.push({
            label:
              item.rag_status === ConstsNodeRagInfoStatus.NodeRagStatusPending
                ? '学习文档'
                : '重新学习',
            key: 'restudy',
            onClick: () => handleRestudy(item),
          });
        }
        menuItems.push(
          {
            label: '文档属性',
            key: 'properties',
            onClick: () => handleProperties(item),
          },
          {
            label: '前台查看',
            key: 'front_doc',
            onClick: () => handleFrontDoc(item.id),
          },
        );
      }

      if (!isEditing) {
        menuItems.push({ label: '重命名', key: 'rename', onClick: renameItem });
      }
      menuItems.push({
        label: '删除',
        color: 'error',
        key: 'delete',
        onClick: () => handleDelete(item),
      });

      return menuItems;
    },
    [
      handleDelete,
      handlePublish,
      handleRestudy,
      handleProperties,
      handleFrontDoc,
    ],
  );
}

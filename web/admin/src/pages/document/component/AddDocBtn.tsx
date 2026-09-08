import TreeMenu, { TreeMenuItem } from '@/components/Drag/DragTree/TreeMenu';
import { Box, Button } from '@mui/material';
import { useState } from 'react';
import DocAddByCustomText from './DocAddByCustomText';

interface InputContentProps {
  refresh?: () => void;
  disabled?: boolean;
  context?: React.ReactElement<{ onClick?: any; 'aria-describedby'?: any }>;
  createLocal?: (node: {
    id: string;
    name: string;
    type: 1 | 2;
    emoji?: string;
    parentId?: string | null;
    content_type?: string;
  }) => void;
  scrollTo?: (id: string) => void;
}

const AddDocBtn = ({
  refresh,
  disabled = false,
  context,
  createLocal,
  scrollTo,
}: InputContentProps) => {
  const [customDocOpen, setCustomDocOpen] = useState(false);
  const [docFileKey, setDocFileKey] = useState<1 | 2>(1);

  const menuItems: TreeMenuItem[] = [
    {
      key: 'docFile',
      label: '创建文件夹',
      onClick: () => {
        setDocFileKey(1);
        setCustomDocOpen(true);
      },
    },
    {
      key: 'next-line',
      label: '创建文档',
      onClick: () => {
        setDocFileKey(2);
        setCustomDocOpen(true);
      },
    },
  ];

  return (
    <Box>
      <TreeMenu
        menu={menuItems}
        context={
          context || (
            <Button variant='contained' disabled={disabled}>
              创建文档
            </Button>
          )
        }
      />
      <DocAddByCustomText
        type={docFileKey}
        open={customDocOpen}
        refresh={refresh}
        onCreated={node => {
          createLocal?.(node);
          scrollTo?.(node.id);
        }}
        onClose={() => setCustomDocOpen(false)}
      />
    </Box>
  );
};

export default AddDocBtn;
